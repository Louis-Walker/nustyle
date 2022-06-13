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

			for _, artist := range artists {
				trackIDs := getNewestTracks(artist)

				snapshotID, err := spo.AddTracksToPlaylist(ctx, PLAYLIST_ID, trackIDs...)
				if err != nil {
					fmt.Println(err)
				} else {
					artistdb.UpdateLastTrack(artistsDB, artist.SUI)
					fmt.Printf("Updated: %v, SID: %v", artist.Name, snapshotID)
				}
			}

			time.Sleep(60 * time.Minute)
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
