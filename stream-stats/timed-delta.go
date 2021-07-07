package stream_stats

import (
	"time"
)

type TimedDelta struct {
	lookback time.Duration
	times    []time.Time
	values   []float64
	delta    float64
}

func (tm *TimedDelta) Insert(timestamp time.Time, value float64) float64 {
	tm.times = append(tm.times, timestamp)
	tm.values = append(tm.values, value)
	cutIndex := -1
	for i, t := range tm.times {
		if timestamp.Sub(t) > tm.lookback {
			cutIndex = i
		} else {
			break
		}
	}
	cutIndex += 1
	if cutIndex > 0 {
		tm.values = tm.values[cutIndex:]
		tm.times = tm.times[cutIndex:]
	}
	if len(tm.values) > 1 {
		tm.delta = tm.values[len(tm.values)-1] - tm.values[0]
	}
	return tm.delta
}

func (tm *TimedDelta) Values() []float64 {
	return tm.values
}

func (tm *TimedDelta) Times() []time.Time {
	return tm.times
}

func (tm *TimedDelta) Len() int {
	return len(tm.values)
}
func (tm *TimedDelta) Range() time.Duration {
	if len(tm.times) > 2 {
		return tm.times[len(tm.times)-1].Sub(tm.times[0])
	} else {
		return time.Duration(0)
	}
}
func (tm *TimedDelta) Delta() float64 {
	return tm.delta
}

func NewTimedDelta(lookback time.Duration) *TimedDelta {
	return &TimedDelta{
		lookback: lookback,
		times:    make([]time.Time, 0),
		values:   make([]float64, 0),
		delta:    0.0,
	}
}

