package spotifyService

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"

	logger "example.com/nustyle/logger"
	m "example.com/nustyle/model"
)

type SpotifyService struct {
	auth         *spotifyauth.Authenticator
	Client       *spotify.Client
	ch           chan *spotify.Client
	state        string
	REDIRECT_URL string
	ctx          context.Context
}

func New(redirectURL string) (*SpotifyService, error) {
	var s = &SpotifyService{
		auth:         newAuth(redirectURL),
		ch:           make(chan *spotify.Client),
		state:        "abc123",
		REDIRECT_URL: redirectURL,
		ctx:          context.Background(),
	}

	http.HandleFunc("/auth", func(w http.ResponseWriter, r *http.Request) { completeAuth(w, r, *s) })
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {})

	go func() {
		err := http.ListenAndServe(":8080", nil)
		logger.Psave("initSpotifyClient", err)
	}()

	url := s.auth.AuthURL(s.state)
	fmt.Println("Login URL:\n", url)

	client := <-s.ch

	user, err := client.CurrentUser(s.ctx)
	logger.Psave("initSpotifyClient", err)
	fmt.Println("You are logged in as:", user.ID)

	s.Client = client

	return s, err
}

func (s *SpotifyService) GetNewestTracks(a m.Artist, t *spotify.PlaylistTrackPage) []spotify.ID {
	var newTracks []spotify.ID

	albums, err := s.Client.GetArtistAlbums(s.ctx, spotify.ID(a.SUI), []spotify.AlbumType{1, 2})
	logger.Psave("getNewestTracks", err)

	if albums.Albums == nil {
		return newTracks
	}

	isNew := func(tDate time.Time, lDate time.Time) bool {
		lElapsed := lDate.Truncate(time.Hour * 24).Add(-(time.Minute * 1))

		if tDate.After(lElapsed) {
			return true
		}

		return false
	}

	for i, album := range albums.Albums {
		// Limit amount of checks... don't need to check whole library
		if i >= 4 {
			break
		}

		if isNew(album.ReleaseDateTime(), a.LastTrackDateTime) {
			tracks, err := s.Client.GetAlbumTracks(s.ctx, album.ID)
			logger.Psave("getNewestTracks", err)

			for _, track := range tracks.Tracks {
				if !trackAdded(t, track.ID) {
					newTracks = append(newTracks, track.ID)
				}
			}
		}
	}

	return newTracks
}

func (s *SpotifyService) UpdatePlaylist(pid spotify.ID, uid string) {
	c := s.Client

	playlist, err := c.GetPlaylist(s.ctx, pid)
	logger.Psave("updatePlaylist", err)
	oldName := playlist.Name

	// CHANGE NAME
	_, nowMonth, nowDay := time.Now().Date()
	newName := fmt.Sprintf("Nustyle %v/%v [DEV]", nowDay, int(nowMonth))

	err = c.ChangePlaylistName(s.ctx, pid, newName)
	logger.Psave("updatePlaylist", err)

	// COPY TO NEW PLAYLIST
	var fp *spotify.FullPlaylist
	fp, err = c.CreatePlaylistForUser(s.ctx, uid, oldName, "", false, false)
	logger.Psave("updatePlaylist", err)

	var tracks *spotify.PlaylistItemPage
	tracks, err = c.GetPlaylistItems(s.ctx, pid)
	logger.Psave("updatePlaylist", err)

	var trackIDs []spotify.ID
	for i := 0; i < tracks.Total; i++ {
		tID := tracks.Items[i].Track.Track.ID
		trackIDs = append(trackIDs, tID)
	}
	s.Client.AddTracksToPlaylist(s.ctx, fp.ID, trackIDs...)

	//CLEAN MAIN PLAYLIST
	_, err = s.Client.RemoveTracksFromPlaylist(s.ctx, pid, trackIDs...)
	logger.Psave("updatePlaylist", err)
}

func trackAdded(tracks *spotify.PlaylistTrackPage, id spotify.ID) bool {
	for i := 0; i < tracks.Total; i++ {
		if tracks.Tracks[i].Track.ID == id {
			return true
		}
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

func completeAuth(w http.ResponseWriter, r *http.Request, s SpotifyService) {
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
