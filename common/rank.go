package common

import (
	"sort"
)

// ranker is a helper type for the rank function.
type ranker struct {
	f []float64 // Data to be ranked.
	r []int     // A list of indexes into f that reflects rank order after sorting.
}

func Rank(data []float64) []float64 {
	rank := ranker{}
	return rank.rank(data)
}

// ranker satisfies the sort.Interface without mutating the reference slice, f.
func (r ranker) Len() int           { return len(r.f) }
func (r ranker) Less(i, j int) bool { return r.f[r.r[i]] < r.f[r.r[j]] }
func (r ranker) Swap(i, j int)      { r.r[i], r.r[j] = r.r[j], r.r[i] }

// rank returns the sample ranks of the values in a vector. Ties (i.e.,
// equal values) are handled by ranking them as the mean rank of coequals.
func (r *ranker) rank(f []float64) []float64 {
	if len(f) == 0 {
		return nil
	}

	r.f = f
	if len(r.r) < len(f) {
		r.r = make([]int, len(f))
	} else {
		r.r = r.r[:len(f)]
	}

	for i := range r.r {
		r.r[i] = i
	}
	sort.Sort(r)
	rl := make([]float64, len(f))
	for i, j := range r.r {
		rl[j] = float64(i)
	}

	var (
		prev = r.f[r.r[0]]

		first int
		same  bool
	)
	for i, j := range r.r[1:] {
		if r.f[j] == prev {
			if !same {
				first = i
			}
			same = true
		} else if same {
			v := (rl[r.r[i]] + rl[r.r[first]]) / 2
			for k := first; k <= i; k++ {
				rl[r.r[k]] = v
			}
			same = false
		}
		prev = r.f[j]
	}

	return rl
}

