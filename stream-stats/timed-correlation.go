package stream_stats

import (
	"time"
)

type TimedCorrelation struct {
	lookback   time.Duration
	times      []time.Time
	xs         []float64
	ys         []float64
	sumX       float64
	sumY       float64
	sumXY      float64
	covariance float64
}

func (tm *TimedCorrelation) Insert(timestamp time.Time, x, y float64) float64 {
	tm.times = append(tm.times, timestamp)
	tm.xs = append(tm.xs, x)
	tm.ys = append(tm.ys, y)
	tm.sumX += x
	tm.sumY += y
	tm.sumXY += x * y
	cutIndex := -1
	for i, t := range tm.times {
		if timestamp.Sub(t) > tm.lookback {
			cutIndex = i
		} else {
			break
		}
	}
	cutIndex += 1
	if cutIndex > 0 {
		for i, xi := range tm.xs[:cutIndex] {
			yi := tm.ys[i]
			tm.sumX -= xi
			tm.sumY -= yi
			tm.sumXY -= xi * yi
		}
		tm.xs = tm.xs[cutIndex:]
		tm.ys = tm.ys[cutIndex:]
		tm.times = tm.times[cutIndex:]
	}
	if len(tm.xs) > 0 {
		tm.covariance = tm.sumXY/float64(len(tm.xs)) - tm.sumX/float64(len(tm.xs))*tm.sumY/float64(len(tm.xs))
	}
	return tm.covariance
}

func (tm *TimedCorrelation) Values() ([]float64, []float64) {
	return tm.xs, tm.ys
}

func (tm *TimedCorrelation) Times() []time.Time {
	return tm.times
}

func (tm *TimedCorrelation) Len() int {
	return len(tm.times)
}

func (tm *TimedCorrelation) Range() time.Duration {
	if len(tm.times) > 2 {
		return tm.times[len(tm.times)-1].Sub(tm.times[0])
	} else {
		return time.Duration(0)
	}
}

func (tm *TimedCorrelation) Correlation() float64 {
	return tm.covariance
}

func NewTimedCorrelation(lookback time.Duration) *TimedCorrelation {
	return &TimedCorrelation{
		lookback: lookback,
		times:    make([]time.Time, 0),
		xs:       make([]float64, 0),
		ys:       make([]float64, 0),
	}
}
