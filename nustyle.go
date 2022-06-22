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
	"example.com/nustyle/playlist"
)

type Nustyle struct {
	pathToDb, redirectUrl, userId string
	playlistId                    spotify.ID
	playlistService               *playlist.Service
	artistsDB                     *sql.DB
	playlistTracks                *spotify.PlaylistTrackPage
	Ctx                           context.Context
}

func main() {
	var nu Nustyle = Nustyle{
		pathToDb:    "./artistdb/artistsDEV.db",
		playlistId:  "3uzLhwcuH1KpmeCPWMqnQl",
		redirectUrl: "http://localhost:8080/auth",
		userId:      "m05hi",
		Ctx:         context.Background(),
	}

	// Context passed to spotify.Client must refresh for every re-authorization (1 hour)
	var cancel context.CancelFunc
	nu.Ctx, cancel = context.WithTimeout(nu.Ctx, time.Minute*60)

	prodCheck(&nu) //Check environment

	// Connections
	var err error
	nu.artistsDB = artistdb.OpenConn(nu.pathToDb)
	nu.playlistService, err = playlist.New(nu.Ctx, nu.redirectUrl)
	logger.Psave("main", err)

	spo := nu.playlistService.Client // Easier short hand

	// Main playlist crawler
	go func() {
		for {
			fmt.Println("[NU] Initiating Release Crawler")
			artists := artistdb.GetAllArtists(nu.artistsDB)
			artistsUpdated := 0

			var err error
			nu.playlistTracks, err = spo.GetPlaylistTracks(nu.Ctx, nu.playlistId)
			logger.Psave("main/Main playlist crawler", err)

			for _, artist := range artists {
				trackIDs := nu.playlistService.GetNewestTracks(nu.Ctx, artist, nu.playlistTracks)

				if len(trackIDs) > 0 {
					_, err := spo.AddTracksToPlaylist(nu.Ctx, nu.playlistId, trackIDs...)
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
			weeklyUpdater(nu)
			defer cancel()
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
	// Only updates playlist if its past 5pm on monday
	if int(time.Now().Weekday()) == 1 && time.Now().Hour() > 17 && len(nu.playlistTracks.Tracks) > 20 {
		nu.playlistService.UpdatePlaylist(nu.Ctx, nu.playlistId, nu.userId)
		fmt.Printf("[NU] New Playlist Created - Main Playlist Cleared\n")
	}
}
