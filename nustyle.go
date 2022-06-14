package main

import (
	"context"
	"fmt"
	"time"

	"example.com/nustyle/artistdb"
)

const REDIRECT_URL = "http://localhost:8080/auth"
const USER_ID = "m05hi"
const PLAYLIST_ID = "0TdRzSP9GMdDcnuZd7wSTE"
const SPOTIFY_ID = "b1c55051e57c47c28659d3e0d12fc875"
const SPOTIFY_SECRET = "bc64e49696ec4182bd92514b24c15ddd"

var ctx = context.Background()
var spo = initSpotifyClient()

func main() {
	artistsDB := artistdb.OpenConn("./artistdb/artists.db")

	go func() {
		for {
			artists := artistdb.GetAllArtists(artistsDB)

			fmt.Println("[X] Initiating Release Scanner [X]")
			for _, artist := range artists {
				trackIDs := getNewestTracks(artist)

				if len(trackIDs) > 0 {
					snapshotID, err := spo.AddTracksToPlaylist(ctx, PLAYLIST_ID, trackIDs...)
					if err != nil {
						fmt.Printf("ERROR: %v", err)
					} else {
						artistdb.UpdateLastTrack(artistsDB, artist.SUI)
						fmt.Printf("Updated: %v, SID: %v", artist.Name, snapshotID)
					}
				}

				fmt.Println(artist.Name)
				time.Sleep(time.Second / 3)
			}

			fmt.Printf("[X] Scan Successful - %v Artists Scanned [X]\n", len(artists))
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
		}

		time.Sleep(time.Hour * 12)
	}
}

func printError(err error) {
	if err != nil {
		fmt.Printf("ERROR: %v", err)
	}
}
