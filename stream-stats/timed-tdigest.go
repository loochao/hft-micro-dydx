package stream_stats

import (
	"github.com/geometrybase/hft-micro/tdigest"
	"time"
)

type TimedTDigest struct {
	lookback       time.Duration
	subInterval    time.Duration
	times          []time.Time
	subTDs         []*tdigest.TDigest
	currentSubTD   *tdigest.TDigest
	subTDStartTime *time.Time
	subTDEndTime   *time.Time
	rollingTD      *tdigest.TDigest
}

func (tm *TimedTDigest) Insert(timestamp time.Time, value float64) (err error) {
	tm.subTDEndTime = &timestamp
	if tm.subTDStartTime == nil {
		//第一次添加数据,以此为起点
		tm.subTDStartTime = &timestamp
	} else {
		if timestamp.Sub(*tm.subTDStartTime) >= tm.subInterval {
			//需要forward sub td
			tm.times = append(tm.times, *tm.subTDStartTime)
			tm.subTDs = append(tm.subTDs, tm.currentSubTD)
			tm.subTDStartTime = &timestamp
			tm.currentSubTD, _ = tdigest.New()
		}
	}

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
		tm.subTDs = tm.subTDs[cutIndex:]
		tm.times = tm.times[cutIndex:]
		tm.rollingTD, _ = tdigest.New()
		for _, td := range tm.subTDs {
			err = tm.rollingTD.Merge(td)
			if err != nil {
				return
			}
		}
		err = tm.rollingTD.Merge(tm.currentSubTD)
		if err != nil {
			return
		}
	}
	err = tm.currentSubTD.Add(value)
	if err != nil {
		return
	}
	err = tm.rollingTD.Add(value)
	return
}

func (tm *TimedTDigest) SubTDs() []*tdigest.TDigest {
	return tm.subTDs
}
func (tm *TimedTDigest) Times() []time.Time {
	return tm.times
}
func (tm *TimedTDigest) Len() int {
	return len(tm.times)
}
func (tm *TimedTDigest) Range() time.Duration {
	if tm.subTDEndTime != nil && tm.subTDStartTime != nil {
		if len(tm.times) > 0 {
			return tm.subTDEndTime.Sub(tm.times[0])
		} else {
			return tm.subTDEndTime.Sub(*tm.subTDStartTime)
		}
	} else {
		return time.Duration(0)
	}
}
func (tm *TimedTDigest) Quantile(q float64) float64 {
	return tm.rollingTD.Quantile(q)
}
func NewTimedTDigest(lookback, subInterval time.Duration) *TimedTDigest {
	rollingTD, _ := tdigest.New()
	subTD, _ := tdigest.New()
	return &TimedTDigest{
		currentSubTD: subTD,
		rollingTD:    rollingTD,
		lookback:     lookback,
		subInterval:  subInterval,
		times:        make([]time.Time, 0),
		subTDs:       make([]*tdigest.TDigest, 0),
	}
}
