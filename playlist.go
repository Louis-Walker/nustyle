package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/zmb3/spotify/v2"
)

type Playlist struct {
	ID     spotify.ID
	Tracks []spotify.PlaylistTrack
}

func NewPlaylist(id spotify.ID) (*Playlist, error) {
	var p = &Playlist{
		ID: id,
	}

	return p, err
}

func (p *Playlist) Playlister(db *sql.DB, spo *spotify.Client) {
	for {
		artists := GetAllArtists(artistsDB)
		artistsUpdated := 0
		ctx := context.Background()

		ptp, err := spo.GetPlaylistTracks(ctx, p.ID)
		if err != nil {
			logger("Playlist/Playlister", err)
		}
		p.Tracks = ptp.Tracks

		p.weeklyUpdater(ctx, spo)

		fmt.Println("[NU] Initiating Release Crawler")
		for _, a := range artists {
			trackIDs, trackArtists := p.getNewestTracks(ctx, db, spo, a)

			if len(trackIDs) > 0 {
				_, err := spo.AddTracksToPlaylist(ctx, p.ID, trackIDs...)
				if err != nil {
					logger("Playlist/Playlister", err)
				}

				UpdateLastTrack(artistsDB, trackArtists)
				fmt.Printf("%v | Updated %v tracks\n", a.Name, len(trackIDs))
				artistsUpdated += 1
			} else {
				fmt.Println(a.Name)
			}
			// time.Sleep(time.Second / 3)
		}

		fmt.Printf("[NU] Crawl Completed At %v - %v/%v Artists Updated\n", time.Now().Format("06-01-02 15:04:05"), artistsUpdated, len(artists))
		defer ctx.Done()
		time.Sleep(time.Minute * 28)
	}
}

func (p *Playlist) getNewestTracks(ctx context.Context, db *sql.DB, spo *spotify.Client, a Artist) (newTracks, artists []spotify.ID) {
	albums, err := spo.GetArtistAlbums(ctx, spotify.ID(a.SUI), []spotify.AlbumType{1, 2})
	if err != nil {
		logger("Playlist/GetNewestTracks", err)
		return
	} else {
		albumCounter := 0
		for _, album := range albums.Albums {
			// Limit amount of checks but theres 3 album types and type: album is first.
			// Fixes singles not getting checked if artist has more than 4 albums but also has a new single.
			if albumCounter >= 4 && album.AlbumType == "album" {
				continue
			}

			if isNew(album.ReleaseDateTime(), a.LastTrackDateTime) && albumCounter <= 8 {
				tracks, err := spo.GetAlbumTracks(ctx, album.ID)
				if err != nil {
					logger("Playlist/GetNewestTracks", err)
				}

				for _, track := range tracks.Tracks {
					if !isAdded(p.Tracks, track.ID) {
						if !isExtended(track.Name) && isLengthy(track.Duration) && !(isSimilarName(p.Tracks, track.Name, track.ID)) {
							newTracks = append(newTracks, track.ID)

							// Add track to p.Tracks instead of calling API again to refresh slice
							p.Tracks = append(p.Tracks, spotify.PlaylistTrack{
								Track: spotify.FullTrack{
									SimpleTrack: spotify.SimpleTrack{
										ID:   track.ID,
										Name: track.Name,
									},
								},
							})

							for _, a := range track.Artists {
								artists = append(artists, a.ID)
							}
						} else {
							var artists []string
							for _, a := range track.Artists {
								artists = append(artists, fmt.Sprintf("%v", a.Name))
							}

							t, err := spo.GetTrack(ctx, track.ID)
							if err != nil {
								logger("Playlist/GetNewestTracks", err)
							}
							imgs := t.Album.Images

							InsertTrackReview(db, TrackReview{
								Name:      track.Name,
								SUI:       track.ID,
								Artists:   artists,
								ImageURL:  imgs[len(imgs)-2].URL,
								DateAdded: DateTimeFormat(time.Now()),
								Status:    1,
							})
						}
					}
				}

				albumCounter++
			}
		}

		return
	}
}

func (p *Playlist) weeklyUpdater(ctx context.Context, spo *spotify.Client) {
	// Only updates playlist if its past 5pm on monday
	if int(time.Now().Weekday()) == 1 && time.Now().Hour() >= 17 && len(p.Tracks) > 16 {
		playlist, err := spo.GetPlaylist(ctx, p.ID)
		if err != nil {
			logger("Playlist/UpdatePlaylist", err)
		}
		oldName := playlist.Name

		// CHANGE NAME
		nowYear, nowMonth, nowDay := time.Now().Date()
		newName := fmt.Sprintf("Nustyle %v/%v/%v", nowDay, int(nowMonth), fmt.Sprint(nowYear)[2:4])

		err = spo.ChangePlaylistName(ctx, playlistID, newName)
		if err != nil {
			logger("Playlist/UpdatePlaylist", err)
		}

		// COPY TO NEW PLAYLIST
		var fp *spotify.FullPlaylist
		fp, err = spo.CreatePlaylistForUser(ctx, userID, oldName, "", false, false)
		if err != nil {
			logger("Playlist/UpdatePlaylist", err)
		}

		var tracks *spotify.PlaylistItemPage
		tracks, err = spo.GetPlaylistItems(ctx, playlistID)
		if err != nil {
			logger("Playlist/UpdatePlaylist", err)
		}

		var trackIDs []spotify.ID
		for i := 0; i < tracks.Total; i++ {
			tID := tracks.Items[i].Track.Track.ID
			trackIDs = append(trackIDs, tID)
		}
		spo.AddTracksToPlaylist(ctx, fp.ID, trackIDs...)

		//CLEAN MAIN PLAYLIST
		_, err = spo.RemoveTracksFromPlaylist(ctx, playlistID, trackIDs...)
		if err != nil {
			logger("Playlist/UpdatePlaylist", err)
		}

		fmt.Printf("[NU] New Playlist Created - Main Playlist Cleared\n")
	}
}

// Helper Functions
func isAdded(tracks []spotify.PlaylistTrack, id spotify.ID) bool {
	a := false
	for i := 0; i < len(tracks); i++ {
		t := tracks[i].Track
		//&& || t.Album.ReleaseDateTime().Before(time.Now().Truncate(time.Hour*24).Add(-time.Minute*30))
		// Work around for still adding track released today without refreshing playlist for weeklyUpdater
		if t.ID == id {
			a = true
		}
	}
	return a
}

func isNew(tDate, lDate time.Time) bool {
	e := false
	lElapsed := time.Now().Truncate(time.Hour * 24).Add(-(time.Minute * 30))
	if tDate.After(lElapsed) {
		e = true
	}
	return e
}

func isExtended(t string) bool {
	e := false
	if strings.Contains(t, "Extended") || strings.Contains(t, "extended") {
		e = true
	}
	return e
}

func isLengthy(t int) bool {
	l := false
	if (t > 90*1000) && (t < 320*1000) {
		l = true
	}
	return l
}

func isSimilarName(tracks []spotify.PlaylistTrack, name string, sui spotify.ID) bool {
	s := false
	for i := 0; i < len(tracks); i++ {
		t := tracks[i].Track

		if t.Name == name {
			s = true
		}
	}
	return s
}
