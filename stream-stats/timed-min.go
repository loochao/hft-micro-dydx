package stream_stats

import (
	"math"
	"sort"
	"time"
)

type TimedMin struct {
	lookback     time.Duration
	times        []time.Time
	values       []float64
	sortedValues []float64
	sortedTimes  []time.Time
	min          float64
	offset       time.Duration
}

func (tm *TimedMin) insert(t time.Time, v float64) {
	i := sort.SearchFloat64s(tm.sortedValues, v)
	tm.sortedValues = append(tm.sortedValues, 0)
	copy(tm.sortedValues[i+1:], tm.sortedValues[i:])
	tm.sortedValues[i] = v
	tm.sortedTimes = append(tm.sortedTimes, time.Time{})
	copy(tm.sortedTimes[i+1:], tm.sortedTimes[i:])
	tm.sortedTimes[i] = t
	return
}

func (tm *TimedMin) delete(value float64) {
	i := sort.SearchFloat64s(tm.sortedValues, value)
	if i == len(tm.sortedValues) {
		tm.sortedValues = tm.sortedValues[:i]
		tm.sortedTimes = tm.sortedTimes[:i]
	} else {
		tm.sortedValues = append(tm.sortedValues[:i], tm.sortedValues[i+1:]...)
		tm.sortedTimes = append(tm.sortedTimes[:i], tm.sortedTimes[i+1:]...)
	}
}

func (tm *TimedMin) Insert(timestamp time.Time, value float64) {
	tm.times = append(tm.times, timestamp)
	tm.values = append(tm.values, value)
	tm.insert(timestamp, value)
	cutIndex := -1
	for i, t := range tm.times {
		if timestamp.Sub(t) > tm.lookback {
			cutIndex = i
			tm.delete(tm.values[i])
		} else {
			break
		}
	}
	cutIndex += 1
	if cutIndex > 0 {
		tm.values = tm.values[cutIndex:]
		tm.times = tm.times[cutIndex:]
	}
	if len(tm.sortedValues) > 0 {
		tm.min = tm.sortedValues[0]
		tm.offset = tm.times[len(tm.times)-1].Sub(tm.sortedTimes[0])
	} else {
		tm.min = math.NaN()
		tm.offset = time.Duration(0)
	}
}

func (tm *TimedMin) Values() []float64 {
	return tm.values
}

func (tm *TimedMin) Times() []time.Time {
	return tm.times
}

func (tm *TimedMin) Len() int {
	return len(tm.values)
}

func (tm *TimedMin) Range() time.Duration {
	if len(tm.times) > 2 {
		return tm.times[len(tm.times)-1].Sub(tm.times[0])
	} else {
		return time.Duration(0)
	}
}
func (tm *TimedMin) Min() float64 {
	return tm.min
}

func (tm *TimedMin) Offset() time.Duration {
	return tm.offset
}

func NewTimedMin(lookback time.Duration) *TimedMin {
	return &TimedMin{
		lookback:     lookback,
		times:        make([]time.Time, 0),
		values:       make([]float64, 0),
		sortedValues: make([]float64, 0),
		sortedTimes:  make([]time.Time, 0),
	}
}
