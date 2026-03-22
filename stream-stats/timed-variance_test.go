package stream_stats

import (
	"github.com/montanaflynn/stats"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
	"time"
)

func TestNewTimedVariance(t *testing.T) {
	tv := NewTimedVariance(time.Hour*24)
	startTime := time.Time{}
	for i := time.Duration(0); i < time.Hour*10000; i += time.Hour {
		tv.Insert(startTime.Add(i), rand.Float64())
		v, _ := stats.Variance(tv.Values())
		assert.InDeltaf(t, v, tv.variance, 1e-10, "")
	}
}
