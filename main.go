package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/zmb3/spotify/v2"
)

const (
	userID   = "m05hi"
	localURL = "http://localhost:8080"
)

var (
	pathToDB, redirectURL, username, password string
	auth                                      *AuthSpotify
	playlist                                  *Playlist
	artistsDB                                 *sql.DB
	playlistID                                spotify.ID
	err                                       error
)

func main() {
	pathToDB = os.Getenv("PATH_TO_DB")
	redirectURL = os.Getenv("REDIRECT_URL")
	playlistID = spotify.ID(os.Getenv("PLAYLIST_ID"))
	username = os.Getenv("NU_USERNAME")
	password = os.Getenv("NU_PASSWORD")

	// Connections
	auth = NewAuthSpotify(redirectURL)
	artistsDB = OpenArtistDB(pathToDB)
	playlist, err = NewPlaylist(playlistID)
	if err != nil {
		logger("main/main", err)
	}

	// Routes
	http.HandleFunc("/", handleRoot)
	http.HandleFunc("/dashboard", basicAuth(handleDashboard))
	http.HandleFunc("/auth/spotify", func(w http.ResponseWriter, r *http.Request) {
		handleAuthSpotify(w, r, auth.URL)
	})
	http.HandleFunc("/auth/spotify/callback", func(w http.ResponseWriter, r *http.Request) {
		completeAuthSpotify(w, r, auth)
	})
	http.HandleFunc("/artist/add", addArtist)
	http.HandleFunc("/artist/remove", removeArtist)

	go func() {
		port := os.Getenv("PORT")
		if port == "" {
			log.Fatalf("No PORT specified!")
		}

		fmt.Println("Listening on port: " + port)
		err := http.ListenAndServe(fmt.Sprintf(":%v", port), nil)
		if err != nil {
			logger("main/server", err)
		}
	}()

	// Spotify Authentication
	ctx := context.Background()
	client := <-auth.ch

	user, err := client.CurrentUser(ctx)
	if err != nil {
		logger("main/server", err)
	}

	fmt.Println("You are logged in as:", user.ID)
	defer ctx.Done()

	// Semi-hourly crawler for releases
	go playlist.Playlister(client)

	// Local laziness
	if !(isProd()) {
		cmd := exec.Command("cmd", "/c", "start", localURL)
		cmd.Start()
	}

	exit := make(chan string)
	for {
		select {
		case <-exit:
			os.Exit(0)
		}
	}
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "<p>Sign into <a href='auth/spotify'>Spotify</a></p>")
}

func handleDashboard(w http.ResponseWriter, r *http.Request) {

}

func addArtist(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	SUI := spotify.ID(r.URL.Query().Get("sui"))
	AddArtist(artistsDB, Artist{name, SUI, time.Now()})
}

func removeArtist(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	SUI := spotify.ID(r.URL.Query().Get("sui"))
	RemoveArtist(artistsDB, Artist{name, SUI, time.Now()})
}

func isProd() bool {
	return len(os.Args) > 1
}
