package main

import (
	"context"
	"database/sql"
	"fmt"
	"html/template"
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
	viewPath = "web/views/"
)

var (
	pathToDB, redirectURL, username, password string
	auth                                      *AuthSpotify
	client                                    *spotify.Client
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
	http.HandleFunc("/artist/add", addArtistBySUI)
	http.HandleFunc("/artist/remove", removeArtistBySUI)

	// Handle web resources
	cssFS := http.FileServer(http.Dir("./web/css"))
	http.Handle("/css/", http.StripPrefix("/css", cssFS))
	jsFS := http.FileServer(http.Dir("./web/js"))
	http.Handle("/js/", http.StripPrefix("/js", jsFS))

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

	// Local laziness
	if !(isProd()) {
		cmd := exec.Command("cmd", "/c", "start", localURL)
		cmd.Start()
	}

	// Spotify Authentication
	ctx := context.Background()
	client = <-auth.ch

	user, err := client.CurrentUser(ctx)
	if err != nil {
		logger("main/server", err)
	}

	fmt.Println("You are logged in as:", user.ID)
	defer ctx.Done()

	// Semi-hourly crawler for releases
	//go playlist.Playlister(client)

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

type DashboardData struct {
	PageTitle string
	Artists   []Artist
}

func handleDashboard(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles(viewPath + "dashboard.html"))
	data := DashboardData{
		PageTitle: "Dashboard",
		Artists:   GetAllArtists(artistsDB),
	}
	tmpl.Execute(w, data)
}

func addArtistBySUI(w http.ResponseWriter, r *http.Request) {
	SUI := spotify.ID(r.URL.Query().Get("sui"))
	a, err := client.GetArtist(context.Background(), SUI)
	if err != nil {
		logger("main/addArtistBySUI", err)
		http.Error(w, "Not a valid URI", http.StatusNotFound)
	} else {
		name := a.Name
		nowString := time.Now().Format("2006-01-02 15:04:05+00:00")
		now, _ := time.Parse("2006-01-02 15:04:05+00:00", nowString)

		err = AddArtist(artistsDB, Artist{name, SUI, now})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			fmt.Println(err.Error())
		}
	}
}

func removeArtistBySUI(w http.ResponseWriter, r *http.Request) {
	SUI := spotify.ID(r.URL.Query().Get("sui"))
	err := RemoveArtist(artistsDB, SUI)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func isProd() bool {
	return len(os.Args) > 1
}
