package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
)

type Playlist struct {
	ID          spotify.ID
	auth        *spotifyauth.Authenticator
	Client      *spotify.Client
	ch          chan *spotify.Client
	state       string
	redirectURL string
	url         string
	Tracks      []spotify.PlaylistTrack
}

func NewPlaylist(redirectURL string, id spotify.ID) (*Playlist, error) {
	var p = &Playlist{
		ID:          id,
		auth:        newAuth(redirectURL),
		ch:          make(chan *spotify.Client),
		state:       fmt.Sprint(time.Now().Unix() * rand.Int63()),
		redirectURL: redirectURL,
	}

	ctx := context.Background()

	http.HandleFunc("/auth", func(w http.ResponseWriter, r *http.Request) { handleAuth(w, r, *p) })
	http.HandleFunc("/auth/callback", func(w http.ResponseWriter, r *http.Request) { completeAuth(w, r, *p) })

	p.url = p.auth.AuthURL(p.state)

	client := <-p.ch

	user, err := client.CurrentUser(ctx)
	if err != nil {
		logger("Playlist/New", err)
	}

	fmt.Println("You are logged in as:", user.ID)
	p.Client = client
	defer ctx.Done()
	return p, err
}

func (p *Playlist) Playlister() {
	spo := p.Client // Easier short hand

	for {
		artists := GetAllArtists(artistsDB)
		artistsUpdated := 0
		ctx := context.Background()

		p.weeklyUpdater(ctx)

		ptp, err := spo.GetPlaylistTracks(ctx, p.ID)
		if err != nil {
			logger("Playlist/Playlister", err)
		}
		p.Tracks = ptp.Tracks

		fmt.Println("[NU] Initiating Release Crawler")
		for _, artist := range artists {
			trackIDs, trackArtists := p.getNewestTracks(ctx, artist)

			if len(trackIDs) > 0 {
				_, err := spo.AddTracksToPlaylist(ctx, p.ID, trackIDs...)
				if err != nil {
					logger("Playlist/Playlister", err)
				}

				UpdateLastTrack(artistsDB, trackArtists)
				fmt.Printf("%v | Updated %v tracks\n", artist.Name, len(trackIDs))
				artistsUpdated += 1
			} else {
				fmt.Println(artist.Name)
			}
			// time.Sleep(time.Second / 3)
		}

		fmt.Printf("[NU] Crawl Completed At %v - %v/%v Artists Updated\n", time.Now().Format("06-01-02 15:04:05"), artistsUpdated, len(artists))
		defer ctx.Done()
		time.Sleep(time.Minute * 30)
	}
}

func (p *Playlist) getNewestTracks(ctx context.Context, a Artist) ([]spotify.ID, []spotify.ID) {
	newTracks := []spotify.ID{}
	artists := []spotify.ID{}

	albums, err := p.Client.GetArtistAlbums(ctx, spotify.ID(a.SUI), []spotify.AlbumType{1, 2})
	if err != nil {
		logger("Playlist/GetNewestTracks", err)
		return newTracks, artists
	} else {
		albumCounter := 0
		for _, album := range albums.Albums {
			// Limit amount of checks but theres 3 album types and type: album is first.
			// Fixes singles not getting checked if artist has more than 4 albums but also has a new single.
			if albumCounter >= 4 && album.AlbumType == "album" {
				continue
			}

			if isNew(album.ReleaseDateTime(), a.LastTrackDateTime) && albumCounter <= 8 {
				tracks, err := p.Client.GetAlbumTracks(ctx, album.ID)
				if err != nil {
					logger("Playlist/GetNewestTracks", err)
				}

				for _, track := range tracks.Tracks {
					if !isAdded(p.Tracks, track.ID, track.Name) && !isExtended(track.Name) {
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
					}
				}

				albumCounter++
			}
		}

		return newTracks, artists
	}
}

