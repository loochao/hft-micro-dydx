package main

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"sync/atomic"
	"time"
)

func startXYStrategy(
	ctx context.Context,
	xSymbol, ySymbol string,
	config Config,
	xOrderOffset Offset,
	yOrderOffset Offset,
	xExchange common.UsdExchange,
	yExchange common.UsdExchange,
	xAccountCh chan common.Balance,
	yAccountCh chan common.Balance,
	xPositionCh chan common.Position,
	yPositionCh chan common.Position,
	xOrderRequestCh chan common.OrderRequest,
	yOrderRequestCh chan common.OrderRequest,
	xOrderCh chan common.Order,
	yOrderCh chan common.Order,
	xOrderErrorCh chan common.OrderError,
	yOrderErrorCh chan common.OrderError,
	xSystemStatusCh chan common.SystemStatus,
	ySystemStatusCh chan common.SystemStatus,
	xDepthCh chan common.Depth,
	yDepthCh chan common.Depth,
	saveCh chan *XYStrategy,
) (err error) {

	xBiasInMs := float64(config.DepthXBias / time.Millisecond)
	yBiasInMs := float64(config.DepthYBias / time.Millisecond)
	minTimeDeltaInMs := float64(config.DepthMinTimeDelta / time.Millisecond)
	maxTimeDeltaInMs := float64(config.DepthMaxTimeDelta / time.Millisecond)

	strat := XYStrategy{
		xExchange:               xExchange,
		yExchange:               yExchange,
		xLeverage:               config.XExchange.Leverage,
		yLeverage:               config.YExchange.Leverage,
		xSymbol:                 xSymbol,
		ySymbol:                 ySymbol,
		targetWeight:            config.TargetWeights[xSymbol],
		maxOrderValue:           config.MaxOrderValues[xSymbol],
		xyDepthMatchSum:         common.NewRollingSum(config.DepthReportCount),
		xyDepthMatchWindow:      float64(config.DepthReportCount),
		xyDepthMatchRatio:       0.0,
		config:                  config,
		xOrderOffset:            xOrderOffset,
		yOrderOffset:            yOrderOffset,
		xAccountCh:              xAccountCh,
		yAccountCh:              yAccountCh,
		xPositionCh:             xPositionCh,
		yPositionCh:             yPositionCh,
		xOrderCh:                xOrderCh,
		yOrderCh:                yOrderCh,
		xOrderErrorCh:           xOrderErrorCh,
		yOrderErrorCh:           yOrderErrorCh,
		xOrderRequestCh:         xOrderRequestCh,
		yOrderRequestCh:         yOrderRequestCh,
		xSystemStatusCh:         xSystemStatusCh,
		ySystemStatusCh:         ySystemStatusCh,
		xDepthCh:                xDepthCh,
		yDepthCh:                yDepthCh,
		saveCh:                  saveCh,
		xPositionUpdateTime:     time.Time{},
		yPositionUpdateTime:     time.Time{},
		xDepth:                  nil,
		yDepth:                  nil,
		xDepthTime:              time.Time{},
		yDepthTime:              time.Time{},
		xDepthFilter:            common.NewTimeFilter(config.DepthXDecay, xBiasInMs, minTimeDeltaInMs, maxTimeDeltaInMs),
		yDepthFilter:            common.NewTimeFilter(config.DepthYDecay, yBiasInMs, minTimeDeltaInMs, maxTimeDeltaInMs),
		xWalkedDepth:            common.WalkedDepthBBMAA{},
		yWalkedDepth:            common.WalkedDepthBBMAA{},
		xAccount:                nil,
		yAccount:                nil,
		xPosition:               nil,
		yPosition:               nil,
		xOrderSilentTime:        time.Now().Add(config.EnterSilent),
		xCancelSilentTime:       time.Time{},
		yOrderSilentTime:        time.Time{},
		xLastFilledBuyPrice:     nil,
		xLastFilledSellPrice:    nil,
		yLastFilledBuyPrice:     nil,
		yLastFilledSellPrice:    nil,
		xOrder:                  nil,
		yOrder:                  nil,
		xOrderError:             common.OrderError{},
		yOrderError:             common.OrderError{},
		enterStep:               0,
		usdAvailable:            0,
		logSilentTime:           time.Time{},
		xWalkDepthTimer:         time.NewTimer(time.Hour * 9999),
		yWalkDepthTimer:         time.NewTimer(time.Hour * 9999),
		spreadWalkTimer:         time.NewTimer(time.Hour * 9999),
		realisedSpreadTimer:     time.NewTimer(time.Hour * 9999),
		xOpenOrderCheckTimer:    time.NewTimer(time.Hour * 9999),
		yOpenOrderCheckTimer:    time.NewTimer(time.Hour * 9999),
		saveTimer:               time.NewTimer(config.EnterSilent),
		spreadTime:              time.Time{},
		spread:                  nil,
		shortEnterTimedMedian:   common.NewTimedMedian(config.SpreadLookback),
		longEnterTimedMedian:    common.NewTimedMedian(config.SpreadLookback),
		xTimedPositionChange:    common.NewTimedSum(config.TurnoverLookback),
		yTimedPositionChange:    common.NewTimedSum(config.TurnoverLookback),
		expectedChanSendingTime: time.Nanosecond * 300,
		depthMatchCount:         0,
		depthCount:              0,
		xDepthExpireCount:       0,
		yDepthExpireCount:       0,
		shortLastEnter:          0,
		longLastEnter:           0,
		adjustedAgeDiff:         0,
		stateOutputCh:           nil,
		error:                   nil,
		xSizeDiff:               0,
		ySizeDiff:               0,
		xSize:                   0,
		ySize:                   0,
		xValue:                  0,
		yValue:                  0,
		xAbsValue:               0,
		yAbsValue:               0,
		midPrice:                0,
		enterValue:              0,
		size:                    0,
		orderSide:               common.OrderSideUnknown,
		xCancelOrderParam:       common.CancelOrderParam{Symbol: xSymbol},
		yCancelOrderParam:       common.CancelOrderParam{Symbol: ySymbol},
		stopped:                 0,
		tradeDir:                1,
	}
	strat.yTickSize, err = yExchange.GetTickSize(ySymbol)
	if err != nil {
		return
	}
	strat.yStepSize, err = yExchange.GetStepSize(ySymbol)
	if err != nil {
		return
	}
	strat.yMultiplier, err = yExchange.GetMultiplier(ySymbol)
	if err != nil {
		return
	}
	strat.yMinNotional, err = yExchange.GetMinNotional(ySymbol)
	if err != nil {
		return
	}

	strat.xTickSize, err = xExchange.GetTickSize(xSymbol)
	if err != nil {
		return
	}
	strat.xStepSize, err = xExchange.GetStepSize(xSymbol)
	if err != nil {
		return
	}
	strat.xMultiplier, err = xExchange.GetMultiplier(xSymbol)
	if err != nil {
		return
	}
	strat.xMinNotional, err = xExchange.GetMinNotional(xSymbol)
	if err != nil {
		return
	}
	strat.xyMergedSpotStepSize = common.MergedStepSize(strat.xStepSize*strat.xMultiplier, strat.yStepSize*strat.yMultiplier)

	go strat.startLoop(ctx)
	return
}

