package main

import (
	"github.com/geometrybase/hft-micro/common"
	stream_stats "github.com/geometrybase/hft-micro/stream-stats"
	"time"
)

type XYStrategy struct {
	xExchange common.UsdExchange
	yExchange common.UsdExchange

	xSymbol string
	ySymbol string

	config Config

	reduceOnly      bool
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
	xyTickerCh      chan common.Ticker

	xSystemStatus common.SystemStatus
	ySystemStatus common.SystemStatus

	stats *stream_stats.XYSimplifiedTickerStats

	xPositionUpdateTime time.Time
	yPositionUpdateTime time.Time

	xTicker   common.Ticker
	yTicker   common.Ticker
	xMidPrice float64
	yMidPrice float64

	xNextTicker common.Ticker
	yNextTicker common.Ticker
	nextTicker  common.Ticker

	xTickerTime       time.Time
	yTickerTime       time.Time
	xTickerTimeDelta  time.Duration
	yTickerTimeDelta  time.Duration
	xyTickerTimeDelta time.Duration

	xLeverage float64
	yLeverage float64

	xAccount             common.Balance
	yAccount             common.Balance
	xPosition            common.Position
	yPosition            common.Position
	xOrderSilentTime     time.Time
	yOrderSilentTime     time.Time
	xFundingRate         common.FundingRate
	yFundingRate         common.FundingRate
	xyFundingRate        *float64
	xAdjustedFundingRate *float64
	yAdjustedFundingRate *float64
	xLastFilledBuyPrice  *float64
	xLastFilledSellPrice *float64
	yLastFilledBuyPrice  *float64
	yLastFilledSellPrice *float64

	enterStep    float64
	enterTarget  float64
	usdAvailable float64

	logSilentTime       time.Time
	realisedSpreadTimer *time.Timer

	xTimedPositionChange *common.TimedSum
	yTimedPositionChange *common.TimedSum

	tickerMatchCount int
	tickerCount      int

	spreadWalkTimer        *time.Timer
	spreadShortTimedMedian *common.TimedMedian
	spreadLongTimedMedian  *common.TimedMedian

	spreadReady       bool
	spreadTickerTime  time.Time
	spreadEventTime   time.Time
	spreadLastShort   float64
	spreadLastLong    float64
	spreadMedianShort float64
	spreadMedianLong  float64
	//strategyOutputCh  chan XYStrategy

	xTickSize        float64
	yTickSize        float64
	xStepSize        float64
	yStepSize        float64
	xMinSize         float64
	yMinSize         float64
	xMultiplier      float64
	yMultiplier      float64
	xMinNotional     float64
	yMinNotional     float64
	xyMergedStepSize float64

	//error error

	isXSpot bool
	isYSpot bool

	offsetFactor      float64
	offsetStep        float64
	thresholdShortTop float64
	thresholdShortBot float64
	thresholdLongBot  float64
	thresholdLongTop  float64

	maxPosValue  float64
	maxPosSize   float64
	maxOrderSize float64

	//xSize                  float64
	//ySize                  float64
	//xValue                 float64
	//yValue                 float64
	//xAbsValue              float64
	//yAbsValue              float64
	//xyMidPrice             float64

	enterValue             float64
	targetWeight           float64
	targetValue            float64
	realisedSpread         *float64
	referenceSpread        float64
	referenceXPrice        float64
	referenceYPrice        float64
	adjustedRealisedSpread *float64

	tdSpreadEnterOffset float64
	tdSpreadExitOffset  float64

	xOrder         common.Order
	yOrder         common.Order
	xNewOrderParam common.NewOrderParam
	yNewOrderParam common.NewOrderParam
	xOrderError    common.OrderError
	yOrderError    common.OrderError

	stopped int32

	fundingRateSettleSilent bool
	xFundingRateCheckTimer  *time.Timer
	yFundingRateCheckTimer  *time.Timer

	xExchangeID common.ExchangeID
	yExchangeID common.ExchangeID

	hedgeCheckTimer    *time.Timer
	hedgeCheckStopTime time.Time

	lastEnterTime time.Time

	tdSpreadMiddle   float64
	tdSpreadShortTop float64
	tdSpreadLongBot  float64
	tdSpreadLongTop  float64
	tdSpreadExitBot  float64

	xFundingRateFactor *float64
	yFundingRateFactor *float64

	xySuccessRatioTM *stream_stats.TimedMean
	xSlippageTM      *stream_stats.TimedMean
	ySlippageTM      *stream_stats.TimedMean

	xSlippage            float64
	ySlippage            float64
	xSlippageTMPath      string
	ySlippageTMPath      string
	xySuccessRatioTMPath string
}
