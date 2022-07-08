package main

import (
	"time"

	"github.com/zmb3/spotify/v2"
)

type TrackReview struct {
	ID        int
	Name      string
	SUI       spotify.ID
	DateAdded time.Time
	Status    int
}
