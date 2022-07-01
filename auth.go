package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
)

type Auth struct {
	client *spotifyauth.Authenticator
	state  string
	ch     chan *spotify.Client
	URL    string
}

func NewAuth(redirectURL string) *Auth {
	c := spotifyauth.New(
		spotifyauth.WithRedirectURL(redirectURL),
		spotifyauth.WithScopes(
			spotifyauth.ScopeUserReadPrivate,
			spotifyauth.ScopePlaylistModifyPublic,
			spotifyauth.ScopePlaylistModifyPrivate,
		),
		spotifyauth.WithClientID(os.Getenv("SPOTIFY_ID")),
		spotifyauth.WithClientSecret(os.Getenv("SPOTIFY_SECRET")),
	)

	a := &Auth{
		client: c,
		state:  fmt.Sprint(time.Now().Unix() * rand.Int63()),
		ch:     make(chan *spotify.Client),
	}
	a.URL = a.client.AuthURL(a.state)

	return a
}

func handleAuth(w http.ResponseWriter, r *http.Request, authURL string) {
	http.Redirect(w, r, authURL, http.StatusFound)
}

func completeAuth(w http.ResponseWriter, r *http.Request, auth *Auth) {
	ctx := context.Background()

	tok, err := auth.client.Token(ctx, auth.state, r)
	if err != nil {
		http.Error(w, "Couldn't get token", http.StatusForbidden)
		log.Fatal(err)
	}

	if st := r.FormValue("state"); st != auth.state {
		http.NotFound(w, r)
		log.Fatalf("State mismatch: %s != %s\n", st, auth.state)
	}

	client := spotify.New(auth.client.Client(ctx, tok), spotify.WithRetry(true))
	fmt.Fprintf(w, "Login Completed!")
	auth.ch <- client
}