func (strat *XYStrategy) Stop() {
	if atomic.CompareAndSwapInt32(&strat.stopped, 0, 1) {
		logger.Debugf("stopped")
		strat.tryCancelXOpenOrder("end")
	}
}

func (strat *XYStrategy) startLoop(ctx context.Context) {
	defer strat.xWalkDepthTimer.Stop()
	defer strat.yWalkDepthTimer.Stop()
	defer strat.spreadWalkTimer.Stop()
	defer strat.realisedSpreadTimer.Stop()
	defer strat.saveTimer.Stop()
	defer strat.Stop()
	var nextXPos, nextYPos common.Position
	strat.xOpenOrder = &common.NewOrderParam{}
	strat.yOpenOrder = &common.NewOrderParam{}
	strat.tryCancelXOpenOrder("start")
	strat.tryCancelYOpenOrder("start")
	strat.xOrderSilentTime = time.Now().Add(strat.config.EnterSilent)
	strat.yOrderSilentTime = time.Now().Add(strat.config.EnterSilent)
	for {
		select {
		case <-ctx.Done():
			return
		case strat.xSystemStatus = <-strat.xSystemStatusCh:
			if strat.xSystemStatus != common.SystemStatusReady {
				strat.tryCancelXOpenOrder("xSystemStatus not ready")
			}
			break
		case strat.ySystemStatus = <-strat.ySystemStatusCh:
			if strat.ySystemStatus != common.SystemStatusReady {
				strat.tryCancelXOpenOrder("ySystemStatus not ready")
			}
			break
		case <-strat.saveTimer.C:
			strat.handleSave()
			break
		case <-strat.xOpenOrderCheckTimer.C:
			if strat.xOpenOrder != nil {
				if !strat.isXOpenOrderOk() {
					strat.tryCancelXOpenOrder("x open order not ok")
				}
				strat.xOpenOrderCheckTimer.Reset(strat.config.OrderCheckInterval)
			} else {
				strat.xOpenOrderCheckTimer.Reset(time.Hour * 9999)
			}
			break
		case <-strat.yOpenOrderCheckTimer.C:
			if strat.yOpenOrder != nil {
				if !strat.isYOpenOrderOk() {
					strat.tryCancelYOpenOrder("y open order not ok")
				}
				strat.yOpenOrderCheckTimer.Reset(strat.config.OrderCheckInterval)
			} else {
				strat.yOpenOrderCheckTimer.Reset(time.Hour * 9999)
			}
			break
		case strat.xAccount = <-strat.xAccountCh:
			strat.updateEnterStep()
			break
		case strat.yAccount = <-strat.yAccountCh:
			strat.updateEnterStep()
			break
		case nextXPos = <-strat.xPositionCh:
			strat.handleXPosition(nextXPos)
			break
		case nextYPos = <-strat.yPositionCh:
			strat.handleYPosition(nextYPos)
			break
		case strat.xOrder = <-strat.xOrderCh:
			strat.handleXOrder()
			break
		case strat.yOrder = <-strat.yOrderCh:
			strat.handleYOrder()
			break
		case strat.xOrderError = <-strat.xOrderErrorCh:
			strat.handleXOrderError()
			break
		case strat.yOrderError = <-strat.yOrderErrorCh:
			strat.handleYOrderError()
			break
		case <-strat.xWalkDepthTimer.C:
			strat.walkXDepth()
			break
		case <-strat.yWalkDepthTimer.C:
			strat.walkYDepth()
			break
		case <-strat.spreadWalkTimer.C:
			strat.walkSpread()
			break
		case strat.xNextDepth = <-strat.xDepthCh:
			strat.handleXDepth()
			break
		case strat.yNextDepth = <-strat.yDepthCh:
			strat.handleYDepth()
			break
		case <-strat.realisedSpreadTimer.C:
			strat.handleRealisedSpread()
			break
		}
	}
}
func (strat *XYStrategy) handleSave() {
	if strat.xSystemStatus != common.SystemStatusReady ||
		strat.ySystemStatus != common.SystemStatusReady ||
		strat.xPosition == nil ||
		strat.yPosition == nil ||
		strat.xAccount == nil ||
		strat.yAccount == nil {
		return
	}
	select {
	case strat.saveCh <- strat:
	default:
		logger.Debugf("strat.saveCh <- strat failed %s %s ch len %d", strat.xSymbol, strat.ySymbol, len(strat.saveCh))
	}
	strat.saveTimer.Reset(strat.config.InternalInflux.SaveInterval)
}

