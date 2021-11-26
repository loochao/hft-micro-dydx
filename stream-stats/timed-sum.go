package stream_stats

import "time"

type TimedSum struct {
	Lookback time.Duration
	Times    []time.Time
	Values   []float64
	Sum      float64
}

func (tm *TimedSum) Insert(timestamp time.Time, value float64) float64 {
	tm.Times = append(tm.Times, timestamp)
	tm.Values = append(tm.Values, value)
	tm.Sum += value
	cutIndex := -1
	for i, t := range tm.Times {
		if timestamp.Sub(t) > tm.Lookback {
			cutIndex = i
			tm.Sum -= tm.Values[i]
		} else {
			break
		}
	}
	//需要offset 1
	cutIndex += 1
	if cutIndex > 0 {
		tm.Values = tm.Values[cutIndex:]
		tm.Times = tm.Times[cutIndex:]
	}
	return tm.Sum
}

func (tm *TimedSum) Len() int {
	return len(tm.Values)
}

func (tm *TimedSum) Range() time.Duration {
	if len(tm.Times) > 2 {
		return tm.Times[len(tm.Times)-1].Sub(tm.Times[0])
	} else {
		return time.Duration(0)
	}
}

func LoadOrCreateTimeSum(path string, lookback time.Duration) {
	tm := &TimedSum{
		Lookback: lookback,
		Times:    make([]time.Time, 0),
		Values:   make([]float64, 0),
		Sum:      0,
	}

	td := NewTimedTDigestWithCompression(lookback, subInterval, compression)
}

func NewTimedSum(lookback time.Duration) *TimedSum {
	return &TimedSum{
		Lookback: lookback,
		Times:    make([]time.Time, 0),
		Values:   make([]float64, 0),
		Sum:      0,
	}
}
