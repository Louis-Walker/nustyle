package main

import (
	"context"
	"database/sql"
	"fmt"
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
)

func main() {
	if isProd() {
		pathToDB = "./artistdb/artists.db"
		redirectURL = "http://quiet-reaches-27997.herokuapp.com/auth/callback"
		pid = "0TdRzSP9GMdDcnuZd7wSTE"
	} else {
		pathToDB = "./artistdb/artistsDEV.db"
		redirectURL = "http://localhost:8080/auth/callback"
		pid = "3uzLhwcuH1KpmeCPWMqnQl"

		cmd := exec.Command("cmd", "/c", "start", "http://localhost:8080")
		cmd.Start()
	}
	userID = "m05hi"

	// Connections
	var err error
	artistsDB = OpenArtistDB(pathToDB)
	playlist, err = NewPlaylist(context.Background(), redirectURL, pid)
	if err != nil {
		cLog("main", err)
	}

	spo := playlist.Client // Easier short hand

	// Main playlist crawler
	go func() {
		for {
			fmt.Println("[NU] Initiating Release Crawler")
			artists := GetAllArtists(artistsDB)
			artistsUpdated := 0

			var err error
			playlist.Tracks, err = spo.GetPlaylistTracks(context.Background(), playlist.ID)
			if err != nil {
				cLog("main/Main playlist crawler", err)
			}

			for _, artist := range artists {
				trackIDs := playlist.GetNewestTracks(context.Background(), artist, playlist.Tracks)

				if len(trackIDs) > 0 {
					_, err := spo.AddTracksToPlaylist(context.Background(), playlist.ID, trackIDs...)
					if err != nil {
						cLog("main/Main playlist crawler", err)
					}

					UpdateLastTrack(artistsDB, artist.SUI)
					fmt.Printf("Updated: %v, Tracks: %v\n", artist.Name, len(trackIDs))
					artistsUpdated += 1
				} else {
					fmt.Println(artist.Name)
				}
				// time.Sleep(time.Second / 3)
			}

			fmt.Printf("[NU] Crawl Completed At %v - %v/%v Artists Updated\n", time.Now().Format("06-01-02 15:04:05"), artistsUpdated, len(artists))
			weeklyUpdater()
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

func isProd() bool {
	if len(os.Args) > 1 {
		return true
	} else {
		return false
	}
}

func weeklyUpdater() {
	// Only updates playlist if its past 5pm on monday
	if int(time.Now().Weekday()) == 1 && time.Now().Hour() > 17 && len(playlist.Tracks.Tracks) > 20 {
		playlist.UpdatePlaylist(context.Background(), playlist.ID, userID)
		fmt.Printf("[NU] New Playlist Created - Main Playlist Cleared\n")
	}
}