func (strat *XYStrategy) hedgeYPosition() {
	if strat.tradeDir <= 0 {
		return
	}
	if strat.xSystemStatus != common.SystemStatusReady ||
		strat.ySystemStatus != common.SystemStatusReady {
		if time.Now().Sub(strat.logSilentTime) > 0 {
			strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
			logger.Debugf("hedgeYPosition xSystemStatus %v ySystemStatus %v", strat.xSystemStatus, strat.ySystemStatus)
		}
		return
	}
	if strat.yPosition == nil ||
		strat.xPosition == nil ||
		strat.spread == nil ||
		time.Now().Sub(strat.yPositionUpdateTime) > strat.config.BalancePositionMaxAge ||
		time.Now().Sub(strat.yOrderSilentTime) < 0 {
		//if time.Now().Sub(strat.logSilentTime) > 0 {
		//	strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
		//	logger.Debugf("hedgeYPosition skipped order silent time %v positionUpdateTime %v", time.Now().Sub(strat.yOrderSilentTime), time.Now().Sub(strat.yPositionUpdateTime))
		//}
		return
	}
	strat.ySizeDiff = strat.yPosition.GetSize()
	if math.Abs(strat.ySizeDiff) < strat.yStepSize {
		return
	}
	//如y下单也加上控制，以限下单太大，造成市场冲击
	if strat.ySizeDiff*strat.yMultiplier > strat.maxOrderValue/strat.yWalkedDepth.MidPrice {
		strat.ySizeDiff = strat.maxOrderValue / strat.yWalkedDepth.MidPrice / strat.yMultiplier
	}
	strat.ySizeDiff = math.Floor(strat.ySizeDiff/strat.yStepSize) * strat.yStepSize

	if strat.ySizeDiff < strat.yStepSize {
		return
	} else if strat.ySizeDiff*strat.yMultiplier*strat.yWalkedDepth.MidPrice < strat.yMinNotional {
		return
	}

	strat.yNewOrderParam = common.NewOrderParam{
		Symbol:     strat.ySymbol,
		Side:       common.OrderSideSell,
		Type:       common.OrderTypeMarket,
		Size:       strat.ySizeDiff,
		ReduceOnly: true,
		ClientID:   strat.yExchange.GenerateClientID(),
	}
	if !strat.config.DryRun {
		//logger.Debugf("sending strat.yOrderRequestCh <- common.OrderRequest %s", strat.yNewOrderParam.Symbol)
		select {
		case strat.yOrderRequestCh <- common.OrderRequest{
			New: &strat.yNewOrderParam,
		}:
			//logger.Debugf("sent strat.yOrderRequestCh <- common.OrderRequest %s", strat.yNewOrderParam.Symbol)
			strat.yOrderSilentTime = time.Now().Add(strat.config.OrderSilent)
			strat.yPositionUpdateTime = time.Unix(0, 0)
		}
	} else {
		strat.yOrderSilentTime = time.Now().Add(strat.config.OrderSilent)
		strat.yPositionUpdateTime = time.Unix(0, 0)
	}
}
func (strat *XYStrategy) hedgeXPosition() {
	if strat.tradeDir >= 0 {
		return
	}
	if strat.xSystemStatus != common.SystemStatusReady ||
		strat.ySystemStatus != common.SystemStatusReady {
		if time.Now().Sub(strat.logSilentTime) > 0 {
			strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
			logger.Debugf("hedgeYPosition xSystemStatus %v ySystemStatus %v", strat.xSystemStatus, strat.ySystemStatus)
		}
		return
	}
	if strat.yPosition == nil ||
		strat.xPosition == nil ||
		strat.spread == nil ||
		time.Now().Sub(strat.xPositionUpdateTime) > strat.config.BalancePositionMaxAge ||
		time.Now().Sub(strat.xOrderSilentTime) < 0 {
		return
	}
	strat.xSizeDiff = strat.xPosition.GetSize()
	if math.Abs(strat.xSizeDiff) < strat.xStepSize {
		return
	}
	if strat.xSizeDiff*strat.xMultiplier > strat.maxOrderValue/strat.xWalkedDepth.MidPrice {
		strat.xSizeDiff = strat.maxOrderValue / strat.xWalkedDepth.MidPrice / strat.xMultiplier
	}
	strat.xSizeDiff = math.Floor(strat.xSizeDiff/strat.xStepSize) * strat.xStepSize

	if strat.xSizeDiff < strat.xStepSize {
		return
	} else if strat.xSizeDiff*strat.xMultiplier*strat.xWalkedDepth.MidPrice < strat.xMinNotional {
		return
	}
	strat.xNewOrderParam = common.NewOrderParam{
		Symbol:     strat.xSymbol,
		Side:       common.OrderSideSell,
		Type:       common.OrderTypeMarket,
		Size:       strat.xSizeDiff,
		ReduceOnly: true,
		ClientID:   strat.xExchange.GenerateClientID(),
	}
	if !strat.config.DryRun {
		//logger.Debugf("sending strat.yOrderRequestCh <- common.OrderRequest %s", strat.yNewOrderParam.Symbol)
		select {
		case strat.xOrderRequestCh <- common.OrderRequest{
			New: &strat.xNewOrderParam,
		}:
			//logger.Debugf("sent strat.yOrderRequestCh <- common.OrderRequest %s", strat.yNewOrderParam.Symbol)
			strat.xOrderSilentTime = time.Now().Add(strat.config.OrderSilent)
			strat.xPositionUpdateTime = time.Unix(0, 0)
		}
	} else {
		strat.xOrderSilentTime = time.Now().Add(strat.config.OrderSilent)
		strat.xPositionUpdateTime = time.Unix(0, 0)
	}
}

