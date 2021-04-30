package common

import "time"

type Signal struct {
	Value  float64
	Weight float64
	Name   string
	Time   time.Time
}