func (p *Playlist) weeklyUpdater(ctx context.Context) {
	// Only updates playlist if its past 5pm on monday
	if int(time.Now().Weekday()) == 1 && time.Now().Hour() >= 17 && len(p.Tracks) > 20 {
		p.updatePlaylist(ctx, userID)
		fmt.Printf("[NU] New Playlist Created - Main Playlist Cleared\n")
	}
}

func (p *Playlist) updatePlaylist(ctx context.Context, uid string) {
	spo := p.Client

	playlist, err := spo.GetPlaylist(ctx, p.ID)
	if err != nil {
		logger("Playlist/UpdatePlaylist", err)
	}
	oldName := playlist.Name

	// CHANGE NAME
	_, nowMonth, nowDay := time.Now().Date()
	newName := fmt.Sprintf("Nustyle %v/%v", nowDay, int(nowMonth))

	err = spo.ChangePlaylistName(ctx, pid, newName)
	if err != nil {
		logger("Playlist/UpdatePlaylist", err)
	}

	// COPY TO NEW PLAYLIST
	var fp *spotify.FullPlaylist
	fp, err = spo.CreatePlaylistForUser(ctx, uid, oldName, "", false, false)
	if err != nil {
		logger("Playlist/UpdatePlaylist", err)
	}

	var tracks *spotify.PlaylistItemPage
	tracks, err = spo.GetPlaylistItems(ctx, pid)
	if err != nil {
		logger("Playlist/UpdatePlaylist", err)
	}

	var trackIDs []spotify.ID
	for i := 0; i < tracks.Total; i++ {
		tID := tracks.Items[i].Track.Track.ID
		trackIDs = append(trackIDs, tID)
	}
	p.Client.AddTracksToPlaylist(ctx, fp.ID, trackIDs...)

	//CLEAN MAIN PLAYLIST
	_, err = p.Client.RemoveTracksFromPlaylist(ctx, pid, trackIDs...)
	if err != nil {
		logger("Playlist/UpdatePlaylist", err)
	}
}

func isAdded(tracks []spotify.PlaylistTrack, id spotify.ID, name string) bool {
	for i := 0; i < len(tracks); i++ {
		t := tracks[i].Track
		if t.ID == id || t.Name == name {
			return true
		}
	}

	return false
}

func isNew(tDate time.Time, lDate time.Time) bool {
	lElapsed := lDate.Truncate(time.Hour * 24).Add(-(time.Minute * 30))

	if tDate.After(lElapsed) {
		return true
	}

	return false
}

func isExtended(t string) bool {
	if strings.Contains(t, "Extended") || strings.Contains(t, "extended") {
		return true
	}

	return false
}

func newAuth(redirectURL string) *spotifyauth.Authenticator {
	return spotifyauth.New(
		spotifyauth.WithRedirectURL(redirectURL),
		spotifyauth.WithScopes(
			spotifyauth.ScopeUserReadPrivate,
			spotifyauth.ScopePlaylistModifyPublic,
			spotifyauth.ScopePlaylistModifyPrivate,
		),
		spotifyauth.WithClientID(os.Getenv("SPOTIFY_ID")),
		spotifyauth.WithClientSecret(os.Getenv("SPOTIFY_SECRET")),
	)
}

func handleAuth(w http.ResponseWriter, r *http.Request, p Playlist) {
	http.Redirect(w, r, p.url, http.StatusFound)
}

func completeAuth(w http.ResponseWriter, r *http.Request, p Playlist) {
	ctx := context.Background()
	tok, err := p.auth.Token(ctx, p.state, r)
	if err != nil {
		http.Error(w, "Couldn't get token", http.StatusForbidden)
		log.Fatal(err)
	}
	if st := r.FormValue("state"); st != p.state {
		http.NotFound(w, r)
		log.Fatalf("State mismatch: %s != %s\n", st, p.state)
	}

	// use the token to get an authenticated client
	client := spotify.New(p.auth.Client(ctx, tok), spotify.WithRetry(true))
	fmt.Fprintf(w, "Login Completed!")
	p.ch <- client
}
