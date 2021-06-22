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
	orderOffset Offset,
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
		xExchange:               xExchange,
		yExchange:               yExchange,
		isXSpot:                 xExchange.IsSpot(),
		isYSpot:                 yExchange.IsSpot(),
		xLeverage:               config.XExchange.Leverage,
		yLeverage:               config.YExchange.Leverage,
		xSymbol:                 xSymbol,
		ySymbol:                 ySymbol,
		enterScale:              config.EnterScales[xSymbol],
		config:                  config,
		orderOffset:             orderOffset,
		xAccountCh:              xAccountCh,
		yAccountCh:              yAccountCh,
		xPositionCh:             xPositionCh,
		yPositionCh:             yPositionCh,
		xFundingRateCh:          xFundingRateCh,
		yFundingRateCh:          yFundingRateCh,
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
		xDepthFilter:            common.NewDepthFilter(config.DepthXDecay, xBiasInMs, minTimeDeltaInMs, maxTimeDeltaInMs),
		yDepthFilter:            common.NewDepthFilter(config.DepthYDecay, yBiasInMs, minTimeDeltaInMs, maxTimeDeltaInMs),
		xWalkedDepth:            common.WalkedDepthBAM{},
		yWalkedDepth:            common.WalkedDepthBAM{},
		xAccount:                nil,
		yAccount:                nil,
		xPosition:               nil,
		yPosition:               nil,
		xOrderSilentTime:        time.Time{},
		yOrderSilentTime:        time.Time{},
		xFundingRate:            nil,
		yFundingRate:            nil,
		xyFundingRate:           nil,
		xLastFilledBuyPrice:     nil,
		xLastFilledSellPrice:    nil,
		yLastFilledBuyPrice:     nil,
		yLastFilledSellPrice:    nil,
		xOrder:                  nil,
		yOrder:                  nil,
		xOrderError:             common.OrderError{},
		yOrderError:             common.OrderError{},
		enterStep:               0,
		enterTarget:             0,
		usdtAvailable:           0,
		logSilentTime:           time.Time{},
		xWalkDepthTimer:         time.NewTimer(time.Hour * 9999),
		yWalkDepthTimer:         time.NewTimer(time.Hour * 9999),
		spreadWalkTimer:         time.NewTimer(time.Hour * 9999),
		realisedSpreadTimer:     time.NewTimer(time.Hour * 9999),
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
		spreadReport:            nil,
		stateOutputCh:           nil,
		error:                   nil,
		xSizeDiff:               0,
		ySizeDiff:               0,
		offsetFactor:            0,
		shortTop:                0,
		shortBot:                0,
		longBot:                 0,
		longTop:                 0,
		xSize:                   0,
		ySize:                   0,
		xValue:                  0,
		yValue:                  0,
		xAbsValue:               0,
		yAbsValue:               0,
		midPrice:                0,
		enterValue:              0,
		targetValue:             0,
		size:                    0,
		orderSide:               common.OrderSideUnknown,
		xCancelOrderParam:       common.CancelOrderParam{Symbol: xSymbol},
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
	defer strat.spreadWalkTimer.Stop()
	defer strat.realisedSpreadTimer.Stop()
	defer strat.saveTimer.Stop()
	var nextXPos, nextYPos common.Position
	strat.xOrderSilentTime = time.Now().Add(-time.Millisecond)
	strat.tryCancelXOpenOrder("start")
	strat.xOrderSilentTime = time.Now().Add(strat.config.EnterSilent)
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
		case <-strat.spreadWalkTimer.C:
			strat.walkSpread()
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
	if strat.xSystemStatus != common.SystemStatusReady ||
		strat.ySystemStatus != common.SystemStatusReady {
		if time.Now().Sub(strat.logSilentTime) > 0 {
			strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
			logger.Debugf("hedgeYPosition xSystemStatus %v ySystemStatus %v", strat.xSystemStatus, strat.ySystemStatus)
		}
		return
	}
	if !strat.tradable ||
		strat.yPosition == nil ||
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
	strat.ySizeDiff = -strat.xPosition.GetSize()*strat.xMultiplier/strat.yMultiplier - strat.yPosition.GetSize()
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

func (strat *XYStrategy) updateEnterStepAndTarget() {
	if strat.xAccount == nil || strat.yAccount == nil {
		return
	}
	strat.enterStep = (strat.xAccount.GetFree() + strat.yAccount.GetFree()) * strat.config.EnterFreePct * strat.enterScale
	if strat.enterStep < strat.config.EnterMinimalStep {
		strat.enterStep = strat.config.EnterMinimalStep
	}
	strat.enterTarget = strat.enterStep * strat.config.EnterTargetFactor * strat.enterScale
	strat.usdtAvailable = math.Min(strat.xAccount.GetFree()*strat.xLeverage, strat.yAccount.GetFree()*strat.yLeverage)
}

