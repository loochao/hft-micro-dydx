package okexv5_usdtswap

import "github.com/geometrybase/hft-micro/common"

const (
	ExchangeID = common.OkexV5UsdtSwap

	TdModeCash     = "cash"
	TdModeIsolated = "isolated"
	TdModeCross    = "cross"

	OrderSideBuy  = "buy"
	OrderSideSell = "sell"

	OrderTypeMarket     = "market"
	OrderTypeLimit      = "limit"
	OrderTypePostOnly = "post_only"
	OrderTypeFOK = "fok"
	OrderTypeIOC = "ioc"
	TimeLayout = "2006-01-02T15:04:05.999Z"

	OrderStateLive = "live"
	OrderStateCanceled = "canceled"
	OrderStatePartiallyFilled = "partially_filled"
	OrderStateFilled = "filled"

	ServiceTypeWebsocket = 0
	ServiceTypeSpot      = 1
	ServiceTypeDeliver   = 2
	ServiceTypePerpetual = 3
	ServiceTypeOptions   = 4
	ServiceTypeTrade     = 5

	StateScheduled = "scheduled"
	StateOngoing   = "ongoing"
	StateCompleted = "completed"
	StateCanceled  = "canceled"
)
