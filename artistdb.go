package main

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func OpenArtistDB(path string) *sql.DB {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		cLog("artistsdb/OpenArtistDB", err)
	}

	db.SetMaxOpenConns(1)

	return db
}

func GetAllArtists(db *sql.DB) []Artist {
	var artists []Artist

	ar, err := db.Query("SELECT * FROM Artists")
	if err != nil {
		cLog("artistsdb/GetAllArtists", err)
	}
	defer ar.Close()

	for ar.Next() {
		var (
			id                int
			lastTrackDateTime string
			a                 Artist
		)

		err := ar.Scan(&id, &a.Name, &a.SUI, &lastTrackDateTime)
		if err != nil {
			cLog("artistsdb/GetAllArtists", err)
		}

		a.LastTrackDateTime, err = time.Parse("2006-01-02 15:04:05+00:00", lastTrackDateTime)
		if err != nil {
			cLog("artistsdb/GetAllArtists", err)
		}

		artists = append(artists, a)
	}

	return artists
}

func UpdateLastTrack(db *sql.DB, SUI string) {
	stmt, err := db.Prepare("UPDATE Artists SET LastTrackDateTime = ? WHERE SUI = ?")
	if err != nil {
		cLog("artistsdb/UpdateLastTrack", err)
	}

	currentTime := time.Now().Format("2006-01-02 15:04:05+00:00")

	_, err = stmt.Exec(currentTime, SUI)
	if err != nil {
		cLog("artistsdb/UpdateLastTrack", err)
	}
}

func AddArtist(db *sql.DB, a Artist) {
	if !artistExists(db, a.SUI) {
		stmt, err := db.Prepare("INSERT INTO Artists(Name, SUI, LastTrackDateTime) VALUES (?, ?, ?)")
		if err != nil {
			cLog("artistsdb/AddArtist", err)
		}

		_, err = stmt.Exec(a.Name, a.SUI, a.LastTrackDateTime)
		if err != nil {
			cLog("artistsdb/AddArtist", err)
		}
	}
}

func artistExists(db *sql.DB, SUI string) bool {
	stmt, err := db.Prepare("SELECT count(*) FROM Artists WHERE SUI = ?")
	if err != nil {
		cLog("artistsdb/artistExists", err)
	}

	var count int

	err = stmt.QueryRow(SUI).Scan(&count)
	if err != nil {
		cLog("artistsdb/artistExists", err)
	}

	if count == 0 {
		return false
	}

	return true
}