func (strat *XYStrategy) walkSpread() {
	if strat.xWalkedDepth.Symbol == "" || strat.yWalkedDepth.Symbol == "" {
		return
	}
	//需要用ema time delta 对age diff进行修正
	strat.adjustedAgeDiff = strat.xWalkedDepth.Time.Sub(strat.yWalkedDepth.Time) + time.Duration(strat.xDepthFilter.TimeDeltaEma-strat.yDepthFilter.TimeDeltaEma)*time.Millisecond
	//取新一点的时间为spread time
	if strat.xWalkedDepth.Time.Sub(strat.yWalkedDepth.Time) < 0 {
		//需要对时间进行补偿
		strat.spreadTime = strat.yWalkedDepth.Time.Add(time.Millisecond * time.Duration(strat.yDepthFilter.TimeDeltaEma))
	} else {
		//需要对时间进行补偿
		strat.spreadTime = strat.xWalkedDepth.Time.Add(time.Millisecond * time.Duration(strat.xDepthFilter.TimeDeltaEma))
	}
	if strat.adjustedAgeDiff > strat.config.DepthMaxAgeDiffBias {
		strat.yDepthExpireCount++
		//logger.Debugf("%s x expire y %v %v %v", xSymbol, xDepthTime.Sub(yDepthTime), adjustedAgeDiff, -time.Duration(xDepthFilter.TimeDeltaEma-yDepthFilter.TimeDeltaEma)*time.Millisecond)
	} else if strat.adjustedAgeDiff < -strat.config.DepthMaxAgeDiffBias {
		//logger.Debugf("%s y expire x %v %v %v", xSymbol, xDepthTime.Sub(yDepthTime), adjustedAgeDiff, -time.Duration(xDepthFilter.TimeDeltaEma-yDepthFilter.TimeDeltaEma)*time.Millisecond)
		strat.xDepthExpireCount++
	}
	strat.depthMatchCount++
	strat.shortLastEnter = (strat.yWalkedDepth.BidPrice - strat.xWalkedDepth.BidPrice) / strat.xWalkedDepth.BidPrice
	strat.longLastEnter = (strat.yWalkedDepth.AskPrice - strat.xWalkedDepth.AskPrice) / strat.xWalkedDepth.AskPrice

	strat.shortEnterTimedMedian.Insert(strat.spreadTime, strat.shortLastEnter)
	strat.longEnterTimedMedian.Insert(strat.spreadTime, strat.longLastEnter)

	if strat.shortEnterTimedMedian.Len() < strat.config.SpreadMinDepthCount {
		return
	}
	if strat.shortEnterTimedMedian.Range() < strat.config.SpreadLookback/2 {
		return
	}
	strat.spread = &common.XYSpread{
		ShortLastEnter:   strat.shortLastEnter,
		ShortLastLeave:   strat.longLastEnter,
		ShortMedianEnter: strat.shortEnterTimedMedian.Median(),
		ShortMedianLeave: strat.longEnterTimedMedian.Median(),

		LongLastEnter:   strat.longLastEnter,
		LongLastLeave:   strat.shortLastEnter,
		LongMedianEnter: strat.longEnterTimedMedian.Median(),
		LongMedianLeave: strat.shortEnterTimedMedian.Median(),
		EventTime:       strat.spreadTime,
		ParseTime:       time.Now(),
	}
	strat.hedgeYPosition()
	strat.updateXOrder()
}

func (strat *XYStrategy) walkXDepth() {
	strat.error = common.WalkDepthWithMultiplier(strat.xDepth, strat.xMultiplier, strat.config.DepthTakerImpact, &strat.xWalkedDepth)
	if strat.error != nil {
		if time.Now().Sub(strat.logSilentTime) > 0 {
			logger.Debugf("x common.WalkDepthWithMultiplier error %v %s", strat.error, strat.xSymbol)
			strat.logSilentTime = time.Now().Add(time.Minute)
		}
	} else {
		strat.spreadWalkTimer.Reset(strat.config.SpreadWalkDelay)
	}
}

