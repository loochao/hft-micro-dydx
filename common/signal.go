package common

import "time"

type Signal struct {
	Name  string
	Value float64
	Time  time.Time
}
