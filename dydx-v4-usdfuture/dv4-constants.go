package dydx_v4_usdfuture

import "github.com/geometrybase/hft-micro/common"

const (
	DydxV4UsdFutureExchangeID = common.DydxV4UsdFuture

	IndexerRestURL = "https://indexer.dydx.trade"
	IndexerWsURL   = "wss://indexer.dydx.trade/v4/ws"
	ValidatorURL   = "https://dydx-ops-rpc.kingnodes.com:443"
	ChainID        = "dydx-mainnet-1"

	TimeLayout = "2006-01-02T15:04:05.000Z"

	OrderSideBuy  = "BUY"
	OrderSideSell = "SELL"

	OrderTypeLimit  = "LIMIT"
	OrderTypeMarket = "MARKET"

	OrderStatusOpen                 = "OPEN"
	OrderStatusFilled               = "FILLED"
	OrderStatusCanceled             = "CANCELED"
	OrderStatusBestEffortCanceled   = "BEST_EFFORT_CANCELED"
	OrderStatusPending              = "PENDING"
	OrderStatusUntriggered          = "UNTRIGGERED"
	OrderStatusBestEffortOpened     = "BEST_EFFORT_OPENED"

	OrderTimeInForceGTT = "GTT"
	OrderTimeInForceFOK = "FOK"
	OrderTimeInForceIOC = "IOC"

	WsChannelOrderbook   = "v4_orderbook"
	WsChannelTrades      = "v4_trades"
	WsChannelMarkets     = "v4_markets"
	WsChannelSubaccounts = "v4_subaccounts"

	DepthReadPoolSize = 1 << 13
	DepthReadMsgSize  = 1 << 12

	UserReadPoolSize = 1 << 10
	UserReadMsgSize  = 1 << 15
)