func (strat *XYStrategy) walkYDepth() {
	strat.error = common.WalkDepthWithMultiplier(strat.yDepth, strat.yMultiplier, strat.config.DepthTakerImpact, &strat.yWalkedDepth)
	if strat.error != nil {
		if time.Now().Sub(strat.logSilentTime) > 0 {
			logger.Debugf("y common.WalkDepthWithMultiplier error %v %s", strat.error, strat.ySymbol)
			strat.logSilentTime = time.Now().Add(time.Minute)
		}
	} else {
		strat.spreadWalkTimer.Reset(strat.config.SpreadWalkDelay)
	}
}

func (strat *XYStrategy) handleXDepth() {
	if strat.xDepth.GetTime().Sub(strat.xDepthTime) < 0 {
		return
	}
	strat.xDepthTime = strat.xDepth.GetTime()
	if !strat.xDepthFilter.Filter(strat.xDepth) && strat.yDepth != nil {
		strat.adjustedAgeDiff = strat.xDepthTime.Sub(strat.yDepthTime) + time.Duration(strat.xDepthFilter.TimeDeltaEma-strat.yDepthFilter.TimeDeltaEma)*time.Millisecond
		if strat.adjustedAgeDiff > strat.config.DepthMaxAgeDiffBias {
			//taker已经过期
			strat.yDepthExpireCount++
			//logger.Debugf("%s x expire y %v %v %v", xSymbol, xDepthTime.Sub(yDepthTime), adjustedAgeDiff, -time.Duration(xDepthFilter.TimeDeltaEma-yDepthFilter.TimeDeltaEma)*time.Millisecond)
		} else if strat.adjustedAgeDiff < -strat.config.DepthMaxAgeDiffBias {
			//maker已经过期
			strat.xDepthExpireCount++
			//logger.Debugf("%s y expire x %v %v %v", xSymbol, xDepthTime.Sub(yDepthTime), adjustedAgeDiff, -time.Duration(xDepthFilter.TimeDeltaEma-yDepthFilter.TimeDeltaEma)*time.Millisecond)
		} else {
			strat.xWalkDepthTimer.Reset(strat.config.DepthWalkDelay)
		}
	}
	strat.depthCount++
	if strat.depthCount > strat.config.DepthReportCount {
		strat.xDepthFilter.GenerateReport()
		strat.yDepthFilter.GenerateReport()
		strat.spreadReport = &common.XYSpreadReport{
			MatchRatio:        float64(strat.depthMatchCount) / float64(strat.depthCount),
			XSymbol:           strat.xSymbol,
			YSymbol:           strat.ySymbol,
			XTimeDeltaEma:     strat.xDepthFilter.TimeDeltaEma,
			YTimeDeltaEma:     strat.yDepthFilter.TimeDeltaEma,
			XTimeDelta:        strat.xDepthFilter.TimeDelta,
			YTimeDelta:        strat.yDepthFilter.TimeDelta,
			XDepthFilterRatio: strat.xDepthFilter.Report.FilterRatio,
			YDepthFilterRatio: strat.yDepthFilter.Report.FilterRatio,
			XExpireRatio:      float64(strat.xDepthExpireCount) / float64(strat.depthCount),
			YExpireRatio:      float64(strat.yDepthExpireCount) / float64(strat.depthCount),
		}
		strat.depthMatchCount = 0
		strat.depthCount = 0
		strat.yDepthExpireCount = 0
		strat.xDepthExpireCount = 0
	}
}

