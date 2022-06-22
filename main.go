package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/zmb3/spotify/v2"
)

type Nustyle struct {
	pathToDB, redirectURL, userID string
	playlistID                    spotify.ID
	playlist                      *Playlist
	artistsDB                     *sql.DB
	playlistTracks                *spotify.PlaylistTrackPage
}

func main() {
	var nu Nustyle = Nustyle{
		pathToDB:    "./artistdb/artistsDEV.db",
		playlistID:  "3uzLhwcuH1KpmeCPWMqnQl",
		redirectURL: "http://localhost:8080/auth/callback",
		userID:      "m05hi",
	}

	prodCheck(&nu) //Check environment

	// Connections
	var err error
	nu.artistsDB = OpenArtistDB(nu.pathToDB)
	nu.playlist, err = NewPlaylist(context.Background(), nu.redirectURL)
	cLog("main", err)

	spo := nu.playlist.Client // Easier short hand

	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("nustyle"),
		newrelic.ConfigLicense("eu01xx1a029386e43e45ccac269ae30a573eNRAL"),
		newrelic.ConfigDistributedTracerEnabled(true),
	)

	fmt.Println(app)

	// Main playlist crawler
	go func() {
		for {
			fmt.Println("[NU] Initiating Release Crawler")
			artists := GetAllArtists(nu.artistsDB)
			artistsUpdated := 0

			var err error
			nu.playlistTracks, err = spo.GetPlaylistTracks(context.Background(), nu.playlistID)
			cLog("main/Main playlist crawler", err)

			for _, artist := range artists {
				trackIDs := nu.playlist.GetNewestTracks(context.Background(), artist, nu.playlistTracks)

				if len(trackIDs) > 0 {
					_, err := spo.AddTracksToPlaylist(context.Background(), nu.playlistID, trackIDs...)
					cLog("main/Main playlist crawler", err)

					UpdateLastTrack(nu.artistsDB, artist.SUI)
					fmt.Printf("Updated: %v, Tracks: %v\n", artist.Name, len(trackIDs))
					artistsUpdated += 1
				} else {
					fmt.Println(artist.Name)
				}
				// time.Sleep(time.Second / 3)
			}

			fmt.Printf("[NU] Crawl Completed At %v - %v/%v Artists Updated\n", time.Now().Format("06-01-02 15:04:05"), artistsUpdated, len(artists))
			weeklyUpdater(nu)
			time.Sleep(time.Minute * 30)
		}
	}()

	for {
		var input string
		fmt.Scan(&input)

		if input == "terminate" {
			break
		}
	}
}

func prodCheck(nu *Nustyle) {
	if len(os.Args) > 1 {
		if os.Args[1] == "-p" {
			nu.pathToDB = "./artistdb/artists.db"
			nu.redirectURL = "http://quiet-reaches-27997.herokuapp.com/auth/callback"
			nu.playlistID = "0TdRzSP9GMdDcnuZd7wSTE"
		}
	}
}

func weeklyUpdater(nu Nustyle) {
	// Only updates playlist if its past 5pm on monday
	if int(time.Now().Weekday()) == 1 && time.Now().Hour() > 17 && len(nu.playlistTracks.Tracks) > 20 {
		nu.playlist.UpdatePlaylist(context.Background(), nu.playlistID, nu.userID)
		fmt.Printf("[NU] New Playlist Created - Main Playlist Cleared\n")
	}
}
