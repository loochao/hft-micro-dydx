package stream_stats

import (
	"encoding/json"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"time"
)

type TimedMeanWithRingBuffer struct {
	Lookback    int64               `json:"lookback"`
	Times       *common.Int64Ring   `json:"times"`
	Values      *common.Float64Ring `json:"values"`
	Sum         float64             `json:"sum"`
	Mean        float64             `json:"mean"`
	CurrentTS   int64               `json:"currentTS"`
	ContentSize int                 `json:"ContentSize"`
}

func (tm *TimedMeanWithRingBuffer) Insert(timestamp time.Time, value float64) float64 {
	tm.CurrentTS = timestamp.UnixNano()
	tm.Times.Enqueue(tm.CurrentTS)
	tm.Values.Enqueue(value)
	tm.Sum += value
	peekTS := tm.Times.Peek()
	for peekTS != nil && tm.CurrentTS-*peekTS > tm.Lookback {
		tm.Times.Dequeue()
		v := tm.Values.Dequeue()
		tm.Sum -= *v
		peekTS = tm.Times.Peek()
	}
	tm.ContentSize = tm.Times.ContentSize()
	if tm.ContentSize > 0 {
		tm.Mean = tm.Sum / float64(tm.ContentSize)
	} else {
		tm.Mean = 0
	}
	return tm.Mean
}

func (tm *TimedMeanWithRingBuffer) Len() int {
	return tm.ContentSize
}

func (tm *TimedMeanWithRingBuffer) Range() time.Duration {
	tailTs := tm.Times.Peek()
	if tailTs != nil {
		return time.Duration(tm.CurrentTS - *tailTs)
	} else {
		return time.Duration(0)
	}
}

func (tm *TimedMeanWithRingBuffer) Load(tsPath string) error {
	tsBytes, err := os.ReadFile(tsPath)
	if err != nil {
		return err
	} else {
		return json.Unmarshal(tsBytes, tm)
	}
}

func (tm *TimedMeanWithRingBuffer) Save(tsPath string) error {
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

func LoadOrCreateTimeMeanWithRingBuffer(tmPath string, lookback time.Duration) *TimedMeanWithRingBuffer {
	tm := NewTimedMeanWithRingBuffer(lookback)
	err := tm.Load(tmPath)
	if err != nil {
		logger.Debugf("tm.Load %s error %v", tmPath, err)
		tm = NewTimedMeanWithRingBuffer(lookback)
	}
	tm.Lookback = lookback.Nanoseconds()
	return tm
}

func NewTimedMeanWithRingBuffer(lookback time.Duration) *TimedMeanWithRingBuffer {
	return &TimedMeanWithRingBuffer{
		Lookback: lookback.Nanoseconds(),
		Times:    common.NewInt64Ring(1024),
		Values:   common.NewFloat64Ring(1024),
	}
}