func (strat *XYStrategy) handleYDepth() {
	if strat.yDepth.GetTime().Sub(strat.yDepthTime) < 0 {
		return
	}
	strat.yDepthTime = strat.yDepth.GetTime()
	if !strat.yDepthFilter.Filter(strat.yDepth) && strat.xDepth != nil {
		strat.adjustedAgeDiff = strat.xDepthTime.Sub(strat.yDepthTime) + time.Duration(strat.xDepthFilter.TimeDeltaEma-strat.yDepthFilter.TimeDeltaEma)*time.Millisecond
		if strat.adjustedAgeDiff < -strat.config.DepthMaxAgeDiffBias {
			//maker已经过期
			//logger.Debugf("%s y expire x %v %v %v", xSymbol, xDepthTime.Sub(yDepthTime), adjustedAgeDiff, -time.Duration(xDepthFilter.TimeDeltaEma-yDepthFilter.TimeDeltaEma)*time.Millisecond)
			strat.xDepthExpireCount++
		} else if strat.adjustedAgeDiff > strat.config.DepthMaxAgeDiffBias {
			//logger.Debugf("%s x expire y %v %v %v", xSymbol, xDepthTime.Sub(yDepthTime), adjustedAgeDiff, -time.Duration(xDepthFilter.TimeDeltaEma-yDepthFilter.TimeDeltaEma)*time.Millisecond)
			//taker已经过期
			strat.yDepthExpireCount++
		} else {
			strat.yWalkDepthTimer.Reset(strat.config.DepthWalkDelay)
		}
	}
	strat.depthCount++
	if strat.depthCount > strat.config.DepthReportCount {
		strat.xDepthFilter.GenerateReport()
		strat.yDepthFilter.GenerateReport()
		strat.spreadReport = &common.XYSpreadReport{
			MatchRatio:        float64(strat.depthMatchCount) / float64(strat.depthCount),
			XSymbol:           strat.xSymbol,
			YSymbol:           strat.ySymbol,
			XTimeDeltaEma:     strat.xDepthFilter.TimeDeltaEma,
			YTimeDeltaEma:     strat.yDepthFilter.TimeDeltaEma,
			XTimeDelta:        strat.xDepthFilter.TimeDelta,
			YTimeDelta:        strat.yDepthFilter.TimeDelta,
			XDepthFilterRatio: strat.xDepthFilter.Report.FilterRatio,
			YDepthFilterRatio: strat.yDepthFilter.Report.FilterRatio,
			XExpireRatio:      float64(strat.xDepthExpireCount) / float64(strat.depthCount),
			YExpireRatio:      float64(strat.yDepthExpireCount) / float64(strat.depthCount),
		}
		strat.depthMatchCount = 0
		strat.depthCount = 0
		strat.yDepthExpireCount = 0
		strat.xDepthExpireCount = 0
	}
}

