package main

import (
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"time"
)

func (strat *XYStrategy) updateXPosition() {
	if strat.xSystemStatus != common.SystemStatusReady ||
		strat.ySystemStatus != common.SystemStatusReady {
		if time.Now().Sub(strat.logSilentTime) > 0 {
			strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
			logger.Debugf("%10s updateXPosition xSystemStatus %v ySystemStatus %v", strat.xSymbol, strat.xSystemStatus, strat.ySystemStatus)
		}
		return
	}

	if !strat.spreadReady ||
		!strat.targetWeightUpdated.True() ||
		strat.xPosition == nil ||
		strat.yPosition == nil ||
		strat.xyFundingRate == nil ||
		strat.xFundingRateFactor == nil ||
		strat.enterTarget == 0 {
		//if time.Now().Sub(strat.logSilentTime) > 0 {
		//strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
		//logger.Debugf("%s %v %v %v %v %v %v %v",
		//	strat.xSymbol,
		//	strat.spreadReady,
		//	!strat.targetWeightUpdated.True(),
		//	strat.xPosition == nil,
		//	strat.yPosition == nil,
		//	strat.xyFundingRate == nil,
		//	strat.xFundingRateFactor == nil,
		//	strat.enterTarget == 0,
		//)
		//}
		return
	}

	xSize := strat.xPosition.GetSize() * strat.xMultiplier
	ySize := strat.yPosition.GetSize() * strat.yMultiplier
	xValue := 0.0
	yValue := 0.0
	if strat.isXSpot || strat.xPosition.GetPrice() == 0 {
		xValue = xSize * strat.xMidPrice
	} else {
		xValue = xSize * strat.xPosition.GetPrice()
	}
	if strat.isYSpot || strat.yPosition.GetPrice() == 0 {
		yValue = ySize * strat.yMidPrice
	} else {
		yValue = ySize * strat.yPosition.GetPrice()
	}
	xAbsValue := math.Abs(xValue)
	yAbsValue := math.Abs(yValue)
	xyMidPrice := (strat.xMidPrice + strat.yMidPrice) * 0.5

	strat.offsetFactor = (xAbsValue + yAbsValue/strat.config.HedgeRatio) * 0.5 / strat.enterTarget
	strat.offsetStep = math.Min(strat.enterStep/strat.enterTarget, strat.offsetFactor)
	if strat.offsetFactor > 1.0 {
		strat.offsetFactor = 1.0
	}
	if strat.offsetStep > 1.0 {
		strat.offsetStep = 1.0
	}

	strat.tdSpreadMiddle = strat.stats.SpreadMiddle.Load()
	strat.tdSpreadEnterOffset = strat.stats.SpreadEnterOffset.Load()
	strat.tdSpreadExitOffset = strat.stats.SpreadLeaveOffset.Load()

	if xSize >= 0 {
		strat.thresholdShortTop = strat.tdSpreadMiddle + strat.config.ShortEnterThreshold + strat.tdSpreadEnterOffset*strat.offsetFactor - *strat.xyFundingRate**strat.xFundingRateFactor
		strat.thresholdShortBot = strat.tdSpreadMiddle + strat.config.ShortLeaveThreshold + strat.tdSpreadExitOffset*(strat.offsetFactor-strat.offsetStep) - *strat.xyFundingRate**strat.xFundingRateFactor
		strat.thresholdLongBot = strat.tdSpreadMiddle + strat.config.LongEnterThreshold - *strat.xyFundingRate**strat.xFundingRateFactor
		strat.thresholdLongTop = strat.tdSpreadMiddle + strat.config.LongLeaveThreshold - *strat.xyFundingRate**strat.xFundingRateFactor
	} else {
		strat.thresholdShortTop = strat.tdSpreadMiddle + strat.config.ShortEnterThreshold - *strat.xyFundingRate**strat.xFundingRateFactor
		strat.thresholdShortBot = strat.tdSpreadMiddle + strat.config.ShortLeaveThreshold - *strat.xyFundingRate**strat.xFundingRateFactor
		strat.thresholdLongBot = strat.tdSpreadMiddle + strat.config.LongEnterThreshold - strat.tdSpreadEnterOffset*strat.offsetFactor - *strat.xyFundingRate**strat.xFundingRateFactor
		strat.thresholdLongTop = strat.tdSpreadMiddle + strat.config.LongLeaveThreshold - strat.tdSpreadExitOffset*(strat.offsetFactor-strat.offsetStep) - *strat.xyFundingRate**strat.xFundingRateFactor
	}

	//if math.IsNaN(strat.thresholdLongBot) && time.Now().Sub(strat.logSilentTime) > 0 {
	//	strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
	//	logger.Debugf("%s enterTarget %f targetWeight %f enterTargetFactor %f", strat.xSymbol, strat.enterTarget, strat.targetWeight, strat.config.EnterTargetFactor)
	//}

	if time.Now().Sub(strat.xPositionUpdateTime) > strat.config.AccountMaxAge ||
		time.Now().Sub(strat.yPositionUpdateTime) > strat.config.AccountMaxAge ||
		strat.xAccount == nil ||
		strat.yAccount == nil ||
		strat.fundingRateSettleSilent ||
		time.Now().Sub(strat.spreadTickerTime) > strat.config.SpreadMaxAge ||
		time.Now().Sub(strat.xOrderSilentTime) < 0 {
		//if time.Now().Sub(strat.logSilentTime) > 0 {
		//strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
		//logger.Debugf("%s %v %v %v %v %v %v %v %v %v %v",
		//	strat.xSymbol,
		//	strat.config.AccountMaxAge,
		//	strat.xPositionUpdateTime, time.Now().Sub(strat.xPositionUpdateTime) > strat.config.AccountMaxAge,
		//	strat.yPositionUpdateTime, time.Now().Sub(strat.yPositionUpdateTime) > strat.config.AccountMaxAge,
		//	strat.xAccount == nil,
		//	strat.yAccount == nil,
		//	strat.fundingRateSettleSilent,
		//	time.Now().Sub(strat.spreadTickerTime) > strat.config.SpreadMaxAge,
		//	time.Now().Sub(strat.xOrderSilentTime) < 0,
		//)
		//}
		return
	}

	if math.Abs(xSize+ySize/strat.config.HedgeRatio)*xyMidPrice > strat.enterStep*0.8 {
		if time.Now().Sub(strat.logSilentTime) > 0 {
			strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
			logger.Debugf(
				"%s %s unhedged value %f > 0.8*enterStep %f",
				strat.xSymbol, strat.ySymbol, math.Abs(xValue+yValue), strat.enterStep*0.8,
			)
		}
		strat.hedgeXPosition()
		if time.Now().Sub(strat.hedgeCheckStopTime) > 0 {
			strat.hedgeYPosition()
		}
		return
	}

	if strat.spreadMedianLong < strat.thresholdShortBot &&
		strat.spreadLastLong < strat.spreadMedianLong &&
		xSize >= strat.xMinSize*strat.xMultiplier {
		strat.enterValue = math.Min(math.Max(4*strat.enterStep, xAbsValue*0.5), math.Min(xAbsValue, yAbsValue/strat.config.HedgeRatio))

		//两步，第一步看x的分布，用一个td之后的bidSize, 第二步不能超过y的td之后askSize的流动性
		tdXBidValue := strat.stats.XBidSize.Load() * strat.xMultiplier * xyMidPrice
		tdYAskValue := strat.stats.YAskSize.Load() * strat.yMultiplier * xyMidPrice
		if strat.enterValue > tdXBidValue {
			strat.enterValue = tdXBidValue
		}
		if strat.enterValue > tdYAskValue {
			strat.enterValue = tdYAskValue
		}
		xSizeDiff := strat.enterValue / xyMidPrice

		if xSizeDiff > strat.maxOrderSize {
			xSizeDiff = strat.maxOrderSize
		}

		if strat.config.BestSizeFactor > 0 {
			//限开仓大小限制到best bid ask size, 主要关心Y的深度，保证Y的深度足够
			xSizeDiff = math.Min(strat.yTicker.GetAskSize()*strat.yMultiplier*strat.config.BestSizeFactor, xSizeDiff)
		}
		xSizeDiff = math.Round(xSizeDiff/strat.xyMergedStepSize) * strat.xyMergedStepSize

		strat.enterValue = xSizeDiff * xyMidPrice
		if xAbsValue-strat.enterValue < strat.xyMergedStepSize*1.005 ||
			yAbsValue-strat.enterValue < strat.xyMergedStepSize*1.005 ||
			xSizeDiff > xSize {
			//两种情况都把x全平，间接y全平
			xSizeDiff = xSize
		}
		strat.enterValue = xSizeDiff * xyMidPrice
		xSizeDiff = math.Floor(xSizeDiff/strat.xMultiplier/strat.xStepSize) * strat.xStepSize
		if xSizeDiff >= strat.xMinSize && strat.enterValue >= 1.2*strat.xMinNotional {

			xPrice := strat.xTicker.GetBidPrice()
			//防止TickSize太大
			if strat.xTickSize/xPrice < strat.config.EnterSlippage {
				xPrice = xPrice * (1.0 - strat.config.EnterSlippage)
				xPrice = math.Floor(xPrice/strat.xTickSize) * strat.xTickSize
			}

			strat.xNewOrderParam = common.NewOrderParam{
				Symbol:      strat.xSymbol,
				Side:        common.OrderSideSell,
				Type:        common.OrderTypeLimit,
				Price:       xPrice,
				TimeInForce: strat.config.XOrderTimeInForce,
				Size:        xSizeDiff,
				PostOnly:    false,
				ReduceOnly:  true,
				ClientID:    strat.xExchange.GenerateClientID(),
			}
			if !strat.config.DryRun {
				select {
				case strat.xOrderRequestCh <- common.OrderRequest{
					New: &strat.xNewOrderParam,
				}:
				}
			}
			strat.xLastFilledBuyPrice = nil
			strat.xLastFilledSellPrice = nil
			strat.yLastFilledBuyPrice = nil
			strat.yLastFilledSellPrice = nil
			strat.xOrderSilentTime = time.Now().Add(strat.config.XOrderSilent)
			if strat.config.HedgeDelay > 0 {
				strat.hedgeCheckTimer.Reset(strat.config.HedgeDelay)
			}else{
				strat.hedgeCheckTimer.Reset(strat.config.HedgeCheckInterval)
			}
			strat.hedgeCheckStopTime = time.Now().Add(strat.config.HedgeCheckDuration)
			strat.lastEnterTime = strat.spreadTickerTime.Add(strat.config.XOrderSilent)
			strat.referenceSpread = strat.thresholdShortBot
			strat.referenceXPrice = strat.xTicker.GetBidPrice()
			strat.referenceYPrice = strat.yTicker.GetAskPrice()
			logger.Debugf(
				"%10s %10s SHORT BOT REDUCE %f < %f, %f < %f, PRICE %f SIZE %f, XTickerDiff %v YTickerDiff %v X %f %f Y %f %f, Offsets %f %f Fr %f",
				strat.xSymbol, strat.ySymbol,
				strat.spreadLastLong, strat.thresholdShortBot,
				strat.spreadMedianLong, strat.thresholdShortBot,
				xPrice,
				xSizeDiff,
				time.Now().Sub(strat.xTickerTime),
				time.Now().Sub(strat.yTickerTime),
				strat.xTicker.GetBidPrice(),
				strat.xTicker.GetAskPrice(),
				strat.yTicker.GetBidPrice(),
				strat.yTicker.GetAskPrice(),
				strat.tdSpreadEnterOffset,
				strat.tdSpreadExitOffset,
				*strat.xyFundingRate,
			)
		}
	} else if strat.spreadMedianShort > strat.thresholdLongTop &&
		strat.spreadLastShort > strat.spreadMedianShort &&
		xSize <= -strat.xMinSize*strat.xMultiplier {

		strat.enterValue = math.Min(math.Max(4*strat.enterStep, xAbsValue*0.5), math.Min(xAbsValue, yAbsValue/strat.config.HedgeRatio))

		//两步，第一步看x的分布，用一个td之后的askSize, 第二步不能超过y的td之后bidSize的流动性
		tdXAskValue := strat.stats.XAskSize.Load() * strat.xMultiplier * xyMidPrice
		tdYBidValue := strat.stats.YBidSize.Load() * strat.yMultiplier * xyMidPrice
		if strat.enterValue > tdXAskValue {
			strat.enterValue = tdXAskValue
		}
		if strat.enterValue > tdYBidValue {
			strat.enterValue = tdYBidValue
		}

		xSizeDiff := strat.enterValue / xyMidPrice

		if xSizeDiff > strat.maxOrderSize {
			xSizeDiff = strat.maxOrderSize
		}

		if strat.config.BestSizeFactor > 0 {
			xSizeDiff = math.Min(strat.yTicker.GetBidSize()*strat.yMultiplier*strat.config.BestSizeFactor, xSizeDiff)
		}

		xSizeDiff = math.Round(xSizeDiff/strat.xyMergedStepSize) * strat.xyMergedStepSize
		strat.enterValue = xSizeDiff * xyMidPrice
		if xAbsValue-strat.enterValue < strat.xyMergedStepSize*1.005 ||
			yAbsValue-strat.enterValue < strat.xyMergedStepSize*1.005 ||
			xSizeDiff > -xSize {
			xSizeDiff = -xSize
		}
		strat.enterValue = xSizeDiff * xyMidPrice
		xSizeDiff = math.Floor(xSizeDiff/strat.xMultiplier/strat.xStepSize) * strat.xStepSize
		if xSizeDiff >= strat.xMinSize && strat.enterValue >= 1.2*strat.xMinNotional {
			xPrice := strat.xTicker.GetAskPrice()
			if strat.xTickSize/xPrice < strat.config.EnterSlippage {
				xPrice = xPrice * (1.0 + strat.config.EnterSlippage)
				xPrice = math.Ceil(xPrice/strat.xTickSize) * strat.xTickSize
			}
			strat.xNewOrderParam = common.NewOrderParam{
				Symbol:      strat.xSymbol,
				Side:        common.OrderSideBuy,
				Type:        common.OrderTypeLimit,
				Price:       xPrice,
				TimeInForce: strat.config.XOrderTimeInForce,
				Size:        xSizeDiff,
				PostOnly:    false,
				ReduceOnly:  true,
				ClientID:    strat.xExchange.GenerateClientID(),
			}
			if !strat.config.DryRun {
				select {
				case strat.xOrderRequestCh <- common.OrderRequest{
					New: &strat.xNewOrderParam,
				}:
				}
			}
			strat.xLastFilledBuyPrice = nil
			strat.xLastFilledSellPrice = nil
			strat.yLastFilledBuyPrice = nil
			strat.yLastFilledSellPrice = nil
			strat.xOrderSilentTime = time.Now().Add(strat.config.XOrderSilent)
			if strat.config.HedgeDelay > 0 {
				strat.hedgeCheckTimer.Reset(strat.config.HedgeDelay)
			}else{
				strat.hedgeCheckTimer.Reset(strat.config.HedgeCheckInterval)
			}
			strat.hedgeCheckStopTime = time.Now().Add(strat.config.HedgeCheckDuration)
			strat.lastEnterTime = strat.spreadTickerTime.Add(strat.config.XOrderSilent)
			strat.referenceSpread = strat.thresholdLongTop
			strat.referenceXPrice = strat.xTicker.GetAskPrice()
			strat.referenceYPrice = strat.yTicker.GetBidPrice()
			logger.Debugf(
				"%10s %10s LONG TOP REDUCE %f > %f, %f > %f, PRICE %f SIZE %f, XTickerDiff %v YTickerDiff %v X %f %f Y %f %f, Offsets %f %f, Fr %f",
				strat.xSymbol, strat.ySymbol,
				strat.spreadLastShort, strat.thresholdLongTop,
				strat.spreadMedianShort, strat.thresholdLongTop,
				xPrice,
				xSizeDiff,
				time.Now().Sub(strat.xTickerTime),
				time.Now().Sub(strat.yTickerTime),
				strat.xTicker.GetBidPrice(),
				strat.xTicker.GetAskPrice(),
				strat.yTicker.GetBidPrice(),
				strat.yTicker.GetAskPrice(),
				strat.tdSpreadEnterOffset,
				strat.tdSpreadExitOffset,
				*strat.xyFundingRate,
			)
		}
	} else if !strat.reduceOnly &&
		!strat.isYSpot &&
		strat.tdSpreadMiddle > strat.config.SpreadMiddleMin &&
		strat.tdSpreadMiddle < strat.config.SpreadMiddleMax &&
		strat.spreadMedianShort > strat.thresholdShortTop &&
		strat.spreadLastShort > strat.spreadMedianShort &&
		*strat.xyFundingRate > strat.config.FundingRateOpenShortMin &&
		xSize > -strat.xMinSize*strat.xMultiplier &&
		strat.xAccount.GetFree() > strat.config.MinXFree &&
		strat.yAccount.GetFree() > strat.config.MinYFree &&
		xSize < strat.maxPosSize &&
		xAbsValue < strat.maxPosValue &&
		yAbsValue < strat.maxPosValue {

		//if strat.config.EnterWithProfitConfirms &&
		//	!strat.isXSpot &&
		//	xSize > 2*strat.xyMergedStepSize &&
		//	//spot balance has no entry price
		//	strat.xTicker.GetAskPrice() < strat.xPosition.GetPrice() {
		//	return
		//}

		strat.targetValue = math.Max(xAbsValue, yAbsValue/strat.config.HedgeRatio) + strat.enterStep
		if strat.targetValue > strat.enterTarget {
			strat.targetValue = strat.enterTarget
		}
		strat.enterValue = strat.targetValue - math.Max(xAbsValue, yAbsValue/strat.config.HedgeRatio)
		if strat.enterValue < 0 {
			//超过最大仓位了, 不操作
			strat.hedgeXPosition()
			return
		}

		//两步，第一步看x的分布，用一个td之后的askSize, 第二步不能超过y的td之后bidSize的流动性
		tdXAskValue := strat.stats.XAskSize.Load() * strat.xMultiplier * xyMidPrice
		tdYBidValue := strat.stats.YBidSize.Load() * strat.yMultiplier * xyMidPrice
		if strat.enterValue > tdXAskValue {
			strat.enterValue = tdXAskValue
		}
		if strat.enterValue > tdYBidValue {
			strat.enterValue = tdYBidValue
		}

		xSizeDiff := strat.enterValue / xyMidPrice
		if xSizeDiff > strat.maxOrderSize {
			xSizeDiff = strat.maxOrderSize
		}

		if strat.config.BestSizeFactor > 0 {
			xSizeDiff = math.Min(strat.yTicker.GetBidSize()*strat.yMultiplier*strat.config.BestSizeFactor, xSizeDiff)
		}

		xSizeDiff = math.Round(xSizeDiff/strat.xyMergedStepSize) * strat.xyMergedStepSize
		strat.enterValue = xSizeDiff * xyMidPrice
		if strat.enterValue > strat.usdAvailable {
			if time.Now().Sub(strat.logSilentTime) > 0 {
				strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
				logger.Debugf(
					"%10s %10s FAILED SHORT TOP OPEN, ENTRY VALUE %f MORE THAN usdAvailable %f, %f > %f, %f > %f, SIZE %f",
					strat.xSymbol,
					strat.ySymbol,
					strat.enterValue,
					strat.usdAvailable,
					strat.spreadLastShort, strat.thresholdShortTop,
					strat.spreadMedianShort, strat.thresholdShortTop,
					xSizeDiff,
				)
			}
			strat.hedgeXPosition()
			return
		}

		xSizeDiff = math.Floor(xSizeDiff/strat.xMultiplier/strat.xStepSize) * strat.xStepSize
		if strat.enterValue < 1.2*strat.yMinNotional || strat.enterValue < 1.2*strat.xMinNotional || xSizeDiff < strat.xMinSize {
			if time.Now().Sub(strat.logSilentTime) > 0 {
				strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
				logger.Debugf(
					"%10s %10s FAILED SHORT TOP OPEN, ORDER VALUE %f TOO SMALL, %f > %f, %f > %f, SIZE %f",
					strat.xSymbol, strat.ySymbol,
					strat.enterValue,
					strat.spreadLastShort, strat.thresholdShortTop,
					strat.spreadMedianShort, strat.thresholdShortTop,
					xSizeDiff,
				)
			}
			strat.hedgeXPosition()
			return
		}
		xPrice := strat.xTicker.GetAskPrice()
		if strat.xTickSize/xPrice < strat.config.EnterSlippage {
			xPrice = xPrice * (1.0 + strat.config.EnterSlippage)
			xPrice = math.Ceil(xPrice/strat.xTickSize) * strat.xTickSize
		}
		strat.xNewOrderParam = common.NewOrderParam{
			Symbol:      strat.xSymbol,
			Side:        common.OrderSideBuy,
			Type:        common.OrderTypeLimit,
			Price:       xPrice,
			TimeInForce: strat.config.XOrderTimeInForce,
			Size:        xSizeDiff,
			PostOnly:    false,
			ReduceOnly:  false,
			ClientID:    strat.xExchange.GenerateClientID(),
		}
		if !strat.config.DryRun {
			select {
			case strat.xOrderRequestCh <- common.OrderRequest{
				New: &strat.xNewOrderParam,
			}:
			}
		}
		strat.xLastFilledBuyPrice = nil
		strat.xLastFilledSellPrice = nil
		strat.yLastFilledBuyPrice = nil
		strat.yLastFilledSellPrice = nil
		strat.xOrderSilentTime = time.Now().Add(strat.config.XOrderSilent)
		if strat.config.HedgeDelay > 0 {
			strat.hedgeCheckTimer.Reset(strat.config.HedgeDelay)
		}else{
			strat.hedgeCheckTimer.Reset(strat.config.HedgeCheckInterval)
		}
		strat.hedgeCheckStopTime = time.Now().Add(strat.config.HedgeCheckDuration)
		strat.lastEnterTime = strat.spreadTickerTime.Add(strat.config.XOrderSilent)
		strat.referenceSpread = strat.thresholdShortTop
		strat.referenceXPrice = strat.xTicker.GetAskPrice()
		strat.referenceYPrice = strat.yTicker.GetBidPrice()
		logger.Debugf(
			"%10s %10s SHORT TOP OPEN %f > %f, %f > %f, PRICE %f SIZE %f, XTickerDiff %v YTickerDiff %v X %f %f Y %f %f, Offsets %f %f, Fr %f",
			strat.xSymbol, strat.ySymbol,
			strat.spreadLastShort, strat.thresholdShortTop,
			strat.spreadMedianShort, strat.thresholdShortTop,
			xPrice,
			xSizeDiff,
			time.Now().Sub(strat.xTickerTime),
			time.Now().Sub(strat.yTickerTime),
			strat.xTicker.GetBidPrice(),
			strat.xTicker.GetAskPrice(),
			strat.yTicker.GetBidPrice(),
			strat.yTicker.GetAskPrice(),
			strat.tdSpreadEnterOffset,
			strat.tdSpreadExitOffset,
			*strat.xyFundingRate,
		)
	} else if !strat.reduceOnly &&
		!strat.isXSpot &&
		strat.tdSpreadMiddle > strat.config.SpreadMiddleMin &&
		strat.tdSpreadMiddle < strat.config.SpreadMiddleMax &&
		strat.spreadMedianLong < strat.thresholdLongBot &&
		strat.spreadLastLong < strat.spreadMedianLong &&
		*strat.xyFundingRate < strat.config.FundingRateOpenLongMax &&
		xSize < strat.xMinSize*strat.xMultiplier &&
		strat.xAccount.GetFree() > strat.config.MinXFree &&
		strat.yAccount.GetFree() > strat.config.MinYFree &&
		xSize > -strat.maxPosSize &&
		xAbsValue < strat.maxPosValue &&
		yAbsValue < strat.maxPosValue {

		//if strat.config.EnterWithProfitConfirms &&
		//	xSize < -2*strat.xyMergedStepSize &&
		//	strat.xTicker.GetBidPrice() > strat.xPosition.GetPrice() {
		//	return
		//}

		strat.targetValue = math.Max(xAbsValue, yAbsValue/strat.config.HedgeRatio) + strat.enterStep
		if strat.targetValue > strat.enterTarget {
			strat.targetValue = strat.enterTarget
		}
		strat.enterValue = strat.targetValue - math.Max(xAbsValue, yAbsValue/strat.config.HedgeRatio)
		if strat.enterValue < 0 {
			//超过最大仓位了, 不操作
			strat.hedgeXPosition()
			return
		}

		//两步，第一步看x的分布，用一个td之后的bidSize, 第二步不能超过y的td之后askSize的流动性
		tdXBidValue := strat.stats.XBidSize.Load() * strat.xMultiplier * xyMidPrice
		tdYAskValue := strat.stats.YAskSize.Load() * strat.yMultiplier * xyMidPrice
		if strat.enterValue > tdXBidValue {
			strat.enterValue = tdXBidValue
		}
		if strat.enterValue > tdYAskValue {
			strat.enterValue = tdYAskValue
		}
		xSizeDiff := strat.enterValue / xyMidPrice

		xSizeDiff = strat.enterValue / xyMidPrice
		if xSizeDiff > strat.maxOrderSize {
			xSizeDiff = strat.maxOrderSize
		}

		if strat.config.BestSizeFactor > 0 {
			xSizeDiff = math.Min(strat.yTicker.GetAskSize()*strat.yMultiplier*strat.config.BestSizeFactor, xSizeDiff)
		}

		xSizeDiff = math.Round(xSizeDiff/strat.xyMergedStepSize) * strat.xyMergedStepSize
		strat.enterValue = xSizeDiff * xyMidPrice
		if strat.enterValue > strat.usdAvailable {
			if time.Now().Sub(strat.logSilentTime) > strat.config.LogInterval {
				strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
				logger.Debugf(
					"%10s %10s FAILED LONG BOT OPEN, ENTRY VALUE %f MORE THAN usdAvailable %f, %f < %f, %f < %f, SIZE %f",
					strat.xSymbol,
					strat.ySymbol,
					strat.enterValue,
					strat.usdAvailable,
					strat.spreadLastLong, strat.thresholdLongBot,
					strat.spreadMedianLong, strat.thresholdLongBot,
					xSizeDiff,
				)
			}
			strat.hedgeXPosition()
			return
		}
		xSizeDiff = math.Floor(xSizeDiff/strat.xMultiplier/strat.xStepSize) * strat.xStepSize
		if strat.enterValue < 1.2*strat.yMinNotional || strat.enterValue < 1.2*strat.xMinNotional || xSizeDiff < strat.xMinSize {
			if time.Now().Sub(strat.logSilentTime) > 0 {
				strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
				logger.Debugf(
					"%10s %10s FAILED LONG BOT OPEN, ORDER VALUE %f TOO SMALL, %f < %f, %f < %f, SIZE %f",
					strat.xSymbol, strat.ySymbol,
					strat.enterValue,
					strat.spreadLastLong, strat.thresholdLongBot,
					strat.spreadMedianLong, strat.thresholdLongBot,
					xSizeDiff,
				)
			}
			strat.hedgeXPosition()
			return
		}
		xPrice := strat.xTicker.GetBidPrice()
		//防止TickSize太大
		if strat.xTickSize/xPrice < strat.config.EnterSlippage {
			xPrice = xPrice * (1.0 - strat.config.EnterSlippage)
			xPrice = math.Floor(xPrice/strat.xTickSize) * strat.xTickSize
		}
		strat.xNewOrderParam = common.NewOrderParam{
			Symbol:      strat.xSymbol,
			Side:        common.OrderSideSell,
			Type:        common.OrderTypeLimit,
			Price:       xPrice,
			TimeInForce: strat.config.XOrderTimeInForce,
			Size:        xSizeDiff,
			PostOnly:    false,
			ReduceOnly:  false,
			ClientID:    strat.xExchange.GenerateClientID(),
		}
		if !strat.config.DryRun {
			select {
			case strat.xOrderRequestCh <- common.OrderRequest{
				New: &strat.xNewOrderParam,
			}:
			}
		}
		strat.xLastFilledBuyPrice = nil
		strat.xLastFilledSellPrice = nil
		strat.yLastFilledBuyPrice = nil
		strat.yLastFilledSellPrice = nil
		strat.xOrderSilentTime = time.Now().Add(strat.config.XOrderSilent)
		if strat.config.HedgeDelay > 0 {
			strat.hedgeCheckTimer.Reset(strat.config.HedgeDelay)
		}else{
			strat.hedgeCheckTimer.Reset(strat.config.HedgeCheckInterval)
		}
		strat.hedgeCheckStopTime = time.Now().Add(strat.config.HedgeCheckDuration)
		strat.lastEnterTime = strat.spreadTickerTime.Add(strat.config.XOrderSilent)
		strat.referenceSpread = strat.thresholdLongBot
		strat.referenceXPrice = strat.xTicker.GetBidPrice()
		strat.referenceYPrice = strat.yTicker.GetAskPrice()
		logger.Debugf(
			"%10s %10s LONG BOT OPEN %f < %f, %f < %f, PRICE %f SIZE %f, XTickerDiff %v YTickerDiff %v X %f %f Y %f %f, Offsets %f %f, Fr %f",
			strat.xSymbol, strat.ySymbol,
			strat.spreadLastLong, strat.thresholdLongBot,
			strat.spreadMedianLong, strat.thresholdLongBot,
			xPrice,
			xSizeDiff,
			time.Now().Sub(strat.xTickerTime),
			time.Now().Sub(strat.yTickerTime),
			strat.xTicker.GetBidPrice(),
			strat.xTicker.GetAskPrice(),
			strat.yTicker.GetBidPrice(),
			strat.yTicker.GetAskPrice(),
			strat.tdSpreadEnterOffset,
			strat.tdSpreadExitOffset,
			*strat.xyFundingRate,
		)

	}

}

