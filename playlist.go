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

func NewPlaylist(ctx context.Context, redirectURL string, id spotify.ID) (*Playlist, error) {
	var s = &Playlist{
		ID:          id,
		auth:        newAuth(redirectURL),
		ch:          make(chan *spotify.Client),
		state:       "abc123",
		redirectURL: redirectURL,
	}

	http.HandleFunc("/", handleRoot)
	http.HandleFunc("/auth", func(w http.ResponseWriter, r *http.Request) { handleAuth(w, r, *s) })
	http.HandleFunc("/auth/callback", func(w http.ResponseWriter, r *http.Request) { completeAuth(w, r, *s) })

	go func() {
		port := os.Getenv("PORT")

		if port == "" {
			log.Fatalf("No PORT specified!")
		}

		fmt.Println("Listening on port: " + port)
		err := http.ListenAndServe(fmt.Sprintf(":%v", port), nil)
		if err != nil {
			cLog("spotifyService/New", err)
		}
	}()

	s.url = s.auth.AuthURL(s.state)

	client := <-s.ch

	user, err := client.CurrentUser(ctx)
	if err != nil {
		cLog("spotifyService/New", err)
	}

	fmt.Println("You are logged in as:", user.ID)
	s.Client = client
	return s, err
}

func (s *Playlist) GetNewestTracks(ctx context.Context, a Artist, t *spotify.PlaylistTrackPage) []spotify.ID {
	var newTracks []spotify.ID = []spotify.ID{}

	albums, err := s.Client.GetArtistAlbums(ctx, spotify.ID(a.SUI), []spotify.AlbumType{1, 2})
	if err != nil {
		cLog("GetNewestTracks", err)
		return newTracks
	} else {
		for i, album := range albums.Albums {
			if i >= 10 {
				break
			}

			// Limit amount of checks... don't need to check whole library
			if isNew(album.ReleaseDateTime(), a.LastTrackDateTime) {
				tracks, err := s.Client.GetAlbumTracks(ctx, album.ID)
				if err != nil {
					cLog("GetNewestTracks", err)
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

func (s *Playlist) UpdatePlaylist(ctx context.Context, pid spotify.ID, uid string) {
	c := s.Client

	playlist, err := c.GetPlaylist(ctx, pid)
	if err != nil {
		cLog("UpdatePlaylist", err)
	}
	oldName := playlist.Name

	// CHANGE NAME
	_, nowMonth, nowDay := time.Now().Date()
	newName := fmt.Sprintf("Nustyle %v/%v", nowDay, int(nowMonth))

	err = c.ChangePlaylistName(ctx, pid, newName)
	if err != nil {
		cLog("UpdatePlaylist", err)
	}

	// COPY TO NEW PLAYLIST
	var fp *spotify.FullPlaylist
	fp, err = c.CreatePlaylistForUser(ctx, uid, oldName, "", false, false)
	if err != nil {
		cLog("UpdatePlaylist", err)
	}

	var tracks *spotify.PlaylistItemPage
	tracks, err = c.GetPlaylistItems(ctx, pid)
	if err != nil {
		cLog("UpdatePlaylist", err)
	}

	var trackIDs []spotify.ID
	for i := 0; i < tracks.Total; i++ {
		tID := tracks.Items[i].Track.Track.ID
		trackIDs = append(trackIDs, tID)
	}
	s.Client.AddTracksToPlaylist(ctx, fp.ID, trackIDs...)

	//CLEAN MAIN PLAYLIST
	_, err = s.Client.RemoveTracksFromPlaylist(ctx, pid, trackIDs...)
	if err != nil {
		cLog("UpdatePlaylist", err)
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

func handleRoot(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "<p>Sign into <a href='auth'>Spotify</a></p>")
}

func handleAuth(w http.ResponseWriter, r *http.Request, s Playlist) {
	http.Redirect(w, r, s.url, http.StatusFound)
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
