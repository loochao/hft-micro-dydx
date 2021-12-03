package stream_stats

import (
	"encoding/json"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"time"
)

type TimedEMA struct {
	Lookback time.Duration `json:"lookback"`
	Ema      float64       `json:"value"`
	Period   float64       `json:"period"`
	LastTime time.Time     `json:"lastTime"`
}

func (tm *TimedEMA) Insert(timestamp time.Time, value float64) float64 {
	diff := timestamp.Sub(tm.LastTime)
	if diff > 0 {
		tm.Period *= 0.9995
		tm.Period += 0.0005 * (float64(tm.Lookback/diff) + 1)
		if tm.Period < 1 {
			tm.Period = 1.0
		}
		k := 2.0 / (tm.Period + 1.0)
		tm.Ema *= 1 - k
		tm.Ema += k * value
	}
	return tm.Ema
}

func (tm *TimedEMA) Len() int {
	return int(tm.Period)
}

func (tm *TimedEMA) Range() time.Duration {
	return tm.Lookback
}

func (tm *TimedEMA) Load(tsPath string) error {
	tsBytes, err := os.ReadFile(tsPath)
	if err != nil {
		return err
	} else {
		return json.Unmarshal(tsBytes, tm)
	}
}

func (tm *TimedEMA) Save(tsPath string) error {
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

func LoadOrCreateTimeEma(tmPath string, lookback time.Duration) *TimedEMA {
	tm := NewTimedEma(lookback)
	err := tm.Load(tmPath)
	if err != nil {
		logger.Debugf("tm.Load %s error %v", tmPath, err)
		tm = NewTimedEma(lookback)
	}
	tm.Lookback = lookback
	return tm
}

func NewTimedEma(lookback time.Duration) *TimedEMA {
	return &TimedEMA{
		Lookback: lookback,
	}
}
