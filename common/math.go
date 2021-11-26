package common

import (
	"math"
	"time"
)

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

type TimedWalkingDistance struct {
	lookback        time.Duration
	times           []time.Time
	offsets         []float64
	walkingDistance float64
	lastValue       *float64
	lastOffset      float64
}

func (twd *TimedWalkingDistance) Insert(timestamp time.Time, value float64) float64 {
	if twd.lastValue == nil {
		twd.lastValue = new(float64)
		*twd.lastValue = value
		twd.walkingDistance = 0
		return twd.walkingDistance
	}
	twd.lastOffset = math.Abs(value - *twd.lastValue)
	twd.times = append(twd.times, timestamp)
	twd.offsets = append(twd.offsets, twd.lastOffset)
	twd.walkingDistance += twd.lastOffset
	cutIndex := -1
	for i, t := range twd.times {
		if timestamp.Sub(t) > twd.lookback {
			cutIndex = i
			twd.walkingDistance -= twd.offsets[i]
		} else {
			break
		}
	}
	//需要offset 1
	cutIndex += 1
	if cutIndex > 0 {
		twd.offsets = twd.offsets[cutIndex:]
		twd.times = twd.times[cutIndex:]
	}
	return twd.walkingDistance
}
func (twd *TimedWalkingDistance) WalkDistance() float64 {
	return twd.walkingDistance
}
func (twd *TimedWalkingDistance) Len() int {
	return len(twd.offsets)
}
func (twd *TimedWalkingDistance) Range() time.Duration {
	if len(twd.times) > 2 {
		return twd.times[len(twd.times)-1].Sub(twd.times[0])
	} else {
		return time.Duration(0)
	}
}
func NewTimedWalkingDistance(lookback time.Duration) *TimedWalkingDistance {
	return &TimedWalkingDistance{
		lookback:        lookback,
		times:           make([]time.Time, 0),
		offsets:         make([]float64, 0),
		walkingDistance: 0,
	}
}

type RollingSum struct {
	values []float64
	window int
	index  int
	sum    float64
}

func (tm *RollingSum) Insert(value float64) float64 {
	tm.sum += value
	tm.index++
	if tm.index == tm.window {
		tm.index = 0
	}
	tm.sum -= tm.values[tm.index]
	tm.values[tm.index] = value
	return tm.sum
}
func (tm *RollingSum) Sum() float64 {
	return tm.sum
}
func (tm *RollingSum) Values() []float64 {
	return tm.values
}
func NewRollingSum(window int) *RollingSum {
	return &RollingSum{
		values: make([]float64, window),
		index:  -1,
		window: window,
		sum:    0,
	}
}

func EaseInExpo(x float64) float64 {
	if x == 0 {
		return 0
	} else {
		return math.Pow(2, 10*x-10)
	}
}