func (strat *XYStrategy) updateXOrder() {
	if strat.xSystemStatus != common.SystemStatusReady ||
		strat.ySystemStatus != common.SystemStatusReady {
		if time.Now().Sub(strat.logSilentTime) > 0 {
			strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
			logger.Debugf("updateXOrder xSystemStatus %v ySystemStatus %v", strat.xSystemStatus, strat.ySystemStatus)
		}
		return
	}

	if time.Now().Sub(strat.xPositionUpdateTime) > strat.config.BalancePositionMaxAge ||
		time.Now().Sub(strat.yPositionUpdateTime) > strat.config.BalancePositionMaxAge ||
		strat.xAccount == nil ||
		strat.yAccount == nil ||
		strat.xPosition == nil ||
		strat.yPosition == nil ||
		strat.spread == nil ||
		strat.xyFundingRate == nil ||
		time.Now().Sub(strat.spread.EventTime) > strat.config.SpreadTimeToLive ||
		!strat.tradable {
		if time.Now().Sub(strat.spread.EventTime) > strat.config.SpreadTimeToLive {
			strat.tryCancelXOpenOrder("spread time out")
		}
		return
	}

	strat.xSize = strat.xPosition.GetSize() * strat.xMultiplier
	strat.ySize = strat.yPosition.GetSize() * strat.yMultiplier
	strat.xValue = strat.xSize * strat.xWalkedDepth.MidPrice
	strat.yValue = strat.ySize * strat.yWalkedDepth.MidPrice
	strat.xAbsValue = math.Abs(strat.xValue)
	strat.yAbsValue = math.Abs(strat.yValue)
	strat.offsetFactor = (strat.xAbsValue + strat.yAbsValue) * 0.5 / strat.enterTarget
	strat.offsetStep = math.Min(strat.enterStep/strat.enterTarget, strat.offsetFactor)

	strat.shortTop = strat.config.ShortEnterDelta + strat.config.EnterOffsetDelta*strat.offsetFactor
	strat.shortBot = strat.config.ShortExitDelta + strat.config.ExitOffsetDelta*(strat.offsetFactor-strat.offsetStep)
	strat.longBot = strat.config.LongEnterDelta - strat.config.EnterOffsetDelta*strat.offsetFactor
	strat.longTop = strat.config.LongExitDelta - strat.config.ExitOffsetDelta*(strat.offsetFactor-strat.offsetStep)

	strat.midPrice = (strat.xWalkedDepth.MidPrice + strat.yWalkedDepth.MidPrice) * 0.5

	if time.Now().Sub(strat.xOrderSilentTime) < 0 {
		return
	}

	if math.Abs(strat.xValue+strat.yValue) > strat.enterStep*0.8 {
		if time.Now().Sub(strat.logSilentTime) > 0 {
			strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
			logger.Debugf(
				"%s %s unhedged value %f > 0.8*enterStep %f",
				strat.xSymbol, strat.ySymbol, math.Abs(strat.xValue+strat.yValue), strat.enterStep*0.8,
			)
		}
		strat.hedgeYPosition()
		strat.tryCancelXOpenOrder("unhedged value")
		return
	}

	if strat.xOpenOrder != nil {
		if !strat.isXOpenOrderOk() {
			strat.tryCancelXOpenOrder("open order not ok")
		}
		return
	}

	if strat.spread.ShortLastLeave < strat.shortBot &&
		strat.spread.ShortMedianLeave < strat.shortBot &&
		*strat.xyFundingRate < strat.config.MinimalKeepFundingRate &&
		strat.xSize >= strat.xStepSize*strat.xMultiplier {
		strat.enterValue = math.Min(4*strat.enterStep, math.Min(strat.xAbsValue, strat.yAbsValue))
		if *strat.xyFundingRate > strat.config.MinimalKeepFundingRate*0.5 {
			strat.enterValue = math.Min(2*strat.enterStep, math.Min(strat.xAbsValue, strat.yAbsValue))
		}
		strat.size = strat.enterValue / strat.midPrice
		strat.size = math.Round(strat.size/strat.xyMergedSpotStepSize) * strat.xyMergedSpotStepSize
		strat.enterValue = strat.size * strat.midPrice
		if strat.xAbsValue-strat.enterValue < strat.xyMergedSpotStepSize ||
			strat.yAbsValue-strat.enterValue < strat.xyMergedSpotStepSize ||
			strat.size > strat.xSize {
			//两种情况都把x全平，间接y全平
			strat.size = strat.xSize
		}
		strat.size = math.Floor(strat.size/strat.xMultiplier/strat.xStepSize) * strat.xStepSize
		if strat.size > 0 || strat.enterValue > 1.2*strat.xMinNotional {
			strat.price = math.Ceil(strat.xWalkedDepth.AskPrice*(1.0+strat.orderOffset.Top)/strat.xTickSize) * strat.xTickSize
			strat.xNewOrderParam = common.NewOrderParam{
				Symbol:      strat.xSymbol,
				Side:        common.OrderSideSell,
				Type:        common.OrderTypeLimit,
				Price:       strat.price,
				TimeInForce: common.OrderTimeInForceGTC,
				Size:        strat.size,
				PostOnly:    true,
				ReduceOnly:  true,
				ClientID:    strat.xExchange.GenerateClientID(),
			}
			strat.xOpenOrder = &strat.xNewOrderParam
			if !strat.config.DryRun {
				//logger.Debugf("sending strat.xOrderRequestCh <- common.OrderRequest %s", strat.xSymbol)
				select {
				case strat.xOrderRequestCh <- common.OrderRequest{
					New: &strat.xNewOrderParam,
				}:
					//logger.Debugf("sent strat.xOrderRequestCh <- common.OrderRequest %s", strat.xSymbol)
				}
			}
			strat.xLastFilledBuyPrice = nil
			strat.xLastFilledSellPrice = nil
			strat.yLastFilledBuyPrice = nil
			strat.yLastFilledSellPrice = nil
			strat.xOrderSilentTime = time.Now().Add(strat.config.OrderSilent)
			logger.Debugf(
				"%s %s SHORT BOT REDUCE %f < %f, %f < %f, SIZE %f, XDepthDiff %v YDepthDiff %v",
				strat.xSymbol, strat.ySymbol,
				strat.spread.ShortLastLeave, strat.shortBot,
				strat.spread.ShortMedianLeave, strat.shortBot,
				strat.size,
				time.Now().Sub(strat.xDepthTime),
				time.Now().Sub(strat.yDepthTime),
			)
		}
	} else if strat.spread.LongLastLeave > strat.longTop &&
		strat.spread.LongMedianLeave > strat.longTop &&
		*strat.xyFundingRate > -strat.config.MinimalKeepFundingRate &&
		strat.xSize <= -strat.xStepSize*strat.xMultiplier {

		strat.enterValue = math.Min(4*strat.enterStep, math.Min(strat.xAbsValue, strat.yAbsValue))
		if *strat.xyFundingRate < -strat.config.MinimalKeepFundingRate*0.5 {
			strat.enterValue = math.Min(2*strat.enterStep, math.Min(strat.xAbsValue, strat.yAbsValue))
		}
		strat.size = strat.enterValue / strat.midPrice
		strat.size = math.Round(strat.size/strat.xyMergedSpotStepSize) * strat.xyMergedSpotStepSize
		strat.enterValue = strat.size * strat.midPrice
		if strat.xAbsValue-strat.enterValue < strat.xyMergedSpotStepSize ||
			strat.yAbsValue-strat.enterValue < strat.xyMergedSpotStepSize ||
			strat.size > -strat.xSize {
			strat.size = -strat.xSize
		}
		strat.size = math.Floor(strat.size/strat.xMultiplier/strat.xStepSize) * strat.xStepSize
		if strat.size > 0 || strat.enterValue > 1.2*strat.xMinNotional {
			strat.price = math.Floor(strat.xWalkedDepth.BidPrice*(1.0+strat.orderOffset.Bot)/strat.xTickSize) * strat.xTickSize
			strat.xNewOrderParam = common.NewOrderParam{
				Symbol:      strat.xSymbol,
				Side:        common.OrderSideBuy,
				Type:        common.OrderTypeLimit,
				Price:       strat.price,
				TimeInForce: common.OrderTimeInForceGTC,
				Size:        strat.size,
				PostOnly:    true,
				ReduceOnly:  true,
				ClientID:    strat.xExchange.GenerateClientID(),
			}
			strat.xOpenOrder = &strat.xNewOrderParam
			if !strat.config.DryRun {
				//logger.Debugf("sending strat.xOrderRequestCh <- common.OrderRequest %s", strat.xSymbol)
				select {
				case strat.xOrderRequestCh <- common.OrderRequest{
					New: &strat.xNewOrderParam,
				}:
					//logger.Debugf("sent strat.xOrderRequestCh <- common.OrderRequest %s", strat.xSymbol)
				}
			}
			strat.xLastFilledBuyPrice = nil
			strat.xLastFilledSellPrice = nil
			strat.yLastFilledBuyPrice = nil
			strat.yLastFilledSellPrice = nil
			strat.xOrderSilentTime = time.Now().Add(strat.config.OrderSilent)
			logger.Debugf(
				"%s %s LONG TOP REDUCE %f > %f, %f > %f, SIZE %f, XDepthDiff %v YDepthDiff %v",
				strat.xSymbol, strat.ySymbol,
				strat.spread.LongLastLeave, strat.longTop,
				strat.spread.LongMedianLeave, strat.longTop,
				strat.size,
				time.Now().Sub(strat.xDepthTime),
				time.Now().Sub(strat.yDepthTime),
			)
		}
	} else if !strat.isYSpot &&
		strat.spread.ShortLastEnter > strat.shortTop &&
		strat.spread.ShortMedianEnter > strat.shortTop &&
		*strat.xyFundingRate > strat.config.MinimalEnterFundingRate &&
		strat.xSize >= 0 {

		strat.targetValue = math.Max(strat.xAbsValue, strat.yAbsValue) + strat.enterStep
		if strat.targetValue > strat.enterTarget {
			strat.targetValue = strat.enterTarget
		}
		strat.enterValue = strat.targetValue - math.Max(strat.xAbsValue, strat.yAbsValue)
		strat.size = strat.enterValue / strat.midPrice
		strat.size = math.Round(strat.size/strat.xyMergedSpotStepSize) * strat.xyMergedSpotStepSize
		strat.enterValue = strat.size * strat.midPrice

		if strat.enterValue > strat.usdtAvailable {
			if time.Now().Sub(strat.logSilentTime) > 0 {
				strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
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
		strat.size = math.Floor(strat.size/strat.xMultiplier/strat.xStepSize) * strat.xStepSize
		if strat.size <= 0 || strat.enterValue < 1.2*strat.yMinNotional || strat.enterValue < 1.2*strat.xMinNotional {
			if time.Now().Sub(strat.logSilentTime) > 0 {
				strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
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
		strat.price = math.Floor(strat.xWalkedDepth.BidPrice*(1.0+strat.orderOffset.Bot)/strat.xTickSize) * strat.xTickSize
		strat.xNewOrderParam = common.NewOrderParam{
			Symbol:      strat.xSymbol,
			Side:        common.OrderSideBuy,
			Type:        common.OrderTypeLimit,
			Price:       strat.price,
			TimeInForce: common.OrderTimeInForceGTC,
			Size:        strat.size,
			PostOnly:    true,
			ReduceOnly:  false,
			ClientID:    strat.xExchange.GenerateClientID(),
		}
		strat.xOpenOrder = &strat.xNewOrderParam
		if !strat.config.DryRun {
			//logger.Debugf("sending strat.xOrderRequestCh <- common.OrderRequest %s", strat.xSymbol)
			select {
			case strat.xOrderRequestCh <- common.OrderRequest{
				New: &strat.xNewOrderParam,
			}:
				//logger.Debugf("sent strat.xOrderRequestCh <- common.OrderRequest %s", strat.xSymbol)
			}
		}
		strat.xLastFilledBuyPrice = nil
		strat.xLastFilledSellPrice = nil
		strat.yLastFilledBuyPrice = nil
		strat.yLastFilledSellPrice = nil
		strat.xOrderSilentTime = time.Now().Add(strat.config.OrderSilent)
		logger.Debugf(
			"%s %s SHORT TOP OPEN %f > %f, %f > %f, SIZE %f, XDepthDiff %v YDepthDiff %v",
			strat.xSymbol, strat.ySymbol,
			strat.spread.ShortLastEnter, strat.shortTop,
			strat.spread.ShortMedianEnter, strat.shortTop,
			strat.size,
			time.Now().Sub(strat.xDepthTime),
			time.Now().Sub(strat.yDepthTime),
		)
	} else if !strat.isXSpot &&
		strat.spread.LongLastEnter < strat.longBot &&
		strat.spread.LongMedianEnter < strat.longBot &&
		*strat.xyFundingRate < -strat.config.MinimalEnterFundingRate &&
		strat.xSize <= 0 {

		strat.targetValue = math.Max(strat.xAbsValue, strat.yAbsValue) + strat.enterStep
		if strat.targetValue > strat.enterTarget {
			strat.targetValue = strat.enterTarget
		}
		strat.enterValue = strat.targetValue - math.Max(strat.xAbsValue, strat.yAbsValue)
		strat.size = strat.enterValue / strat.midPrice
		strat.size = math.Round(strat.size/strat.xyMergedSpotStepSize) * strat.xyMergedSpotStepSize
		strat.enterValue = strat.size * strat.midPrice
		if strat.enterValue > strat.usdtAvailable {
			if time.Now().Sub(strat.logSilentTime) > strat.config.LogInterval {
				strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
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
		strat.size = math.Floor(strat.size/strat.xMultiplier/strat.xStepSize) * strat.xStepSize
		if strat.size <= 0 || strat.enterValue < 1.2*strat.yMinNotional || strat.enterValue < 1.2*strat.xMinNotional {
			if time.Now().Sub(strat.logSilentTime) > 0 {
				strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
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
		strat.price = math.Ceil(strat.xWalkedDepth.AskPrice*(1.0+strat.orderOffset.Top)/strat.xTickSize) * strat.xTickSize
		strat.xNewOrderParam = common.NewOrderParam{
			Symbol:      strat.xSymbol,
			Side:        common.OrderSideSell,
			Type:        common.OrderTypeLimit,
			Price:       strat.price,
			TimeInForce: common.OrderTimeInForceGTC,
			Size:        strat.size,
			PostOnly:    true,
			ReduceOnly:  false,
			ClientID:    strat.xExchange.GenerateClientID(),
		}
		strat.xOpenOrder = &strat.xNewOrderParam
		if !strat.config.DryRun {
			//logger.Debugf("sending strat.xOrderRequestCh <- common.OrderRequest %s", strat.xSymbol)
			select {
			case strat.xOrderRequestCh <- common.OrderRequest{
				New: &strat.xNewOrderParam,
			}:
				//logger.Debugf("sent strat.xOrderRequestCh <- common.OrderRequest %s", strat.xSymbol)
			}
		}
		strat.xLastFilledBuyPrice = nil
		strat.xLastFilledSellPrice = nil
		strat.yLastFilledBuyPrice = nil
		strat.yLastFilledSellPrice = nil
		strat.xOrderSilentTime = time.Now().Add(strat.config.OrderSilent)
		logger.Debugf(
			"%s %s LONG BOT OPEN %f < %f, %f < %f, SIZE %f, XDepthDiff %v YDepthDiff %v",
			strat.xSymbol, strat.ySymbol,
			strat.spread.LongLastEnter, strat.longBot,
			strat.spread.LongMedianEnter, strat.longBot,
			strat.size,
			time.Now().Sub(strat.xDepthTime),
			time.Now().Sub(strat.yDepthTime),
		)

	}
}

func (strat *XYStrategy) isXOpenOrderOk() bool {

	//检查价格有没有在OFFSET范围内，不在撤掉
	if strat.xOpenOrder.Side == common.OrderSideBuy &&
		strat.xOpenOrder.Price < strat.xWalkedDepth.BidPrice*(1.0+strat.orderOffset.FarBot) {
		logger.Debugf("%s BUY PRICE %f < FAR BOT %f, CANCEL",
			strat.xSymbol,
			strat.xOpenOrder.Price,
			strat.xWalkedDepth.BidPrice*(1.0+strat.orderOffset.FarBot),
		)
		return false
	} else if strat.xOpenOrder.Side == common.OrderSideBuy &&
		strat.xOpenOrder.Price > strat.xWalkedDepth.BidPrice*(1.0+strat.orderOffset.NearBot) {
		logger.Debugf("%s BUY PRICE %f > NEAR BOT %f, CANCEL",
			strat.xSymbol,
			strat.xOpenOrder.Price,
			strat.xWalkedDepth.BidPrice*(1.0+strat.orderOffset.NearBot),
		)
		return false
	} else if strat.xOpenOrder.Side == common.OrderSideSell &&
		strat.xOpenOrder.Price > strat.xWalkedDepth.AskPrice*(1.0+strat.orderOffset.FarTop) {
		logger.Debugf("%s SELL PRICE %f > FAR TOP %f, CANCEL ",
			strat.xSymbol,
			strat.xOpenOrder.Price,
			strat.xWalkedDepth.AskPrice*(1.0+strat.orderOffset.FarTop),
		)
		return false
	} else if strat.xOpenOrder.Side == common.OrderSideSell &&
		strat.xOpenOrder.Price < strat.xWalkedDepth.AskPrice*(1.0+strat.orderOffset.NearTop) {
		logger.Debugf("%s SELL PRICE %f < NEAR TOP %f, CANCEL ",
			strat.xSymbol,
			strat.xOpenOrder.Price,
			strat.xWalkedDepth.AskPrice*(1.0+strat.orderOffset.NearTop),
		)
		return false
	}

	if strat.xOpenOrder.Side == common.OrderSideBuy &&
		!strat.xOpenOrder.ReduceOnly &&
		(strat.yWalkedDepth.BidPrice-strat.xOpenOrder.Price)/strat.xOpenOrder.Price > strat.shortTop {
		//买入开多, 是开空价差, 参考ShortTop
		return true
	} else if strat.xOpenOrder.Side == common.OrderSideSell &&
		strat.xOpenOrder.ReduceOnly &&
		(strat.yWalkedDepth.AskPrice-strat.xOpenOrder.Price)/strat.xOpenOrder.Price < strat.shortBot {
		//卖出平多, 是平空价, 参考ShortBot
		return true
	} else if strat.xOpenOrder.Side == common.OrderSideSell &&
		!strat.xOpenOrder.ReduceOnly &&
		(strat.yWalkedDepth.AskPrice-strat.xOpenOrder.Price)/strat.xOpenOrder.Price < strat.longBot {
		//卖出开空, 是开多价差, 参考LongBot
		return true
	} else if strat.xOpenOrder.Side == common.OrderSideBuy &&
		strat.xOpenOrder.ReduceOnly &&
		(strat.yWalkedDepth.BidPrice-strat.xOpenOrder.Price)/strat.xOpenOrder.Price > strat.longTop {
		//买入平空, 是平多价差, 参考LongTop
		return true
	}
	if strat.xOpenOrder.Side == common.OrderSideBuy {
		logger.Debugf(
			"NOT PROFITABLE %s BUY ORDER, CANCEL", strat.xSymbol,
		)
	} else {
		logger.Debugf(
			"NOT PROFITABLE %s SELL ORDER, CANCEL", strat.xSymbol,
		)
	}
	return false
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
				strat.xOrderSilentTime = time.Now().Add(strat.config.EnterSilent)
				strat.yOrderSilentTime = time.Now()
				if strat.xWalkedDepth.Symbol != "" {
					strat.xTimedPositionChange.Insert(time.Now(), math.Abs(strat.xPosition.GetSize()-nextPos.GetSize())*strat.xWalkedDepth.MidPrice*strat.xMultiplier)
				}
				strat.xPosition = nextPos
				strat.hedgeYPosition()
				logger.Debugf("%s x position change %f -> %f %v", nextPos.GetSymbol(), strat.xPosition.GetSize(), nextPos.GetSize(), nextPos.GetEventTime())
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

func (strat *XYStrategy) tryCancelXOpenOrder(reason string) {
	if time.Now().Sub(strat.xOrderSilentTime) < 0 {
		return
	}
	strat.xOrderSilentTime = time.Now().Add(strat.config.CancelSilent)
	if !strat.config.DryRun {
		//logger.Debugf("sending cancel strat.xOrderRequestCh <- common.OrderRequest %s %s", strat.xSymbol, reason)
		select {
		case strat.xOrderRequestCh <- common.OrderRequest{
			Cancel: &strat.xCancelOrderParam,
		}:
			//logger.Debugf("sent cancel strat.xOrderRequestCh <- common.OrderRequest %s %s", strat.xSymbol, reason)
		}
	}
	strat.xOpenOrder = nil
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
