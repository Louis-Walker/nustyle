package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/zmb3/spotify/v2"
)

var (
	pathToDB, redirectURL, userID string
	playlist                      *Playlist
	artistsDB                     *sql.DB
	pid                           spotify.ID
	err                           error
)

func main() {
	pathToDB = os.Getenv("PATH_TO_DB")
	redirectURL = os.Getenv("REDIRECT_URL")
	pid = spotify.ID(os.Getenv("PLAYLIST_ID"))
	userID = "m05hi"

	go server()

	if !(isProd()) {
		cmd := exec.Command("cmd", "/c", "start", "http://localhost:8080")
		cmd.Start()
	}

	// Connections
	artistsDB = OpenArtistDB(pathToDB)
	playlist, err = NewPlaylist(redirectURL, pid)
	if err != nil {
		logger("main/main", err)
	}

	// Main playlist crawler
	go playlist.Playlister()

	exit := make(chan string)
	for {
		select {
		case <-exit:
			os.Exit(0)
		}
	}
}

func server() {
	http.HandleFunc("/", handleRoot)
	http.HandleFunc("/artist/add", addArtist)
	http.HandleFunc("/artist/remove", removeArtist)

	port := os.Getenv("PORT")
	if port == "" {
		log.Fatalf("No PORT specified!")
	}

	fmt.Println("Listening on port: " + port)
	err := http.ListenAndServe(fmt.Sprintf(":%v", port), nil)
	if err != nil {
		logger("main/server", err)
	}
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "<p>Sign into <a href='auth'>Spotify</a></p>")
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
