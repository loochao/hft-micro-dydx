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
	params := XYParams{
		logInterval:             config.LogInterval,
		depthTakerImpact:        config.DepthTakerImpact,
		depthXDecay:             config.DepthXDecay,
		depthXBias:              config.DepthXBias,
		depthYDecay:             config.DepthYDecay,
		depthYBias:              config.DepthYBias,
		depthMinTimeDelta:       config.DepthMinTimeDelta,
		depthMaxTimeDelta:       config.DepthMaxTimeDelta,
		depthMaxAgeDiffBias:     config.DepthMaxAgeDiffBias,
		depthReportCount:        config.DepthReportCount,
		spreadLookback:          config.SpreadLookback,
		spreadMinDepthCount:     config.SpreadMinDepthCount,
		spreadTimeToLive:        config.SpreadTimeToLive,
		enterTargetFactor:       config.EnterTargetFactor,
		enterMinimalStep:        config.EnterMinimalStep,
		enterFreePct:            config.EnterFreePct,
		longEnterDelta:          config.LongEnterDelta,
		longExitDelta:           config.LongExitDelta,
		shortEnterDelta:         config.ShortEnterDelta,
		shortExitDelta:          config.ShortExitDelta,
		enterOffsetDelta:        config.EnterOffsetDelta,
		exitOffsetDelta:         config.ExitOffsetDelta,
		minimalKeepFundingRate:  config.MinimalKeepFundingRate,
		minimalEnterFundingRate: config.MinimalEnterFundingRate,
		dryRun:                  config.DryRun,
		isXSpot:                 xExchange.IsSpot(),
		isYSpot:                 yExchange.IsSpot(),
		balancePositionMaxAge:   config.BalancePositionMaxAge,
		enterSilent:             config.EnterSilent,
		orderSilent:             config.OrderSilent,
		turnoverLookback:        config.TurnoverLookback,
		xLeverage:               config.XExchange.Leverage,
		yLeverage:               config.YExchange.Leverage,
		saveInterval:            config.InternalInflux.SaveInterval,
		hedgeYDelay: config.HedgeYDelay,
	}

	if params.saveInterval == 0 {
		params.saveInterval = time.Hour
	}

	if _, ok := config.NotTradePairs[xSymbol]; ok {
		params.tradable = false
	} else {
		params.tradable = true
	}

	params.xTickSize, err = xExchange.GetTickSize(xSymbol)
	if err != nil {
		return
	}
	params.xStepSize, err = xExchange.GetStepSize(xSymbol)
	if err != nil {
		return
	}
	params.xMultiplier, err = xExchange.GetMultiplier(xSymbol)
	if err != nil {
		return
	}
	params.xMinNotional, err = xExchange.GetMinNotional(xSymbol)
	if err != nil {
		return
	}

	params.yTickSize, err = yExchange.GetTickSize(ySymbol)
	if err != nil {
		return
	}
	params.yStepSize, err = yExchange.GetStepSize(ySymbol)
	if err != nil {
		return
	}
	params.yMultiplier, err = yExchange.GetMultiplier(ySymbol)
	if err != nil {
		return
	}
	params.yMinNotional, err = yExchange.GetMinNotional(ySymbol)
	if err != nil {
		return
	}

	params.xyMergedSpotStepSize = common.MergedStepSize(params.xStepSize*params.xMultiplier, params.yStepSize*params.yMultiplier)

	xBiasInMs := float64(params.depthXBias / time.Millisecond)
	yBiasInMs := float64(params.depthYBias / time.Millisecond)
	minTimeDeltaInMs := float64(params.depthMinTimeDelta / time.Millisecond)
	maxTimeDeltaInMs := float64(params.depthMaxTimeDelta / time.Millisecond)

	strat := XYStrategy{
		xExchange:                        xExchange,
		yExchange:                        yExchange,
		xSymbol:                          xSymbol,
		ySymbol:                          ySymbol,
		params:                           params,
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
		xDepth:                           nil,
		yDepth:                           nil,
		xDepthTime:                       time.Time{},
		yDepthTime:                       time.Time{},
		xDepthFilter:                     common.NewDepthFilter(params.depthXDecay, xBiasInMs, minTimeDeltaInMs, maxTimeDeltaInMs),
		yDepthFilter:                     common.NewDepthFilter(params.depthYDecay, yBiasInMs, minTimeDeltaInMs, maxTimeDeltaInMs),
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
		xTargetSpotSize:                  nil,
		yTargetSpotSize:                  nil,
		xOrder:                           nil,
		yOrder:                           nil,
		xOrderError:                      common.OrderError{},
		yOrderError:                      common.OrderError{},
		xyTargetSpotSizeUpdateSilentTime: time.Time{},
		enterStep:                        0,
		enterTarget:                      0,
		usdtAvailable:                    0,
		logSilentTime:                    time.Time{},
		xWalkDepthTimer:                  time.NewTimer(time.Hour * 9999),
		yWalkDepthTimer:                  time.NewTimer(time.Hour * 9999),
		realisedSpreadTimer:              time.NewTimer(time.Hour * 9999),
		hedgeYTimer:                      time.NewTimer(time.Hour * 9999),
		saveTimer:                        time.NewTimer(params.saveInterval),
		spreadTime:                       time.Time{},
		spread:                           nil,
		shortEnterTimedMedian:            common.NewTimedMedian(params.spreadLookback),
		longEnterTimedMedian:             common.NewTimedMedian(params.spreadLookback),
		xTimedPositionChange:             common.NewTimedSum(params.turnoverLookback),
		yTimedPositionChange:             common.NewTimedSum(params.turnoverLookback),
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
			strat.saveTimer.Reset(strat.params.saveInterval)
			break
		case <-strat.hedgeYTimer.C:
			strat.changeYPosition()
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
		case strat.xDepth = <-strat.xDepthCh:
			strat.handleXDepth()
			break
		case strat.yDepth = <-strat.yDepthCh:
			strat.handleYDepth()
			break
		case <-strat.realisedSpreadTimer.C:
			strat.handleRealisedSpread()
			break
		}
	}
}

