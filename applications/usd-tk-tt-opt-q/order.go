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
			logger.Debugf("%s updateXPosition xSystemStatus %v ySystemStatus %v", strat.xSymbol, strat.xSystemStatus, strat.ySystemStatus)
		}
		return
	}

	if strat.xPosition == nil ||
		strat.yPosition == nil ||
		strat.spread == nil ||
		strat.quantileMiddle == nil ||
		strat.xyFundingRate == nil {
		//if time.Now().Sub(strat.logSilentTime) > 0 {
		//	strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
		//	logger.Debugf("time.Now().Sub(strat.spread.EventTime) %v", time.Now().Sub(strat.spread.EventTime))
		//}
		return
	}

	strat.xSize = strat.xPosition.GetSize() * strat.xMultiplier
	strat.ySize = strat.yPosition.GetSize() * strat.yMultiplier
	if strat.isXSpot || strat.xPosition.GetPrice() == 0 {
		strat.xValue = strat.xSize * strat.xMidPrice
	} else {
		strat.xValue = strat.xSize * strat.xPosition.GetPrice()
	}
	if strat.isYSpot || strat.yPosition.GetPrice() == 0 {
		strat.yValue = strat.ySize * strat.yMidPrice
	} else {
		strat.yValue = strat.ySize * strat.yPosition.GetPrice()
	}
	strat.xAbsValue = math.Abs(strat.xValue)
	strat.yAbsValue = math.Abs(strat.yValue)

	strat.shortTop = *strat.quantileMiddle + strat.enterOffset - *strat.xyFundingRate*strat.config.FundingRateOffsetFactor
	strat.shortBot = *strat.quantileMiddle - strat.leaveOffset - *strat.xyFundingRate*strat.config.FundingRateOffsetFactor
	strat.longBot = *strat.quantileMiddle - strat.enterOffset - *strat.xyFundingRate*strat.config.FundingRateOffsetFactor
	strat.longTop = *strat.quantileMiddle + strat.leaveOffset - *strat.xyFundingRate*strat.config.FundingRateOffsetFactor

	strat.midPrice = (strat.xMidPrice + strat.yMidPrice) * 0.5

	if time.Now().Sub(strat.xPositionUpdateTime) > strat.config.BalancePositionMaxAge ||
		time.Now().Sub(strat.yPositionUpdateTime) > strat.config.BalancePositionMaxAge ||
		strat.xAccount == nil ||
		strat.yAccount == nil ||
		strat.fundingRateSettleSilent ||
		time.Now().Sub(strat.spread.EventTime) > strat.config.SpreadTimeToEnter {
		//if time.Now().Sub(strat.logSilentTime) > 0 {
		//	strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
		//	logger.Debugf("time.Now().Sub(strat.spread.EventTime) %v", time.Now().Sub(strat.spread.EventTime))
		//}
		return
	}

	if time.Now().Sub(strat.xOrderSilentTime) < 0 {
		return
	}

	if math.Abs(strat.xSize+strat.ySize)*strat.midPrice > strat.enterStep*0.8 {
		if time.Now().Sub(strat.logSilentTime) > 0 {
			strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
			logger.Debugf(
				"%s %s unhedged value %f > 0.8*enterStep %f",
				strat.xSymbol, strat.ySymbol, math.Abs(strat.xValue+strat.yValue), strat.enterStep*0.8,
			)
		}
		strat.hedgeXPosition()
		if time.Now().Sub(strat.hedgeCheckStopTime) > 0 {
			strat.hedgeYPosition()
		}
		return
	}

	if strat.spread.ShortMedianLeave < strat.shortBot &&
		strat.spread.ShortLastLeave < strat.spread.ShortMedianLeave &&
		strat.xSize >= strat.xStepSize*strat.xMultiplier {

		strat.enterValue = math.Min(math.Max(4*strat.enterStep, strat.xAbsValue*0.5), math.Min(strat.xAbsValue, strat.yAbsValue))
		if strat.enterValue > strat.maxOrderValue {
			strat.enterValue = strat.maxOrderValue
		}
		strat.xSizeDiff = strat.enterValue / strat.midPrice

		//限开仓大小限制到best bid ask size
		strat.xSizeDiff = math.Min(strat.xTicker.GetBidSize()*strat.xMultiplier*strat.config.BestSizeFactor, strat.xSizeDiff)
		strat.xSizeDiff = math.Min(strat.yTicker.GetAskSize()*strat.yMultiplier*strat.config.BestSizeFactor, strat.xSizeDiff)
		strat.xSizeDiff = math.Round(strat.xSizeDiff/strat.xyMergedSpotStepSize) * strat.xyMergedSpotStepSize

		strat.enterValue = strat.xSizeDiff * strat.midPrice
		if strat.xAbsValue-strat.enterValue < strat.xyMergedSpotStepSize*1.005 ||
			strat.yAbsValue-strat.enterValue < strat.xyMergedSpotStepSize*1.005 ||
			strat.xSizeDiff > strat.xSize {
			//两种情况都把x全平，间接y全平
			strat.xSizeDiff = strat.xSize
		}
		strat.enterValue = strat.xSizeDiff * strat.midPrice
		strat.xSizeDiff = math.Floor(strat.xSizeDiff/strat.xMultiplier/strat.xStepSize) * strat.xStepSize
		if strat.xSizeDiff > 0 && (!strat.isXSpot || (strat.enterValue >= 1.2*strat.xMinNotional && strat.xSizeDiff >= strat.xMinSize)) {

			strat.xPrice = strat.xTicker.GetBidPrice()
			//防止TickSize太大
			if strat.xTickSize/strat.xPrice < strat.config.EnterSlippage {
				strat.xPrice = strat.xPrice * (1.0 - strat.config.EnterSlippage)
				strat.xPrice = math.Floor(strat.xPrice/strat.xTickSize) * strat.xTickSize
			}

			strat.xNewOrderParam = common.NewOrderParam{
				Symbol:      strat.xSymbol,
				Side:        common.OrderSideSell,
				Type:        common.OrderTypeLimit,
				Price:       strat.xPrice,
				TimeInForce: strat.config.XOrderTimeInForce,
				Size:        strat.xSizeDiff,
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
			strat.lastEnterTime = strat.spread.EventTime.Add(strat.config.XOrderSilent)
			logger.Debugf(
				"%s %s SHORT BOT REDUCE %f < %f, %f < %f, PRICE %f SIZE %f, XTickerDiff %v YTickerDiff %v X %f %f Y %f %f, Offset %f %f",
				strat.xSymbol, strat.ySymbol,
				strat.spread.ShortLastLeave, strat.shortBot,
				strat.spread.ShortMedianLeave, strat.shortBot,
				strat.xPrice,
				strat.xSizeDiff,
				time.Now().Sub(strat.xTickerTime),
				time.Now().Sub(strat.yTickerTime),
				strat.xTicker.GetBidPrice(),
				strat.xTicker.GetAskPrice(),
				strat.yTicker.GetBidPrice(),
				strat.yTicker.GetAskPrice(),
				strat.enterOffset,
				strat.leaveOffset,
			)
		}
	} else if strat.spread.LongMedianLeave > strat.longTop &&
		strat.spread.LongLastLeave > strat.spread.LongMedianLeave &&
		strat.xSize <= -strat.xStepSize*strat.xMultiplier {

		strat.enterValue = math.Min(math.Max(4*strat.enterStep, strat.xAbsValue*0.5), math.Min(strat.xAbsValue, strat.yAbsValue))
		if strat.enterValue > strat.maxOrderValue {
			strat.enterValue = strat.maxOrderValue
		}
		strat.xSizeDiff = strat.enterValue / strat.midPrice
		//限开仓大小限制到best bid ask size
		strat.xSizeDiff = math.Min(strat.xTicker.GetAskSize()*strat.xMultiplier*strat.config.BestSizeFactor, strat.xSizeDiff)
		strat.xSizeDiff = math.Min(strat.yTicker.GetBidSize()*strat.yMultiplier*strat.config.BestSizeFactor, strat.xSizeDiff)

		strat.xSizeDiff = math.Round(strat.xSizeDiff/strat.xyMergedSpotStepSize) * strat.xyMergedSpotStepSize
		strat.enterValue = strat.xSizeDiff * strat.midPrice
		if strat.xAbsValue-strat.enterValue < strat.xyMergedSpotStepSize*1.005 ||
			strat.yAbsValue-strat.enterValue < strat.xyMergedSpotStepSize*1.005 ||
			strat.xSizeDiff > -strat.xSize {
			strat.xSizeDiff = -strat.xSize
		}
		strat.enterValue = strat.xSizeDiff * strat.midPrice
		strat.xSizeDiff = math.Floor(strat.xSizeDiff/strat.xMultiplier/strat.xStepSize) * strat.xStepSize
		if strat.xSizeDiff > 0 && (!strat.isXSpot || (strat.enterValue >= 1.2*strat.xMinNotional && strat.xSizeDiff >= strat.xMinSize)) {
			strat.xPrice = strat.xTicker.GetAskPrice()
			if strat.xTickSize/strat.xPrice < strat.config.EnterSlippage {
				strat.xPrice = strat.xPrice * (1.0 + strat.config.EnterSlippage)
				strat.xPrice = math.Ceil(strat.xPrice/strat.xTickSize) * strat.xTickSize
			}
			strat.xNewOrderParam = common.NewOrderParam{
				Symbol:      strat.xSymbol,
				Side:        common.OrderSideBuy,
				Type:        common.OrderTypeLimit,
				Price:       strat.xPrice,
				TimeInForce: strat.config.XOrderTimeInForce,
				Size:        strat.xSizeDiff,
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
			strat.lastEnterTime = strat.spread.EventTime.Add(strat.config.XOrderSilent)
			logger.Debugf(
				"%s %s LONG TOP REDUCE %f > %f, %f > %f, PRICE %f SIZE %f, XTickerDiff %v YTickerDiff %v X %f %f Y %f %f, Offsets %f %f",
				strat.xSymbol, strat.ySymbol,
				strat.spread.LongLastLeave, strat.longTop,
				strat.spread.LongMedianLeave, strat.longTop,
				strat.xPrice,
				strat.xSizeDiff,
				time.Now().Sub(strat.xTickerTime),
				time.Now().Sub(strat.yTickerTime),
				strat.xTicker.GetBidPrice(),
				strat.xTicker.GetAskPrice(),
				strat.yTicker.GetBidPrice(),
				strat.yTicker.GetAskPrice(),
				strat.enterOffset,
				strat.leaveOffset,
			)
		}
	} else if !strat.config.ReduceOnly &&
		!strat.isYSpot &&
		strat.spread.ShortMedianEnter > strat.shortTop &&
		strat.spread.ShortLastEnter > strat.spread.ShortMedianEnter &&
		*strat.xyFundingRate > strat.config.MinimalEnterFundingRate &&
		strat.xSize > -strat.xStepSize*strat.xMultiplier &&
		strat.xAccount.GetFree() > strat.config.MinimalXFree &&
		strat.yAccount.GetFree() > strat.config.MinimalYFree &&
		strat.xAbsValue < strat.config.MaximalXPosValue &&
		strat.yAbsValue < strat.config.MaximalYPosValue {

		strat.targetValue = math.Max(strat.xAbsValue, strat.yAbsValue) + strat.enterStep
		strat.enterValue = math.Min(
			strat.targetValue,
			math.Min(strat.config.MaximalXPosValue, strat.config.MaximalYPosValue),
		) - math.Max(strat.xAbsValue, strat.yAbsValue)
		if strat.enterValue < 0 {
			//超过最大仓位了, 不操作
			strat.hedgeXPosition()
			return
		}
		if strat.enterValue > strat.maxOrderValue {
			strat.enterValue = strat.maxOrderValue
		}
		strat.xSizeDiff = strat.enterValue / strat.midPrice
		strat.xSizeDiff = math.Min(strat.xTicker.GetAskSize()*strat.xMultiplier*strat.config.BestSizeFactor, strat.xSizeDiff)
		strat.xSizeDiff = math.Min(strat.yTicker.GetBidSize()*strat.yMultiplier*strat.config.BestSizeFactor, strat.xSizeDiff)

		strat.xSizeDiff = math.Round(strat.xSizeDiff/strat.xyMergedSpotStepSize) * strat.xyMergedSpotStepSize
		strat.enterValue = strat.xSizeDiff * strat.midPrice
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
					strat.xSizeDiff,
				)
			}
			strat.hedgeXPosition()
			return
		}

		strat.xSizeDiff = math.Floor(strat.xSizeDiff/strat.xMultiplier/strat.xStepSize) * strat.xStepSize
		if strat.xSizeDiff <= 0 || strat.enterValue < 1.2*strat.yMinNotional || strat.enterValue < 1.2*strat.xMinNotional || strat.xSizeDiff < strat.xMinSize {
			if time.Now().Sub(strat.logSilentTime) > 0 {
				strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
				logger.Debugf(
					"%s %s FAILED SHORT TOP OPEN, ORDER VALUE %f TOO SMALL, %f > %f, %f > %f, SIZE %f",
					strat.xSymbol, strat.ySymbol,
					strat.enterValue,
					strat.spread.ShortLastEnter, strat.shortTop,
					strat.spread.ShortMedianEnter, strat.shortTop,
					strat.xSizeDiff,
				)
			}
			strat.hedgeXPosition()
			return
		}
		strat.xPrice = strat.xTicker.GetAskPrice()
		if strat.xTickSize/strat.xPrice < strat.config.EnterSlippage {
			strat.xPrice = strat.xPrice * (1.0 + strat.config.EnterSlippage)
			strat.xPrice = math.Ceil(strat.xPrice/strat.xTickSize) * strat.xTickSize
		}
		strat.xNewOrderParam = common.NewOrderParam{
			Symbol:      strat.xSymbol,
			Side:        common.OrderSideBuy,
			Type:        common.OrderTypeLimit,
			Price:       strat.xPrice,
			TimeInForce: strat.config.XOrderTimeInForce,
			Size:        strat.xSizeDiff,
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
		strat.lastEnterTime = strat.spread.EventTime.Add(strat.config.XOrderSilent)
		logger.Debugf(
			"%s %s SHORT TOP OPEN %f > %f, %f > %f, PRICE %f SIZE %f, XTickerDiff %v YTickerDiff %v X %f %f Y %f %f, Offsets %f %f",
			strat.xSymbol, strat.ySymbol,
			strat.spread.ShortLastEnter, strat.shortTop,
			strat.spread.ShortMedianEnter, strat.shortTop,
			strat.xPrice,
			strat.xSizeDiff,
			time.Now().Sub(strat.xTickerTime),
			time.Now().Sub(strat.yTickerTime),
			strat.xTicker.GetBidPrice(),
			strat.xTicker.GetAskPrice(),
			strat.yTicker.GetBidPrice(),
			strat.yTicker.GetAskPrice(),
			strat.enterOffset,
			-strat.leaveOffset,
		)
	} else if !strat.config.ReduceOnly &&
		!strat.isXSpot &&
		strat.spread.LongMedianEnter < strat.longBot &&
		strat.spread.LongLastEnter < strat.spread.LongMedianEnter &&
		*strat.xyFundingRate < -strat.config.MinimalEnterFundingRate &&
		strat.xSize < strat.xStepSize*strat.xMultiplier &&
		strat.xAccount.GetFree() > strat.config.MinimalXFree &&
		strat.yAccount.GetFree() > strat.config.MinimalYFree &&
		strat.xAbsValue < strat.config.MaximalXPosValue &&
		strat.yAbsValue < strat.config.MaximalYPosValue {

		strat.targetValue = math.Max(strat.xAbsValue, strat.yAbsValue) + strat.enterStep
		strat.enterValue = math.Min(
			strat.targetValue,
			math.Min(strat.config.MaximalXPosValue, strat.config.MaximalYPosValue),
		) - math.Max(strat.xAbsValue, strat.yAbsValue)
		if strat.enterValue < 0 {
			//超过最大仓位了, 不操作
			strat.hedgeXPosition()
			return
		}
		if strat.enterValue > strat.maxOrderValue {
			strat.enterValue = strat.maxOrderValue
		}
		strat.xSizeDiff = strat.enterValue / strat.midPrice
		strat.xSizeDiff = math.Min(strat.xTicker.GetBidSize()*strat.xMultiplier*strat.config.BestSizeFactor, strat.xSizeDiff)
		strat.xSizeDiff = math.Min(strat.yTicker.GetAskSize()*strat.yMultiplier*strat.config.BestSizeFactor, strat.xSizeDiff)

		strat.xSizeDiff = math.Round(strat.xSizeDiff/strat.xyMergedSpotStepSize) * strat.xyMergedSpotStepSize
		strat.enterValue = strat.xSizeDiff * strat.midPrice
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
					strat.xSizeDiff,
				)
			}
			strat.hedgeXPosition()
			return
		}
		strat.xSizeDiff = math.Floor(strat.xSizeDiff/strat.xMultiplier/strat.xStepSize) * strat.xStepSize
		if strat.xSizeDiff <= 0 || strat.enterValue < 1.2*strat.yMinNotional || strat.enterValue < 1.2*strat.xMinNotional || strat.xSizeDiff < strat.xMinSize {
			if time.Now().Sub(strat.logSilentTime) > 0 {
				strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
				logger.Debugf(
					"%s %s FAILED LONG BOT OPEN, ORDER VALUE %f TOO SMALL, %f < %f, %f < %f, SIZE %f",
					strat.xSymbol, strat.ySymbol,
					strat.enterValue,
					strat.spread.LongLastEnter, strat.longBot,
					strat.spread.LongMedianEnter, strat.longBot,
					strat.xSizeDiff,
				)
			}
			strat.hedgeXPosition()
			return
		}
		strat.xPrice = strat.xTicker.GetBidPrice()
		//防止TickSize太大
		if strat.xTickSize/strat.xPrice < strat.config.EnterSlippage {
			strat.xPrice = strat.xPrice * (1.0 - strat.config.EnterSlippage)
			strat.xPrice = math.Floor(strat.xPrice/strat.xTickSize) * strat.xTickSize
		}
		strat.xNewOrderParam = common.NewOrderParam{
			Symbol:      strat.xSymbol,
			Side:        common.OrderSideSell,
			Type:        common.OrderTypeLimit,
			Price:       strat.xPrice,
			TimeInForce: strat.config.XOrderTimeInForce,
			Size:        strat.xSizeDiff,
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
		strat.lastEnterTime = strat.spread.EventTime.Add(strat.config.XOrderSilent)
		logger.Debugf(
			"%s %s LONG BOT OPEN %f < %f, %f < %f, PRICE %f SIZE %f, XTickerDiff %v YTickerDiff %v X %f %f Y %f %f, Offsets %f %f",
			strat.xSymbol, strat.ySymbol,
			strat.spread.LongLastEnter, strat.longBot,
			strat.spread.LongMedianEnter, strat.longBot,
			strat.xPrice,
			strat.xSizeDiff,
			time.Now().Sub(strat.xTickerTime),
			time.Now().Sub(strat.yTickerTime),
			strat.xTicker.GetBidPrice(),
			strat.xTicker.GetAskPrice(),
			strat.yTicker.GetBidPrice(),
			strat.yTicker.GetAskPrice(),
			-strat.enterOffset,
			strat.leaveOffset,
		)

	}

}

