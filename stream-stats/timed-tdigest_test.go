package stream_stats

import (
	"encoding/json"
	"github.com/geometrybase/hft-micro/logger"
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

func TestSaveTimedTDigest(t *testing.T) {
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
	data, err := json.Marshal(rollingTD)
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%s",data)
	outTD := &TimedTDigest{}
	err = json.Unmarshal(data, &outTD)
	if err != nil {
		t.Fatal(err)
	}
	assert.InDelta(t, rollingTD.Quantile(0.005), outTD.Quantile(0.005), 1e-6)
	assert.InDelta(t, rollingTD.Quantile(0.05), outTD.Quantile(0.05),1e-6)
	assert.InDelta(t, rollingTD.Quantile(0.5), outTD.Quantile(0.5), 1e-6)
	assert.InDelta(t, rollingTD.Quantile(0.95), outTD.Quantile(0.95), 1e-6)
	assert.InDelta(t, rollingTD.Quantile(0.995), outTD.Quantile(0.995), 1e-6)
}
