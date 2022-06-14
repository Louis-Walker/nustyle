package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"example.com/nustyle/artistdb"
	m "example.com/nustyle/model"
)

func main() {
	db := artistdb.OpenConn("../artistdb/artists.db")

	file, err := os.ReadFile("artists.csv")
	if err != nil {
		fmt.Println(err)
	}

	r := csv.NewReader(strings.NewReader(string(file)))
	if err != nil {
		fmt.Println(err)
	}

	a, err := r.ReadAll()
	for i := range a {
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		dateTime, _ := time.Parse("2006-01-02 15:04:05", time.Now().Format("2006-01-02 15:04:05"))

		newArtist := m.Artist{
			Name:              a[i][0],
			SUI:               a[i][1],
			LastTrackDateTime: dateTime,
		}

		artistdb.AddArtist(db, newArtist)
	}
}
