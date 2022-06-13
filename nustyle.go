package main

import (
	"context"
	"fmt"
	"time"

	"github.com/zmb3/spotify/v2"

	"example.com/nustyle/artistdb"

	m "example.com/nustyle/model"
)

const REDIRECT_URL = "http://localhost:8080/auth"
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

	for {
		var input string
		fmt.Scan(&input)

		if input == "terminate" {
			break
		}
	}
}

func getNewestTracks(a m.Artist) []spotify.ID {
	var SUI spotify.ID = spotify.ID(a.SUI)

	albums, err := spo.GetArtistAlbums(ctx, SUI, []spotify.AlbumType{1, 2})
	if err != nil {
		println("SPO/GetArtistAlbums: %v", err)
	}

	var newTracks []spotify.ID

	for _, album := range albums.Albums[:4] {
		if album.ReleaseDateTime().After(a.LastTrackDateTime) {
			tracks, err := spo.GetAlbumTracks(ctx, album.ID)
			if err != nil {
				println("SPO/GetTracks: %v", err)
			}

			for _, track := range tracks.Tracks {
				newTracks = append(newTracks, track.ID)
			}
		}
	}

	return newTracks
}
