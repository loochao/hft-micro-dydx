package common

import "time"

type Float64Slice []float64
type TimeSlice []time.Time

type TimedFloat64s struct {
	lookback time.Duration
	times    []time.Time
	values   []float64
}

func (tm *TimedFloat64s) Insert(timestamp time.Time, value float64) []float64 {
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
	//需要offset 1
	cutIndex += 1
	if cutIndex > 0 {
		tm.values = tm.values[cutIndex:]
		tm.times = tm.times[cutIndex:]
	}
	return tm.values
}
func (tm *TimedFloat64s) Values() []float64 {
	return tm.values
}
func (tm *TimedFloat64s) Times() []time.Time {
	return tm.times
}
func (tm *TimedFloat64s) Len() int {
	return len(tm.values)
}
func (tm *TimedFloat64s) Range() time.Duration {
	if len(tm.times) > 2 {
		return tm.times[len(tm.times)-1].Sub(tm.times[0])
	} else {
		return time.Duration(0)
	}
}
func NewTimedFloat64s(lookback time.Duration) *TimedFloat64s {
	return &TimedFloat64s{
		lookback: lookback,
		times:    make([]time.Time, 0),
		values:   make([]float64, 0),
	}
}

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
	cutIndex := -1
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
	} else {
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
	cutIndex := -1
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
func (tm *TimedMean) Values() []float64 {
	return tm.values
}
func (tm *TimedMean) Times() []time.Time {
	return tm.times
}
func (tm *TimedMean) Len() int {
	return len(tm.values)
}
func (tm *TimedMean) Range() time.Duration {
	if len(tm.times) > 2 {
		return tm.times[len(tm.times)-1].Sub(tm.times[0])
	} else {
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

func NewTimedWeightedMean(lookback time.Duration) *TimedWeightedMean {
	return &TimedWeightedMean{
		lookback: lookback,
		times:    make([]time.Time, 0),
		values:   make([]float64, 0),
		weights:  make([]float64, 0),
		sum:      0,
		mean:     0,
		weight:   0,
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
	cutIndex := -1
	for i, t := range tm.times {
		if timestamp.Sub(t) > tm.lookback {
			cutIndex = i
			tm.sum -= tm.values[i] * tm.weights[i]
			tm.weight -= tm.weights[i]
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
	} else {
		return time.Duration(0)
	}
}

type VPIN struct {
	binSize   float64
	lastPrice float64
	lastDir   float64
	buy       float64
	sell      float64
	total     float64
	values    []float64
	imbalance float64
}

func (v *VPIN) Insert(size, price float64) float64 {
	//first trade default to buy
	if price > v.lastPrice {
		v.buy += size * price
		v.total += size * price
		v.values = append(v.values, size*price)
		v.lastDir = 1.0
	} else if price < v.lastPrice {
		v.sell += size * price
		v.total += size * price
		v.values = append(v.values, -size*price)
		v.lastDir = -1.0
	} else if v.lastDir >= 0 {
		v.buy += size * price
		v.total += size * price
		v.values = append(v.values, size*price)
		v.lastDir = 1.0
	} else if v.lastDir < 0 {
		v.sell += size * price
		v.total += size * price
		v.values = append(v.values, -size*price)
		v.lastDir = -1.0
	}
	v.lastPrice = price
	cutIndex := -1
	for v.total > v.binSize {
		cutIndex++
		if v.values[cutIndex] >= 0 {
			v.total -= v.values[cutIndex]
			v.buy -= v.values[cutIndex]
		} else {
			v.total += v.values[cutIndex]
			v.sell += v.values[cutIndex]
		}
	}
	//需要offset 1
	cutIndex += 1
	if cutIndex > 0 {
		v.values = v.values[cutIndex:]
	}
	v.imbalance = (v.buy - v.sell) / v.total
	return v.imbalance
}
func (v *VPIN) Imbalance() float64 {
	return v.imbalance
}
func (v *VPIN) Len() int {
	return len(v.values)
}
func (v *VPIN) Values() []float64 {
	return v.values
}
func NewVPIN(binSize float64) *VPIN {
	return &VPIN{
		binSize: binSize,
		values:  make([]float64, 0),
	}
}

type TimedMedian struct {
	lookback         time.Duration
	times            []time.Time
	values           []float64
	median           float64
	sortedFloatSlice SortedFloatSlice
}

func (tm *TimedMedian) Insert(timestamp time.Time, value float64) float64 {
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
	tm.median = tm.sortedFloatSlice.Median()
	return tm.median
}

func (tm *TimedMedian) Values() []float64 {
	return tm.values
}

func (tm *TimedMedian) Times() []time.Time {
	return tm.times
}

func (tm *TimedMedian) Len() int {
	return len(tm.values)
}

func (tm *TimedMedian) Range() time.Duration {
	if len(tm.times) > 2 {
		return tm.times[len(tm.times)-1].Sub(tm.times[0])
	} else {
		return time.Duration(0)
	}
}

func (tm *TimedMedian) Median() float64 {
	return tm.median
}

func NewTimedMedian(lookback time.Duration) *TimedMedian {
	return &TimedMedian{
		lookback:         lookback,
		times:            make([]time.Time, 0),
		values:           make([]float64, 0),
		sortedFloatSlice: SortedFloatSlice{},
	}
}
