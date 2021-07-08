package main

import (
	"github.com/geometrybase/hft-micro/common"
	"time"
)

type XYStrategy struct {
	xExchange common.UsdExchange
	yExchange common.UsdExchange

	xExchangeID common.ExchangeID
	yExchangeID common.ExchangeID

	xSymbol string
	ySymbol string

	config Config

	xAccountCh      chan common.Balance
	yAccountCh      chan common.Balance
	xPositionCh     chan common.Position
	yPositionCh     chan common.Position
	xOrderRequestCh chan common.OrderRequest
	yOrderRequestCh chan common.OrderRequest
	xOrderCh        chan common.Order
	yOrderCh        chan common.Order
	xOrderErrorCh   chan common.OrderError
	yOrderErrorCh   chan common.OrderError
	xSystemStatusCh chan common.SystemStatus
	ySystemStatusCh chan common.SystemStatus
	depthCh         chan common.Depth
	saveCh          chan *XYStrategy

	xSystemStatus common.SystemStatus
	ySystemStatus common.SystemStatus

	xPositionUpdateTime time.Time
	yPositionUpdateTime time.Time

	xDepth     common.Depth
	yDepth     common.Depth
	nextDepth  common.Depth
	xNextDepth common.Depth
	yNextDepth common.Depth
	xDepthTime time.Time
	yDepthTime time.Time

	xLeverage float64
	yLeverage float64

	xAccount               common.Balance
	yAccount               common.Balance
	xPosition              common.Position
	yPosition              common.Position
	xOrderSilentTime       time.Time
	yOrderSilentTime       time.Time
	updateTargetSilentTime time.Time

	logSilentTime time.Time
	saveTimer     *time.Timer

	stateOutputCh chan XYStrategy

	xTickSize            float64
	yTickSize            float64
	xStepSize            float64
	yStepSize            float64
	xMultiplier          float64
	yMultiplier          float64
	xMinNotional         float64
	yMinNotional         float64
	xyMergedSpotStepSize float64

	error error

	isXSpot bool
	isYSpot bool

	xSizeDiff float64
	ySizeDiff float64

	xSize        float64
	ySize        float64
	xFreeSize    float64
	yFreeSize    float64
	xAbsSize     float64
	yAbsSize     float64
	xyTargetSize *float64

	xValue    float64
	yValue    float64
	xAbsValue float64
	yAbsValue float64
	midPrice  float64

	xOrder         common.Order
	yOrder         common.Order
	xNewOrderParam common.NewOrderParam
	yNewOrderParam common.NewOrderParam
	xOrderError    common.OrderError
	yOrderError    common.OrderError

	size       float64
	price      float64
	reduceOnly bool
	orderSide  common.OrderSide

	stopped int32
}
