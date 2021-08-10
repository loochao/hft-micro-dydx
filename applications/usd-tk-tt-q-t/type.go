package main

import (
	"github.com/geometrybase/hft-micro/common"
	stream_stats "github.com/geometrybase/hft-micro/stream-stats"
	"os"
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
	xFundingRateCh  chan common.FundingRate
	yFundingRateCh  chan common.FundingRate
	xOrderRequestCh chan common.OrderRequest
	xOrderCh        chan common.Order
	xOrderErrorCh   chan common.OrderError
	xSystemStatusCh chan common.SystemStatus
	ySystemStatusCh chan common.SystemStatus
	xyTickerCh      chan common.Ticker
	saveCh          chan *XYStrategy

	xSystemStatus common.SystemStatus
	ySystemStatus common.SystemStatus

	xPositionUpdateTime time.Time

	xTicker   common.Ticker
	yTicker   common.Ticker
	xMidPrice float64
	yMidPrice float64

	xNextTicker common.Ticker
	yNextTicker common.Ticker
	nextTicker  common.Ticker

	xTickerTime   time.Time
	yTickerTime   time.Time
	xTickerFilter common.TimeFilter
	yTickerFilter common.TimeFilter

	xLeverage float64

	xAccount         common.Balance
	xPosition        common.Position
	xOrderSilentTime time.Time
	xFundingRate     common.FundingRate
	yFundingRate     common.FundingRate
	xyFundingRate    *float64

	enterStep    float64
	enterTarget  float64
	usdAvailable float64

	logSilentTime   time.Time
	spreadWalkTimer *time.Timer
	saveTimer       *time.Timer
	spreadTime      time.Time
	spread          *common.XYSpread

	shortEnterTimedMedian *common.TimedMedian
	longEnterTimedMedian  *common.TimedMedian

	xTimedPositionChange *common.TimedSum

	expectedChanSendingTime time.Duration
	tickerMatchCount        int
	tickerCount             int
	xTickerExpireCount      int
	yTickerExpireCount      int
	shortLastEnter          float64
	longLastEnter           float64
	adjustedAgeDiff         time.Duration
	spreadReport            *common.XYSpreadReport
	stateOutputCh           chan XYStrategy

	xTickSize    float64
	xStepSize    float64
	xMultiplier  float64
	xMinNotional float64

	error error

	isXSpot bool
	isYSpot bool

	xSizeDiff float64

	offsetFactor           float64
	offsetStep             float64
	shortTop               float64
	shortHalfTop           float64
	shortBot               float64
	longBot                float64
	longHalfBot            float64
	longTop                float64
	xSize                  float64
	xValue                 float64
	xAbsValue              float64
	midPrice               float64
	enterValue             float64
	targetWeight           float64
	maxOrderValue          float64
	targetValue            float64
	realisedSpread         *float64
	adjustedRealisedSpread *float64

	xOrder         common.Order
	xNewOrderParam common.NewOrderParam
	xOrderError    common.OrderError

	size       float64
	price      float64
	reduceOnly bool
	orderSide  common.OrderSide

	stopped                 int32
	fundingRateSettleSilent bool
	fundingRateSettleTimer  *time.Timer

	xExchangeID common.ExchangeID
	yExchangeID common.ExchangeID

	lastSpreadEnterTime time.Time

	timedTDigest           *stream_stats.TimedTDigest
	quantileSaveTimer      *time.Timer
	quantileLastSampleTime time.Time
	quantileBytes          []byte
	quantileFile           *os.File
	quantileMiddle         *float64
}
