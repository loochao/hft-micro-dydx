package main

import (
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"strings"
	"time"
)

type XYSpread struct {
	Age              time.Duration
	AgeDiff          time.Duration
	ShortLastEnter   float64
	ShortLastLeave   float64
	ShortMedianEnter float64
	ShortMedianLeave float64
	LongLastEnter    float64
	LongLastLeave    float64
	LongMedianEnter  float64
	LongMedianLeave  float64
	Time             time.Time
}

type SpreadReport struct {
	AgeDiff           time.Duration
	AdjustedAgeDiff   time.Duration
	MatchRatio        float64
	XDepthFilterRatio float64
	YDepthFilterRatio float64
	XTimeDeltaEma     float64
	YTimeDeltaEma     float64
	XTimeDelta        float64
	YTimeDelta        float64
	XMidPrice         float64
	YMidPrice         float64
	XSymbol           string
	YSymbol           string
	XExpireRatio      float64
	YExpireRatio      float64
}

type XYParams struct {
	dryRun   bool
	tradable bool

	isXSpot bool
	isYSpot bool

	logInterval time.Duration

	depthTakerImpact    float64
	depthXDecay         float64
	depthXBias          time.Duration
	depthYDecay         float64
	depthYBias          time.Duration
	depthMinTimeDelta   time.Duration
	depthMaxTimeDelta   time.Duration
	depthMaxAgeDiffBias time.Duration
	depthReportCount    int
	spreadLookback      time.Duration
	spreadTimeToLive    time.Duration
	spreadMinDepthCount int

	enterTargetFactor       float64
	enterMinimalStep        float64
	enterFreePct            float64
	enterScale              float64
	longEnterDelta          float64
	longExitDelta           float64
	shortEnterDelta         float64
	shortExitDelta          float64
	enterOffsetDelta        float64
	exitOffsetDelta         float64
	minimalKeepFundingRate  float64
	minimalEnterFundingRate float64

	xTickSize            float64
	yTickSize            float64
	xStepSize            float64
	yStepSize            float64
	xMultiplier          float64
	yMultiplier          float64
	xMinNotional         float64
	yMinNotional         float64
	xyMergedSpotStepSize float64

	turnoverLookback      time.Duration
	balancePositionMaxAge time.Duration
	enterSilent           time.Duration
	orderSilent           time.Duration
	cancelSilent          time.Duration
	saveInterval          time.Duration

	xLeverage float64
	yLeverage float64
}

type XYStrategy struct {
	xExchange common.Exchange
	yExchange common.Exchange

	xSymbol string
	ySymbol string

	params      XYParams
	orderOffset Offset

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
	xDepthTime   time.Time
	yDepthTime   time.Time
	xDepthFilter common.TimeFilter
	yDepthFilter common.TimeFilter
	xWalkedDepth common.WalkedDepthBAM
	yWalkedDepth common.WalkedDepthBAM

	xAccount                         common.Balance
	yAccount                         common.Balance
	xPosition                        common.Position
	yPosition                        common.Position
	xOrderSilentTime                 time.Time
	yOrderSilentTime                 time.Time
	xFundingRate                     common.FundingRate
	yFundingRate                     common.FundingRate
	xyFundingRate                    *float64
	xLastFilledBuyPrice              *float64
	xLastFilledSellPrice             *float64
	yLastFilledBuyPrice              *float64
	yLastFilledSellPrice             *float64
	//xyTargetSpotSizeUpdateSilentTime time.Time
	enterStep                        float64
	enterTarget                      float64
	usdtAvailable                    float64

	logSilentTime       time.Time
	xWalkDepthTimer     *time.Timer
	yWalkDepthTimer     *time.Timer
	saveTimer           *time.Timer
	realisedSpreadTimer *time.Timer
	spreadTime          time.Time
	spread              *XYSpread

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
	spreadReport            *SpreadReport
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
	price       float64
	reduceOnly bool
	orderSide  common.OrderSide

	xOpenOrder        *common.NewOrderParam
	xCancelOrderParam common.CancelOrderParam
}

type Offset struct {
	FarTop  float64
	Top     float64
	NearTop float64
	NearBot float64
	Bot     float64
	FarBot  float64
}

func NewOffset(msg string) (Offset, error) {
	splits := strings.Split(msg, ",")
	if len(splits) != 10 {
		return Offset{}, fmt.Errorf("bad offsets %s", msg)
	}
	offsets := [10]float64{}
	var err error
	for i, s := range splits {
		offsets[i], err = common.ParseFloat([]byte(s))
		if err != nil {
			return Offset{}, err
		}
	}
	return Offset{
		FarTop:  offsets[9],
		Top:     offsets[7],
		NearTop: offsets[5],
		NearBot: offsets[4],
		Bot:     offsets[2],
		FarBot:  offsets[0],
	}, nil
}
