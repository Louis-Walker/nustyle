package model

import (
	"time"
)

type Artist struct {
	Name              string
	SUI               string
	LastTrackDateTime time.Time
}
