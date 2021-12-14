package main

import (
	"github.com/geometrybase/hft-micro/common"
	stream_stats "github.com/geometrybase/hft-micro/stream-stats"
	"time"
)

type XYStrategy struct {
	xExchange common.UsdExchange

	xSymbol string
	ySymbol string

	config Config

	reduceOnly      bool
	xAccountCh      chan common.Balance
	xPositionCh     chan common.Position
	xFundingRateCh  chan common.FundingRate
	yFundingRateCh  chan common.FundingRate
	xOrderRequestCh chan common.OrderRequest
	xOrderCh        chan common.Order
	xOrderErrorCh   chan common.OrderError
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

	xAccount             common.Balance
	xPosition            common.Position
	xOrderSilentTime     time.Time
	xFundingRate         common.FundingRate
	yFundingRate         common.FundingRate
	xyFundingRate        *float64
	xAdjustedFundingRate *float64
	yAdjustedFundingRate *float64
	xLastFilledBuyPrice  *float64
	xLastFilledSellPrice *float64

	enterStep    float64
	enterTarget  float64
	usdAvailable float64

	logSilentTime       time.Time
	realisedSpreadTimer *time.Timer

	xTimedPositionChange *common.TimedSum

	tickerMatchCount int
	tickerCount      int

	spreadWalkTimer      *time.Timer
	spreadShortTimedMean *common.TimedMean
	spreadLongTimedMean  *common.TimedMean

	spreadReady       bool
	spreadTickerTime  time.Time
	spreadEventTime   time.Time
	spreadLastShort   float64
	spreadLastLong    float64
	spreadMedianShort float64
	spreadMedianLong  float64

	xTickSize    float64
	xStepSize    float64
	xMinSize     float64
	xMultiplier  float64
	yMultiplier  float64
	xMinNotional float64

	offsetFactor      float64
	offsetStep        float64
	thresholdShortTop float64
	thresholdShortBot float64
	thresholdLongBot  float64
	thresholdLongTop  float64

	maxPosValue  float64
	maxPosSize   float64
	maxOrderSize float64

	enterValue             float64
	targetWeight           float64
	targetValue            float64
	referenceXPrice        float64

	tdSpreadEnterOffset float64
	tdSpreadExitOffset  float64

	xOrder         common.Order
	xNewOrderParam common.NewOrderParam
	xOrderError    common.OrderError

	stopped int32

	fundingRateSettleSilent bool
	xFundingRateCheckTimer  *time.Timer

	xExchangeID common.ExchangeID
	yExchangeID common.ExchangeID

	lastEnterTime time.Time

	tdSpreadMiddle   float64
	tdSpreadShortTop float64
	tdSpreadLongBot  float64
	tdSpreadLongTop  float64
	tdSpreadExitBot  float64

	xFundingRateFactor *float64
	yFundingRateFactor *float64

	xSlippageTM     *stream_stats.TimedMean
	xSlippage       float64
	xSlippageTMPath string
}
