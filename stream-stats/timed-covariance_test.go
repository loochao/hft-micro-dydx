package stream_stats

import (
	"github.com/montanaflynn/stats"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
	"time"
)

func TestNewTimedCovariance(t *testing.T) {
	tcv := NewTimedCovariance(time.Hour*24)
	startTime := time.Time{}
	for i := time.Duration(0); i < time.Hour*10000; i += time.Hour {
		tcv.Insert(startTime.Add(i), rand.Float64(), rand.Float64())
		xs, ys := tcv.Values()
		v, _ := stats.CovariancePopulation(xs, ys)
		assert.InDeltaf(t, v, tcv.Covariance(), 1e-10, "")
	}
	stats.Correlation()
}
