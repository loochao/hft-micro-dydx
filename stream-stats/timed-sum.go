package stream_stats

import (
	"encoding/json"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"time"
)

type TimedSum struct {
	Lookback time.Duration
	Times    []time.Time
	Values   []float64
	Sum      float64
}

func (ts *TimedSum) Insert(timestamp time.Time, value float64) float64 {
	ts.Times = append(ts.Times, timestamp)
	ts.Values = append(ts.Values, value)
	ts.Sum += value
	cutIndex := -1
	for i, t := range ts.Times {
		if timestamp.Sub(t) > ts.Lookback {
			cutIndex = i
			ts.Sum -= ts.Values[i]
		} else {
			break
		}
	}
	//需要offset 1
	cutIndex += 1
	if cutIndex > 0 {
		ts.Values = ts.Values[cutIndex:]
		ts.Times = ts.Times[cutIndex:]
	}
	return ts.Sum
}

func (ts *TimedSum) Len() int {
	return len(ts.Values)
}

func (ts *TimedSum) Range() time.Duration {
	if len(ts.Times) > 2 {
		return ts.Times[len(ts.Times)-1].Sub(ts.Times[0])
	} else {
		return time.Duration(0)
	}
}

func (ts *TimedSum) Load(tsPath string) error {
	tsBytes, err := os.ReadFile(tsPath)
	if err != nil {
		return err
	} else {
		return json.Unmarshal(tsBytes, ts)
	}
}

func (ts *TimedSum) Save(tsPath string) error {
	tsBytes, err := json.Marshal(*ts)
	if err != nil {
		return err
	}
	tsFile, err := os.OpenFile(tsPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	_, err = tsFile.Write(tsBytes)
	if err != nil {
		return err
	}
	return tsFile.Close()
}

func LoadOrCreateTimeSum(tsPath string, lookback time.Duration) *TimedSum {
	ts := NewTimedSum(lookback)
	err := ts.Load(tsPath)
	if err != nil {
		logger.Debugf("ts.Load error %v", err)
		ts = NewTimedSum(lookback)
	}
	ts.Lookback = lookback
	return ts
}

func NewTimedSum(lookback time.Duration) *TimedSum {
	return &TimedSum{
		Lookback: lookback,
		Times:    make([]time.Time, 0),
		Values:   make([]float64, 0),
		Sum:      0,
	}
}
