package stream_stats

import (
	"bytes"
	"encoding/json"
	"github.com/geometrybase/hft-micro/tdigest"
	"time"
)

type TimedTDigest struct {
	Lookback       time.Duration      `json:"lookback,omitempty"`
	Compression    uint32             `json:"compression,omitempty"`
	SubInterval    time.Duration      `json:"subInterval,omitempty"`
	Times          []time.Time        `json:"times,omitempty"`
	SubTDs         []*tdigest.TDigest `json:"-"`
	SubTDStartTime *time.Time         `json:"subTDStartTime,omitempty"`
	SubTDEndTime   *time.Time         `json:"subTDEndTime,omitempty"`
	CurrentSubTD   *tdigest.TDigest   `json:"-"`
	RollingTD      *tdigest.TDigest   `json:"-"`
}

func (ttd *TimedTDigest) MarshalJSON() ([]byte, error) {
	var err error
	rollingTD := make([]byte, 0)
	currentSubTD := make([]byte, 0)
	if ttd.RollingTD != nil {
		rollingTD, err = ttd.RollingTD.AsBytes()
		if err != nil {
			return nil, err
		}
	}
	if ttd.CurrentSubTD != nil {
		currentSubTD, err = ttd.CurrentSubTD.AsBytes()
		if err != nil {
			return nil, err
		}
	}
	subTDs := make([][]byte, len(ttd.SubTDs))
	for i := range ttd.SubTDs {
		subTDs[i], err = ttd.SubTDs[i].AsBytes()
		if err != nil {
			return nil, err
		}
	}
	type Alias TimedTDigest
	return json.Marshal(&struct {
		SubTDs       [][]byte `json:"subTDs,omitempty"`
		CurrentSubTD []byte   `json:"currentSubTD,omitempty"`
		RollingTD    []byte   `json:"rollingTD,omitempty"`
		*Alias
	}{
		SubTDs:       subTDs,
		CurrentSubTD: currentSubTD,
		RollingTD:    rollingTD,
		Alias:        (*Alias)(ttd),
	})
}

func (ttd *TimedTDigest) UnmarshalJSON(data []byte) error {
	type Alias TimedTDigest
	aux := &struct {
		SubTDs       [][]byte `json:"subTDs,omitempty"`
		CurrentSubTD []byte   `json:"currentSubTD,omitempty"`
		RollingTD    []byte   `json:"rollingTD,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(ttd),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	} else {
		ttd.CurrentSubTD, err = tdigest.FromBytes(bytes.NewReader(aux.CurrentSubTD))
		if err != nil {
			return err
		}
		ttd.RollingTD, err = tdigest.FromBytes(bytes.NewReader(aux.RollingTD))
		if err != nil {
			return err
		}
		ttd.SubTDs = make([]*tdigest.TDigest, len(aux.SubTDs))
		for i := range aux.SubTDs {
			ttd.SubTDs[i], err = tdigest.FromBytes(bytes.NewReader(aux.SubTDs[i]))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (ttd *TimedTDigest) Insert(timestamp time.Time, value float64) (err error) {
	ttd.SubTDEndTime = &timestamp
	if ttd.SubTDStartTime == nil {
		//第一次添加数据,以此为起点
		ttd.SubTDStartTime = &timestamp
	} else {
		if timestamp.Sub(*ttd.SubTDStartTime) >= ttd.SubInterval {
			//需要forward sub td
			ttd.Times = append(ttd.Times, *ttd.SubTDStartTime)
			ttd.SubTDs = append(ttd.SubTDs, ttd.CurrentSubTD)
			ttd.SubTDStartTime = &timestamp
			ttd.CurrentSubTD, _ = tdigest.New(tdigest.Compression(ttd.Compression))
		}
	}

	cutIndex := -1
	for i, t := range ttd.Times {
		if timestamp.Sub(t) > ttd.Lookback {
			cutIndex = i
		} else {
			break
		}
	}
	cutIndex += 1
	if cutIndex > 0 {
		ttd.SubTDs = ttd.SubTDs[cutIndex:]
		ttd.Times = ttd.Times[cutIndex:]
		ttd.RollingTD, _ = tdigest.New(tdigest.Compression(ttd.Compression))
		for _, td := range ttd.SubTDs {
			err = ttd.RollingTD.Merge(td)
			if err != nil {
				return
			}
		}
		err = ttd.RollingTD.Merge(ttd.CurrentSubTD)
		if err != nil {
			return
		}
	}
	err = ttd.CurrentSubTD.Add(value)
	if err != nil {
		return
	}
	err = ttd.RollingTD.Add(value)
	return
}
func (ttd *TimedTDigest) Len() int {
	return len(ttd.Times)
}
func (ttd *TimedTDigest) Range() time.Duration {
	if ttd.SubTDEndTime != nil && ttd.SubTDStartTime != nil {
		if len(ttd.Times) > 0 {
			return ttd.SubTDEndTime.Sub(ttd.Times[0])
		} else {
			return ttd.SubTDEndTime.Sub(*ttd.SubTDStartTime)
		}
	} else {
		return time.Duration(0)
	}
}
func (ttd *TimedTDigest) Quantile(q float64) float64 {
	return ttd.RollingTD.Quantile(q)
}
func NewTimedTDigest(lookback, subInterval time.Duration) *TimedTDigest {
	rollingTD, _ := tdigest.New()
	subTD, _ := tdigest.New()
	return &TimedTDigest{
		CurrentSubTD: subTD,
		RollingTD:    rollingTD,
		Lookback:     lookback,
		SubInterval:  subInterval,
		Compression:  100,
		Times:        make([]time.Time, 0),
		SubTDs:       make([]*tdigest.TDigest, 0),
	}
}

func NewTimedTDigestWithCompression(lookback, subInterval time.Duration, compression uint32) *TimedTDigest {
	rollingTD, _ := tdigest.New(tdigest.Compression(compression))
	subTD, _ := tdigest.New(tdigest.Compression(compression))
	return &TimedTDigest{
		Compression:  compression,
		CurrentSubTD: subTD,
		RollingTD:    rollingTD,
		Lookback:     lookback,
		SubInterval:  subInterval,
		Times:        make([]time.Time, 0),
		SubTDs:       make([]*tdigest.TDigest, 0),
	}
}
