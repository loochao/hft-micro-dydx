package main

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"time"
)

func startXYStrategy(
	ctx context.Context,
	xSymbol, ySymbol string,
	config Config,
	xExchange common.UsdExchange,
	yExchange common.UsdExchange,
	xAccountCh chan common.Balance,
	yAccountCh chan common.Balance,
	xPositionCh chan common.Position,
	yPositionCh chan common.Position,
	xFundingRateCh chan common.FundingRate,
	yFundingRateCh chan common.FundingRate,
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
		xExchange:                        xExchange,
		yExchange:                        yExchange,
		xSymbol:                          xSymbol,
		ySymbol:                          ySymbol,
		params:                           config,
		xAccountCh:                       xAccountCh,
		yAccountCh:                       yAccountCh,
		xPositionCh:                      xPositionCh,
		yPositionCh:                      yPositionCh,
		xFundingRateCh:                   xFundingRateCh,
		yFundingRateCh:                   yFundingRateCh,
		xOrderCh:                         xOrderCh,
		yOrderCh:                         yOrderCh,
		xOrderErrorCh:                    xOrderErrorCh,
		yOrderErrorCh:                    yOrderErrorCh,
		xOrderRequestCh:                  xOrderRequestCh,
		yOrderRequestCh:                  yOrderRequestCh,
		xSystemStatusCh:                  xSystemStatusCh,
		ySystemStatusCh:                  ySystemStatusCh,
		xDepthCh:                         xDepthCh,
		yDepthCh:                         yDepthCh,
		saveCh:                           saveCh,
		xPositionUpdateTime:              time.Time{},
		yPositionUpdateTime:              time.Time{},
		maxOrderValue:                    config.MaxOrderValues[xSymbol],
		xDepth:                           nil,
		yDepth:                           nil,
		xNextDepth:                       nil,
		yNextDepth:                       nil,
		xDepthTime:                       time.Time{},
		yDepthTime:                       time.Time{},
		xDepthFilter:                     common.NewDepthFilter(config.DepthXDecay, xBiasInMs, minTimeDeltaInMs, maxTimeDeltaInMs),
		yDepthFilter:                     common.NewDepthFilter(config.DepthYDecay, yBiasInMs, minTimeDeltaInMs, maxTimeDeltaInMs),
		xWalkedDepth:                     common.WalkedDepthBAM{},
		yWalkedDepth:                     common.WalkedDepthBAM{},
		xAccount:                         nil,
		yAccount:                         nil,
		xPosition:                        nil,
		yPosition:                        nil,
		xOrderSilentTime:                 time.Time{},
		yOrderSilentTime:                 time.Time{},
		xFundingRate:                     nil,
		yFundingRate:                     nil,
		xyFundingRate:                    nil,
		xLastFilledBuyPrice:              nil,
		xLastFilledSellPrice:             nil,
		yLastFilledBuyPrice:              nil,
		yLastFilledSellPrice:             nil,
		xOrder:                           nil,
		yOrder:                           nil,
		xOrderError:                      common.OrderError{},
		yOrderError:                      common.OrderError{},
		xyTargetPositionUpdateSilentTime: time.Now().Add(config.RestartSilent),
		enterStep:                        0,
		enterTarget:                      0,
		targetWeight:                     config.TargetWeights[xSymbol],
		usdtAvailable:                    0,
		logSilentTime:                    time.Time{},
		xWalkDepthTimer:                  time.NewTimer(time.Hour * 9999),
		yWalkDepthTimer:                  time.NewTimer(time.Hour * 9999),
		realisedSpreadTimer:              time.NewTimer(time.Hour * 9999),
		hedgeTimer:                       time.NewTimer(time.Hour * 9999),
		spreadWalkTimer:                  time.NewTimer(time.Hour * 9999),
		saveTimer:                        time.NewTimer(config.RestartSilent),
		spreadTime:                       time.Time{},
		spread:                           nil,
		shortEnterTimedMedian:            common.NewTimedMedian(config.SpreadLookback),
		longEnterTimedMedian:             common.NewTimedMedian(config.SpreadLookback),
		xTimedPositionChange:             common.NewTimedSum(config.TurnoverLookback),
		yTimedPositionChange:             common.NewTimedSum(config.TurnoverLookback),
		expectedChanSendingTime:          time.Nanosecond * 300,
		depthMatchCount:                  0,
		depthCount:                       0,
		xDepthExpireCount:                0,
		yDepthExpireCount:                0,
		shortLastEnter:                   0,
		longLastEnter:                    0,
		adjustedAgeDiff:                  0,
		spreadReport:                     nil,
		stateOutputCh:                    nil,
		error:                            nil,
		xSizeDiff:                        0,
		ySizeDiff:                        0,
		offsetFactor:                     0,
		shortTop:                         0,
		shortBot:                         0,
		longBot:                          0,
		longTop:                          0,
		xSize:                            0,
		ySize:                            0,
		xValue:                           0,
		yValue:                           0,
		xAbsValue:                        0,
		yAbsValue:                        0,
		midPrice:                         0,
		enterValue:                       0,
		targetValue:                      0,
		size:                             0,
		orderSide:                        common.OrderSideUnknown,
		isXSpot:                          xExchange.IsSpot(),
		isYSpot:                          yExchange.IsSpot(),
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

	strat.xyMergedSpotStepSize = common.MergedStepSize(strat.xStepSize*strat.xMultiplier, strat.yStepSize*strat.yMultiplier)

	if _, ok := config.NotTradePairs[xSymbol]; ok {
		strat.tradable = false
	} else {
		strat.tradable = true
	}

	go strat.startLoop(ctx)
	return
}

