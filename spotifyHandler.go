package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"

	m "example.com/nustyle/model"
)

var (
	auth = spotifyauth.New(
		spotifyauth.WithRedirectURL(REDIRECT_URL),
		spotifyauth.WithScopes(
			spotifyauth.ScopeUserReadPrivate,
			spotifyauth.ScopePlaylistModifyPublic,
			spotifyauth.ScopePlaylistModifyPrivate,
		),
		spotifyauth.WithClientID(os.Getenv("SPOTIFY_ID")),
		spotifyauth.WithClientSecret(os.Getenv("SPOTIFY_SECRET")),
	)
	ch    = make(chan *spotify.Client)
	state = "abc123"
)

func initSpotifyClient() *spotify.Client {
	http.HandleFunc("/auth", completeAuth)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {})

	go func() {
		err := http.ListenAndServe(":8080", nil)
		printError(err)
	}()

	url := auth.AuthURL(state)
	fmt.Println("Please log in to Spotify by visiting the following page in your browser:", url)

	client := <-ch

	user, err := client.CurrentUser(context.Background())
	printError(err)
	fmt.Println("You are logged in as:", user.ID)

	return client
}

func getNewestTracks(a m.Artist) []spotify.ID {
	var SUI spotify.ID = spotify.ID(a.SUI)

	albums, err := spo.GetArtistAlbums(context.Background(), SUI, []spotify.AlbumType{1, 2})
	printError(err)

	var newTracks []spotify.ID

	if albums.Albums == nil {
		return newTracks
	}

	lastTrackBeforeDate := func() time.Time {
		n := time.Now()

		timeBuilder := fmt.Sprintf("%v-%v-%v %v:%v:%v+00:00", n.Year(), int(n.Month()), n.Day(), 0, 0, 0)

		newTime, err := time.Parse("2006-1-2 15:4:5+00:00", timeBuilder)
		if err != nil {
			printError(err)
		}

		newTime = newTime.Add(-time.Minute * 1)

		return newTime
	}()

	for _, album := range albums.Albums[:intLimiter(albums.Total)] {
		if album.ReleaseDateTime().After(lastTrackBeforeDate) {
			tracks, err := spo.GetAlbumTracks(context.Background(), album.ID)
			if err != nil {
				printError(err)
			}

			for _, track := range tracks.Tracks {
				if !trackAdded(playlistTracks, track.ID) {
					newTracks = append(newTracks, track.ID)
				}
			}
		}
	}

	return newTracks
}

func updatePlaylist() {
	playlist, err := spo.GetPlaylist(context.Background(), PLAYLIST_ID)
	printError(err)
	oldName := playlist.Name

	// CHANGE NAME
	_, nowMonth, nowDay := time.Now().Date()
	newName := fmt.Sprintf("Nustyle %v/%v [DEV]", nowDay, int(nowMonth))

	err = spo.ChangePlaylistName(context.Background(), PLAYLIST_ID, newName)
	printError(err)

	// COPY TO NEW PLAYLIST
	var s *spotify.FullPlaylist
	s, err = spo.CreatePlaylistForUser(context.Background(), USER_ID, oldName, "", false, false)
	printError(err)

	var tracks *spotify.PlaylistItemPage
	tracks, err = spo.GetPlaylistItems(context.Background(), PLAYLIST_ID)
	printError(err)

	var trackIDs []spotify.ID
	for i := 0; i < tracks.Total; i++ {
		tID := tracks.Items[i].Track.Track.ID
		trackIDs = append(trackIDs, tID)
	}
	spo.AddTracksToPlaylist(context.Background(), s.ID, trackIDs...)

	//CLEAN MAIN PLAYLIST
	_, err = spo.RemoveTracksFromPlaylist(context.Background(), PLAYLIST_ID, trackIDs...)
	printError(err)
}

func completeAuth(w http.ResponseWriter, r *http.Request) {
	tok, err := auth.Token(r.Context(), state, r)
	if err != nil {
		http.Error(w, "Couldn't get token", http.StatusForbidden)
		log.Fatal(err)
	}
	if st := r.FormValue("state"); st != state {
		http.NotFound(w, r)
		log.Fatalf("State mismatch: %s != %s\n", st, state)
	}

	// use the token to get an authenticated client
	client := spotify.New(auth.Client(r.Context(), tok), spotify.WithRetry(true))
	fmt.Fprintf(w, "Login Completed!")
	ch <- client
}

func trackAdded(tracks *spotify.PlaylistTrackPage, id spotify.ID) bool {
	added := false

	for i := 0; i < tracks.Total; i++ {
		if tracks.Tracks[i].Track.ID == id {
			added = true
		}
	}

	return added
}

func intLimiter(c int) int {
	if c < 4 {
		return c
	} else {
		return 4
	}
}
