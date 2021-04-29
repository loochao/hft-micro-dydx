package common

import "time"

type TimedMedian struct {
	lookback         time.Duration
	times            []time.Time
	values           []float64
	sortedFloatSlice SortedFloatSlice
}

func (tm *TimedMedian) Insert(value float64, timestamp time.Time) float64{
	tm.times = append(tm.times, timestamp)
}

func NewTimedMedian(lookback time.Duration) *TimedMedian {
	return &TimedMedian{
		lookback:         lookback,
		times:            make([]time.Time, 0),
		values:           make([]float64, 0),
		sortedFloatSlice: SortedFloatSlice{},
	}
}