func (strat *XYStrategy) updateEnterStep() {
	if strat.xAccount == nil || strat.yAccount == nil {
		return
	}
	strat.enterStep = (strat.xAccount.GetFree() + strat.yAccount.GetFree()) * strat.config.EnterPct * strat.targetWeight
	if strat.enterStep < strat.config.EnterMinimalStep {
		strat.enterStep = strat.config.EnterMinimalStep
	}
	strat.usdAvailable = math.Min(strat.xAccount.GetFree()*strat.xLeverage, strat.yAccount.GetFree()*strat.yLeverage)
}

func (strat *XYStrategy) handleXPosition(nextPos common.Position) {
	if nextPos.GetSymbol() != strat.xSymbol {
		logger.Debugf("bad next position, symbol %s %s not match %v", nextPos.GetSymbol(), strat.xSymbol, nextPos)
		return
	}
	if strat.xPosition != nil {
		if strat.xPosition == nextPos {
			logger.Debugf("bad strat.xPosition == nextPos pass same pointer")
			return
		}
		if nextPos.GetEventTime().Sub(strat.xPosition.GetEventTime()) >= 0 {
			if math.Abs(strat.xPosition.GetSize()-nextPos.GetSize()) >= strat.xStepSize {
				strat.xOrderSilentTime = time.Now().Add(strat.config.EnterSilent)
				strat.yOrderSilentTime = time.Now()
				if strat.xWalkedDepth.Symbol != "" {
					strat.xTimedPositionChange.Insert(time.Now(), math.Abs(strat.xPosition.GetSize()-nextPos.GetSize())*strat.xWalkedDepth.MidPrice*strat.xMultiplier)
				}
				logger.Debugf("%s x position change %f -> %f %v", nextPos.GetSymbol(), strat.xPosition.GetSize(), nextPos.GetSize(), nextPos.GetEventTime())
				strat.xPosition = nextPos
				strat.hedgeYPosition()
			} else {
				strat.xPosition = nextPos
			}
		}
		strat.xPositionUpdateTime = nextPos.GetParseTime()
	} else {
		strat.xPosition = nextPos
		strat.xPositionUpdateTime = nextPos.GetParseTime()
		logger.Debugf("%s x position change nil -> %f", nextPos.GetSymbol(), nextPos.GetSize())
	}
}

