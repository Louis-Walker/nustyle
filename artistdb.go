package main

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/zmb3/spotify/v2"
)

func OpenArtistDB(path string) (db *sql.DB) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		logger("artistsdb/OpenArtistDB", err)
	}

	db.SetMaxOpenConns(1)

	return
}

func GetAllArtists(db *sql.DB) (artists []Artist) {
	aa, err := db.Query("SELECT * FROM Artists")
	if err != nil {
		logger("artistsdb/GetAllArtists", err)
	}
	defer aa.Close()

	for aa.Next() {
		var (
			id                int
			lastTrackDateTime string
			a                 Artist
		)

		err := aa.Scan(&id, &a.Name, &a.SUI, &lastTrackDateTime)
		if err != nil {
			logger("artistsdb/GetAllArtists", err)
		}

		a.LastTrackDateTime, err = time.Parse("2006-01-02 15:04:05+00:00", lastTrackDateTime)
		if err != nil {
			logger("artistsdb/GetAllArtists", err)
		}

		artists = append(artists, a)
	}

	return
}

func UpdateLastTrack(db *sql.DB, SUIs []spotify.ID) {
	for _, SUI := range SUIs {
		stmt, err := db.Prepare("UPDATE Artists SET LastTrackDateTime = ? WHERE SUI = ?")
		if err != nil {
			logger("artistsdb/UpdateLastTrack", err)
		}

		currentTime := time.Now().Format("2006-01-02 15:04:05+00:00")

		_, err = stmt.Exec(currentTime, SUI)
		if err != nil {
			logger("artistsdb/UpdateLastTrack", err)
		}
	}
}

func AddArtist(db *sql.DB, a Artist) {
	if !artistExists(db, a.SUI) {
		stmt, err := db.Prepare("INSERT INTO Artists(Name, SUI, LastTrackDateTime) VALUES (?, ?, ?)")
		if err != nil {
			logger("artistsdb/AddArtist", err)
		}

		_, err = stmt.Exec(a.Name, a.SUI, a.LastTrackDateTime)
		if err != nil {
			logger("artistsdb/AddArtist", err)
		}
	}
}

func RemoveArtist(db *sql.DB, a Artist) {
	stmt, err := db.Prepare("DELETE FROM Artists WHERE SUI = ?")
	if err != nil {
		logger("artistsdb/RemoveArtist", err)
	}

	_, err = stmt.Exec(a.SUI)
	if err != nil {
		logger("artistsdb/RemoveArtist", err)
	}

}

func artistExists(db *sql.DB, SUI spotify.ID) (exists bool) {
	exists = true

	stmt, err := db.Prepare("SELECT count(*) FROM Artists WHERE SUI = ?")
	if err != nil {
		logger("artistsdb/artistExists", err)
	}

	var count int

	err = stmt.QueryRow(SUI).Scan(&count)
	if err != nil {
		logger("artistsdb/artistExists", err)
	}

	if count == 0 {
		exists = false
	}

	return
}
