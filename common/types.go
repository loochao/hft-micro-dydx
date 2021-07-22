package common

import (
	"compress/gzip"
	"encoding/gob"
	"fmt"
	"math"
	"net/url"
	"os"
	"sort"
	"time"
)

const (
	JsonKeyUnknown uint8 = iota
	JsonKeyEventType
	JsonKeyEventTime
	JsonKeyTransactionTime
	JsonKeySymbol
	JsonKeyLastUpdateId
	JsonKeyBids
	JsonKeyAsks
	JsonKeyMarkPrice
	JsonKeyIndexPrice
	JsonKeyEstimatedSettlePrice
	JsonKeyFundingRate
	JsonKeyNextFundingTime
	JsonKeyStream
	JsonKeyPrice
	JsonKeyQuantity
	JsonKeySide
	JsonKeyTradeTime
	JsonKeyMRID
	JsonKeyID
	JsonKeyVersion
	JsonKeyBidPrice
	JsonKeyAskPrice
	JsonKeyBidSize
	JsonKeyAskSize
	JsonKeySize
)

const (
	OrderTimeInForceGTX  = "GTX"
	OrderRespTypeAck     = "ACK"
	OrderRespTypeResult  = "RESULT"
	OrderRespTypeFull    = "FULL"
	OrderIsIsolatedTrue  = "TRUE"
	OrderIsIsolatedFalse = "FALSE"

	OrderTypeStopLoss        = "STOP_LOSS"
	OrderTypeStopLossLimit   = "STOP_LOSS_LIMIT"
	OrderTypeTakeProfit      = "TAKE_PROFIT"
	OrderTypeTakeProfitLimit = "TAKE_PROFIT_LIMIT"
	OrderTypeLimitMarker     = "LIMIT_MAKER"

	TimeIntervalMinute         = "1m"
	TimeIntervalThreeMinutes   = "3m"
	TimeIntervalFiveMinutes    = "5m"
	TimeIntervalFifteenMinutes = "15m"
	TimeIntervalThirtyMinutes  = "30m"
	TimeIntervalHour           = "1h"
	TimeIntervalTwoHours       = "2h"
	TimeIntervalFourHours      = "4h"
	TimeIntervalSixHours       = "6h"
	TimeIntervalEightHours     = "8h"
	TimeIntervalTwelveHours    = "12h"
	TimeIntervalDay            = "1d"
	TimeIntervalThreeDays      = "3d"
	TimeIntervalWeek           = "1w"
	TimeIntervalMonth          = "1M"
)

type Params interface {
	ToUrlValues() url.Values
}

type ErrorCap struct {
	Code int64  `json:"code"`
	Msg  string `json:"msg"`
}

type Credentials struct {
	Key    string
	Secret string
}

type KLine struct {
	Symbol    string
	Open      float64
	Close     float64
	High      float64
	Low       float64
	Volume    float64
	Timestamp time.Time
}

func (kLine *KLine) ToString() string {
	return fmt.Sprintf("%s T%v O%f H%f L%f C%f V%f",
		kLine.Symbol,
		kLine.Timestamp, kLine.Open, kLine.High,
		kLine.Low, kLine.Close, kLine.Volume,
	)
}

type KLinesMap map[string][]KLine

func (m *KLinesMap) Load(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	g, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	decoder := gob.NewDecoder(g)
	err = decoder.Decode(m)
	if err != nil {
		return err
	}
	err = g.Close()
	if err != nil {
		return err
	}
	err = f.Close()
	if err != nil {
		return err
	}
	return nil
}

func (m *KLinesMap) Save(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	g := gzip.NewWriter(f)
	decoder := gob.NewEncoder(g)
	err = decoder.Encode(m)
	if err != nil {
		return err
	}
	err = g.Close()
	if err != nil {
		return err
	}
	err = f.Close()
	if err != nil {
		return err
	}
	return nil
}

// SortedFloatSlice assumes elements are sorted
type SortedFloatSlice []float64

// Insert into slice maintaining the sort order
func (f SortedFloatSlice) Insert(value float64) SortedFloatSlice {
	i := sort.SearchFloat64s(f, value)
	n := append(f, 0)
	copy(n[i+1:], n[i:])
	n[i] = value
	return n
}

// Delete from slice maintaining the sort order
func (f SortedFloatSlice) Delete(value float64) SortedFloatSlice {
	i := sort.SearchFloat64s(f, value)
	if i == len(f) {
		return f[:i]
	} else {
		return append(f[:i], f[i+1:]...)
	}
}

// Median of the slice
func (f SortedFloatSlice) Median() float64 {
	if len(f) > 0 {
		if len(f)%2 == 1 {
			return f[len(f)/2]
		}
		return (f[len(f)/2] + f[len(f)/2-1]) / 2
	} else {
		return math.NaN()
	}
}
func (f SortedFloatSlice) Min() float64 {
	if len(f) > 0 {
		return f[0]
	} else {
		return math.NaN()
	}
}
func (f SortedFloatSlice) Max() float64 {
	if len(f) > 0 {
		return f[len(f)-1]
	} else {
		return math.NaN()
	}
}

type Float64 float64

func (f Float64) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%.8f", f)), nil
}

type StringFloat string

func (f StringFloat) MarshalJSON() ([]byte, error) {
	return []byte(f), nil
}

type ZeroFundingRate struct {
	Symbol string
}

func (f *ZeroFundingRate) GetSymbol() string {
	return f.Symbol
}

func (f *ZeroFundingRate) GetFundingRate() float64 {
	return 0
}

func (f *ZeroFundingRate) GetNextFundingTime() time.Time {
	return time.Time{}
}
