package main

import (
	"bufio"
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

func main() {
	var err error
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
		cLog("main/main", err)
	}

	// Main playlist crawler
	go Playlister(playlist)

	// Keeps service open until command entered to terminate
	reader := bufio.NewReader(os.Stdin)
	txt, _ := reader.ReadString('\n')
	if txt == "terminate" {
		os.Exit(0)
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
