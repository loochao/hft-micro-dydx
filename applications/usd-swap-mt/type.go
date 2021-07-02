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

	config       Config
	xOrderOffset Offset
	yOrderOffset Offset

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
	xDepthCh        chan common.Depth
	yDepthCh        chan common.Depth
	saveCh          chan *XYStrategy

	xSystemStatus common.SystemStatus
	ySystemStatus common.SystemStatus

	xPositionUpdateTime time.Time
	yPositionUpdateTime time.Time

	xyDepthMatchRatio  float64
	xyDepthMatchWindow float64
	xyDepthMatchSum    *common.RollingSum

	xDepth       common.Depth
	yDepth       common.Depth
	xNextDepth   common.Depth
	yNextDepth   common.Depth
	xDepthTime   time.Time
	yDepthTime   time.Time
	xDepthFilter common.TimeFilter
	yDepthFilter common.TimeFilter
	xWalkedDepth common.WalkedDepthBBMAA
	yWalkedDepth common.WalkedDepthBBMAA

	xLeverage float64
	yLeverage float64

	xAccount          common.Balance
	yAccount          common.Balance
	xPosition         common.Position
	yPosition         common.Position
	xOrderSilentTime  time.Time
	xCancelSilentTime time.Time
	yOrderSilentTime  time.Time
	yCancelSilentTime time.Time

	xLastFilledBuyPrice  *float64
	xLastFilledSellPrice *float64
	yLastFilledBuyPrice  *float64
	yLastFilledSellPrice *float64

	enterStep    float64
	usdAvailable float64

	logSilentTime        time.Time
	xWalkDepthTimer      *time.Timer
	yWalkDepthTimer      *time.Timer
	spreadWalkTimer      *time.Timer
	xOpenOrderCheckTimer *time.Timer //在订单提交之后定期check订单状态
	yOpenOrderCheckTimer *time.Timer //在订单提交之后定期check订单状态
	saveTimer            *time.Timer
	realisedSpreadTimer  *time.Timer
	spreadTime           time.Time
	spread               *XYSpread

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
	stateOutputCh           chan XYStrategy

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

	xSizeDiff float64
	ySizeDiff float64

	xyDelta float64
	yxDelta float64

	xSize          float64
	ySize          float64
	xValue         float64
	yValue         float64
	xAbsValue      float64
	yAbsValue      float64
	midPrice       float64
	enterValue     float64
	targetWeight   float64
	maxOrderValue  float64
	realisedSpread *float64

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

	xOpenOrder        *common.NewOrderParam
	yOpenOrder        *common.NewOrderParam
	xCancelOrderParam common.CancelOrderParam
	yCancelOrderParam common.CancelOrderParam

	XTimeDeltaEma      float64
	YTimeDeltaEma      float64
	XTimeDelta         float64
	YTimeDelta         float64
	XTickerFilterRatio float64
	YTickerFilterRatio float64
	XExpireRatio       float64
	YExpireRatio       float64

	stopped  int32
	tradeDir int
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

type XYSpread struct {
	XYLastEnter   float64
	XYMedianEnter float64
	YXLastEnter   float64
	YXMedianEnter float64
	EventTime     time.Time
	ParseTime     time.Time
}
