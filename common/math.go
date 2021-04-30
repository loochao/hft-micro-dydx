package common

import "time"

type TimedSum struct {
	lookback time.Duration
	times    []time.Time
	values   []float64
	sum      float64
}

func (tm *TimedSum) Insert(timestamp time.Time, value float64) float64 {
	tm.times = append(tm.times, timestamp)
	tm.values = append(tm.values, value)
	tm.sum += value
	cutIndex := 0
	for i, t := range tm.times {
		if timestamp.Sub(t) > tm.lookback {
			cutIndex = i
			tm.sum -= tm.values[i]
		} else {
			break
		}
	}
	//需要offset 1
	cutIndex += 1
	if cutIndex > 0 {
		tm.values = tm.values[cutIndex:]
		tm.times = tm.times[cutIndex:]
	}
	return tm.sum
}

func (tm *TimedSum) Sum() float64 {
	return tm.sum
}
func (tm *TimedSum) Len() int {
	return len(tm.values)
}
func (tm *TimedSum) Range() time.Duration {
	if len(tm.times) > 2 {
		return tm.times[len(tm.times)-1].Sub(tm.times[0])
	}else{
		return time.Duration(0)
	}
}

func NewTimedSum(lookback time.Duration) *TimedSum {
	return &TimedSum{
		lookback: lookback,
		times:    make([]time.Time, 0),
		values:   make([]float64, 0),
		sum:      0,
	}
}

type TimedMean struct {
	lookback time.Duration
	times    []time.Time
	values   []float64
	sum      float64
	mean     float64
}

func (tm *TimedMean) Insert(timestamp time.Time, value float64) float64 {
	tm.times = append(tm.times, timestamp)
	tm.values = append(tm.values, value)
	tm.sum += value
	cutIndex := 0
	for i, t := range tm.times {
		if timestamp.Sub(t) > tm.lookback {
			cutIndex = i
			tm.sum -= tm.values[i]
		} else {
			break
		}
	}
	//需要offset 1
	cutIndex += 1
	if cutIndex > 0 {
		tm.values = tm.values[cutIndex:]
		tm.times = tm.times[cutIndex:]
	}
	if len(tm.values) > 0 {
		tm.mean = tm.sum / float64(len(tm.values))
	} else {
		tm.mean = 0
	}
	return tm.mean
}

func (tm *TimedMean) Mean() float64 {
	return tm.mean
}
func (tm *TimedMean) Len() int {
	return len(tm.values)
}
func (tm *TimedMean) Range() time.Duration {
	if len(tm.times) > 2 {
		return tm.times[len(tm.times)-1].Sub(tm.times[0])
	}else{
		return time.Duration(0)
	}
}

func NewTimedMean(lookback time.Duration) *TimedMean {
	return &TimedMean{
		lookback: lookback,
		times:    make([]time.Time, 0),
		values:   make([]float64, 0),
		sum:      0,
		mean:     0,
	}
}

type TimedWeightedMean struct {
	lookback time.Duration
	times    []time.Time
	values   []float64
	weights  []float64
	sum      float64
	weight   float64
	mean     float64
}

func (tm *TimedWeightedMean) Insert(timestamp time.Time, weight, value float64) float64 {
	tm.times = append(tm.times, timestamp)
	tm.weights = append(tm.weights, weight)
	tm.values = append(tm.values, value)
	tm.sum += value * weight
	tm.weight += weight
	cutIndex := 0
	for i, t := range tm.times {
		if timestamp.Sub(t) > tm.lookback {
			cutIndex = i
			tm.sum -= tm.values[i] * tm.weights[i]
		} else {
			break
		}
	}
	//需要offset 1
	cutIndex += 1
	if cutIndex > 0 {
		tm.values = tm.values[cutIndex:]
		tm.weights = tm.weights[cutIndex:]
		tm.times = tm.times[cutIndex:]
	}
	if tm.weight > 0 {
		tm.mean = tm.sum / tm.weight
	} else {
		tm.mean = 0
	}
	return tm.mean
}

func (tm *TimedWeightedMean) Mean() float64 {
	return tm.mean
}
func (tm *TimedWeightedMean) Len() int {
	return len(tm.values)
}

func (tm *TimedWeightedMean) Range() time.Duration {
	if len(tm.times) > 2 {
		return tm.times[len(tm.times)-1].Sub(tm.times[0])
	}else{
		return time.Duration(0)
	}
}


