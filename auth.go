package main

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
)

type AuthSpotify struct {
	client *spotifyauth.Authenticator
	state  string
	ch     chan *spotify.Client
	URL    string
}

func NewAuthSpotify(redirectURL string) *AuthSpotify {
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

	a := &AuthSpotify{
		client: c,
		state:  fmt.Sprint(time.Now().Unix() * rand.Int63()),
		ch:     make(chan *spotify.Client),
	}
	a.URL = a.client.AuthURL(a.state)

	return a
}

func handleAuthSpotify(w http.ResponseWriter, r *http.Request, authURL string) {
	http.Redirect(w, r, authURL, http.StatusFound)
}

func completeAuthSpotify(w http.ResponseWriter, r *http.Request, auth *AuthSpotify) {
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
	http.Redirect(w, r, r.URL.Hostname()+"/dashboard", http.StatusFound)
	auth.ch <- client
}

func basicAuth(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if ok {
			usernameHash := sha256.Sum256([]byte(username))
			passwordHash := sha256.Sum256([]byte(password))
			expectedUsernameHash := sha256.Sum256([]byte(username))
			expectedPasswordHash := sha256.Sum256([]byte(password))

			usernameMatch := (subtle.ConstantTimeCompare(usernameHash[:], expectedUsernameHash[:]) == 1)
			passwordMatch := (subtle.ConstantTimeCompare(passwordHash[:], expectedPasswordHash[:]) == 1)

			if usernameMatch && passwordMatch {
				next.ServeHTTP(w, r)
				return
			}
		}

		w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}
