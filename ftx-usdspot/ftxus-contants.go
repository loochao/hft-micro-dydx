package ftx_usdspot

import "github.com/geometrybase/hft-micro/common"

const (
	TimeLayout    = "2006-01-02T15:04:05-07:00"
	TradeSideBuy  = "buy"
	TradeSideSell = "sell"

	OrderSideBuy      = "buy"
	OrderSideSell     = "sell"
	OrderStatusOpen   = "open"
	OrderStatusNew    = "new"
	OrderStatusClosed = "closed"
	OrderTypeLimit    = "limit"
	OrderTypeMarket   = "market"
	ExchangeID        = common.FtxUsdFuture
)
