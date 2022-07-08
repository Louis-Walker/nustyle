package main

import (
	"context"
	"database/sql"
	"encoding/json"
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
	userID    = "m05hi"
	localURL  = "http://localhost:8080"
	viewsPath = "web/views/"
)

var (
	pathToDB, redirectURL, expectedUsername, expectedPassword string
	authSpo                                                   *AuthSpotify
	client                                                    *spotify.Client
	playlist                                                  *Playlist
	artistsDB                                                 *sql.DB
	playlistID                                                spotify.ID
	err                                                       error
)

func main() {
	pathToDB = os.Getenv("PATH_TO_DB")
	redirectURL = os.Getenv("REDIRECT_URL")
	playlistID = spotify.ID(os.Getenv("PLAYLIST_ID"))
	expectedUsername = os.Getenv("NU_USERNAME")
	expectedPassword = os.Getenv("NU_PASSWORD")

	// Connections
	authSpo = NewAuthSpotify(redirectURL)
	artistsDB = OpenArtistDB(pathToDB)
	playlist, err = NewPlaylist(playlistID)
	if err != nil {
		logger("main/main", err)
	}

	// Routes
	http.HandleFunc("/", basicAuth(handleRoot))
	http.HandleFunc("/admin", basicAuth(handleAdmin))
	// Auths
	http.HandleFunc("/auth/spotify", func(w http.ResponseWriter, r *http.Request) {
		handleAuthSpotify(w, r, authSpo.URL)
	})
	http.HandleFunc("/auth/spotify/callback", func(w http.ResponseWriter, r *http.Request) {
		completeAuthSpotify(w, r, authSpo)
	})
	// Private API
	http.HandleFunc("/api/artist/add", basicAuth(addArtistBySUI))
	http.HandleFunc("/api/artist/remove", basicAuth(removeArtistBySUI))
	http.HandleFunc("/api/trackreview/add", basicAuth(handleAddTrackReview))
	http.HandleFunc("/api/trackreview/reviewed", basicAuth(HandleReviewedTrackReview))

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
	client = <-authSpo.ch

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

// View Controllers
func handleRoot(w http.ResponseWriter, r *http.Request) {
	type RootData struct {
		PageTitle string
	}

	tmpl, err := template.ParseFiles(layoutPaths("login")...)
	if err != nil {
		logger("main/handleRoot", err)
		http.Error(w, "Internal Server Error", 500)
	}

	err = tmpl.Execute(w, RootData{
		PageTitle: "Home",
	})
	if err != nil {
		logger("main/handleRoot", err)
		http.Error(w, "Internal Server Error", 500)
	}
}

func handleAdmin(w http.ResponseWriter, r *http.Request) {
	type AdminData struct {
		PageTitle    string
		Artists      []Artist
		TotalArtists int
	}

	tmpl, err := template.ParseFiles(layoutPaths("admin")...)
	if err != nil {
		logger("main/handleAdmin", err)
		http.Error(w, "Internal Server Error", 500)
	}

	a := GetAllArtists(artistsDB)
	err = tmpl.Execute(w, AdminData{
		PageTitle:    "Dashboard",
		Artists:      a,
		TotalArtists: len(a),
	})
	if err != nil {
		logger("main/handleAdmin", err)
		http.Error(w, "Internal Server Error", 500)
	}
}

// API Controllers
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

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		res := make(map[string]string)
		res["status"] = "200"
		res["name"] = name
		jsonRes, err := json.Marshal(res)
		if err != nil {
			logger("main/addArtistBySUI", err)
		}

		w.Write(jsonRes)
	}
}

func removeArtistBySUI(w http.ResponseWriter, r *http.Request) {
	SUI := spotify.ID(r.URL.Query().Get("sui"))
	err := RemoveArtist(artistsDB, SUI)
	if err != nil {
		logger("main/removeArtistBySUI", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handleAddTrackReview(w http.ResponseWriter, r *http.Request) {
	SUI := spotify.ID(r.URL.Query().Get("sui"))
	nowString := time.Now().Format("2006-01-02 15:04:05+00:00")
	now, _ := time.Parse("2006-01-02 15:04:05+00:00", nowString)

	n, err := client.GetTrack(context.Background(), SUI)
	if err != nil {
		logger("main/addTrackReview", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	err = AddTrackReview(artistsDB, TrackReview{
		Name:      n.Name,
		SUI:       SUI,
		DateAdded: now,
		Status:    1,
	})
	if err != nil {
		logger("main/addTrackReview", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handleReviewedTrackReview(w http.ResponseWriter, r *http.Request) {
	SUI := spotify.ID(r.URL.Query().Get("sui"))
	s := r.URL.Query().Get("status")
	var statusCode int

	switch s {
	case "approved":
		statusCode = 2
		w.WriteHeader(http.StatusOK)
	case "denied":
		statusCode = 3
		w.WriteHeader(http.StatusOK)
	default:
		statusCode = 1
		logger("main/handleReviewedTrackReview", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	err := UpdateTrackReviewStatus(artistsDB, SUI, statusCode)
	if err != nil {
		logger("main/handleReviewedTrackReview", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Helper Functions
func isProd() bool {
	return len(os.Args) > 1
}

func layoutPaths(viewName string) (p []string) {
	p = append(p, viewsPath+viewName+".html", viewsPath+"header.html", viewsPath+"footer.html")
	return
}
