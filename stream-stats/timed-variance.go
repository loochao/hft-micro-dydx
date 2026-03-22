package stream_stats

import (
	"time"
)

type TimedVariance struct {
	lookback time.Duration
	times    []time.Time
	values   []float64
	sum      float64
	sumSq    float64
	mean     float64
	variance float64
}

func (tm *TimedVariance) Insert(timestamp time.Time, value float64) float64 {
	tm.times = append(tm.times, timestamp)
	tm.values = append(tm.values, value)
	tm.sum += value
	tm.sumSq += value * value
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
		for _, v := range tm.values[:cutIndex] {
			tm.sum -= v
			tm.sumSq -= v * v
		}
		tm.values = tm.values[cutIndex:]
		tm.times = tm.times[cutIndex:]
	}
	if len(tm.values) > 0 {
		tm.mean = tm.sum/float64(len(tm.values))
		tm.variance = (tm.sumSq / float64(len(tm.values))) - tm.mean*tm.mean
	}
	return tm.variance
}

func (tm *TimedVariance) Values() []float64 {
	return tm.values
}

func (tm *TimedVariance) Times() []time.Time {
	return tm.times
}

func (tm *TimedVariance) Len() int {
	return len(tm.values)
}

func (tm *TimedVariance) Range() time.Duration {
	if len(tm.times) > 2 {
		return tm.times[len(tm.times)-1].Sub(tm.times[0])
	} else {
		return time.Duration(0)
	}
}

func (tm *TimedVariance) Variance() float64 {
	return tm.variance
}

func NewTimedVariance(lookback time.Duration) *TimedVariance {
	return &TimedVariance{
		lookback: lookback,
		times:    make([]time.Time, 0),
		values:   make([]float64, 0),
	}
}
