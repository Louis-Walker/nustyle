package artistdb

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"

	logger "example.com/nustyle/logger"
	m "example.com/nustyle/model"
)

func OpenConn(path string) *sql.DB {
	db, err := sql.Open("sqlite3", path)
	logger.Psave("OpenConn", err)

	db.SetMaxOpenConns(1)

	return db
}

func GetAllArtists(db *sql.DB) []m.Artist {
	var artists []m.Artist

	ar, err := db.Query("SELECT * FROM Artists")
	logger.Psave("GetAllArtists", err)
	defer ar.Close()

	for ar.Next() {
		var (
			id                int
			lastTrackDateTime string
			a                 m.Artist
		)

		err := ar.Scan(&id, &a.Name, &a.SUI, &lastTrackDateTime)
		if err != nil {
			logger.Psave("GetAllArtists", err)
		}

		a.LastTrackDateTime, err = time.Parse("2006-01-02 15:04:05+00:00", lastTrackDateTime)
		if err != nil {
			logger.Psave("GetAllArtists", err)
		}

		artists = append(artists, a)
	}

	return artists
}

func UpdateLastTrack(db *sql.DB, SUI string) {
	stmt, err := db.Prepare("UPDATE Artists SET LastTrackDateTime = ? WHERE SUI = ?")
	logger.Psave("UpdateLastTrack", err)

	currentTime := time.Now().Format("2006-01-02 15:04:05+00:00")

	_, err = stmt.Exec(currentTime, SUI)
	logger.Psave("UpdateLastTrack", err)
}

func AddArtist(db *sql.DB, a m.Artist) {
	if !artistExists(db, a.SUI) {
		stmt, err := db.Prepare("INSERT INTO Artists(Name, SUI, LastTrackDateTime) VALUES (?, ?, ?)")
		logger.Psave("AddArtist", err)

		_, err = stmt.Exec(a.Name, a.SUI, a.LastTrackDateTime)
		logger.Psave("AddArtist", err)
	}
}

func artistExists(db *sql.DB, SUI string) bool {
	stmt, err := db.Prepare("SELECT count(*) FROM Artists WHERE SUI = ?")
	logger.Psave("artistExists", err)

	var count int

	err = stmt.QueryRow(SUI).Scan(&count)
	logger.Psave("artistExists", err)

	if count == 0 {
		return false
	}

	return true
}
