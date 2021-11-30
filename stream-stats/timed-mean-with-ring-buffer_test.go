package stream_stats

import (
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
	"time"
)

func TestNewTimedMeanWithRingBuffer(t *testing.T) {
	tm1 := NewTimedMeanWithRingBuffer(time.Hour)
	tm2 := NewTimedMean(time.Hour)
	startTime := time.Now()
	for i := time.Duration(0); i < time.Hour*10; i += time.Minute {
		tt := startTime.Add(i)
		rr := rand.Float64()
		tm1.Insert(tt, rr)
		tm2.Insert(tt, rr)
	}
	assert.Equal(t, tm2.Mean, tm1.Mean)
}

func BenchmarkTimedMeanWithRingBuffer(b *testing.B) {
	tm := NewTimedMeanWithRingBuffer(time.Second*5)
	startTime := time.Now()
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		for i := time.Duration(0); i < time.Second; i+= time.Millisecond{
			t := startTime.Add(time.Second * time.Duration(n)+i)
			tm.Insert(t, 0)
		}
	}
}

var benchmarkTimedMeanOut *TimedMean

func BenchmarkTimedMean(b *testing.B) {
	tm := NewTimedMean(time.Second * 5)
	startTime := time.Now()
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		for i := time.Duration(0); i < time.Second; i+= time.Millisecond{
			t := startTime.Add(time.Second * time.Duration(n)+i)
			tm.Insert(t, 0)
		}
	}
	benchmarkTimedMeanOut = tm
}

func BenchmarkSliceAppend(b *testing.B) {
	s := make([]float64, 0)
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		s = append(s, 0)
		if len(s) > 100 {
			s = s[1:]
		}
	}
}

func BenchmarkSliceAppend2(b *testing.B) {
	s := make([]float64, 0)
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		s = append(s, 0)
		s[len(s)-1] = float64(n)
	}
}
