package stream_stats

import "time"

type TimedMax struct {
	lookback         time.Duration
	times            []time.Time
	values           []float64
	max              float64
	sortedFloatSlice SortedFloatSlice
}

func (tm *TimedMax) Insert(timestamp time.Time, value float64) float64 {
	tm.times = append(tm.times, timestamp)
	tm.values = append(tm.values, value)
	tm.sortedFloatSlice = tm.sortedFloatSlice.Insert(value)
	cutIndex := -1
	for i, t := range tm.times {
		if timestamp.Sub(t) > tm.lookback {
			cutIndex = i
			tm.sortedFloatSlice = tm.sortedFloatSlice.Delete(tm.values[i])
		} else {
			break
		}
	}
	cutIndex += 1
	if cutIndex > 0 {
		tm.values = tm.values[cutIndex:]
		tm.times = tm.times[cutIndex:]
	}
	tm.max = tm.sortedFloatSlice.Max()
	return tm.max
}
func (tm *TimedMax) Values() []float64 {
	return tm.values
}
func (tm *TimedMax) Times() []time.Time {
	return tm.times
}
func (tm *TimedMax) Len() int {
	return len(tm.values)
}
func (tm *TimedMax) Range() time.Duration {
	if len(tm.times) > 2 {
		return tm.times[len(tm.times)-1].Sub(tm.times[0])
	} else {
		return time.Duration(0)
	}
}
func (tm *TimedMax) Max() float64 {
	return tm.max
}
func NewTimedMax(lookback time.Duration) *TimedMax {
	return &TimedMax{
		lookback:         lookback,
		times:            make([]time.Time, 0),
		values:           make([]float64, 0),
		sortedFloatSlice: SortedFloatSlice{},
	}
}

