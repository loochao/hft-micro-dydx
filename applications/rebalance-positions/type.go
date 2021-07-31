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
	xPositionCh     chan common.Position
	xOrderRequestCh chan common.OrderRequest
	xOrderCh        chan common.Order
	xOrderErrorCh   chan common.OrderError
	xSystemStatusCh chan common.SystemStatus
	ySystemStatusCh chan common.SystemStatus

	xSystemStatus common.SystemStatus
	ySystemStatus common.SystemStatus

	xPositionUpdateTime time.Time

	xLeverage float64

	xAccount         common.Balance
	xPosition        common.Position
	xOrderSilentTime time.Time

	orderValue   float64
	usdAvailable float64

	logSilentTime   time.Time

	xTimedPositionChange *common.TimedSum

	xTickSize    float64
	xStepSize    float64
	xMultiplier  float64
	xMinNotional float64

	error error

	isXSpot bool
	isYSpot bool

	xSizeDiff float64

	xOrder         common.Order
	xNewOrderParam common.NewOrderParam
	xOrderError    common.OrderError

	yOrder         common.Order
	yNewOrderParam common.NewOrderParam
	yOrderError    common.OrderError

	size       float64
	price      float64
	reduceOnly bool
	orderSide  common.OrderSide

	stopped                 int32
}