func (strat *XYStrategy) hedgeXPosition() {
	//如果lastSpreadEnterTime没有更新，说明没有信号触发，就需要检查对冲的情况
	if time.Now().Sub(strat.lastEnterTime) > strat.config.XEnterTimeout {
		//如果已经没有信号对冲，重新检查x y的仓位，对冲较小的
		if math.Abs(strat.xPosition.GetSize()*strat.xMultiplier) < math.Abs(strat.yPosition.GetSize()*strat.yMultiplier) {
			//X的size比Y小，不用操作X
			return
		}
		strat.xSizeDiff = -strat.yPosition.GetSize()*strat.yMultiplier/strat.xMultiplier - strat.xPosition.GetSize()
		//如y下单也加上控制，以限下单太大，造成市场冲击
		if strat.xSizeDiff*strat.xMultiplier < -strat.maxOrderValue/strat.xTicker.GetBidPrice() {
			strat.xSizeDiff = -strat.maxOrderValue / strat.xTicker.GetBidPrice() / strat.xMultiplier
		} else if strat.xSizeDiff*strat.xMultiplier > strat.maxOrderValue/strat.xTicker.GetAskPrice() {
			strat.xSizeDiff = strat.maxOrderValue / strat.xTicker.GetAskPrice() / strat.xMultiplier
		}

		if strat.xSizeDiff >= 0 {
			strat.xSizeDiff = math.Floor(strat.xSizeDiff/strat.xStepSize) * strat.xStepSize
		} else {
			strat.xSizeDiff = math.Ceil(strat.xSizeDiff/strat.xStepSize) * strat.xStepSize
		}

		if strat.isXSpot {
			if math.Abs(strat.xSizeDiff) < strat.xStepSize {
				return
			} else if strat.xSizeDiff < 0 && -strat.xSizeDiff*strat.xMultiplier*strat.xTicker.GetBidPrice() < 1.2*strat.xMinNotional {
				return
			} else if strat.xSizeDiff > 0 && strat.xSizeDiff*strat.xMultiplier*strat.xTicker.GetAskPrice() < 1.2*strat.xMinNotional {
				return
			}
		} else {
			//期货以close仓位，没有minNotional限制
			if math.Abs(strat.xSizeDiff) < strat.xStepSize {
				return
			} else if strat.xSizeDiff < 0 && strat.xPosition.GetSize() <= 0 && -strat.xSizeDiff*strat.xMultiplier*strat.xTicker.GetBidPrice() < 1.2*strat.xMinNotional {
				return
			} else if strat.xSizeDiff > 0 && strat.xPosition.GetSize() >= 0 && strat.xSizeDiff*strat.xMultiplier*strat.xTicker.GetAskPrice() < 1.2*strat.xMinNotional {
				return
			}
		}

		if strat.xSizeDiff < 0 {
			strat.orderSide = common.OrderSideSell
			strat.xSizeDiff = -strat.xSizeDiff
			strat.xPrice = strat.xTicker.GetBidPrice()
			//防止TickSize太大
			if strat.xTickSize/strat.xPrice < strat.config.EnterSlippage {
				strat.xPrice = strat.xPrice * (1.0 - strat.config.EnterSlippage)
				strat.xPrice = math.Floor(strat.xPrice/strat.xTickSize) * strat.xTickSize
			}
		} else {
			strat.orderSide = common.OrderSideBuy
			strat.xPrice = strat.xTicker.GetAskPrice()
			//防止TickSize太大
			if strat.xTickSize/strat.xPrice < strat.config.EnterSlippage {
				strat.xPrice = strat.xPrice * (1.0 + strat.config.EnterSlippage)
				strat.xPrice = math.Ceil(strat.xPrice/strat.xTickSize) * strat.xTickSize
			}
		}
		strat.xNewOrderParam = common.NewOrderParam{
			Symbol:      strat.xSymbol,
			Side:        strat.orderSide,
			Type:        common.OrderTypeLimit,
			Price:       strat.xPrice,
			TimeInForce: strat.config.XOrderTimeInForce,
			Size:        strat.xSizeDiff,
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
			"%s %s REVERSE HEDGE X BY Y, SIZE X %f Y %f, ORDER SIDE %s SIZE %f PRICE %f",
			strat.xSymbol, strat.ySymbol,
			strat.xPosition.GetSize()*strat.xMultiplier,
			strat.yPosition.GetSize()*strat.yMultiplier,
			strat.orderSide,
			strat.xSizeDiff,
			strat.xPrice,
		)
	}
}
