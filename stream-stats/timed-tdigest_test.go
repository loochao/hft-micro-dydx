package stream_stats

import (
	"github.com/geometrybase/hft-micro/tdigest"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
	"time"
)

func TestNewTimedTDigest(t *testing.T) {
	td, _ := tdigest.New()
	rollingTD := NewTimedTDigest(time.Hour*24, time.Hour)
	startTime := time.Time{}
	times := make([]time.Time, 0)
	values := make([]float64, 0)
	for i := time.Duration(0); i < time.Hour*100; i += time.Minute {
		times = append(times, startTime.Add(i))
		values = append(values, rand.NormFloat64())
	}
	for i := 0; i < 1440; i++ {
		err := rollingTD.Insert(times[i], values[i])
		if err != nil {
			t.Fatal(err)
		}
		err = td.Add(values[i])
		if err != nil {
			t.Fatal(err)
		}
	}
	assert.Equal(t, td.Quantile(0.005), rollingTD.Quantile(0.005))
	assert.Equal(t, td.Quantile(0.05), rollingTD.Quantile(0.05))
	assert.Equal(t, td.Quantile(0.2), rollingTD.Quantile(0.2))
	assert.Equal(t, td.Quantile(0.5), rollingTD.Quantile(0.5))
	assert.Equal(t, td.Quantile(0.8), rollingTD.Quantile(0.8))
	assert.Equal(t, td.Quantile(0.95), rollingTD.Quantile(0.95))
	assert.Equal(t, td.Quantile(0.995), rollingTD.Quantile(0.995))
	for i := 1440; i < len(times); i++ {
		err := rollingTD.Insert(times[i], values[i])
		if err != nil {
			t.Fatal(err)
		}
	}
	td, _ = tdigest.New()
	for i := len(times) - 1440; i < len(times); i++ {
		err := td.Add(values[i])
		if err != nil {
			t.Fatal(err)
		}
	}
	assert.Equal(t, td.Quantile(0.005), rollingTD.Quantile(0.005))
	assert.Equal(t, td.Quantile(0.05), rollingTD.Quantile(0.05))
	assert.Equal(t, td.Quantile(0.2), rollingTD.Quantile(0.2))
	assert.Equal(t, td.Quantile(0.5), rollingTD.Quantile(0.5))
	assert.Equal(t, td.Quantile(0.8), rollingTD.Quantile(0.8))
	assert.Equal(t, td.Quantile(0.95), rollingTD.Quantile(0.95))
	assert.Equal(t, td.Quantile(0.995), rollingTD.Quantile(0.995))
}
