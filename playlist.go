package main

import (
	"context"
	"fmt"
	"log"
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
	Tracks      *spotify.PlaylistTrackPage
}

func NewPlaylist(redirectURL string, id spotify.ID) (*Playlist, error) {
	var p = &Playlist{
		ID:          id,
		auth:        newAuth(redirectURL),
		ch:          make(chan *spotify.Client),
		state:       "abc123",
		redirectURL: redirectURL,
	}

	ctx := context.Background()

	http.HandleFunc("/auth", func(w http.ResponseWriter, r *http.Request) { handleAuth(w, r, *p) })
	http.HandleFunc("/auth/callback", func(w http.ResponseWriter, r *http.Request) { completeAuth(w, r, *p) })

	p.url = p.auth.AuthURL(p.state)

	client := <-p.ch

	user, err := client.CurrentUser(ctx)
	if err != nil {
		cLog("Playlist/New", err)
	}

	fmt.Println("You are logged in as:", user.ID)
	p.Client = client
	defer ctx.Done()
	return p, err
}

func Playlister(p *Playlist) {
	spo := p.Client // Easier short hand

	for {
		fmt.Println("[NU] Initiating Release Crawler")
		artists := GetAllArtists(artistsDB)
		artistsUpdated := 0
		ctx := context.Background()

		var err error
		playlist.Tracks, err = spo.GetPlaylistTracks(ctx, p.ID)
		if err != nil {
			cLog("Playlist/Playlister", err)
		}

		for _, artist := range artists {
			trackIDs := p.getNewestTracks(ctx, artist, p.Tracks)

			if len(trackIDs) > 0 {
				_, err := spo.AddTracksToPlaylist(ctx, p.ID, trackIDs...)
				if err != nil {
					cLog("Playlist/Playlister", err)
				}

				UpdateLastTrack(artistsDB, artist.SUI)
				fmt.Printf("%v | Updated %v tracks\n", artist.Name, len(trackIDs))
				artistsUpdated += 1
			} else {
				fmt.Println(artist.Name)
			}
			// time.Sleep(time.Second / 3)
		}

		fmt.Printf("[NU] Crawl Completed At %v - %v/%v Artists Updated\n", time.Now().Format("06-01-02 15:04:05"), artistsUpdated, len(artists))
		weeklyUpdater(ctx)
		defer ctx.Done()
		time.Sleep(time.Minute * 30)
	}
}

func (p *Playlist) getNewestTracks(ctx context.Context, a Artist, t *spotify.PlaylistTrackPage) []spotify.ID {
	newTracks := []spotify.ID{}

	albums, err := p.Client.GetArtistAlbums(ctx, spotify.ID(a.SUI), []spotify.AlbumType{1, 2})
	if err != nil {
		cLog("Playlist/GetNewestTracks", err)
		return newTracks
	} else {
		albumCounter := 0
		for _, album := range albums.Albums {
			albumCounter++

			// Limit amount of checks but theres 3 album types and type: album is first.
			// Fixes singles not getting checked if artist has more than 4 albums but also has a new single.
			if albumCounter >= 4 && album.AlbumType == "album" {
				continue
			}

			if isNew(album.ReleaseDateTime(), a.LastTrackDateTime) {
				tracks, err := p.Client.GetAlbumTracks(ctx, album.ID)
				if err != nil {
					cLog("Playlist/GetNewestTracks", err)
				}

				for _, track := range tracks.Tracks {
					if !isAdded(t, track.ID, track.Name) && !isExtended(track.Name) {
						newTracks = append(newTracks, track.ID)
					}
				}
			}
		}

		return newTracks
	}
}

func (p *Playlist) updatePlaylist(ctx context.Context, pid spotify.ID, uid string) {
	spo := p.Client

	playlist, err := spo.GetPlaylist(ctx, pid)
	if err != nil {
		cLog("Playlist/UpdatePlaylist", err)
	}
	oldName := playlist.Name

	// CHANGE NAME
	_, nowMonth, nowDay := time.Now().Date()
	newName := fmt.Sprintf("Nustyle %v/%v", nowDay, int(nowMonth))

	err = spo.ChangePlaylistName(ctx, pid, newName)
	if err != nil {
		cLog("Playlist/UpdatePlaylist", err)
	}

	// COPY TO NEW PLAYLIST
	var fp *spotify.FullPlaylist
	fp, err = spo.CreatePlaylistForUser(ctx, uid, oldName, "", false, false)
	if err != nil {
		cLog("Playlist/UpdatePlaylist", err)
	}

	var tracks *spotify.PlaylistItemPage
	tracks, err = spo.GetPlaylistItems(ctx, pid)
	if err != nil {
		cLog("Playlist/UpdatePlaylist", err)
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
		cLog("Playlist/UpdatePlaylist", err)
	}
}

func weeklyUpdater(ctx context.Context) {
	// Only updates playlist if its past 5pm on monday
	if int(time.Now().Weekday()) == 1 && time.Now().Hour() > 17 && len(playlist.Tracks.Tracks) > 20 {
		playlist.updatePlaylist(context.Background(), playlist.ID, userID)
		fmt.Printf("[NU] New Playlist Created - Main Playlist Cleared\n")
	}
}

func isAdded(tracks *spotify.PlaylistTrackPage, id spotify.ID, name string) bool {
	for i := 0; i < tracks.Total; i++ {
		t := tracks.Tracks[i].Track
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

func handleAuth(w http.ResponseWriter, r *http.Request, s Playlist) {
	http.Redirect(w, r, s.url, http.StatusFound)
}

func completeAuth(w http.ResponseWriter, r *http.Request, s Playlist) {
	tok, err := s.auth.Token(r.Context(), s.state, r)
	if err != nil {
		http.Error(w, "Couldn't get token", http.StatusForbidden)
		log.Fatal(err)
	}
	if st := r.FormValue("state"); st != s.state {
		http.NotFound(w, r)
		log.Fatalf("State mismatch: %s != %s\n", st, s.state)
	}

	// use the token to get an authenticated client
	client := spotify.New(s.auth.Client(r.Context(), tok), spotify.WithRetry(true))
	fmt.Fprintf(w, "Login Completed!")
	s.ch <- client
}