func (strat *XYStrategy) startLoop(ctx context.Context) {
	defer strat.xWalkDepthTimer.Stop()
	defer strat.yWalkDepthTimer.Stop()
	defer strat.realisedSpreadTimer.Stop()
	defer strat.saveTimer.Stop()
	var nextXPos, nextYPos common.Position
	for {
		select {
		case <-ctx.Done():
			return
		case strat.xSystemStatus = <-strat.xSystemStatusCh:
			break
		case strat.ySystemStatus = <-strat.ySystemStatusCh:
			break
		case <-strat.saveTimer.C:
			select {
			case strat.saveCh <- strat:
			default:
				logger.Debugf("strat.saveCh <- strat failed %s %s ch len %d", strat.xSymbol, strat.ySymbol, len(strat.saveCh))
			}
			strat.saveTimer.Reset(strat.params.InternalInflux.SaveInterval)
			break
		case <-strat.hedgeTimer.C:
			strat.hedgeXPosition()
			strat.hedgeYPosition()
			strat.hedgeCounter--
			if strat.hedgeCounter > 0 {
				strat.hedgeTimer.Reset(strat.params.HedgeCheckInterval)
			} else {
				strat.hedgeTimer.Reset(time.Hour * 9999)
			}
			break
		case <-strat.spreadWalkTimer.C:
			strat.walkSpread()
			break
		case strat.xAccount = <-strat.xAccountCh:
			strat.updateEnterStepAndTarget()
			break
		case strat.yAccount = <-strat.yAccountCh:
			strat.updateEnterStepAndTarget()
			break
		case nextXPos = <-strat.xPositionCh:
			strat.handleXPosition(nextXPos)
			break
		case nextYPos = <-strat.yPositionCh:
			strat.handleYPosition(nextYPos)
			break
		case strat.xFundingRate = <-strat.xFundingRateCh:
			strat.handleFundingRate()
			break
		case strat.yFundingRate = <-strat.yFundingRateCh:
			strat.handleFundingRate()
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

func (strat *XYStrategy) hedgeXPosition() {
	//logger.Debugf("hedgeYPosition %s", strat.ySymbol)
	if strat.xSystemStatus != common.SystemStatusReady ||
		strat.ySystemStatus != common.SystemStatusReady {
		if time.Now().Sub(strat.logSilentTime) > 0 {
			strat.logSilentTime = time.Now().Add(strat.params.LogInterval)
			logger.Debugf("hedgeYPosition failed xSystemStatus %v ySystemStatus %v", strat.xSystemStatus, strat.ySystemStatus)
		}
		return
	}
	if !strat.tradable ||
		strat.yPosition == nil ||
		strat.xPosition == nil ||
		strat.xTargetPositionSize == nil ||
		time.Now().Sub(strat.yPositionUpdateTime) > strat.params.BalancePositionMaxAge ||
		time.Now().Sub(strat.xPositionUpdateTime) > strat.params.BalancePositionMaxAge ||
		time.Now().Sub(strat.xOrderSilentTime) < 0 {
		return
	}
	strat.xSizeDiff = *strat.xTargetPositionSize/strat.xMultiplier - strat.xPosition.GetSize()
	if math.Abs(strat.xSizeDiff) < strat.xStepSize {
		return
	}
	strat.xSizeDiff = math.Round(strat.xSizeDiff/strat.xStepSize) * strat.xStepSize

	if strat.isXSpot {
		if math.Abs(strat.xSizeDiff) < strat.xStepSize {
			return
		} else if strat.xSizeDiff < 0 && -strat.xSizeDiff*strat.xMultiplier*strat.xWalkedDepth.MidPrice < strat.xMinNotional {
			return
		} else if strat.xSizeDiff > 0 && strat.xSizeDiff*strat.xMultiplier*strat.xWalkedDepth.MidPrice < strat.xMinNotional {
			return
		}
	} else {
		//期货以close仓位，没有minNotional限制
		if math.Abs(strat.xSizeDiff) < strat.xStepSize {
			return
		} else if strat.xSizeDiff < 0 && strat.xPosition.GetSize() <= 0 &&
			-strat.xSizeDiff*strat.xMultiplier*strat.xWalkedDepth.MidPrice < strat.xMinNotional {
			return
		} else if strat.xSizeDiff > 0 && strat.xPosition.GetSize() >= 0 &&
			strat.xSizeDiff*strat.xMultiplier*strat.xWalkedDepth.MidPrice < strat.xMinNotional {
			return
		}
	}

	strat.reduceOnly = false
	if strat.xSizeDiff*strat.xPosition.GetSize() < 0 && math.Abs(strat.xSizeDiff) <= math.Abs(strat.xPosition.GetSize()) {
		strat.reduceOnly = true
	}
	strat.orderSide = common.OrderSideBuy
	if strat.xSizeDiff < 0 {
		strat.orderSide = common.OrderSideSell
		strat.xSizeDiff = -strat.xSizeDiff
	}
	strat.xNewOrderParam = common.NewOrderParam{
		Symbol:     strat.xSymbol,
		Side:       strat.orderSide,
		Type:       common.OrderTypeMarket,
		Size:       strat.xSizeDiff,
		ReduceOnly: strat.reduceOnly,
		ClientID:   strat.xExchange.GenerateClientID(),
	}
	if !strat.params.DryRun {
		select {
		case strat.xOrderRequestCh <- common.OrderRequest{
			New: &strat.xNewOrderParam,
		}:
			strat.xOrderSilentTime = time.Now().Add(strat.params.OrderSilent)
			strat.xPositionUpdateTime = time.Unix(0, 0)
		}
	} else {
		strat.xOrderSilentTime = time.Now().Add(strat.params.OrderSilent)
		strat.xPositionUpdateTime = time.Unix(0, 0)
	}
}

func (strat *XYStrategy) hedgeYPosition() {
	//logger.Debugf("hedgeYPosition %s", strat.ySymbol)
	if strat.xSystemStatus != common.SystemStatusReady ||
		strat.ySystemStatus != common.SystemStatusReady {
		if time.Now().Sub(strat.logSilentTime) > 0 {
			strat.logSilentTime = time.Now().Add(strat.params.LogInterval)
			logger.Debugf("hedgeYPosition failed xSystemStatus %v ySystemStatus %v", strat.xSystemStatus, strat.ySystemStatus)
		}
		return
	}
	if !strat.tradable ||
		strat.yPosition == nil ||
		strat.xPosition == nil ||
		strat.xTargetPositionSize == nil ||
		time.Now().Sub(strat.yPositionUpdateTime) > strat.params.BalancePositionMaxAge ||
		time.Now().Sub(strat.xPositionUpdateTime) > strat.params.BalancePositionMaxAge ||
		time.Now().Sub(strat.yOrderSilentTime) < 0 {
		return
	}
	strat.ySizeDiff = *strat.yTargetPositionSize/strat.yMultiplier - strat.yPosition.GetSize()
	if math.Abs(strat.ySizeDiff) < strat.yStepSize {
		return
	}
	strat.ySizeDiff = math.Round(strat.ySizeDiff/strat.yStepSize) * strat.yStepSize

	if strat.isYSpot {
		if math.Abs(strat.ySizeDiff) < strat.yStepSize {
			return
		} else if strat.ySizeDiff < 0 && -strat.ySizeDiff*strat.yMultiplier*strat.yWalkedDepth.MidPrice < strat.yMinNotional {
			return
		} else if strat.ySizeDiff > 0 && strat.ySizeDiff*strat.yMultiplier*strat.yWalkedDepth.MidPrice < strat.yMinNotional {
			return
		}
	} else {
		//期货以close仓位，没有minNotional限制
		if math.Abs(strat.ySizeDiff) < strat.yStepSize {
			return
		} else if strat.ySizeDiff < 0 && strat.yPosition.GetSize() <= 0 && -strat.ySizeDiff*strat.yMultiplier*strat.yWalkedDepth.MidPrice < strat.yMinNotional {
			return
		} else if strat.ySizeDiff > 0 && strat.yPosition.GetSize() >= 0 && strat.ySizeDiff*strat.yMultiplier*strat.yWalkedDepth.MidPrice < strat.yMinNotional {
			return
		}
	}

	strat.reduceOnly = false
	if strat.ySizeDiff*strat.yPosition.GetSize() < 0 && math.Abs(strat.ySizeDiff) <= math.Abs(strat.yPosition.GetSize()) {
		strat.reduceOnly = true
	}
	strat.orderSide = common.OrderSideBuy
	if strat.ySizeDiff < 0 {
		strat.orderSide = common.OrderSideSell
		strat.ySizeDiff = -strat.ySizeDiff
	}
	strat.yNewOrderParam = common.NewOrderParam{
		Symbol:     strat.ySymbol,
		Side:       strat.orderSide,
		Type:       common.OrderTypeMarket,
		Size:       strat.ySizeDiff,
		ReduceOnly: strat.reduceOnly,
		ClientID:   strat.yExchange.GenerateClientID(),
	}
	if !strat.params.DryRun {
		select {
		case strat.yOrderRequestCh <- common.OrderRequest{
			New: &strat.yNewOrderParam,
		}:
			strat.yOrderSilentTime = time.Now().Add(strat.params.OrderSilent)
			strat.yPositionUpdateTime = time.Unix(0, 0)
		}
	} else {
		strat.yOrderSilentTime = time.Now().Add(strat.params.OrderSilent)
		strat.yPositionUpdateTime = time.Unix(0, 0)
	}
	return
}

func (strat *XYStrategy) updateEnterStepAndTarget() {
	if strat.xAccount == nil || strat.yAccount == nil {
		return
	}
	strat.enterStep = (strat.xAccount.GetFree() + strat.yAccount.GetFree()) * strat.params.EnterFreePct * strat.targetWeight
	if strat.enterStep < strat.params.EnterMinimalStep {
		strat.enterStep = strat.params.EnterMinimalStep
	}
	strat.enterTarget = strat.enterStep * strat.params.EnterTargetFactor * strat.targetWeight
	strat.usdtAvailable = math.Min(strat.xAccount.GetFree()*strat.params.XExchange.Leverage, strat.yAccount.GetFree()*strat.params.YExchange.Leverage)
}

func (strat *XYStrategy) updateTargetPositionSize() {
	if strat.xSystemStatus != common.SystemStatusReady ||
		strat.ySystemStatus != common.SystemStatusReady {
		if time.Now().Sub(strat.logSilentTime) > 0 {
			strat.logSilentTime = time.Now().Add(strat.params.LogInterval)
			logger.Debugf("updateTargetPositionSize failed xSystemStatus %v ySystemStatus %v", strat.xSystemStatus, strat.ySystemStatus)
		}
		return
	}
	if time.Now().Sub(strat.xyTargetPositionUpdateSilentTime) < 0 ||
		time.Now().Sub(strat.xOrderSilentTime) < 0 ||
		time.Now().Sub(strat.yOrderSilentTime) < 0 ||
		time.Now().Sub(strat.xPositionUpdateTime) > strat.params.BalancePositionMaxAge ||
		time.Now().Sub(strat.yPositionUpdateTime) > strat.params.BalancePositionMaxAge ||
		strat.xAccount == nil ||
		strat.yAccount == nil ||
		strat.xPosition == nil ||
		strat.yPosition == nil ||
		strat.spread == nil ||
		strat.xyFundingRate == nil ||
		time.Now().Sub(strat.spread.EventTime) > strat.params.SpreadTimeToLive ||
		!strat.tradable {
		return
	}
	strat.xSize = strat.xPosition.GetSize() * strat.xMultiplier
	strat.ySize = strat.yPosition.GetSize() * strat.yMultiplier
	strat.xValue = strat.xSize * strat.xWalkedDepth.MidPrice
	strat.yValue = strat.ySize * strat.yWalkedDepth.MidPrice
	if math.Abs(strat.xValue+strat.yValue) > strat.enterStep*0.8 {
		if time.Now().Sub(strat.logSilentTime) > 0 {
			strat.logSilentTime = time.Now().Add(strat.params.LogInterval)
			logger.Debugf(
				"%s %s unhedged value %f > 0.8*enterStep %f",
				strat.xSymbol, strat.ySymbol, math.Abs(strat.xValue+strat.yValue), strat.enterStep*0.8,
			)
		}
		return
	}
	strat.xAbsValue = math.Abs(strat.xValue)
	strat.yAbsValue = math.Abs(strat.yValue)
	strat.offsetFactor = (strat.xAbsValue + strat.yAbsValue) * 0.5 / strat.enterTarget
	strat.offsetStep = math.Min(strat.enterStep/strat.enterTarget, strat.offsetFactor)

	strat.shortTop = strat.params.ShortEnterDelta + strat.params.EnterOffsetDelta*strat.offsetFactor
	strat.shortBot = strat.params.ShortExitDelta + strat.params.ExitOffsetDelta*(strat.offsetFactor-strat.offsetStep)
	strat.longBot = strat.params.LongEnterDelta - strat.params.EnterOffsetDelta*strat.offsetFactor
	strat.longTop = strat.params.LongExitDelta - strat.params.ExitOffsetDelta*(strat.offsetFactor-strat.offsetStep)
	strat.midPrice = (strat.xWalkedDepth.MidPrice + strat.yWalkedDepth.MidPrice) * 0.5

	if strat.xAbsValue-strat.yAbsValue >= strat.xyMergedSpotStepSize {
		if strat.xTargetPositionSize == nil {
			strat.xTargetPositionSize = new(float64)
			strat.yTargetPositionSize = new(float64)
		}
		*strat.yTargetPositionSize = strat.ySize
		*strat.xTargetPositionSize = -strat.ySize
	} else if strat.xAbsValue-strat.yAbsValue <= -strat.xyMergedSpotStepSize {
		if strat.xTargetPositionSize == nil {
			strat.xTargetPositionSize = new(float64)
			strat.yTargetPositionSize = new(float64)
		}
		*strat.xTargetPositionSize = strat.xSize
		*strat.yTargetPositionSize = -strat.xSize
	} else {
		if strat.xTargetPositionSize == nil {
			strat.xTargetPositionSize = new(float64)
			strat.yTargetPositionSize = new(float64)
		}
		*strat.xTargetPositionSize = strat.xSize
		*strat.yTargetPositionSize = strat.ySize
	}

	if strat.spread.ShortLastLeave < strat.shortBot &&
		strat.spread.ShortMedianLeave < strat.shortBot &&
		strat.spread.ShortLastLeave < strat.spread.ShortMedianLeave &&
		*strat.xyFundingRate < strat.params.MinimalKeepFundingRate &&
		strat.xSize >= strat.xStepSize*strat.xMultiplier {

		strat.enterValue = math.Min(2*strat.enterStep, math.Min(strat.xAbsValue, strat.yAbsValue))
		if *strat.xyFundingRate > strat.params.MinimalKeepFundingRate*0.5 {
			strat.enterValue = math.Min(strat.enterStep, math.Min(strat.xAbsValue, strat.yAbsValue))
		}
		if strat.enterValue > strat.maxOrderValue {
			strat.enterValue = strat.maxOrderValue
		}
		strat.size = strat.enterValue / strat.midPrice
		strat.size = math.Round(strat.size/strat.xyMergedSpotStepSize) * strat.xyMergedSpotStepSize
		strat.enterValue = strat.size * strat.midPrice
		if strat.xAbsValue-strat.enterValue < strat.xyMergedSpotStepSize || strat.yAbsValue-strat.enterValue < strat.xyMergedSpotStepSize {
			//两种情况都把x全平，间接y全平
			strat.size = strat.xSize
		}
		//谁小以谁为准
		if strat.xAbsValue < strat.yAbsValue {
			*strat.xTargetPositionSize -= strat.size
			if *strat.xTargetPositionSize < 0 {
				*strat.xTargetPositionSize = 0
			}
			*strat.yTargetPositionSize = -*strat.xTargetPositionSize
		} else {
			*strat.yTargetPositionSize += strat.size
			if *strat.yTargetPositionSize > 0 {
				*strat.yTargetPositionSize = 0
			}
			*strat.xTargetPositionSize = -*strat.yTargetPositionSize
		}
		strat.xyTargetPositionUpdateSilentTime = time.Now().Add(strat.params.EnterSilent)
		strat.xLastFilledBuyPrice = nil
		strat.xLastFilledSellPrice = nil
		strat.yLastFilledBuyPrice = nil
		strat.yLastFilledSellPrice = nil
		strat.hedgeTimer.Reset(strat.params.HedgeCheckInterval)
		strat.hedgeCounter = time.Minute / strat.params.HedgeCheckInterval
		strat.hedgeXPosition()
		strat.hedgeYPosition()
		logger.Debugf(
			"%s %s SHORT BOT REDUCE %f < %f, %f < %f, SIZE %f,  XDepthDiff %v YDepthDiff %v SpreadDiff %v",
			strat.xSymbol, strat.ySymbol,
			strat.spread.ShortLastLeave, strat.shortBot,
			strat.spread.ShortMedianLeave, strat.shortBot,
			strat.size,
			time.Now().Sub(strat.xDepthTime),
			time.Now().Sub(strat.yDepthTime),
			time.Now().Sub(strat.spread.EventTime),
		)
	} else if strat.spread.LongLastLeave > strat.longTop &&
		strat.spread.LongMedianLeave > strat.longTop &&
		strat.spread.LongLastLeave > strat.spread.LongMedianLeave &&
		*strat.xyFundingRate > -strat.params.MinimalKeepFundingRate &&
		strat.xSize <= -strat.xStepSize*strat.xMultiplier {
		strat.enterValue = math.Min(2*strat.enterStep, math.Min(strat.xAbsValue, strat.yAbsValue))
		if *strat.xyFundingRate < -strat.params.MinimalKeepFundingRate*0.5 {
			strat.enterValue = math.Min(strat.enterStep, math.Min(strat.xAbsValue, strat.yAbsValue))
		}
		if strat.enterValue > strat.maxOrderValue {
			strat.enterValue = strat.maxOrderValue
		}
		strat.size = strat.enterValue / strat.midPrice
		strat.size = math.Round(strat.size/strat.xyMergedSpotStepSize) * strat.xyMergedSpotStepSize
		strat.enterValue = strat.size * strat.midPrice
		if strat.xAbsValue-strat.enterValue < strat.xyMergedSpotStepSize || strat.yAbsValue-strat.enterValue < strat.xyMergedSpotStepSize {
			strat.size = -strat.xSize
		}
		//谁小以谁为准
		if strat.xAbsValue < strat.yAbsValue {
			*strat.xTargetPositionSize += strat.size
			if *strat.xTargetPositionSize > 0 {
				*strat.xTargetPositionSize = 0
			}
			*strat.yTargetPositionSize = -*strat.xTargetPositionSize
		} else {
			*strat.yTargetPositionSize -= strat.size
			if *strat.yTargetPositionSize < 0 {
				*strat.yTargetPositionSize = 0
			}
			*strat.xTargetPositionSize = -*strat.yTargetPositionSize
		}
		strat.xyTargetPositionUpdateSilentTime = time.Now().Add(strat.params.EnterSilent)
		strat.xLastFilledBuyPrice = nil
		strat.xLastFilledSellPrice = nil
		strat.yLastFilledBuyPrice = nil
		strat.yLastFilledSellPrice = nil
		strat.hedgeTimer.Reset(strat.params.HedgeCheckInterval)
		strat.hedgeCounter = time.Minute / strat.params.HedgeCheckInterval
		strat.hedgeXPosition()
		strat.hedgeYPosition()
		logger.Debugf(
			"%s %s LONG TOP REDUCE %f > %f, %f > %f, SIZE -%f, XDepthDiff %v YDepthDiff %v SpreadDiff %v",
			strat.xSymbol, strat.ySymbol,
			strat.spread.LongLastLeave, strat.longTop,
			strat.spread.LongMedianLeave, strat.longTop,
			strat.size,
			time.Now().Sub(strat.xDepthTime),
			time.Now().Sub(strat.yDepthTime),
			time.Now().Sub(strat.spread.EventTime),
		)
	} else if !strat.params.ReduceOnly &&
		!strat.isYSpot &&
		strat.spread.ShortLastEnter > strat.shortTop &&
		strat.spread.ShortMedianEnter > strat.shortTop &&
		strat.spread.ShortLastEnter > strat.spread.ShortMedianEnter &&
		*strat.xyFundingRate > strat.params.MinimalEnterFundingRate &&
		strat.xSize >= 0 {

		strat.targetValue = math.Max(strat.xAbsValue, strat.yAbsValue) + strat.enterStep
		if strat.targetValue > strat.enterTarget {
			strat.targetValue = strat.enterTarget
		}
		strat.enterValue = strat.targetValue - math.Max(strat.xAbsValue, strat.yAbsValue)
		if strat.enterValue > strat.maxOrderValue {
			strat.enterValue = strat.maxOrderValue
		}
		strat.size = strat.enterValue / strat.midPrice
		strat.size = math.Round(strat.size/strat.xyMergedSpotStepSize) * strat.xyMergedSpotStepSize
		strat.enterValue = strat.size * strat.midPrice

		if strat.enterValue > strat.usdtAvailable {
			if time.Now().Sub(strat.logSilentTime) > 0 {
				strat.logSilentTime = time.Now().Add(strat.params.LogInterval)
				logger.Debugf(
					"%s %s FAILED SHORT TOP OPEN, ENTRY VALUE %f MORE THAN usdtAvailable %f, %f > %f, %f > %f, SIZE %f",
					strat.xSymbol,
					strat.ySymbol,
					strat.enterValue,
					strat.usdtAvailable,
					strat.spread.ShortLastEnter, strat.shortTop,
					strat.spread.ShortMedianEnter, strat.shortTop,
					strat.size,
				)
			}
			return
		}
		if strat.enterValue < strat.yMinNotional || strat.enterValue < strat.xMinNotional {
			if time.Now().Sub(strat.logSilentTime) > 0 {
				strat.logSilentTime = time.Now().Add(strat.params.LogInterval)
				logger.Debugf(
					"%s %s FAILED SHORT TOP OPEN, ORDER VALUE %f TOO SMALL, %f > %f, %f > %f, SIZE %f",
					strat.xSymbol, strat.ySymbol,
					strat.enterValue,
					strat.spread.ShortLastEnter, strat.shortTop,
					strat.spread.ShortMedianEnter, strat.shortTop,
					strat.size,
				)
			}
			return
		}
		//谁小以谁为准
		if strat.xAbsValue < strat.yAbsValue {
			*strat.xTargetPositionSize += strat.size
			*strat.yTargetPositionSize = -*strat.xTargetPositionSize
		} else {
			*strat.yTargetPositionSize -= strat.size
			*strat.xTargetPositionSize = -*strat.yTargetPositionSize
		}
		strat.xyTargetPositionUpdateSilentTime = time.Now().Add(strat.params.EnterSilent)
		strat.xLastFilledBuyPrice = nil
		strat.xLastFilledSellPrice = nil
		strat.yLastFilledBuyPrice = nil
		strat.yLastFilledSellPrice = nil
		strat.hedgeTimer.Reset(strat.params.HedgeCheckInterval)
		strat.hedgeCounter = time.Minute / strat.params.HedgeCheckInterval
		strat.hedgeXPosition()
		strat.hedgeYPosition()
		logger.Debugf(
			"%s %s SHORT TOP OPEN %f > %f, %f > %f, SIZE %f, XDepthDiff %v YDepthDiff %v SpreadDiff %v",
			strat.xSymbol, strat.ySymbol,
			strat.spread.ShortLastEnter, strat.shortTop,
			strat.spread.ShortMedianEnter, strat.shortTop,
			strat.size,
			time.Now().Sub(strat.xDepthTime),
			time.Now().Sub(strat.yDepthTime),
			time.Now().Sub(strat.spread.EventTime),
		)
	} else if !strat.params.ReduceOnly &&
		!strat.isXSpot &&
		strat.spread.LongLastEnter < strat.longBot &&
		strat.spread.LongMedianEnter < strat.longBot &&
		strat.spread.LongLastEnter < strat.spread.LongMedianEnter &&
		*strat.xyFundingRate < -strat.params.MinimalEnterFundingRate &&
		strat.xSize <= 0 {

		strat.targetValue = math.Max(strat.xAbsValue, strat.yAbsValue) + strat.enterStep
		if strat.targetValue > strat.enterTarget {
			strat.targetValue = strat.enterTarget
		}
		strat.enterValue = strat.targetValue - math.Max(strat.xAbsValue, strat.yAbsValue)
		if strat.enterValue > strat.maxOrderValue {
			strat.enterValue = strat.maxOrderValue
		}
		strat.size = strat.enterValue / strat.midPrice
		strat.size = math.Round(strat.size/strat.xyMergedSpotStepSize) * strat.xyMergedSpotStepSize
		strat.enterValue = strat.size * strat.midPrice
		if strat.enterValue > strat.usdtAvailable {
			if time.Now().Sub(strat.logSilentTime) > strat.params.LogInterval {
				strat.logSilentTime = time.Now().Add(strat.params.LogInterval)
				logger.Debugf(
					"%s %s FAILED LONG BOT OPEN, ENTRY VALUE %f MORE THAN usdtAvailable %f, %f < %f, %f < %f, SIZE %f",
					strat.xSymbol,
					strat.ySymbol,
					strat.enterValue,
					strat.usdtAvailable,
					strat.spread.LongLastEnter, strat.longBot,
					strat.spread.LongMedianEnter, strat.longBot,
					strat.size,
				)
			}
			return
		}
		if strat.enterValue < strat.yMinNotional || strat.enterValue < strat.xMinNotional {
			if time.Now().Sub(strat.logSilentTime) > 0 {
				strat.logSilentTime = time.Now().Add(strat.params.LogInterval)
				logger.Debugf(
					"%s %s FAILED LONG BOT OPEN, ORDER VALUE %f TOO SMALL, %f < %f, %f < %f, SIZE %f",
					strat.xSymbol, strat.ySymbol,
					strat.enterValue,
					strat.spread.LongLastEnter, strat.longBot,
					strat.spread.LongMedianEnter, strat.longBot,
					strat.size,
				)
			}
			return
		}
		//谁小以谁为准
		if strat.xAbsValue < strat.yAbsValue {
			*strat.xTargetPositionSize -= strat.size
			*strat.yTargetPositionSize = -*strat.xTargetPositionSize
		} else {
			*strat.yTargetPositionSize += strat.size
			*strat.xTargetPositionSize = -*strat.yTargetPositionSize
		}
		strat.xyTargetPositionUpdateSilentTime = time.Now().Add(strat.params.EnterSilent)
		strat.xLastFilledBuyPrice = nil
		strat.xLastFilledSellPrice = nil
		strat.yLastFilledBuyPrice = nil
		strat.yLastFilledSellPrice = nil
		strat.hedgeTimer.Reset(strat.params.HedgeCheckInterval)
		strat.hedgeCounter = time.Minute / strat.params.HedgeCheckInterval
		strat.hedgeXPosition()
		strat.hedgeYPosition()
		logger.Debugf(
			"%s %s LONG BOT OPEN %f < %f, %f < %f, SIZE -%f, XDepthDiff %v YDepthDiff %v SpreadDiff %v",
			strat.xSymbol, strat.ySymbol,
			strat.spread.LongLastEnter, strat.longBot,
			strat.spread.LongMedianEnter, strat.longBot,
			strat.size,
			time.Now().Sub(strat.xDepthTime),
			time.Now().Sub(strat.yDepthTime),
			time.Now().Sub(strat.spread.EventTime),
		)
	}
}

func (strat *XYStrategy) handleXPosition(nextPos common.Position) {
	if nextPos.GetSymbol() != strat.xSymbol {
		logger.Debugf("bad next position, symbol %s %s not match %v", nextPos.GetSymbol(), strat.xSymbol, nextPos)
		return
	}
	if !strat.tradable {
		return
	}
	if strat.xPosition != nil {
		if strat.xPosition == nextPos {
			logger.Debugf("bad strat.xPosition == nextPos pass same pointer")
			return
		}
		if nextPos.GetEventTime().Sub(strat.xPosition.GetEventTime()) >= 0 {
			if strat.xPosition.GetSize() != nextPos.GetSize() {
				if strat.xWalkedDepth.Symbol != "" {
					strat.xTimedPositionChange.Insert(time.Now(), math.Abs(strat.xPosition.GetSize()-nextPos.GetSize())*strat.xWalkedDepth.MidPrice*strat.xMultiplier)
				}
				logger.Debugf("%s x position change %f -> %f %v", nextPos.GetSymbol(), strat.xPosition.GetSize(), nextPos.GetSize(), nextPos.GetEventTime())
			}
			strat.xPosition = nextPos
			strat.hedgeYPosition()
		}
		strat.xPositionUpdateTime = nextPos.GetParseTime()
	} else {
		strat.xPosition = nextPos
		strat.xPositionUpdateTime = nextPos.GetParseTime()
		logger.Debugf("%s x position change nil -> %f", nextPos.GetSymbol(), nextPos.GetSize())
	}
	//if strat.markedYBidPrice == nil &&
	//	strat.markedYAskPrice == nil {
	//	strat.hedgeYPosition()
	//}
}

func (strat *XYStrategy) handleYPosition(nextPos common.Position) {
	if nextPos.GetSymbol() != strat.ySymbol {
		logger.Debugf("bad next position, symbol %s %s not match %v", nextPos.GetSymbol(), strat.ySymbol, nextPos)
		return
	}
	if !strat.tradable {
		return
	}
	if strat.yPosition != nil {
		if strat.yPosition == nextPos {
			logger.Debugf("bad strat.yPosition == nextPos pass same pointer")
			return
		}
		if nextPos.GetEventTime().Sub(strat.yPosition.GetEventTime()) >= 0 {
			if strat.yPosition.GetSize() != nextPos.GetSize() {
				if strat.yWalkedDepth.Symbol != "" {
					strat.yTimedPositionChange.Insert(time.Now(), math.Abs(strat.yPosition.GetSize()-nextPos.GetSize())*strat.yWalkedDepth.MidPrice*strat.yMultiplier)
				}
				logger.Debugf("%s y position change %f -> %f %v", nextPos.GetSymbol(), strat.yPosition.GetSize(), nextPos.GetSize(), nextPos.GetEventTime())
			}
			strat.yPosition = nextPos
			strat.hedgeYPosition()
		}
		strat.yPositionUpdateTime = nextPos.GetParseTime()
	} else {
		strat.yPosition = nextPos
		strat.yPositionUpdateTime = nextPos.GetParseTime()
		logger.Debugf("%s y position change nil -> %f", nextPos.GetSymbol(), nextPos.GetSize())
	}
	//if strat.markedYBidPrice == nil &&
	//	strat.markedYAskPrice == nil {
	//	strat.hedgeYPosition()
	//}
}
