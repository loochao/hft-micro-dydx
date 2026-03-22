package stream_stats

import (
	"encoding/json"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"time"
)

type TimedWeightedMean struct {
	Lookback time.Duration `json:"lookback"`
	Times    []time.Time   `json:"times"`
	Values   []float64     `json:"values"`
	Weights  []float64     `json:"weights"`
	Sum      float64       `json:"sum"`
	Mean     float64       `json:"mean"`
	Weight   float64       `json:"weight"`
}

func (tm *TimedWeightedMean) Insert(timestamp time.Time, value, weight float64) float64 {
	tm.Times = append(tm.Times, timestamp)
	tm.Values = append(tm.Values, value)
	tm.Weights = append(tm.Weights, weight)
	tm.Sum += value * weight
	tm.Weight += weight
	cutIndex := -1
	for i, t := range tm.Times {
		if timestamp.Sub(t) > tm.Lookback {
			cutIndex = i
			tm.Sum -= tm.Values[i] * tm.Weights[i]
			tm.Weight -= tm.Weights[i]
		} else {
			break
		}
	}
	//需要offset 1
	cutIndex += 1
	if cutIndex > 0 {
		tm.Times = tm.Times[cutIndex:]
		tm.Values = tm.Values[cutIndex:]
		tm.Weights = tm.Weights[cutIndex:]
	}
	if tm.Weight != 0 {
		tm.Mean = tm.Sum / tm.Weight
	} else {
		tm.Mean = 0
	}
	return tm.Mean
}

func (tm *TimedWeightedMean) Len() int {
	return len(tm.Values)
}

func (tm *TimedWeightedMean) Range() time.Duration {
	if len(tm.Times) > 2 {
		return tm.Times[len(tm.Times)-1].Sub(tm.Times[0])
	} else {
		return time.Duration(0)
	}
}

func (tm *TimedWeightedMean) Load(tsPath string) error {
	tsBytes, err := os.ReadFile(tsPath)
	if err != nil {
		return err
	} else {
		return json.Unmarshal(tsBytes, tm)
	}
}

func (tm *TimedWeightedMean) Save(tsPath string) error {
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

func LoadOrCreateTimedWeightedMean(tmPath string, lookback time.Duration) *TimedWeightedMean {
	tm := NewTimedWeightedMean(lookback)
	err := tm.Load(tmPath)
	if err != nil {
		logger.Debugf("tm.Load %s error %v", tmPath, err)
		tm = NewTimedWeightedMean(lookback)
	}
	tm.Lookback = lookback
	return tm
}

func NewTimedWeightedMean(lookback time.Duration) *TimedWeightedMean {
	return &TimedWeightedMean{
		Lookback: lookback,
		Times:    make([]time.Time, 0),
		Values:   make([]float64, 0),
		Weights:  make([]float64, 0),
		Weight:   0,
		Sum:      0,
		Mean:     0,
	}
}
