package bybit_usdtfuture

import "github.com/geometrybase/hft-micro/common"

const (
	SymbolStatusTrading = "Trading"
	ExchangeID          = common.BybitUsdtFuture
	PositionSideBuy     = "Buy"
	PositionSideSell    = "Sell"

	OrderTypeLimit  = "Limit"
	OrderTypeMarket = "Market"
	OrderSideBuy    = "Buy"
	OrderSideSell   = "Sell"

	TimeInForceGoodTillCancel    = "GoodTillCancel"
	TimeInForceImmediateOrCancel = "ImmediateOrCancel"
	TimeInForceFillOrKill        = "FillOrKill"
	TimeInForcePostOnly          = "PostOnly"

	OrderStatusCreated         = "Created"
	OrderStatusRejected        = "Rejected"
	OrderStatusNew             = "New"
	OrderStatusPartiallyFilled = "PartiallyFilled"
	OrderStatusFilled          = "Filled"
	OrderStatusCancelled       = "Cancelled"
	OrderStatusPendingCancel   = "PendingCancel"
)
