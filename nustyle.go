package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"example.com/nustyle/artistdb"
	"github.com/zmb3/spotify/v2"
)

var (
	REDIRECT_URL            = "http://localhost:8080/auth"
	PLAYLIST_ID  spotify.ID = "3uzLhwcuH1KpmeCPWMqnQl"
	USER_ID                 = "m05hi"

	ctx            = context.Background()
	spo            *spotify.Client
	artistsDB      *sql.DB
	playlistTracks *spotify.PlaylistTrackPage
)

func main() {
	if len(os.Args) > 1 {
		if os.Args[1] == "-p" {
			REDIRECT_URL = "http://localhost:8080/auth"
			PLAYLIST_ID = "0TdRzSP9GMdDcnuZd7wSTE"

			fmt.Println("[NU] Initialising in PRODUCTION mode. Do you wish to continue? [y/n]")
			var input string
			_, err := fmt.Scan(&input)
			printError(err)

			if input == "n" {
				os.Exit(1)
			}
		}
	}

	spo = initSpotifyClient()
	artistsDB = artistdb.OpenConn("./artistdb/artists.db")

	go func() {
		for {
			fmt.Println("[NU] Initiating Release Scanner")
			artists := artistdb.GetAllArtists(artistsDB)

			var err error
			playlistTracks, err = spo.GetPlaylistTracks(context.Background(), PLAYLIST_ID)
			if err != nil {
				printError(err)
			}

			for _, artist := range artists {
				trackIDs := getNewestTracks(artist)

				if len(trackIDs) > 0 {
					snapshotID, err := spo.AddTracksToPlaylist(context.Background(), PLAYLIST_ID, trackIDs...)
					if err != nil {
						fmt.Printf("ERROR: %v", err)
					} else {
						artistdb.UpdateLastTrack(artistsDB, artist.SUI)
						fmt.Printf("Updated: %v, SID: %v", artist.Name, snapshotID)
					}
				}

				fmt.Println(artist.Name)
				// time.Sleep(time.Second / 3)
			}

			fmt.Printf("[NU] Scan Completed At %v - %v Artists Scanned\n", time.Now().Format("2006-01-02 3:4:5 pm"), len(artists))
			time.Sleep(time.Minute * 30)
		}
	}()

	go weeklyUpdater()

	for {
		var input string
		fmt.Scan(&input)

		if input == "terminate" {
			break
		}
	}
}

func weeklyUpdater() {
	isOldPlaylist := false

	for {
		if time.Now().Day() == 2 {
			isOldPlaylist = true
		}

		if isOldPlaylist && time.Now().Day() == 1 {
			updatePlaylist()
			fmt.Printf("[NU] New Playlist Created - Main Playlist Cleared\n")
		}

		time.Sleep(time.Hour * 12)
	}
}

func printError(err error) {
	if err != nil {
		fmt.Printf("ERROR: %v", err)
	}
}
