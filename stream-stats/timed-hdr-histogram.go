package stream_stats

import (
	"encoding/json"
	"github.com/geometrybase/hft-micro/hdrhistogram"
	"time"
)

type TimedHdrHistogram struct {
	Lookback                       time.Duration             `json:"lookback,omitempty"`
	SubInterval                    time.Duration             `json:"subInterval,omitempty"`
	Times                          []time.Time               `json:"times,omitempty"`
	SubHists                       []*hdrhistogram.Histogram `json:"-"`
	SubHistStartTime               *time.Time                `json:"subHistStartTime,omitempty"`
	SubHistEndTime                 *time.Time                `json:"subHistEndTime,omitempty"`
	CurrentSubHist                 *hdrhistogram.Histogram   `json:"-"`
	RollingHist                    *hdrhistogram.Histogram   `json:"-"`
	LowestDiscernibleValue         int64                     `json:"lowestDiscernibleValue"`
	HighestTrackableValue          int64                     `json:"highestTrackableValue"`
	NumberOfSignificantValueDigits int                       `json:"numberOfSignificantValueDigits"`
}

func (hh *TimedHdrHistogram) MarshalJSON() ([]byte, error) {
	var err error
	rollingHist := make([]byte, 0)
	currentSubHist := make([]byte, 0)
	if hh.RollingHist != nil {
		rollingHist, err = hh.RollingHist.Encode(hdrhistogram.V2CompressedEncodingCookieBase)
		if err != nil {
			return nil, err
		}
	}
	if hh.CurrentSubHist != nil {
		currentSubHist, err = hh.CurrentSubHist.Encode(hdrhistogram.V2CompressedEncodingCookieBase)
		if err != nil {
			return nil, err
		}
	}
	subHists := make([][]byte, len(hh.SubHists))
	for i := range hh.SubHists {
		subHists[i], err = hh.SubHists[i].Encode(hdrhistogram.V2CompressedEncodingCookieBase)
		if err != nil {
			return nil, err
		}
	}
	type Alias TimedHdrHistogram
	return json.Marshal(&struct {
		SubHists       [][]byte `json:"subHists,omitempty"`
		CurrentSubHist []byte   `json:"currentSubHist,omitempty"`
		RollingHist    []byte   `json:"rollingHist,omitempty"`
		*Alias
	}{
		SubHists:       subHists,
		CurrentSubHist: currentSubHist,
		RollingHist:    rollingHist,
		Alias:          (*Alias)(hh),
	})
}

func (hh *TimedHdrHistogram) UnmarshalJSON(data []byte) error {
	type Alias TimedHdrHistogram
	aux := &struct {
		SubHists       [][]byte `json:"subHists,omitempty"`
		CurrentSubHist []byte   `json:"currentSubHist,omitempty"`
		RollingHist    []byte   `json:"rollingHist,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(hh),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	} else {

		hh.CurrentSubHist, err = hdrhistogram.Decode(aux.CurrentSubHist)
		if err != nil {
			return err
		}
		hh.RollingHist, err = hdrhistogram.Decode(aux.RollingHist)
		if err != nil {
			return err
		}
		hh.SubHists = make([]*hdrhistogram.Histogram, len(aux.SubHists))
		for i := range aux.SubHists {
			hh.SubHists[i], err = hdrhistogram.Decode(aux.SubHists[i])
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (hh *TimedHdrHistogram) Insert(timestamp time.Time, value int64) (err error) {
	hh.SubHistEndTime = &timestamp
	if hh.SubHistStartTime == nil {
		//第一次添加数据,以此为起点
		hh.SubHistStartTime = &timestamp
	} else {
		if timestamp.Sub(*hh.SubHistStartTime) >= hh.SubInterval {
			//需要forward sub td
			hh.Times = append(hh.Times, *hh.SubHistStartTime)
			hh.SubHists = append(hh.SubHists, hh.CurrentSubHist)
			hh.SubHistStartTime = &timestamp
			hh.CurrentSubHist = hdrhistogram.New(hh.LowestDiscernibleValue, hh.HighestTrackableValue, hh.NumberOfSignificantValueDigits)
		}
	}

	cutIndex := -1
	for i, t := range hh.Times {
		if timestamp.Sub(t) > hh.Lookback {
			cutIndex = i
		} else {
			break
		}
	}
	cutIndex += 1
	if cutIndex > 0 {
		hh.SubHists = hh.SubHists[cutIndex:]
		hh.Times = hh.Times[cutIndex:]
		hh.RollingHist = hdrhistogram.New(hh.LowestDiscernibleValue, hh.HighestTrackableValue, hh.NumberOfSignificantValueDigits)
		for _, td := range hh.SubHists {
			_ = hh.RollingHist.Merge(td)
		}
		_ = hh.RollingHist.Merge(hh.CurrentSubHist)
	}
	err = hh.CurrentSubHist.RecordValue(value)
	if err != nil {
		return
	}
	err = hh.RollingHist.RecordValue(value)
	return
}

func (hh *TimedHdrHistogram) Len() int {
	return len(hh.Times)
}

func (hh *TimedHdrHistogram) Range() time.Duration {
	if hh.SubHistEndTime != nil && hh.SubHistStartTime != nil {
		if len(hh.Times) > 0 {
			return hh.SubHistEndTime.Sub(hh.Times[0])
		} else {
			return hh.SubHistEndTime.Sub(*hh.SubHistStartTime)
		}
	} else {
		return time.Duration(0)
	}
}

func (hh *TimedHdrHistogram) Quantile(q float64) int64 {
	return hh.RollingHist.ValueAtPercentile(q)
}

func NewTimedHdrHistogram(
	lowestDiscernibleValue, highestTrackableValue int64,
	numberOfSignificantValueDigits int,
	lookback, subInterval time.Duration,
) *TimedHdrHistogram {
	return &TimedHdrHistogram{
		LowestDiscernibleValue:         lowestDiscernibleValue,
		HighestTrackableValue:          highestTrackableValue,
		NumberOfSignificantValueDigits: numberOfSignificantValueDigits,
		CurrentSubHist:                 hdrhistogram.New(lowestDiscernibleValue, highestTrackableValue, numberOfSignificantValueDigits),
		RollingHist:                    hdrhistogram.New(lowestDiscernibleValue, highestTrackableValue, numberOfSignificantValueDigits),
		Lookback:                       lookback,
		SubInterval:                    subInterval,
		Times:                          make([]time.Time, 0),
		SubHists:                       make([]*hdrhistogram.Histogram, 0),
	}
}