func (strat *XYStrategy) hedgeXPosition() {
	//如果lastSpreadEnterTime没有更新，说明没有信号触发，就需要检查对冲的情况
	if time.Now().Sub(strat.lastEnterTime) > strat.config.XEnterTimeout {

		//如果已经没有信号对冲，重新检查x y的仓位，对冲较小的
		if math.Abs(strat.xPosition.GetSize()*strat.xMultiplier) < math.Abs(strat.yPosition.GetSize()*strat.yMultiplier/strat.config.HedgeRatio) {
			//X的size比Y小，不用操作X
			return
		}
		xSizeDiff := -strat.yPosition.GetSize()*strat.yMultiplier/strat.config.HedgeRatio/strat.xMultiplier - strat.xPosition.GetSize()

		//下单也加上控制，以防下单太大，造成市场冲击
		if strat.stats.Ready.True() {
			tdXBidSize := strat.stats.XBidSize.Load()
			tdXAskSize := strat.stats.XAskSize.Load()
			if xSizeDiff < -tdXBidSize {
				xSizeDiff = -tdXBidSize
			} else if xSizeDiff > tdXAskSize {
				xSizeDiff = tdXAskSize
			}
		}

		if xSizeDiff > strat.maxOrderSize {
			xSizeDiff = strat.maxOrderSize
		} else if xSizeDiff < -strat.maxOrderSize {
			xSizeDiff = -strat.maxOrderSize
		}

		if xSizeDiff >= 0 {
			xSizeDiff = math.Floor(xSizeDiff/strat.xStepSize) * strat.xStepSize
		} else {
			xSizeDiff = math.Ceil(xSizeDiff/strat.xStepSize) * strat.xStepSize
		}

		if strat.isXSpot {
			if math.Abs(xSizeDiff) < strat.xStepSize {
				return
			} else if xSizeDiff < 0 && -xSizeDiff*strat.xMultiplier*strat.xTicker.GetBidPrice() < 1.2*strat.xMinNotional {
				return
			} else if xSizeDiff > 0 && xSizeDiff*strat.xMultiplier*strat.xTicker.GetAskPrice() < 1.2*strat.xMinNotional {
				return
			}
		} else {
			//期货可以close仓位，没有minNotional限制
			if math.Abs(xSizeDiff) < strat.xStepSize {
				return
			} else if xSizeDiff < 0 && strat.xPosition.GetSize() <= 0 && -xSizeDiff*strat.xMultiplier*strat.xTicker.GetBidPrice() < 1.2*strat.xMinNotional {
				return
			} else if xSizeDiff > 0 && strat.xPosition.GetSize() >= 0 && xSizeDiff*strat.xMultiplier*strat.xTicker.GetAskPrice() < 1.2*strat.xMinNotional {
				return
			}
		}

		var xPrice float64
		var orderSide common.OrderSide
		if xSizeDiff < 0 {
			orderSide = common.OrderSideSell
			xSizeDiff = -xSizeDiff
			xPrice = strat.xTicker.GetBidPrice()
			//防止TickSize太大
			if strat.xTickSize/xPrice < strat.config.EnterSlippage {
				xPrice = xPrice * (1.0 - strat.config.EnterSlippage)
				xPrice = math.Floor(xPrice/strat.xTickSize) * strat.xTickSize
			}
		} else {
			orderSide = common.OrderSideBuy
			xPrice = strat.xTicker.GetAskPrice()
			//防止TickSize太大
			if strat.xTickSize/xPrice < strat.config.EnterSlippage {
				xPrice = xPrice * (1.0 + strat.config.EnterSlippage)
				xPrice = math.Ceil(xPrice/strat.xTickSize) * strat.xTickSize
			}
		}
		strat.xNewOrderParam = common.NewOrderParam{
			Symbol:      strat.xSymbol,
			Side:        orderSide,
			Type:        common.OrderTypeLimit,
			Price:       xPrice,
			TimeInForce: strat.config.XOrderTimeInForce,
			Size:        xSizeDiff,
			PostOnly:    false,
			ReduceOnly:  true,
			ClientID:    strat.xExchange.GenerateClientID(),
		}
		if !strat.config.DryRun {
			select {
			case strat.xOrderRequestCh <- common.OrderRequest{
				New: &strat.xNewOrderParam,
			}:
			}
		}
		strat.xLastFilledBuyPrice = nil
		strat.xLastFilledSellPrice = nil
		strat.yLastFilledBuyPrice = nil
		strat.yLastFilledSellPrice = nil
		strat.xOrderSilentTime = time.Now().Add(strat.config.XOrderSilent)
		logger.Debugf(
			"%10s %10s REVERSE HEDGE X BY Y, SIZE X %f Y %f, ORDER SIDE %s SIZE %f PRICE %f",
			strat.xSymbol, strat.ySymbol,
			strat.xPosition.GetSize()*strat.xMultiplier,
			strat.yPosition.GetSize()*strat.yMultiplier,
			orderSide,
			xSizeDiff,
			xPrice,
		)
	}
}

