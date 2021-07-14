package kucoin_usdtspot

import (
	"github.com/geometrybase/hft-micro/common"
	"time"
)

const (
	CandleType1Min   = "1min"
	CandleType3Min   = "3min"
	CandleType5Min   = "5min"
	CandleType15Min  = "15min"
	CandleType30Min  = "30min"
	CandleType1Hour  = "1hour"
	CandleType4Hour  = "4hour"
	CandleType6Hour  = "6hour"
	CandleType8Hour  = "8hour"
	CandleType12Hour = "12hour"
	CandleType1Day   = "1day"
	CandleType1Week  = "1week"

	OrderStatusOpen  = "open"
	OrderStatusMatch = "match"
	OrderStatusDone  = "done"

	OrderTypeOpen     = "open"
	OrderTypeMatch    = "match"
	OrderTypeFilled   = "filled"
	OrderTypeCanceled = "canceled"
	OrderTypeUpdate   = "update"

	SystemStatusOpen       = "open"
	SystemStatusCancelOnly = "cancelonly"
	SystemStatusClose      = "close"

	TradeSideBuy  = "buy"
	TradeSideSell = "sell"
	ExchangeID    = common.KucoinUsdtSpot
)

var CandleTypeDurations = map[string]time.Duration{
	CandleType1Min:   time.Minute,
	CandleType3Min:   time.Minute * 3,
	CandleType5Min:   time.Minute * 5,
	CandleType15Min:  time.Minute * 15,
	CandleType30Min:  time.Minute * 30,
	CandleType1Hour:  time.Hour,
	CandleType4Hour:  time.Hour * 4,
	CandleType6Hour:  time.Hour * 6,
	CandleType8Hour:  time.Hour * 8,
	CandleType12Hour: time.Hour * 12,
	CandleType1Day:   time.Hour * 24,
	CandleType1Week:  time.Hour * 168,
}
