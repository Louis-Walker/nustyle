package artistdb

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"

	m "example.com/nustyle/model"
)

func OpenConn(path string) *sql.DB {
	db, err := sql.Open("sqlite3", path)
	checkErr(err)

	db.SetMaxOpenConns(1)

	return db
}

func GetAllArtists(db *sql.DB) []m.Artist {
	var artists []m.Artist

	ar, err := db.Query("SELECT * FROM Artists")
	if err != nil {
		checkErr(err)
	}
	defer ar.Close()

	for ar.Next() {
		var (
			id                int
			lastTrackDateTime string
			a                 m.Artist
		)

		err := ar.Scan(&id, &a.Name, &a.SUI, &lastTrackDateTime)
		if err != nil {
			checkErr(err)
		}

		a.LastTrackDateTime, err = time.Parse("2006-01-02 15:04:05", lastTrackDateTime)
		if err != nil {
			checkErr(err)
		}

		artists = append(artists, a)
	}

	return artists
}

func UpdateLastTrack(db *sql.DB, SUI string) {
	stmt, err := db.Prepare("UPDATE Artists SET LastTrackDateTime = ? WHERE SUI = ?")
	if err != nil {
		checkErr(err)
	}

	currentTime := time.Now().Format("2006-01-02 15:04:05")

	res, err := stmt.Exec(currentTime, SUI)
	if err != nil {
		checkErr(err)
	}
	println(res)
}

func AddArtist(db *sql.DB, a m.Artist) {
	if !artistExists(db, a.SUI) {
		stmt, err := db.Prepare("INSERT INTO Artists(Name, SUI, LastTrackDateTime) VALUES (?, ?, ?)")
		if err != nil {
			checkErr(err)
		}
		res, err := stmt.Exec(a.Name, a.SUI, a.LastTrackDateTime)
		if err != nil {
			checkErr(err)
		}
		fmt.Println(res)
	}
}

func artistExists(db *sql.DB, SUI string) bool {
	stmt, err := db.Prepare("SELECT count(*) FROM Artists WHERE SUI = ?")
	if err != nil {
		checkErr(err)
	}

	var count int

	err = stmt.QueryRow(SUI).Scan(&count)
	if err != nil {
		log.Fatal(err)
	}

	if count == 0 {
		return false
	}

	return true
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}
