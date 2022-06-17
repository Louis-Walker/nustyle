package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/zmb3/spotify/v2"

	"example.com/nustyle/artistdb"
	logger "example.com/nustyle/logger"
	ss "example.com/nustyle/spotifyService"
)

type Nustyle struct {
	DB_LOCATION    string
	PLAYLIST_ID    spotify.ID
	REDIRECT_URL   string
	USER_ID        string
	spotifyService *ss.SpotifyService
	artistsDB      *sql.DB
	playlistTracks *spotify.PlaylistTrackPage
}

func main() {
	var nu Nustyle = Nustyle{
		DB_LOCATION:  "./artistdb/artistsDEV.db",
		PLAYLIST_ID:  "3uzLhwcuH1KpmeCPWMqnQl",
		REDIRECT_URL: "http://localhost:8080/auth",
		USER_ID:      "m05hi",
	}

	prodCheck(nu) //Check environment

	// Connections
	var err error
	nu.artistsDB = artistdb.OpenConn(nu.DB_LOCATION)
	nu.spotifyService, err = ss.New(nu.REDIRECT_URL)
	logger.Psave("main", err)

	spo := nu.spotifyService.Client // Easier short hand

	// Main playlist crawler
	go func() {
		for {
			fmt.Println("[NU] Initiating Release Crawler")
			ctx := context.Background()
			defer ctx.Done()
			artists := artistdb.GetAllArtists(nu.artistsDB)
			artistsUpdated := 0

			var err error
			nu.playlistTracks, err = spo.GetPlaylistTracks(ctx, nu.PLAYLIST_ID)
			logger.Psave("main/Main playlist crawler", err)

			for _, artist := range artists {
				trackIDs := nu.spotifyService.GetNewestTracks(artist, nu.playlistTracks)

				if len(trackIDs) > 0 {
					_, err := spo.AddTracksToPlaylist(ctx, nu.PLAYLIST_ID, trackIDs...)
					logger.Psave("main/Main playlist crawler", err)

					artistdb.UpdateLastTrack(nu.artistsDB, artist.SUI)
					fmt.Printf("Updated: %v, Tracks: %v", artist.Name, len(trackIDs))
					artistsUpdated += 1
				}

				fmt.Println(artist.Name)
				// time.Sleep(time.Second / 3)
			}

			fmt.Printf("[NU] Crawl Completed At %v - %v/%v Updated\n", time.Now().Format("06-01-02 15:04:05"), artistsUpdated, len(artists))
			time.Sleep(time.Minute * 30)
		}
	}()

	go weeklyUpdater(nu)

	for {
		var input string
		fmt.Scan(&input)

		if input == "terminate" {
			break
		}
	}
}

func prodCheck(nu Nustyle) {
	if len(os.Args) > 1 {
		if os.Args[1] == "-p" {
			nu.DB_LOCATION = "./artistdb/artists.db"
			nu.REDIRECT_URL = "http://localhost:8080/auth"
			nu.PLAYLIST_ID = "0TdRzSP9GMdDcnuZd7wSTE"

			fmt.Println("[NU] Initialising in PRODUCTION mode. Do you wish to continue? [y/n]")
			var input string
			_, err := fmt.Scan(&input)
			logger.Psave("prodCheck", err)

			if input == "n" {
				os.Exit(1)
			}
		}
	}
}

func weeklyUpdater(nu Nustyle) {
	for {
		// Only updates playlist if its past 5pm on monday
		if time.Now().Day() == 1 && time.Now().Hour() > 17 && nu.playlistTracks.Total > 20 {
			nu.spotifyService.UpdatePlaylist(nu.PLAYLIST_ID, nu.USER_ID)
			fmt.Printf("[NU] New Playlist Created - Main Playlist Cleared\n")
		}

		time.Sleep(time.Hour * 12)
	}
}
