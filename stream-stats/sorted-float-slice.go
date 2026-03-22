package stream_stats

import (
	"math"
	"sort"
)

// SortedFloatSlice assumes elements are sorted
type SortedFloatSlice []float64

// Insert into slice maintaining the sort order
func (f SortedFloatSlice) Insert(value float64) SortedFloatSlice {
	i := sort.SearchFloat64s(f, value)
	n := append(f, 0)
	copy(n[i+1:], n[i:])
	n[i] = value
	return n
}

// Delete from slice maintaining the sort order
func (f SortedFloatSlice) Delete(value float64) SortedFloatSlice {
	i := sort.SearchFloat64s(f, value)
	if i == len(f) {
		return f[:i]
	} else {
		return append(f[:i], f[i+1:]...)
	}
}

// Median of the slice
func (f SortedFloatSlice) Median() float64 {
	if len(f) > 0 {
		if len(f)%2 == 1 {
			return f[len(f)/2]
		}
		return (f[len(f)/2] + f[len(f)/2-1]) / 2
	} else {
		return math.NaN()
	}
}
func (f SortedFloatSlice) Min() float64 {
	if len(f) > 0 {
		return f[0]
	} else {
		return math.NaN()
	}
}
func (f SortedFloatSlice) Max() float64 {
	if len(f) > 0 {
		return f[len(f)-1]
	} else {
		return math.NaN()
	}
}
