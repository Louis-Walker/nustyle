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
	pathToDb, redirectUrl, userId string
	playlistId                    spotify.ID
	spotifyService                *ss.SpotifyService
	artistsDB                     *sql.DB
	playlistTracks                *spotify.PlaylistTrackPage
}

func main() {
	var nu Nustyle = Nustyle{
		pathToDb:    "./artistdb/artists.db",
		playlistId:  "3uzLhwcuH1KpmeCPWMqnQl",
		redirectUrl: "http://localhost:8080/auth",
		userId:      "m05hi",
	}

	prodCheck(&nu) //Check environment

	// Connections
	var err error
	nu.artistsDB = artistdb.OpenConn(nu.pathToDb)
	nu.spotifyService, err = ss.New(nu.redirectUrl)
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
			nu.playlistTracks, err = spo.GetPlaylistTracks(ctx, nu.playlistId)
			logger.Psave("main/Main playlist crawler", err)

			for _, artist := range artists {
				trackIDs := nu.spotifyService.GetNewestTracks(artist, nu.playlistTracks)

				if len(trackIDs) > 0 {
					_, err := spo.AddTracksToPlaylist(ctx, nu.playlistId, trackIDs...)
					logger.Psave("main/Main playlist crawler", err)

					artistdb.UpdateLastTrack(nu.artistsDB, artist.SUI)
					fmt.Printf("Updated: %v, Tracks: %v\n", artist.Name, len(trackIDs))
					artistsUpdated += 1
				} else {
					fmt.Println(artist.Name)
				}
				// time.Sleep(time.Second / 3)
			}

			fmt.Printf("[NU] Crawl Completed At %v - %v/%v Artists Updated\n", time.Now().Format("06-01-02 15:04:05"), artistsUpdated, len(artists))
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

func prodCheck(nu *Nustyle) {
	if len(os.Args) > 1 {
		if os.Args[1] == "-p" {
			nu.pathToDb = "./artistdb/artists.db"
			nu.redirectUrl = "http://localhost:8080/auth"
			nu.playlistId = "0TdRzSP9GMdDcnuZd7wSTE"

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
			nu.spotifyService.UpdatePlaylist(nu.playlistId, nu.userId)
			fmt.Printf("[NU] New Playlist Created - Main Playlist Cleared\n")
		}

		time.Sleep(time.Hour * 12)
	}
}
