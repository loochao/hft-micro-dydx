package stream_stats

import (
	"testing"
	"time"
)

func BenchmarkTimedMean2(b *testing.B) {
	tm := NewTimedMean(time.Second * 3)
	startTime := time.Now()
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		for i := time.Duration(0); i < time.Second; i += time.Nanosecond {
			t := startTime.Add(time.Second*time.Duration(n) + i)
			tm.Insert(t, float64(i))
		}
	}
}

//1000000000 BenchmarkTimedMean2-16    	       1	505170705297 ns/op	186245372616 B/op	     200 allocs/op
func BenchmarkTimedEma(b *testing.B) {
	tm := NewTimedEma(time.Second * 3)
	startTime := time.Now()
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		for i := time.Duration(0); i < time.Second; i += time.Nanosecond {
			t := startTime.Add(time.Second*time.Duration(n) + i)
			tm.Insert(t, float64(i))
		}
	}
}