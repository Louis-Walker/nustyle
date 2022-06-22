package playlist

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

	logger "example.com/nustyle/logger"
	m "example.com/nustyle/model"
)

type Service struct {
	auth        *spotifyauth.Authenticator
	Client      *spotify.Client
	ch          chan *spotify.Client
	state       string
	redirectUrl string
}

func New(ctx context.Context, redirectURL string) (*Service, error) {
	var s = &Service{
		auth:        newAuth(redirectURL),
		ch:          make(chan *spotify.Client),
		state:       "abc123",
		redirectUrl: redirectURL,
	}

	http.HandleFunc("/auth", func(w http.ResponseWriter, r *http.Request) { completeAuth(w, r, *s) })
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {})

	go func() {
		err := http.ListenAndServe(":8080", nil)
		logger.Psave("spotifyService/New", err)
	}()

	url := s.auth.AuthURL(s.state)
	fmt.Println("Login URL:\n", url)

	client := <-s.ch

	user, err := client.CurrentUser(ctx)
	logger.Psave("spotifyService/New", err)
	fmt.Println("You are logged in as:", user.ID)

	s.Client = client

	return s, err
}

func (s *Service) GetNewestTracks(ctx context.Context, a m.Artist, t *spotify.PlaylistTrackPage) []spotify.ID {
	var newTracks []spotify.ID = []spotify.ID{}

	albums, err := s.Client.GetArtistAlbums(ctx, spotify.ID(a.SUI), []spotify.AlbumType{1, 2})
	if err != nil {
		logger.Psave("GetNewestTracks", err)
		return newTracks
	} else {
		for i, album := range albums.Albums {
			// Limit amount of checks... don't need to check whole library
			if isNew(album.ReleaseDateTime(), a.LastTrackDateTime) && !(i >= 4) {
				tracks, err := s.Client.GetAlbumTracks(ctx, album.ID)
				logger.Psave("GetNewestTracks", err)

				for _, track := range tracks.Tracks {
					if !isAdded(t, track.ID, track.Name) && !isExtended(track.Name) {
						newTracks = append(newTracks, track.ID)
					}
				}
			} else {
				break
			}
		}

		return newTracks
	}
}

func (s *Service) UpdatePlaylist(ctx context.Context, pid spotify.ID, uid string) {
	c := s.Client

	playlist, err := c.GetPlaylist(ctx, pid)
	logger.Psave("UpdatePlaylist", err)
	oldName := playlist.Name

	// CHANGE NAME
	_, nowMonth, nowDay := time.Now().Date()
	newName := fmt.Sprintf("Nustyle %v/%v", nowDay, int(nowMonth))

	err = c.ChangePlaylistName(ctx, pid, newName)
	logger.Psave("UpdatePlaylist", err)

	// COPY TO NEW PLAYLIST
	var fp *spotify.FullPlaylist
	fp, err = c.CreatePlaylistForUser(ctx, uid, oldName, "", false, false)
	logger.Psave("UpdatePlaylist", err)

	var tracks *spotify.PlaylistItemPage
	tracks, err = c.GetPlaylistItems(ctx, pid)
	logger.Psave("UpdatePlaylist", err)

	var trackIDs []spotify.ID
	for i := 0; i < tracks.Total; i++ {
		tID := tracks.Items[i].Track.Track.ID
		trackIDs = append(trackIDs, tID)
	}
	s.Client.AddTracksToPlaylist(ctx, fp.ID, trackIDs...)

	//CLEAN MAIN PLAYLIST
	_, err = s.Client.RemoveTracksFromPlaylist(ctx, pid, trackIDs...)
	logger.Psave("UpdatePlaylist", err)
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

func completeAuth(w http.ResponseWriter, r *http.Request, s Service) {
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
