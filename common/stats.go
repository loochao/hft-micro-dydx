package common

import "time"

type TimeDelta struct {
	EventTime time.Time
	Value     time.Duration
}

type Spread struct {
	EventTime time.Time
	Value     float64
}