func (strat *XYStrategy) hedgeYPosition() {
	if strat.xSystemStatus != common.SystemStatusReady ||
		strat.ySystemStatus != common.SystemStatusReady {
		if time.Now().Sub(strat.logSilentTime) > 0 {
			strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
			logger.Debugf("%10s hedgeYPosition xSystemStatus %v ySystemStatus %v", strat.xSymbol, strat.xSystemStatus, strat.ySystemStatus)
		}
		return
	}
	if strat.yPosition == nil ||
		strat.xPosition == nil ||
		time.Now().Sub(strat.yPositionUpdateTime) > strat.config.AccountMaxAge ||
		time.Now().Sub(strat.yOrderSilentTime) < 0 {
		//if time.Now().Sub(strat.logSilentTime) > 0 {
		//	strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
		//	logger.Debugf("hedgeYPosition skipped order silent time %v positionUpdateTime %v", time.Now().Sub(strat.yOrderSilentTime), time.Now().Sub(strat.yPositionUpdateTime))
		//}
		return
	}
	var ySizeDiff float64
	if time.Now().Sub(strat.lastEnterTime) < strat.config.XEnterTimeout {
		ySizeDiff = -strat.xPosition.GetSize()*strat.xMultiplier*strat.config.HedgeRatio/strat.yMultiplier - strat.yPosition.GetSize()
	} else {
		//其他时间对冲小的size, 防止出现一边爆仓的情况
		if math.Abs(strat.xPosition.GetSize()*strat.xMultiplier) > math.Abs(strat.yPosition.GetSize()*strat.yMultiplier/strat.config.HedgeRatio) {
			//Y的size比X小，不用操作Y
			return
		} else {
			ySizeDiff = -strat.xPosition.GetSize()*strat.xMultiplier*strat.config.HedgeRatio/strat.yMultiplier - strat.yPosition.GetSize()
		}
	}
	if math.Abs(ySizeDiff) < strat.yStepSize {
		return
	}

	//下单也加上控制，以限下单太大，造成市场冲击
	if strat.stats.Ready.True() {
		tdYBidSize := strat.stats.YBidSize.Load()
		tdYAskSize := strat.stats.YAskSize.Load()
		if ySizeDiff < -tdYBidSize {
			ySizeDiff = -tdYBidSize
		} else if ySizeDiff > tdYAskSize {
			ySizeDiff = tdYAskSize
		}
	}

	if ySizeDiff > strat.maxOrderSize {
		ySizeDiff = strat.maxOrderSize
	} else if ySizeDiff < -strat.maxOrderSize {
		ySizeDiff = -strat.maxOrderSize
	}

	if ySizeDiff >= 0 {
		ySizeDiff = math.Floor(ySizeDiff/strat.yStepSize) * strat.yStepSize
	} else {
		ySizeDiff = math.Ceil(ySizeDiff/strat.yStepSize) * strat.yStepSize
	}

	if strat.isYSpot {
		if math.Abs(ySizeDiff) < strat.yStepSize || math.Abs(ySizeDiff) < strat.yMinSize {
			return
		} else if ySizeDiff < 0 && -ySizeDiff*strat.yMultiplier*strat.yTicker.GetBidPrice() < 1.2*strat.yMinNotional {
			return
		} else if ySizeDiff > 0 && ySizeDiff*strat.yMultiplier*strat.yTicker.GetAskPrice() < 1.2*strat.yMinNotional {
			return
		}
	} else {
		//期货以close仓位，没有minNotional限制
		if math.Abs(ySizeDiff) < strat.yStepSize || math.Abs(ySizeDiff) < strat.yMinSize {
			return
		} else if ySizeDiff < 0 && strat.yPosition.GetSize() <= 0 && -ySizeDiff*strat.yMultiplier*strat.yTicker.GetBidPrice() < 1.2*strat.yMinNotional {
			return
		} else if ySizeDiff > 0 && strat.yPosition.GetSize() >= 0 && ySizeDiff*strat.yMultiplier*strat.yTicker.GetAskPrice() < 1.2*strat.yMinNotional {
			return
		}
	}

	reduceOnly := false
	if ySizeDiff*strat.yPosition.GetSize() < 0 && math.Abs(ySizeDiff)*0.995 <= math.Abs(strat.yPosition.GetSize()) {
		reduceOnly = true
	}
	orderSide := common.OrderSideBuy
	if strat.config.HedgeByLimit {
		yPrice := 0.0
		if ySizeDiff < 0 {
			orderSide = common.OrderSideSell
			ySizeDiff = -ySizeDiff
			yPrice = strat.yTicker.GetBidPrice()
			//防止TickSize太大
			if strat.yTickSize/yPrice < strat.config.EnterSlippage {
				yPrice = yPrice * (1.0 - strat.config.EnterSlippage)
				yPrice = math.Floor(yPrice/strat.yTickSize) * strat.yTickSize
			}
		} else {
			orderSide = common.OrderSideBuy
			yPrice = strat.yTicker.GetAskPrice()
			//防止TickSize太大
			if strat.yTickSize/yPrice < strat.config.EnterSlippage {
				yPrice = yPrice * (1.0 + strat.config.EnterSlippage)
				yPrice = math.Ceil(yPrice/strat.yTickSize) * strat.yTickSize
			}
		}
		strat.yNewOrderParam = common.NewOrderParam{
			Symbol:      strat.ySymbol,
			Side:        orderSide,
			Type:        common.OrderTypeLimit,
			Price:       yPrice,
			TimeInForce: strat.config.YOrderTimeInForce,
			Size:        ySizeDiff,
			ReduceOnly:  reduceOnly,
			ClientID:    strat.yExchange.GenerateClientID(),
		}
	} else {
		if ySizeDiff < 0 {
			orderSide = common.OrderSideSell
			ySizeDiff = -ySizeDiff
		}
		strat.yNewOrderParam = common.NewOrderParam{
			Symbol:     strat.ySymbol,
			Side:       orderSide,
			Type:       common.OrderTypeMarket,
			Size:       ySizeDiff,
			ReduceOnly: reduceOnly,
			ClientID:   strat.yExchange.GenerateClientID(),
		}
	}
	if !strat.config.DryRun {
		select {
		case strat.yOrderRequestCh <- common.OrderRequest{
			New: &strat.yNewOrderParam,
		}:
			strat.yOrderSilentTime = time.Now().Add(strat.config.YOrderSilent)
			strat.yPositionUpdateTime = time.Unix(0, 0)
		}
	} else {
		strat.yOrderSilentTime = time.Now().Add(strat.config.YOrderSilent)
		strat.yPositionUpdateTime = time.Unix(0, 0)
	}
}
