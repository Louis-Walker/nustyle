package main

import (
	"time"

	"github.com/zmb3/spotify/v2"
)

type Artist struct {
	Name              string
	SUI               spotify.ID
	LastTrackDateTime time.Time
}
