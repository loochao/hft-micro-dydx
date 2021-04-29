package common

import "time"

type TimedMedian struct {
	lookback         time.Duration
	times            []time.Time
	values           []float64
	sortedFloatSlice SortedFloatSlice
}

func (tm *TimedMedian) Insert(value float64, timestamp time.Time) float64 {
	tm.times = append(tm.times, timestamp)
	tm.values = append(tm.values, value)
	tm.sortedFloatSlice = tm.sortedFloatSlice.Insert(value)
	cutIndex := 0
	for i, t := range tm.times {
		if timestamp.Sub(t) > tm.lookback {
			cutIndex = i
		} else {
			break
		}
	}
	if cutIndex > 0 {
		for _, value = range tm.values[:cutIndex] {
			tm.sortedFloatSlice = tm.sortedFloatSlice.Delete(value)
		}
		tm.values = tm.values[cutIndex:]
		tm.times = tm.times[cutIndex:]
	}
	return tm.sortedFloatSlice.Median()
}

func (tm *TimedMedian) Median() float64 {
	return tm.sortedFloatSlice.Median()
}

func NewTimedMedian(lookback time.Duration) *TimedMedian {
	return &TimedMedian{
		lookback:         lookback,
		times:            make([]time.Time, 0),
		values:           make([]float64, 0),
		sortedFloatSlice: SortedFloatSlice{},
	}
}


