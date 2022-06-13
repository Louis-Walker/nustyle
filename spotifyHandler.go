package main

import (
	"fmt"
	"log"
	"net/http"
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
		spotifyauth.WithClientID(SPOTIFY_ID),
		spotifyauth.WithClientSecret(SPOTIFY_SECRET),
	)
	ch    = make(chan *spotify.Client)
	state = "abc123"
)

func initSpotifyClient() *spotify.Client {
	http.HandleFunc("/auth", completeAuth)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Got req for:", r.URL.String())
	})
	go func() {
		err := http.ListenAndServe(":8080", nil)
		printError(err)
	}()

	url := auth.AuthURL(state)
	fmt.Println("Please log in to Spotify by visiting the following page in your browser:", url)

	client := <-ch

	user, err := client.CurrentUser(ctx)
	printError(err)
	fmt.Println("You are logged in as:", user.ID)

	return client
}

func getNewestTracks(a m.Artist) []spotify.ID {
	var SUI spotify.ID = spotify.ID(a.SUI)

	albums, err := spo.GetArtistAlbums(ctx, SUI, []spotify.AlbumType{1, 2})
	printError(err)

	var newTracks []spotify.ID

	for _, album := range albums.Albums[:4] {
		if album.ReleaseDateTime().After(a.LastTrackDateTime) {
			tracks, err := spo.GetAlbumTracks(ctx, album.ID)
			printError(err)

			for _, track := range tracks.Tracks {
				newTracks = append(newTracks, track.ID)
			}
		}
	}

	return newTracks
}

func updatePlaylist() {
	playlist, err := spo.GetPlaylist(ctx, PLAYLIST_ID)
	printError(err)
	oldName := playlist.Name

	// CHANGE NAME
	_, nowMonth, nowDay := time.Now().Date()
	newName := fmt.Sprintf("Nustyle %v/%v", nowDay, int(nowMonth))

	err = spo.ChangePlaylistName(ctx, PLAYLIST_ID, newName)
	printError(err)

	// COPY TO NEW PLAYLIST
	var s *spotify.FullPlaylist
	s, err = spo.CreatePlaylistForUser(ctx, USER_ID, oldName, "", false, false)
	printError(err)

	var tracks *spotify.PlaylistItemPage
	tracks, err = spo.GetPlaylistItems(ctx, PLAYLIST_ID)
	printError(err)

	var trackIDs []spotify.ID
	for i := 0; i < tracks.Total; i++ {
		tID := tracks.Items[i].Track.Track.ID
		trackIDs = append(trackIDs, tID)
	}
	spo.AddTracksToPlaylist(ctx, s.ID, trackIDs...)

	//CLEAN MAIN PLAYLIST
	_, err = spo.RemoveTracksFromPlaylist(ctx, PLAYLIST_ID, trackIDs...)
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
	client := spotify.New(auth.Client(r.Context(), tok))
	fmt.Fprintf(w, "Login Completed!")
	ch <- client
}
