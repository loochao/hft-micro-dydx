package benchmarks

import (
	"testing"
	"time"
)

var GlobalTimer *time.Timer

func BenchmarkTimerReset(b *testing.B) {
	timer := time.NewTimer(time.Hour)
	for i := 0; i < b.N; i++ {
		timer.Reset(time.Hour)
		select {
		case <- timer.C:
		default:
		}
	}
	GlobalTimer = timer
}

func BenchmarkTimerReset2(b *testing.B) {
	timer := time.NewTimer(time.Hour)
	for i := 0; i < b.N; i++ {
		timer.Reset(time.Hour)
	}
	GlobalTimer = timer
}