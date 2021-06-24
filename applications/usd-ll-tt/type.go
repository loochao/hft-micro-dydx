package main

import (
	"github.com/geometrybase/hft-micro/common"
	"time"
)

type XYStrategy struct {
	xExchange common.UsdExchange
	yExchange common.UsdExchange

	xSymbol  string
	ySymbol  string
	tradable bool
	isXSpot  bool
	isYSpot  bool

	params Config

	xLeverage float64
	yLeverage float64

	xTickSize            float64
	yTickSize            float64
	xStepSize            float64
	yStepSize            float64
	xMultiplier          float64
	yMultiplier          float64
	xMinNotional         float64
	yMinNotional         float64
	xyMergedSpotStepSize float64

	xAccountCh      chan common.Balance
	yAccountCh      chan common.Balance
	xPositionCh     chan common.Position
	yPositionCh     chan common.Position
	xFundingRateCh  chan common.FundingRate
	yFundingRateCh  chan common.FundingRate
	xOrderRequestCh chan common.OrderRequest
	yOrderRequestCh chan common.OrderRequest
	xOrderCh        chan common.Order
	yOrderCh        chan common.Order
	xOrderErrorCh   chan common.OrderError
	yOrderErrorCh   chan common.OrderError
	xSystemStatusCh chan common.SystemStatus
	ySystemStatusCh chan common.SystemStatus
	xDepthCh        chan common.Depth
	yDepthCh        chan common.Depth
	saveCh          chan *XYStrategy

	xSystemStatus common.SystemStatus
	ySystemStatus common.SystemStatus

	xPositionUpdateTime time.Time
	yPositionUpdateTime time.Time

	xDepth       common.Depth
	yDepth       common.Depth
	xNextDepth   common.Depth
	yNextDepth   common.Depth
	xDepthTime   time.Time
	yDepthTime   time.Time
	xDepthFilter common.TimeFilter
	yDepthFilter common.TimeFilter
	xWalkedDepth common.WalkedDepthBAM
	yWalkedDepth common.WalkedDepthBAM

	xAccount             common.Balance
	yAccount             common.Balance
	xPosition            common.Position
	yPosition            common.Position
	xOrderSilentTime     time.Time
	yOrderSilentTime     time.Time
	xFundingRate         common.FundingRate
	yFundingRate         common.FundingRate
	xyFundingRate        *float64
	xLastFilledBuyPrice  *float64
	xLastFilledSellPrice *float64
	yLastFilledBuyPrice  *float64
	yLastFilledSellPrice *float64
	xyEnterSilentTime    time.Time
	enterStep            float64
	enterTarget          float64
	enterScale           float64
	usdtAvailable        float64

	logSilentTime       time.Time
	xWalkDepthTimer     *time.Timer
	yWalkDepthTimer     *time.Timer
	spreadWalkTimer     *time.Timer
	saveTimer           *time.Timer
	realisedSpreadTimer *time.Timer
	hedgeYTimer         *time.Timer
	hedgeCounter        time.Duration
	spreadTime          time.Time
	spread              *common.XYSpread

	shortEnterTimedMedian *common.TimedMedian
	longEnterTimedMedian  *common.TimedMedian

	xTimedPositionChange *common.TimedSum
	yTimedPositionChange *common.TimedSum

	expectedChanSendingTime time.Duration
	depthMatchCount         int
	depthCount              int
	xDepthExpireCount       int
	yDepthExpireCount       int
	shortLastEnter          float64
	longLastEnter           float64
	adjustedAgeDiff         time.Duration
	spreadReport            *common.XYSpreadReport
	stateOutputCh           chan XYStrategy

	error error

	xSizeDiff float64
	ySizeDiff float64

	offsetFactor   float64
	offsetStep     float64
	shortTop       float64
	shortBot       float64
	longBot        float64
	longTop        float64
	xSize          float64
	ySize          float64
	xValue         float64
	yValue         float64
	xAbsValue      float64
	yAbsValue      float64
	midPrice       float64
	enterValue     float64
	targetValue    float64
	realisedSpread *float64

	xOrder         common.Order
	yOrder         common.Order
	xNewOrderParam common.NewOrderParam
	yNewOrderParam common.NewOrderParam
	xOrderError    common.OrderError
	yOrderError    common.OrderError

	size       float64
	reduceOnly bool
	orderSide  common.OrderSide
}
