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
			logger.Debugf("updateXPosition xSystemStatus %v ySystemStatus %v", strat.xSystemStatus, strat.ySystemStatus)
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
		strat.quantileMiddle == nil ||
		strat.fundingRateSettleSilent ||
		time.Now().Sub(strat.spread.EventTime) > strat.config.SpreadTimeToEnter {
		//if time.Now().Sub(strat.logSilentTime) > 0 {
		//	strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
		//	logger.Debugf("time.Now().Sub(strat.spread.EventTime) %v", time.Now().Sub(strat.spread.EventTime))
		//}
		return
	}

	strat.xSize = strat.xPosition.GetSize() * strat.xMultiplier
	strat.ySize = strat.yPosition.GetSize() * strat.yMultiplier
	strat.xValue = strat.xSize * strat.xMidPrice
	strat.yValue = strat.ySize * strat.yMidPrice
	strat.xAbsValue = math.Abs(strat.xValue)
	strat.yAbsValue = math.Abs(strat.yValue)
	strat.offsetFactor = (strat.xAbsValue + strat.yAbsValue) * 0.5 / strat.enterTarget
	strat.offsetStep = math.Min(strat.enterStep/strat.enterTarget, strat.offsetFactor)

	if strat.xSize >= 0 {
		strat.shortTop = *strat.quantileMiddle + strat.config.ShortEnterDelta + strat.config.EnterOffsetDelta*strat.offsetFactor - *strat.xyFundingRate*strat.config.FrOffsetFactor
		strat.shortBot = *strat.quantileMiddle + strat.config.ShortExitDelta + strat.config.ExitOffsetDelta*(strat.offsetFactor-strat.offsetStep) - *strat.xyFundingRate*strat.config.FrOffsetFactor
		strat.longBot = *strat.quantileMiddle + strat.config.LongEnterDelta - *strat.xyFundingRate*strat.config.FrOffsetFactor
		strat.longTop = *strat.quantileMiddle + strat.config.LongExitDelta - *strat.xyFundingRate*strat.config.FrOffsetFactor
	} else {
		strat.shortTop = *strat.quantileMiddle + strat.config.ShortEnterDelta - *strat.xyFundingRate*strat.config.FrOffsetFactor
		strat.shortBot = *strat.quantileMiddle + strat.config.ShortExitDelta - *strat.xyFundingRate*strat.config.FrOffsetFactor
		strat.longBot = *strat.quantileMiddle + strat.config.LongEnterDelta - strat.config.EnterOffsetDelta*strat.offsetFactor - *strat.xyFundingRate*strat.config.FrOffsetFactor
		strat.longTop = *strat.quantileMiddle + strat.config.LongExitDelta - strat.config.ExitOffsetDelta*(strat.offsetFactor-strat.offsetStep) - *strat.xyFundingRate*strat.config.FrOffsetFactor
	}

	strat.midPrice = (strat.xMidPrice + strat.yMidPrice) * 0.5
	if math.IsNaN(strat.longBot) && time.Now().Sub(strat.logSilentTime) > 0 {
		strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
		logger.Debugf("%s enterTarget %f targetWeight %f EnterTargetFactor %f", strat.xSymbol, strat.enterTarget, strat.targetWeight, strat.config.EnterTargetFactor)
	}

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
		if time.Now().Sub(strat.hedgeCheckStopTime) > 0 {
			strat.hedgeYPosition()
		}
		return
	}

	if strat.spread.ShortLastLeave < strat.shortBot &&
		strat.spread.ShortMedianLeave < strat.shortBot &&
		strat.spread.ShortLastLeave < strat.spread.ShortMedianLeave &&
		*strat.xyFundingRate < strat.config.MinimalKeepFundingRate &&
		strat.xSize >= strat.xStepSize*strat.xMultiplier {
		strat.enterValue = math.Min(math.Max(4*strat.enterStep, strat.xAbsValue*0.5), math.Min(strat.xAbsValue, strat.yAbsValue))
		if *strat.xyFundingRate > strat.config.MinimalKeepFundingRate*0.5 {
			strat.enterValue = math.Min(math.Max(2*strat.enterStep, strat.xAbsValue*0.25), math.Min(strat.xAbsValue, strat.yAbsValue))
		}
		if strat.enterValue > strat.maxOrderValue {
			strat.enterValue = strat.maxOrderValue
		}
		strat.size = strat.enterValue / strat.midPrice

		//限开仓大小限制到best bid ask size
		strat.size = math.Min(strat.xTicker.GetBidSize()*strat.xMultiplier*strat.config.BestSizeFactor, strat.size)
		strat.size = math.Min(strat.yTicker.GetAskSize()*strat.yMultiplier*strat.config.BestSizeFactor, strat.size)
		strat.size = math.Round(strat.size/strat.xyMergedSpotStepSize) * strat.xyMergedSpotStepSize

		strat.enterValue = strat.size * strat.midPrice
		if strat.xAbsValue-strat.enterValue < strat.xyMergedSpotStepSize*1.005 ||
			strat.yAbsValue-strat.enterValue < strat.xyMergedSpotStepSize*1.005 ||
			strat.size > strat.xSize {
			//两种情况都把x全平，间接y全平
			strat.size = strat.xSize
		}
		strat.size = math.Floor(strat.size/strat.xMultiplier/strat.xStepSize) * strat.xStepSize
		if strat.size > 0 && (!strat.isXSpot || strat.enterValue > 1.2*strat.xMinNotional) {
			strat.price = strat.xTicker.GetBidPrice()
			strat.xNewOrderParam = common.NewOrderParam{
				Symbol:      strat.xSymbol,
				Side:        common.OrderSideSell,
				Type:        common.OrderTypeLimit,
				Price:       strat.price,
				TimeInForce: common.OrderTimeInForceFOK,
				Size:        strat.size,
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
			strat.hedgeCheckTimer.Reset(strat.config.HedgeDelay)
			strat.hedgeCheckStopTime = time.Now().Add(strat.config.HedgeCheckDuration)
			logger.Debugf(
				"%s %s SHORT BOT REDUCE %f < %f, %f < %f, PRICE %f SIZE %f, XTickerDiff %v YTickerDiff %v",
				strat.xSymbol, strat.ySymbol,
				strat.spread.ShortLastLeave, strat.shortBot,
				strat.spread.ShortMedianLeave, strat.shortBot,
				strat.price,
				strat.size,
				time.Now().Sub(strat.xTickerTime),
				time.Now().Sub(strat.yTickerTime),
			)
		}
		return
	} else if strat.spread.LongLastLeave > strat.longTop &&
		strat.spread.LongMedianLeave > strat.longTop &&
		strat.spread.LongLastLeave > strat.spread.LongMedianLeave &&
		*strat.xyFundingRate > -strat.config.MinimalKeepFundingRate &&
		strat.xSize <= -strat.xStepSize*strat.xMultiplier {

		strat.enterValue = math.Min(math.Max(4*strat.enterStep, strat.xAbsValue*0.5), math.Min(strat.xAbsValue, strat.yAbsValue))
		if *strat.xyFundingRate < -strat.config.MinimalKeepFundingRate*0.5 {
			strat.enterValue = math.Min(math.Max(2*strat.enterStep, strat.xAbsValue*0.25), math.Min(strat.xAbsValue, strat.yAbsValue))
		}
		if strat.enterValue > strat.maxOrderValue {
			strat.enterValue = strat.maxOrderValue
		}
		strat.size = strat.enterValue / strat.midPrice
		//限开仓大小限制到best bid ask size
		strat.size = math.Min(strat.xTicker.GetAskSize()*strat.xMultiplier*strat.config.BestSizeFactor, strat.size)
		strat.size = math.Min(strat.yTicker.GetBidSize()*strat.yMultiplier*strat.config.BestSizeFactor, strat.size)

		strat.size = math.Round(strat.size/strat.xyMergedSpotStepSize) * strat.xyMergedSpotStepSize
		strat.enterValue = strat.size * strat.midPrice
		if strat.xAbsValue-strat.enterValue < strat.xyMergedSpotStepSize*1.005 ||
			strat.yAbsValue-strat.enterValue < strat.xyMergedSpotStepSize*1.005 ||
			strat.size > -strat.xSize {
			strat.size = -strat.xSize
		}
		strat.size = math.Floor(strat.size/strat.xMultiplier/strat.xStepSize) * strat.xStepSize
		if strat.size > 0 && (!strat.isXSpot || strat.enterValue > 1.2*strat.xMinNotional) {
			strat.price = strat.xTicker.GetAskPrice()
			strat.xNewOrderParam = common.NewOrderParam{
				Symbol:      strat.xSymbol,
				Side:        common.OrderSideBuy,
				Type:        common.OrderTypeLimit,
				Price:       strat.price,
				TimeInForce: common.OrderTimeInForceFOK,
				Size:        strat.size,
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
			strat.hedgeCheckTimer.Reset(strat.config.HedgeDelay)
			strat.hedgeCheckStopTime = time.Now().Add(strat.config.HedgeCheckDuration)
			logger.Debugf(
				"%s %s LONG TOP REDUCE %f > %f, %f > %f, PRICE %f SIZE %f, XTickerDiff %v YTickerDiff %v X %f %f Y %f %f",
				strat.xSymbol, strat.ySymbol,
				strat.spread.LongLastLeave, strat.longTop,
				strat.spread.LongMedianLeave, strat.longTop,
				strat.price,
				strat.size,
				time.Now().Sub(strat.xTickerTime),
				time.Now().Sub(strat.yTickerTime),
				strat.xTicker.GetBidPrice(),
				strat.xTicker.GetAskPrice(),
				strat.yTicker.GetBidPrice(),
				strat.yTicker.GetAskPrice(),
			)
		}
		return
	} else if !strat.config.ReduceOnly &&
		!strat.isYSpot &&
		strat.spread.ShortLastEnter > strat.shortTop &&
		strat.spread.ShortMedianEnter > strat.shortTop &&
		strat.spread.ShortLastEnter > strat.spread.ShortMedianEnter &&
		*strat.xyFundingRate > strat.config.MinimalEnterFundingRate &&
		strat.xSize > -strat.xStepSize*strat.xMultiplier {

		strat.targetValue = math.Max(strat.xAbsValue, strat.yAbsValue) + strat.enterStep
		if strat.targetValue > strat.enterTarget {
			strat.targetValue = strat.enterTarget
		}
		strat.enterValue = strat.targetValue - math.Max(strat.xAbsValue, strat.yAbsValue)
		if strat.enterValue > strat.maxOrderValue {
			strat.enterValue = strat.maxOrderValue
		}
		strat.size = strat.enterValue / strat.midPrice
		strat.size = math.Min(strat.xTicker.GetAskSize()*strat.xMultiplier*strat.config.BestSizeFactor, strat.size)
		strat.size = math.Min(strat.yTicker.GetBidSize()*strat.yMultiplier*strat.config.BestSizeFactor, strat.size)

		strat.size = math.Round(strat.size/strat.xyMergedSpotStepSize) * strat.xyMergedSpotStepSize
		strat.enterValue = strat.size * strat.midPrice
		if strat.enterValue > strat.usdAvailable {
			if time.Now().Sub(strat.logSilentTime) > 0 {
				strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
				logger.Debugf(
					"%s %s FAILED SHORT TOP OPEN, ENTRY VALUE %f MORE THAN usdAvailable %f, %f > %f, %f > %f, SIZE %f",
					strat.xSymbol,
					strat.ySymbol,
					strat.enterValue,
					strat.usdAvailable,
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
		strat.price = strat.xTicker.GetAskPrice()
		strat.xNewOrderParam = common.NewOrderParam{
			Symbol:      strat.xSymbol,
			Side:        common.OrderSideBuy,
			Type:        common.OrderTypeLimit,
			Price:       strat.price,
			TimeInForce: common.OrderTimeInForceFOK,
			Size:        strat.size,
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
		strat.hedgeCheckTimer.Reset(strat.config.HedgeDelay)
		strat.hedgeCheckStopTime = time.Now().Add(strat.config.HedgeCheckDuration)
		logger.Debugf(
			"%s %s SHORT TOP OPEN %f > %f, %f > %f, PRICE %f SIZE %f, XTickerDiff %v YTickerDiff %v X %f %f Y %f %f",
			strat.xSymbol, strat.ySymbol,
			strat.spread.ShortLastEnter, strat.shortTop,
			strat.spread.ShortMedianEnter, strat.shortTop,
			strat.price,
			strat.size,
			time.Now().Sub(strat.xTickerTime),
			time.Now().Sub(strat.yTickerTime),
			strat.xTicker.GetBidPrice(),
			strat.xTicker.GetAskPrice(),
			strat.yTicker.GetBidPrice(),
			strat.yTicker.GetAskPrice(),
		)
	} else if !strat.config.ReduceOnly &&
		!strat.isXSpot &&
		strat.spread.LongLastEnter < strat.longBot &&
		strat.spread.LongMedianEnter < strat.longBot &&
		strat.spread.LongLastEnter < strat.spread.LongMedianEnter &&
		*strat.xyFundingRate < -strat.config.MinimalEnterFundingRate &&
		strat.xSize < strat.xStepSize*strat.xMultiplier {

		strat.targetValue = math.Max(strat.xAbsValue, strat.yAbsValue) + strat.enterStep
		if strat.targetValue > strat.enterTarget {
			strat.targetValue = strat.enterTarget
		}
		strat.enterValue = strat.targetValue - math.Max(strat.xAbsValue, strat.yAbsValue)
		if strat.enterValue > strat.maxOrderValue {
			strat.enterValue = strat.maxOrderValue
		}
		strat.size = strat.enterValue / strat.midPrice
		strat.size = math.Min(strat.xTicker.GetBidSize()*strat.xMultiplier*strat.config.BestSizeFactor, strat.size)
		strat.size = math.Min(strat.yTicker.GetAskSize()*strat.yMultiplier*strat.config.BestSizeFactor, strat.size)

		strat.size = math.Round(strat.size/strat.xyMergedSpotStepSize) * strat.xyMergedSpotStepSize
		strat.enterValue = strat.size * strat.midPrice
		if strat.enterValue > strat.usdAvailable {
			if time.Now().Sub(strat.logSilentTime) > strat.config.LogInterval {
				strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
				logger.Debugf(
					"%s %s FAILED LONG BOT OPEN, ENTRY VALUE %f MORE THAN usdAvailable %f, %f < %f, %f < %f, SIZE %f",
					strat.xSymbol,
					strat.ySymbol,
					strat.enterValue,
					strat.usdAvailable,
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
		strat.price = strat.xTicker.GetBidPrice()
		strat.xNewOrderParam = common.NewOrderParam{
			Symbol:      strat.xSymbol,
			Side:        common.OrderSideSell,
			Type:        common.OrderTypeLimit,
			Price:       strat.price,
			TimeInForce: common.OrderTimeInForceFOK,
			Size:        strat.size,
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
		strat.hedgeCheckTimer.Reset(strat.config.HedgeDelay)
		strat.hedgeCheckStopTime = time.Now().Add(strat.config.HedgeCheckDuration)
		logger.Debugf(
			"%s %s LONG BOT OPEN %f < %f, %f < %f, PRICE %f SIZE %f, XTickerDiff %v YTickerDiff %v X %f %f Y %f %f",
			strat.xSymbol, strat.ySymbol,
			strat.spread.LongLastEnter, strat.longBot,
			strat.spread.LongMedianEnter, strat.longBot,
			strat.price,
			strat.size,
			time.Now().Sub(strat.xTickerTime),
			time.Now().Sub(strat.yTickerTime),
			strat.xTicker.GetBidPrice(),
			strat.xTicker.GetAskPrice(),
			strat.yTicker.GetBidPrice(),
			strat.yTicker.GetAskPrice(),
		)

	}
}
