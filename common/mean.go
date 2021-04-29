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

type TimedWeightedMean struct {
	lookback    time.Duration
	times       []time.Time
	values      []float64
	weights     []float64
	TotalValue  float64
	TotalWeight float64
	lastMean    float64
}

func (tm *TimedWeightedMean) Insert(timestamp time.Time, weight, value float64, ) float64 {
	tm.times = append(tm.times, timestamp)
	tm.values = append(tm.values, value*weight)
	tm.weights = append(tm.weights, weight)
	tm.TotalValue += value * weight
	tm.TotalWeight += weight
	cutIndex := 0
	for i, t := range tm.times {
		if timestamp.Sub(t) > tm.lookback {
			cutIndex = i
		} else {
			break
		}
	}
	if cutIndex > 0 {
		for i, value := range tm.values[:cutIndex] {
			tm.TotalValue -= value
			tm.TotalWeight -= tm.weights[i]
		}
		tm.values = tm.values[cutIndex:]
		tm.weights = tm.weights[cutIndex:]
		tm.times = tm.times[cutIndex:]
	}
	if tm.TotalWeight != 0 {
		tm.lastMean = tm.TotalValue / tm.TotalWeight
	}
	return tm.lastMean
}

func (tm *TimedWeightedMean) Median() float64 {
	return tm.lastMean
}

func NewTimedWeightedMean(lookback time.Duration) *TimedWeightedMean {
	return &TimedWeightedMean{
		lookback: lookback,
		times:    make([]time.Time, 0),
		values:   make([]float64, 0),
		weights:  make([]float64, 0),
	}
}
