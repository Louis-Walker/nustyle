package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"

	"github.com/zmb3/spotify/v2"
)

var (
	pathToDB, redirectURL, userID string
	playlist                      *Playlist
	artistsDB                     *sql.DB
	pid                           spotify.ID
)

// pathToDB = "./artistdb/artists.db"
// redirectURL = "http://quiet-reaches-27997.herokuapp.com/auth/callback"
// pid = "0TdRzSP9GMdDcnuZd7wSTE"

func main() {
	var err error
	pathToDB = os.Getenv("PATH_TO_DB")
	redirectURL = os.Getenv("REDIRECT_URL")
	pid = spotify.ID(os.Getenv("PLAYLIST_ID"))
	userID = "m05hi"

	go server()

	// Connections
	artistsDB = OpenArtistDB(pathToDB)
	playlist, err = NewPlaylist(context.Background(), redirectURL, pid)
	if err != nil {
		cLog("main", err)
	}

	// Main playlist crawler
	go playlist.Playlister()

	if !(isProd()) {
		cmd := exec.Command("cmd", "/c", "start", "http://localhost:8080")
		cmd.Start()
	}
}

func server() {
	http.HandleFunc("/", handleRoot)

	port := os.Getenv("PORT")
	if port == "" {
		log.Fatalf("No PORT specified!")
	}

	fmt.Println("Listening on port: " + port)
	err := http.ListenAndServe(fmt.Sprintf(":%v", port), nil)
	if err != nil {
		cLog("main/server", err)
	}
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "<p>Sign into <a href='auth'>Spotify</a></p>")
}

func isProd() bool {
	if len(os.Args) > 1 {
		return true
	} else {
		return false
	}
}
