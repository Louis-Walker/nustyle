package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/zmb3/spotify/v2"
)

const (
	createTrackReviewsQuery = "CREATE TABLE TrackReviews ( ID integer PRIMARY KEY, Name text, SUI text, Artists text, ImageURL text, DateAdded text, Status integer );"
)

func OpenArtistDB(path string) (db *sql.DB) {
	if _, err := os.Stat(path); err != nil {
		fmt.Println("Artist database does not exist. Creating new database.")
		err := createDB(path)
		if err != nil {
			log.Fatal(err)
		} else {
			fmt.Println("Artist database successfully created.")
		}
	}

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		logger("artistsdb/OpenArtistDB", err)
	}

	db.SetMaxOpenConns(1)

	// Production db doesn't contain table yet
	res, err := db.Exec("SELECT name FROM sqlite_master WHERE type='table' AND name='TrackReviews'")
	if err != nil {
		logger("artistsdb/OpenArtistDB", err)
	}

	s, err := res.LastInsertId()
	if err != nil {
		logger("artistsdb/OpenArtistDB", err)
	}

	if s == 0 {
		err = createTable(db, createTrackReviewsQuery)
		if err != nil {
			log.Fatal(err)
		}
	}

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

func InsertArtist(db *sql.DB, a Artist) error {
	if !recordExistsBySUI(db, "Artists", a.SUI) {
		stmt, err := db.Prepare("INSERT INTO Artists(Name, SUI, LastTrackDateTime) VALUES (?, ?, ?)")
		if err != nil {
			logger("artistsdb/AddArtist", err)
		}

		_, err = stmt.Exec(a.Name, a.SUI, a.LastTrackDateTime)
		if err != nil {
			logger("artistsdb/AddArtist", err)
		}
	} else {
		err = errors.New("Already exists")
	}

	return err
}

func DeleteArtist(db *sql.DB, SUI spotify.ID) error {
	if recordExistsBySUI(db, "Artists", SUI) {
		stmt, err := db.Prepare("DELETE FROM Artists WHERE SUI = ?")
		if err != nil {
			logger("artistsdb/RemoveArtist", err)
		}

		_, err = stmt.Exec(SUI)
		if err != nil {
			logger("artistsdb/RemoveArtist", err)
		}
	} else {
		err = errors.New("Doesn't exist")
	}

	return err
}

func InsertTrackReview(db *sql.DB, t TrackReview) (err error) {
	if !(recordExistsBySUI(db, "TrackReviews", t.SUI)) {
		stmt, err := db.Prepare("INSERT INTO TrackReviews(Name, SUI, Artists, ImageURL, DateAdded, Status) VALUES (?, ?, ?, ?, ?, ?)")
		if err != nil {
			logger("artistsdb/AddTrackReview", err)
		}

		var artistsString string
		for _, a := range t.Artists {
			artistsString += fmt.Sprintf("%v,", a)
		}

		_, err = stmt.Exec(t.Name, t.SUI, artistsString, t.ImageURL, t.DateAdded, t.Status)
		if err != nil {
			logger("artistsdb/AddTrackReview", err)
		}
	}

	return
}

func UpdateTrackReviewStatus(db *sql.DB, SUI spotify.ID, s int) (err error) {
	if recordExistsBySUI(db, "TrackReviews", SUI) {
		stmt, err := db.Prepare("UPDATE TrackReviews SET Status = ? WHERE SUI = ?")
		if err != nil {
			logger("artistsdb/UpdateTrackReviewStatus", err)
		}

		_, err = stmt.Exec(s, SUI)
		if err != nil {
			logger("artistsdb/UpdateTrackReviewStatus", err)
		}
	} else {
		err = errors.New("Doesn't exist")
	}

	return err
}

func recordExistsBySUI(db *sql.DB, tableName string, SUI spotify.ID) (exists bool) {
	exists = false
	q := fmt.Sprintf("SELECT count(*) FROM %v WHERE SUI = ?", tableName)

	stmt, err := db.Prepare(q)
	if err != nil {
		logger("artistsdb/recordExistsBySUI", err)
	}

	var count int

	err = stmt.QueryRow(SUI).Scan(&count)
	if err != nil {
		logger("artistsdb/recordExistsBySUI", err)
	}

	if count == 1 {
		exists = true
	}

	return
}

func createDB(p string) (err error) {
	err = os.WriteFile(p, nil, 0777)
	if err != nil {
		log.Fatal("createDB - failed to write file.")
	}

	db := OpenArtistDB(p)
	err = createTable(db, "CREATE TABLE Artists ( ID integer PRIMARY KEY, Name text, SUI text, LastTrackDateTime text );")
	if err != nil {
		log.Fatal("createDB - failed to create Artists table.")
	}
	err = createTable(db, createTrackReviewsQuery)
	if err != nil {
		log.Fatal("createDB - failed to create TrackReviews table.")
	}
	return
}

func createTable(db *sql.DB, q string) (err error) {
	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()

	_, err = db.ExecContext(ctx, q)
	return
}
