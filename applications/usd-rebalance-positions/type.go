package main

import (
	"github.com/geometrybase/hft-micro/common"
	"time"
)

type XYStrategy struct {
	xExchange common.UsdExchange
	yExchange common.UsdExchange

	xSymbol string
	ySymbol string

	config Config

	xAccountCh      chan common.Balance
	yAccountCh      chan common.Balance
	xPositionCh     chan common.Position
	yPositionCh     chan common.Position
	xOrderRequestCh chan common.OrderRequest
	yOrderRequestCh chan common.OrderRequest
	yOrderCh        chan common.Order
	yOrderErrorCh   chan common.OrderError
	xOrderCh        chan common.Order
	xOrderErrorCh   chan common.OrderError
	xSystemStatusCh chan common.SystemStatus
	ySystemStatusCh chan common.SystemStatus

	xSystemStatus common.SystemStatus
	ySystemStatus common.SystemStatus

	xPositionUpdateTime time.Time
	yPositionUpdateTime time.Time

	xTickerCh chan common.Ticker
	yTickerCh chan common.Ticker

	xSize float64
	ySize float64

	xValue float64
	yValue float64

	xAbsValue float64
	yAbsValue float64

	xAccount        common.Balance
	xPosition       common.Position
	OrderSilentTime time.Time
	yAccount        common.Balance
	yPosition       common.Position

	enterValue   float64
	usdAvailable float64

	logSilentTime time.Time

	xStepSize float64
	yStepSize float64

	xMultiplier  float64
	yMultiplier  float64
	xMinNotional float64
	yMinNotional float64
	xNextTicker  common.Ticker
	yNextTicker  common.Ticker

	xTickerTime time.Time
	yTickerTime time.Time

	xTicker   common.Ticker
	yTicker   common.Ticker
	xMidPrice float64
	yMidPrice float64

	error error

	isXSpot bool
	isYSpot bool

	xOrder         common.Order
	xNewOrderParam common.NewOrderParam
	xOrderError    common.OrderError

	yOrder         common.Order
	yNewOrderParam common.NewOrderParam
	yOrderError    common.OrderError

	xOrderSize float64
	yOrderSize float64
	xOrderSide common.OrderSide
	yOrderSide common.OrderSide

	stopped int32
}
