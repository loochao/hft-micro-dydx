package main

import (
	"github.com/geometrybase/hft-micro/common"
	"time"
)

type XYStrategy struct {
	xExchange common.UsdExchange

	xSymbol string

	config Config

	xAccountCh      chan common.Balance
	xPositionCh     chan common.Position
	xOrderRequestCh chan common.OrderRequest
	xOrderCh        chan common.Order
	xOrderErrorCh   chan common.OrderError
	xSystemStatusCh chan common.SystemStatus

	xSystemStatus common.SystemStatus

	xPositionUpdateTime time.Time

	xTickerCh chan common.Ticker

	xSize float64

	xValue float64

	xAbsValue float64

	xAccount        common.Balance
	xPosition       common.Position
	OrderSilentTime time.Time

	enterValue   float64

	logSilentTime time.Time

	xStepSize float64

	xMultiplier  float64
	xMinNotional float64
	xNextTicker  common.Ticker

	xTickerTime time.Time

	xTicker   common.Ticker
	xMidPrice float64

	error error

	isXSpot bool

	xOrder         common.Order
	xNewOrderParam common.NewOrderParam
	xOrderError    common.OrderError

	xOrderSize float64
	xOrderSide common.OrderSide

	stopped int32
}
