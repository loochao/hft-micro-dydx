package main

import (
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"strings"
	"time"
)

type XYStrategy struct {
	xExchange common.UsdExchange
	yExchange common.UsdExchange

	xSymbol string
	ySymbol string
	isXSpot bool
	isYSpot bool

	config Config

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
	xWalkedDepth common.WalkedDepthBMA
	yWalkedDepth common.WalkedDepthBMA

	xAccount         common.Balance
	yAccount         common.Balance
	xPosition        common.Position
	yPosition        common.Position
	xOrderSilentTime time.Time
	yOrderSilentTime time.Time
	xFundingRate     common.FundingRate
	yFundingRate     common.FundingRate

	xyFundingRate        *float64
	xLastFilledBuyPrice  *float64
	xLastFilledSellPrice *float64
	yLastFilledBuyPrice  *float64
	yLastFilledSellPrice *float64
	xyEnterSilentTime    time.Time
	enterStep            float64
	enterTarget          float64
	targetWeight         float64
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

	xyDepthMatchRatio  float64
	xyDepthMatchWindow float64
	xyDepthMatchSum    *common.RollingSum

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
	orderOffset             Offset

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

	fundingRateSettleSilent bool
	fundingRateSettleTimer  *time.Timer

	xOrder         common.Order
	yOrder         common.Order
	xNewOrderParam common.NewOrderParam
	yNewOrderParam common.NewOrderParam
	xOrderError    common.OrderError
	yOrderError    common.OrderError

	size       float64
	reduceOnly bool
	orderSide  common.OrderSide
	price      float64
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
	if len(splits) != 6 {
		return Offset{}, fmt.Errorf("bad offsets %s", msg)
	}
	offsets := [6]float64{}
	var err error
	for i, s := range splits {
		offsets[i], err = common.ParseFloat([]byte(s))
		if err != nil {
			return Offset{}, err
		}
	}
	return Offset{
		FarTop:  offsets[5],
		Top:     offsets[4],
		NearTop: offsets[3],
		NearBot: offsets[2],
		Bot:     offsets[1],
		FarBot:  offsets[0],
	}, nil
}