func (strat *XYStrategy) changeYPosition() {
	if strat.xSystemStatus != common.SystemStatusReady ||
		strat.ySystemStatus != common.SystemStatusReady {
		return
	}
	if !strat.params.tradable ||
		strat.yPosition == nil ||
		strat.yTargetSpotSize == nil ||
		strat.spread == nil ||
		time.Now().Sub(strat.yPositionUpdateTime) > strat.params.balancePositionMaxAge ||
		time.Now().Sub(strat.yOrderSilentTime) < 0 {
		return
	}
	strat.ySizeDiff = *strat.yTargetSpotSize/strat.params.yMultiplier - strat.yPosition.GetSize()
	if math.Abs(strat.ySizeDiff) < strat.params.yStepSize {
		return
	}
	strat.ySizeDiff = math.Round(strat.ySizeDiff/strat.params.yStepSize) * strat.params.yStepSize

	if strat.params.isYSpot {
		if math.Abs(strat.ySizeDiff) < strat.params.yStepSize {
			return
		} else if strat.ySizeDiff < 0 && -strat.ySizeDiff*strat.params.yMultiplier*strat.yWalkedDepth.MidPrice < strat.params.yMinNotional {
			return
		} else if strat.ySizeDiff > 0 && strat.ySizeDiff*strat.params.yMultiplier*strat.yWalkedDepth.MidPrice < strat.params.yMinNotional {
			return
		}
	} else {
		//期货以close仓位，没有minNotional限制
		if math.Abs(strat.ySizeDiff) < strat.params.yStepSize {
			return
		} else if strat.ySizeDiff < 0 && strat.yPosition.GetSize() <= 0 && -strat.ySizeDiff*strat.params.yMultiplier*strat.yWalkedDepth.MidPrice < strat.params.yMinNotional {
			return
		} else if strat.ySizeDiff > 0 && strat.yPosition.GetSize() >= 0 && strat.ySizeDiff*strat.params.yMultiplier*strat.yWalkedDepth.MidPrice < strat.params.yMinNotional {
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
	if !strat.params.dryRun {
		logger.Debugf("sending strat.yOrderRequestCh <- common.OrderRequest %s", strat.yNewOrderParam.Symbol)
		select {
		case strat.yOrderRequestCh <- common.OrderRequest{
			New: &strat.yNewOrderParam,
		}:
			logger.Debugf("sent strat.yOrderRequestCh <- common.OrderRequest %s", strat.yNewOrderParam.Symbol)
			strat.yOrderSilentTime = time.Now().Add(strat.params.orderSilent)
			strat.yPositionUpdateTime = time.Unix(0, 0)
			//default:
			//	logger.Debugf("strat.yOrderRequestCh <- common.OrderRequest %s failed, ch len %d", strat.ySymbol, len(strat.yOrderRequestCh))
		}
	} else {
		strat.yOrderSilentTime = time.Now().Add(strat.params.orderSilent)
		strat.yPositionUpdateTime = time.Unix(0, 0)
	}
	return
}

func (strat *XYStrategy) changeXPosition() {
	if strat.xSystemStatus != common.SystemStatusReady ||
		strat.ySystemStatus != common.SystemStatusReady {
		return
	}
	if !strat.params.tradable ||
		strat.xPosition == nil ||
		strat.xTargetSpotSize == nil ||
		strat.spread == nil ||
		time.Now().Sub(strat.xPositionUpdateTime) > strat.params.balancePositionMaxAge ||
		time.Now().Sub(strat.xOrderSilentTime) < 0 {
		return
	}
	strat.xSizeDiff = *strat.xTargetSpotSize/strat.params.xMultiplier - strat.xPosition.GetSize()
	if math.Abs(strat.xSizeDiff) < strat.params.xStepSize {
		return
	}
	strat.xSizeDiff = math.Round(strat.xSizeDiff/strat.params.xStepSize) * strat.params.xStepSize
	if strat.params.isXSpot {
		if math.Abs(strat.xSizeDiff) < strat.params.xStepSize {
			return
		} else if strat.xSizeDiff < 0 && -strat.xSizeDiff*strat.params.xMultiplier*strat.xWalkedDepth.MidPrice < strat.params.xMinNotional {
			return
		} else if strat.xSizeDiff > 0 && strat.xSizeDiff*strat.params.xMultiplier*strat.xWalkedDepth.MidPrice < strat.params.xMinNotional {
			return
		}
	} else {
		if math.Abs(strat.xSizeDiff) < strat.params.xStepSize {
			return
		} else if strat.xSizeDiff < 0 && strat.xPosition.GetSize() <= 0 && -strat.xSizeDiff*strat.params.xMultiplier*strat.xWalkedDepth.MidPrice < strat.params.xMinNotional {
			return
		} else if strat.xSizeDiff > 0 && strat.xPosition.GetSize() >= 0 && strat.xSizeDiff*strat.params.xMultiplier*strat.xWalkedDepth.MidPrice < strat.params.xMinNotional {
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
	if !strat.params.dryRun {
		logger.Debugf("sending strat.xOrderRequestCh <- common.OrderRequest %s", strat.xNewOrderParam.Symbol)
		select {
		case strat.xOrderRequestCh <- common.OrderRequest{
			New: &strat.xNewOrderParam,
		}:
			logger.Debugf("sent strat.xOrderRequestCh <- common.OrderRequest %s", strat.xNewOrderParam.Symbol)
			strat.xOrderSilentTime = time.Now().Add(strat.params.orderSilent)
			strat.xPositionUpdateTime = time.Unix(0, 0)
			//default:
			//	logger.Debugf("strat.xOrderRequestCh <- common.OrderRequest %s failed, ch len %d", strat.xSymbol, len(strat.xOrderRequestCh))
		}
	} else {
		strat.xOrderSilentTime = time.Now().Add(strat.params.orderSilent)
		strat.xPositionUpdateTime = time.Unix(0, 0)
	}
}

func (strat *XYStrategy) updateTargetByPosition() {
	if strat.xPosition == nil || strat.yPosition == nil {
		return
	}
	if time.Now().Sub(strat.xyTargetSpotSizeUpdateSilentTime) < 0 {
		return
	}
	if strat.xTargetSpotSize == nil {
		strat.xTargetSpotSize = new(float64)
	}
	if strat.yTargetSpotSize == nil {
		strat.yTargetSpotSize = new(float64)
	}
	if math.Abs(strat.xPosition.GetSize()*strat.params.xMultiplier)-math.Abs(strat.yPosition.GetSize()*strat.params.yMultiplier) >= strat.params.xyMergedSpotStepSize {
		*strat.yTargetSpotSize = strat.yPosition.GetSize() * strat.params.yMultiplier
		*strat.xTargetSpotSize = -*strat.yTargetSpotSize
	} else if math.Abs(strat.xPosition.GetSize()*strat.params.xMultiplier)-math.Abs(strat.yPosition.GetSize()*strat.params.yMultiplier) <= -strat.params.xyMergedSpotStepSize {
		*strat.xTargetSpotSize = strat.xPosition.GetSize() * strat.params.xMultiplier
		*strat.yTargetSpotSize = -*strat.xTargetSpotSize
	} else {
		*strat.xTargetSpotSize = strat.xPosition.GetSize() * strat.params.xMultiplier
		*strat.yTargetSpotSize = strat.yPosition.GetSize() * strat.params.yMultiplier
	}
	strat.changeXPosition()
	if strat.params.hedgeYDelay <= 0 {
		strat.changeYPosition()
	}else{
		strat.hedgeYTimer.Reset(strat.params.hedgeYDelay)
	}
}

func (strat *XYStrategy) updateEnterStepAndTarget() {
	if strat.xAccount == nil || strat.yAccount == nil {
		return
	}
	strat.enterStep = (strat.xAccount.GetFree() + strat.yAccount.GetFree()) * strat.params.enterFreePct
	if strat.enterStep < strat.params.enterMinimalStep {
		strat.enterStep = strat.params.enterMinimalStep
	}
	strat.enterTarget = strat.enterStep * strat.params.enterTargetFactor
	strat.usdtAvailable = math.Min(strat.xAccount.GetFree()*strat.params.xLeverage, strat.yAccount.GetFree()*strat.params.yLeverage)
}
func (strat *XYStrategy) updateTarget() {
	if strat.xSystemStatus != common.SystemStatusReady ||
		strat.ySystemStatus != common.SystemStatusReady {
		return
	}
	if time.Now().Sub(strat.xyTargetSpotSizeUpdateSilentTime) < 0 ||
		time.Now().Sub(strat.xPositionUpdateTime) > strat.params.balancePositionMaxAge ||
		time.Now().Sub(strat.yPositionUpdateTime) > strat.params.balancePositionMaxAge ||
		strat.xAccount == nil ||
		strat.yAccount == nil ||
		strat.xPosition == nil ||
		strat.yPosition == nil ||
		strat.spread == nil ||
		strat.xyFundingRate == nil ||
		time.Now().Sub(strat.spread.Time) > strat.params.spreadTimeToLive ||
		!strat.params.tradable {
		return
	}

	strat.xSize = strat.xPosition.GetSize() * strat.params.xMultiplier
	strat.ySize = strat.yPosition.GetSize() * strat.params.yMultiplier
	strat.xValue = strat.xSize * strat.xWalkedDepth.MidPrice
	strat.yValue = strat.ySize * strat.yWalkedDepth.MidPrice
	if math.Abs(strat.xValue+strat.yValue) > strat.enterStep*0.8 {
		if time.Now().Sub(strat.logSilentTime) > 0 {
			strat.logSilentTime = time.Now().Add(strat.params.logInterval)
			logger.Debugf(
				"%s %s unhedged value %f > 0.8*enterStep %f",
				strat.xSymbol, strat.ySymbol, math.Abs(strat.xValue+strat.yValue), strat.enterStep*0.8,
			)
		}
		strat.updateTargetByPosition()
		return
	}
	strat.xAbsValue = math.Abs(strat.xValue)
	strat.yAbsValue = math.Abs(strat.yValue)
	strat.offsetFactor = (strat.xAbsValue + strat.yAbsValue) * 0.5 / strat.enterTarget
	strat.offsetStep = math.Min(strat.enterStep/strat.enterTarget, strat.offsetFactor)

	strat.shortTop = strat.params.shortEnterDelta + strat.params.enterOffsetDelta*strat.offsetFactor
	strat.shortBot = strat.params.shortExitDelta + strat.params.exitOffsetDelta*(strat.offsetFactor-strat.offsetStep)
	strat.longBot = strat.params.longEnterDelta - strat.params.enterOffsetDelta*strat.offsetFactor
	strat.longTop = strat.params.longExitDelta - strat.params.exitOffsetDelta*(strat.offsetFactor-strat.offsetStep)
	//if time.Now().Sub(strat.logSilentTime) > 0 {
	//	strat.logSilentTime = time.Now().Add(strat.params.logInterval)
	//	logger.Debugf(
	//		"%s enterTarget %f enterStep %f offsetFactor %f xTargetSpotSize %f yTargetSpotSize %f",
	//		strat.xSymbol, strat.enterTarget, strat.enterStep, strat.offsetFactor,
	//		*strat.xTargetSpotSize, *strat.yTargetSpotSize,
	//	)
	//}

	strat.midPrice = (strat.xWalkedDepth.MidPrice + strat.yWalkedDepth.MidPrice) * 0.5
	if strat.spread.ShortLastLeave < strat.shortBot &&
		strat.spread.ShortMedianLeave < strat.shortBot &&
		strat.spread.ShortLastLeave < strat.spread.ShortMedianLeave &&
		*strat.xyFundingRate < strat.params.minimalKeepFundingRate &&
		strat.xSize >= strat.params.xStepSize {

		strat.enterValue = math.Min(4*strat.enterStep, math.Min(strat.xAbsValue, strat.yAbsValue))
		if *strat.xyFundingRate > strat.params.minimalKeepFundingRate*0.5 {
			strat.enterValue = math.Min(2*strat.enterStep, math.Min(strat.xAbsValue, strat.yAbsValue))
		}
		strat.size = strat.enterValue / strat.midPrice
		strat.size = math.Round(strat.size/strat.params.xyMergedSpotStepSize) * strat.params.xyMergedSpotStepSize
		strat.enterValue = strat.size * strat.midPrice

		if strat.xAbsValue-strat.enterValue < strat.params.xyMergedSpotStepSize || strat.yAbsValue-strat.enterValue < strat.params.xyMergedSpotStepSize {
			//两种情况都把x全平，间接y全平
			strat.size = strat.xSize
		}
		//谁小以谁为准
		if strat.xAbsValue <= strat.yAbsValue {
			*strat.xTargetSpotSize -= strat.size
			if *strat.xTargetSpotSize < 0 {
				*strat.xTargetSpotSize = 0
			}
			*strat.yTargetSpotSize = -*strat.xTargetSpotSize
		} else {
			*strat.yTargetSpotSize += strat.size
			if *strat.yTargetSpotSize > 0 {
				*strat.yTargetSpotSize = 0
			}
			*strat.xTargetSpotSize = -*strat.yTargetSpotSize
		}
		strat.xyTargetSpotSizeUpdateSilentTime = time.Now().Add(strat.params.enterSilent)
		strat.xOrderSilentTime = time.Now()
		strat.yOrderSilentTime = time.Now()
		strat.xLastFilledBuyPrice = nil
		strat.xLastFilledSellPrice = nil
		strat.yLastFilledBuyPrice = nil
		strat.yLastFilledSellPrice = nil
		logger.Debugf(
			"%s %s SHORT BOT REDUCE %f < %f, %f < %f, SIZE %f, TARGET X %f TARGET Y %f, XDepthDiff %v YDepthDiff %v",
			strat.xSymbol, strat.ySymbol,
			strat.spread.ShortLastLeave, strat.shortBot,
			strat.spread.ShortMedianLeave, strat.shortBot,
			strat.size,
			*strat.xTargetSpotSize,
			*strat.yTargetSpotSize,
			time.Now().Sub(strat.xDepthTime),
			time.Now().Sub(strat.yDepthTime),
		)
		strat.changeXPosition()
		if strat.params.hedgeYDelay <= 0 {
			strat.changeYPosition()
		}else{
			strat.hedgeYTimer.Reset(strat.params.hedgeYDelay)
		}
	} else if strat.spread.LongLastLeave > strat.longTop &&
		strat.spread.LongMedianLeave > strat.longTop &&
		strat.spread.LongLastLeave > strat.spread.LongMedianLeave &&
		*strat.xyFundingRate > -strat.params.minimalKeepFundingRate &&
		strat.xSize <= -strat.params.xStepSize {

		strat.enterValue = math.Min(4*strat.enterStep, math.Min(strat.xAbsValue, strat.yAbsValue))
		if *strat.xyFundingRate < -strat.params.minimalKeepFundingRate*0.5 {
			strat.enterValue = math.Min(2*strat.enterStep, math.Min(strat.xAbsValue, strat.yAbsValue))
		}
		strat.size = strat.enterValue / strat.midPrice
		strat.size = math.Round(strat.size/strat.params.xyMergedSpotStepSize) * strat.params.xyMergedSpotStepSize
		strat.enterValue = strat.size * strat.midPrice
		if strat.xAbsValue-strat.enterValue < strat.params.xyMergedSpotStepSize || strat.yAbsValue-strat.enterValue < strat.params.xyMergedSpotStepSize {
			strat.size = -strat.xSize
		}
		//谁小以谁为准
		if strat.xAbsValue <= strat.yAbsValue {
			*strat.xTargetSpotSize += strat.size
			if *strat.xTargetSpotSize > 0 {
				*strat.xTargetSpotSize = 0
			}
			*strat.yTargetSpotSize = -*strat.xTargetSpotSize
		} else {
			*strat.yTargetSpotSize -= strat.size
			if *strat.yTargetSpotSize < 0 {
				*strat.yTargetSpotSize = 0
			}
			*strat.xTargetSpotSize = -*strat.yTargetSpotSize
		}
		strat.xyTargetSpotSizeUpdateSilentTime = time.Now().Add(strat.params.enterSilent)
		strat.xOrderSilentTime = time.Now()
		strat.yOrderSilentTime = time.Now()
		strat.xLastFilledBuyPrice = nil
		strat.xLastFilledSellPrice = nil
		strat.yLastFilledBuyPrice = nil
		strat.yLastFilledSellPrice = nil
		logger.Debugf(
			"%s %s LONG TOP REDUCE %f > %f, %f > %f, SIZE %f, TARGET X %f, TARGET Y %f, XDepthDiff %v YDepthDiff %v",
			strat.xSymbol, strat.ySymbol,
			strat.spread.LongLastLeave, strat.longTop,
			strat.spread.LongMedianLeave, strat.longTop,
			strat.size,
			*strat.xTargetSpotSize,
			*strat.yTargetSpotSize,
			time.Now().Sub(strat.xDepthTime),
			time.Now().Sub(strat.yDepthTime),
		)
		strat.changeXPosition()
		if strat.params.hedgeYDelay <= 0 {
			strat.changeYPosition()
		}else{
			strat.hedgeYTimer.Reset(strat.params.hedgeYDelay)
		}
	} else if !strat.params.isYSpot &&
		strat.spread.ShortLastEnter > strat.shortTop &&
		strat.spread.ShortMedianEnter > strat.shortTop &&
		strat.spread.ShortLastEnter > strat.spread.ShortMedianEnter &&
		*strat.xyFundingRate > strat.params.minimalEnterFundingRate &&
		strat.xSize >= 0 {

		strat.targetValue = math.Max(strat.xAbsValue, strat.yAbsValue) + strat.enterStep
		if strat.targetValue > strat.enterTarget {
			strat.targetValue = strat.enterTarget
		}
		strat.enterValue = strat.targetValue - math.Max(strat.xAbsValue, strat.yAbsValue)
		strat.size = strat.enterValue / strat.midPrice
		strat.size = math.Round(strat.size/strat.params.xyMergedSpotStepSize) * strat.params.xyMergedSpotStepSize
		strat.enterValue = strat.size * strat.midPrice

		if strat.enterValue > strat.usdtAvailable {
			if time.Now().Sub(strat.logSilentTime) > 0 {
				strat.logSilentTime = time.Now().Add(strat.params.logInterval)
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
		if strat.enterValue < strat.params.yMinNotional || strat.enterValue < strat.params.xMinNotional {
			if time.Now().Sub(strat.logSilentTime) > 0 {
				strat.logSilentTime = time.Now().Add(strat.params.logInterval)
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
		//谁大以谁为准
		if strat.xAbsValue >= strat.yAbsValue {
			*strat.xTargetSpotSize += strat.size
			*strat.yTargetSpotSize = -*strat.xTargetSpotSize
		} else {
			*strat.yTargetSpotSize -= strat.size
			*strat.xTargetSpotSize = -*strat.yTargetSpotSize
		}
		strat.xyTargetSpotSizeUpdateSilentTime = time.Now().Add(strat.params.enterSilent)
		strat.xOrderSilentTime = time.Now()
		strat.yOrderSilentTime = time.Now()
		strat.xLastFilledBuyPrice = nil
		strat.xLastFilledSellPrice = nil
		strat.yLastFilledBuyPrice = nil
		strat.yLastFilledSellPrice = nil
		logger.Debugf(
			"%s %s SHORT TOP OPEN %f > %f, %f > %f, SIZE %f, TARGET X %f, TARGET Y %f, XDepthDiff %v YDepthDiff %v",
			strat.xSymbol, strat.ySymbol,
			strat.spread.ShortLastEnter, strat.shortTop,
			strat.spread.ShortMedianEnter, strat.shortTop,
			strat.size,
			*strat.xTargetSpotSize,
			*strat.yTargetSpotSize,
			time.Now().Sub(strat.xDepthTime),
			time.Now().Sub(strat.yDepthTime),
		)
		strat.changeXPosition()
		if strat.params.hedgeYDelay <= 0 {
			strat.changeYPosition()
		}else{
			strat.hedgeYTimer.Reset(strat.params.hedgeYDelay)
		}
	} else if !strat.params.isXSpot &&
		strat.spread.LongLastEnter < strat.longBot &&
		strat.spread.LongMedianEnter < strat.longBot &&
		strat.spread.LongLastEnter < strat.spread.LongMedianEnter &&
		*strat.xyFundingRate < -strat.params.minimalEnterFundingRate &&
		strat.xSize <= 0 {

		strat.targetValue = math.Max(strat.xAbsValue, strat.yAbsValue) + strat.enterStep
		if strat.targetValue > strat.enterTarget {
			strat.targetValue = strat.enterTarget
		}
		strat.enterValue = strat.targetValue - math.Max(strat.xAbsValue, strat.yAbsValue)
		strat.size = strat.enterValue / strat.midPrice
		strat.size = math.Round(strat.size/strat.params.xyMergedSpotStepSize) * strat.params.xyMergedSpotStepSize
		strat.enterValue = strat.size * strat.midPrice
		if strat.enterValue > strat.usdtAvailable {
			if time.Now().Sub(strat.logSilentTime) > strat.params.logInterval {
				strat.logSilentTime = time.Now().Add(strat.params.logInterval)
				logger.Debugf(
					"%s %s FAILED SHORT TOP OPEN, ENTRY VALUE %f MORE THAN usdtAvailable %f, %f < %f, %f < %f, SIZE %f",
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
		if strat.enterValue < strat.params.yMinNotional || strat.enterValue < strat.params.xMinNotional {
			if time.Now().Sub(strat.logSilentTime) > 0 {
				strat.logSilentTime = time.Now().Add(strat.params.logInterval)
				logger.Debugf(
					"%s %s FAILED SHORT TOP OPEN, ORDER VALUE %f TOO SMALL, %f < %f, %f < %f, SIZE %f",
					strat.xSymbol, strat.ySymbol,
					strat.enterValue,
					strat.spread.LongLastEnter, strat.longBot,
					strat.spread.LongMedianEnter, strat.longBot,
					strat.size,
				)
			}
			return
		}
		//谁大以谁为准
		if strat.xAbsValue >= strat.yAbsValue {
			*strat.xTargetSpotSize -= strat.size
			*strat.yTargetSpotSize = -*strat.xTargetSpotSize
		} else {
			*strat.yTargetSpotSize += strat.size
			*strat.xTargetSpotSize = -*strat.yTargetSpotSize
		}
		strat.xyTargetSpotSizeUpdateSilentTime = time.Now().Add(strat.params.enterSilent)
		strat.xOrderSilentTime = time.Now()
		strat.yOrderSilentTime = time.Now()
		strat.xLastFilledBuyPrice = nil
		strat.xLastFilledSellPrice = nil
		strat.yLastFilledBuyPrice = nil
		strat.yLastFilledSellPrice = nil
		logger.Debugf(
			"%s %s LONG BOT OPEN %f < %f, %f < %f, SIZE %f, TARGET X %f, TARGET Y %f, XDepthDiff %v YDepthDiff %v",
			strat.xSymbol, strat.ySymbol,
			strat.spread.LongLastEnter, strat.longBot,
			strat.spread.LongMedianEnter, strat.longBot,
			strat.size,
			*strat.xTargetSpotSize,
			*strat.yTargetSpotSize,
			time.Now().Sub(strat.xDepthTime),
			time.Now().Sub(strat.yDepthTime),
		)
		strat.changeXPosition()
		if strat.params.hedgeYDelay <= 0 {
			strat.changeYPosition()
		}else{
			strat.hedgeYTimer.Reset(strat.params.hedgeYDelay)
		}
	}
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
			if strat.xPosition.GetSize() != nextPos.GetSize() {
				if strat.xWalkedDepth.Symbol != "" {
					strat.xTimedPositionChange.Insert(time.Now(), math.Abs(strat.xPosition.GetSize()-nextPos.GetSize())*strat.xWalkedDepth.MidPrice*strat.params.xMultiplier)
				}
				logger.Debugf("%s x position change %f -> %f %v", nextPos.GetSymbol(), strat.xPosition.GetSize(), nextPos.GetSize(), nextPos.GetEventTime())
			}
			strat.xPosition = nextPos
		}
		strat.xPositionUpdateTime = nextPos.GetParseTime()
	} else {
		strat.xPosition = nextPos
		strat.xPositionUpdateTime = nextPos.GetParseTime()
		logger.Debugf("%s x position change nil -> %f", nextPos.GetSymbol(), nextPos.GetSize())
	}
	strat.updateTargetByPosition()
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
			if strat.yPosition.GetSize() != nextPos.GetSize() {
				if strat.yWalkedDepth.Symbol != "" {
					strat.yTimedPositionChange.Insert(time.Now(), math.Abs(strat.yPosition.GetSize()-nextPos.GetSize())*strat.yWalkedDepth.MidPrice*strat.params.yMultiplier)
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
	strat.updateTargetByPosition()
}