func (strat *XYStrategy) handleYPosition(nextPos common.Position) {
	if nextPos.GetSymbol() != strat.ySymbol {
		logger.Debugf("bad next position, symbol %s %s not match %v", nextPos.GetSymbol(), strat.ySymbol, nextPos)
		return
	}
	if strat.yPosition != nil {
		if strat.yPosition == nextPos {
			logger.Debugf("bad strat.yPosition == nextPos pass same pointer")
			return
		}
		if nextPos.GetEventTime().Sub(strat.yPosition.GetEventTime()) >= 0 {
			if math.Abs(strat.yPosition.GetSize()-nextPos.GetSize()) >= strat.xStepSize {
				if strat.yWalkedDepth.Symbol != "" {
					strat.yTimedPositionChange.Insert(time.Now(), math.Abs(strat.yPosition.GetSize()-nextPos.GetSize())*strat.yWalkedDepth.MidPrice*strat.yMultiplier)
				}
				logger.Debugf("%s y position change %f -> %f %v", nextPos.GetSymbol(), strat.yPosition.GetSize(), nextPos.GetSize(), nextPos.GetEventTime())
			}
			strat.yPosition = nextPos
		}
		strat.yPositionUpdateTime = nextPos.GetParseTime()
	} else {
		strat.yPosition = nextPos
		strat.yPositionUpdateTime = nextPos.GetParseTime()
		logger.Debugf("%s y position change nil -> %f", nextPos.GetSymbol(), nextPos.GetSize())
	}
}
