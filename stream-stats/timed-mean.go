package stream_stats

import (
	"encoding/json"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"time"
)

type TimedMean struct {
	Lookback time.Duration `json:"lookback"`
	Times    []time.Time   `json:"times"`
	Values   []float64     `json:"values"`
	Sum      float64       `json:"sum"`
	Mean     float64       `json:"mean"`
}

func (tm *TimedMean) Insert(timestamp time.Time, value float64) float64 {
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
	if len(tm.Values) > 0 {
		tm.Mean = tm.Sum / float64(len(tm.Values))
	} else {
		tm.Mean = 0
	}
	return tm.Mean
}

func (tm *TimedMean) Len() int {
	return len(tm.Values)
}

func (tm *TimedMean) Range() time.Duration {
	if len(tm.Times) > 2 {
		return tm.Times[len(tm.Times)-1].Sub(tm.Times[0])
	} else {
		return time.Duration(0)
	}
}

func (tm *TimedMean) Load(tsPath string) error {
	tsBytes, err := os.ReadFile(tsPath)
	if err != nil {
		return err
	} else {
		return json.Unmarshal(tsBytes, tm)
	}
}

func (tm *TimedMean) Save(tsPath string) error {
	tsBytes, err := json.Marshal(*tm)
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

func LoadOrCreateTimeMean(tmPath string, lookback time.Duration) *TimedMean {
	tm := NewTimedMean(lookback)
	err := tm.Load(tmPath)
	if err != nil {
		logger.Debugf("tm.Load %s error %v", tmPath, err)
		tm = NewTimedMean(lookback)
	}
	tm.Lookback = lookback
	return tm
}

func NewTimedMean(lookback time.Duration) *TimedMean {
	return &TimedMean{
		Lookback: lookback,
		Times:    make([]time.Time, 0),
		Values:   make([]float64, 0),
		Sum:      0,
		Mean:     0,
	}
}
