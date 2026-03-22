package main

import (
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"time"
)

func (strat *XYStrategy) changeYPosition() {
	//logger.Debugf("changeYPosition %s", strat.ySymbol)
	if strat.xSystemStatus != common.SystemStatusReady ||
		strat.ySystemStatus != common.SystemStatusReady {
		if time.Now().Sub(strat.logSilentTime) > 0 {
			strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
			logger.Debugf("changeYPosition failed xSystemStatus %v ySystemStatus %v", strat.xSystemStatus, strat.ySystemStatus)
		}
		return
	}
	if strat.yPosition == nil ||
		strat.xPosition == nil ||
		time.Now().Sub(strat.yPositionUpdateTime) > strat.config.BalancePositionMaxAge ||
		time.Now().Sub(strat.xPositionUpdateTime) > strat.config.BalancePositionMaxAge ||
		time.Now().Sub(strat.yOrderSilentTime) < 0 {
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
		select {
		case strat.yOrderRequestCh <- common.OrderRequest{
			New: &strat.yNewOrderParam,
		}:
			strat.yOrderSilentTime = time.Now().Add(strat.config.OrderSilent)
			strat.yPositionUpdateTime = time.Unix(0, 0)
		}
	} else {
		strat.yOrderSilentTime = time.Now().Add(strat.config.OrderSilent)
		strat.yPositionUpdateTime = time.Unix(0, 0)
	}
	return
}

func (strat *XYStrategy) updateEnterStepAndTarget() {
	if strat.xAccount == nil || strat.yAccount == nil {
		return
	}
	strat.enterStep = (strat.xAccount.GetFree() + strat.yAccount.GetFree()) * strat.config.EnterFreePct * strat.targetWeight
	if strat.enterStep < strat.config.EnterMinimalStep {
		strat.enterStep = strat.config.EnterMinimalStep
	}
	strat.enterTarget = strat.enterStep * strat.config.EnterTargetFactor * strat.targetWeight
	strat.usdtAvailable = math.Min(strat.xAccount.GetFree()*strat.config.XExchange.Leverage, strat.yAccount.GetFree()*strat.config.YExchange.Leverage)
}

func (strat *XYStrategy) changeXPosition() {
	if strat.xSystemStatus != common.SystemStatusReady ||
		strat.ySystemStatus != common.SystemStatusReady {
		if time.Now().Sub(strat.logSilentTime) > 0 {
			strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
			logger.Debugf("changeXPosition failed xSystemStatus %v ySystemStatus %v", strat.xSystemStatus, strat.ySystemStatus)
		}
		return
	}
	if time.Now().Sub(strat.xyEnterSilentTime) < 0 ||
		time.Now().Sub(strat.xOrderSilentTime) < 0 ||
		time.Now().Sub(strat.xPositionUpdateTime) > strat.config.BalancePositionMaxAge ||
		time.Now().Sub(strat.yPositionUpdateTime) > strat.config.BalancePositionMaxAge ||
		strat.xAccount == nil ||
		strat.yAccount == nil ||
		strat.xPosition == nil ||
		strat.yPosition == nil ||
		strat.spread == nil ||
		strat.xyFundingRate == nil ||
		strat.fundingRateSettleSilent ||
		time.Now().Sub(strat.spread.EventTime) > strat.config.SpreadTimeToLive ||
		strat.config.EnterDepthMatchRatio > strat.xyDepthMatchRatio {
		if strat.config.EnterDepthMatchRatio > strat.xyDepthMatchRatio &&
			time.Now().Sub(strat.xOrderSilentTime) > 0 {
			strat.xOrderSilentTime = time.Now().Add(strat.config.EnterSilent)
			logger.Debugf("%s match ratio %f < %f, silent %v", strat.xSymbol, strat.xyDepthMatchRatio, strat.config.EnterDepthMatchRatio, strat.config.EnterSilent)
		}
		return
	}

	strat.xSize = strat.xPosition.GetSize() * strat.xMultiplier
	strat.ySize = strat.yPosition.GetSize() * strat.yMultiplier
	strat.xValue = strat.xSize * strat.xWalkedDepth.MidPrice
	strat.yValue = strat.ySize * strat.yWalkedDepth.MidPrice
	if math.Abs(strat.xValue+strat.yValue) > strat.enterStep*0.8 {
		if time.Now().Sub(strat.logSilentTime) > 0 {
			strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
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

	strat.shortTop = strat.config.ShortEnterDelta + strat.config.EnterOffsetDelta*strat.offsetFactor - *strat.xyFundingRate*strat.config.FrOffsetFactor
	strat.shortBot = strat.config.ShortExitDelta + strat.config.ExitOffsetDelta*(strat.offsetFactor-strat.offsetStep) - *strat.xyFundingRate*strat.config.FrOffsetFactor
	strat.longBot = strat.config.LongEnterDelta - strat.config.EnterOffsetDelta*strat.offsetFactor - *strat.xyFundingRate*strat.config.FrOffsetFactor
	strat.longTop = strat.config.LongExitDelta - strat.config.ExitOffsetDelta*(strat.offsetFactor-strat.offsetStep) - *strat.xyFundingRate*strat.config.FrOffsetFactor

	strat.midPrice = (strat.xWalkedDepth.MidPrice + strat.yWalkedDepth.MidPrice) * 0.5

	if strat.spread.ShortLastLeave < strat.shortBot &&
		strat.spread.ShortMedianLeave < strat.shortBot &&
		strat.spread.ShortLastLeave < strat.spread.ShortMedianLeave &&
		*strat.xyFundingRate < strat.config.MinimalKeepFundingRate &&
		strat.xSize >= strat.xStepSize*strat.xMultiplier {
		strat.enterValue = math.Min(2*strat.enterStep, math.Min(strat.xAbsValue, strat.yAbsValue))
		if *strat.xyFundingRate > strat.config.MinimalKeepFundingRate*0.5 {
			strat.enterValue = math.Min(strat.enterStep, math.Min(strat.xAbsValue, strat.yAbsValue))
		}
		strat.size = strat.enterValue / strat.midPrice
		strat.size = math.Round(strat.size/strat.xyMergedSpotStepSize) * strat.xyMergedSpotStepSize
		strat.enterValue = strat.size * strat.midPrice
		if strat.xAbsValue-strat.enterValue < strat.xyMergedSpotStepSize || strat.yAbsValue-strat.enterValue < strat.xyMergedSpotStepSize {
			//两种情况都把x全平，间接y全平
			strat.size = strat.xSize
		}
		strat.xSizeDiff = strat.size / strat.xMultiplier
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
			if math.Abs(strat.xSizeDiff) < strat.xStepSize {
				return
			} else if strat.xSizeDiff < 0 && strat.xPosition.GetSize() <= 0 && -strat.xSizeDiff*strat.xMultiplier*strat.xWalkedDepth.MidPrice < strat.xMinNotional {
				return
			} else if strat.xSizeDiff > 0 && strat.xPosition.GetSize() >= 0 && strat.xSizeDiff*strat.xMultiplier*strat.xWalkedDepth.MidPrice < strat.xMinNotional {
				return
			}
		}
		strat.price = math.Floor((strat.xWalkedDepth.BidPrice+strat.orderOffset.Top*strat.config.FokOffsetFactor)/strat.xTickSize) * strat.xTickSize
		strat.xNewOrderParam = common.NewOrderParam{
			Symbol:      strat.xSymbol,
			Side:        common.OrderSideSell,
			Type:        common.OrderTypeLimit,
			TimeInForce: common.OrderTimeInForceFOK,
			Size:        strat.xSizeDiff,
			Price:       strat.price,
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
		strat.xyEnterSilentTime = time.Now().Add(strat.config.EnterSilent)
		strat.xOrderSilentTime = time.Now().Add(strat.config.OrderSilent)
		strat.yOrderSilentTime = time.Now()
		strat.xPositionUpdateTime = time.Unix(0, 0)
		strat.xLastFilledBuyPrice = nil
		strat.xLastFilledSellPrice = nil
		strat.yLastFilledBuyPrice = nil
		strat.yLastFilledSellPrice = nil
		strat.hedgeYTimer.Reset(strat.config.HedgeYDelay)
		strat.hedgeCounter = strat.config.HedgeCheckCount
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
		*strat.xyFundingRate > -strat.config.MinimalKeepFundingRate &&
		strat.xSize <= -strat.xStepSize*strat.xMultiplier {
		strat.enterValue = math.Min(2*strat.enterStep, math.Min(strat.xAbsValue, strat.yAbsValue))
		if *strat.xyFundingRate < -strat.config.MinimalKeepFundingRate*0.5 {
			strat.enterValue = math.Min(strat.enterStep, math.Min(strat.xAbsValue, strat.yAbsValue))
		}
		strat.size = strat.enterValue / strat.midPrice
		strat.size = math.Round(strat.size/strat.xyMergedSpotStepSize) * strat.xyMergedSpotStepSize
		strat.enterValue = strat.size * strat.midPrice
		if strat.xAbsValue-strat.enterValue < strat.xyMergedSpotStepSize || strat.yAbsValue-strat.enterValue < strat.xyMergedSpotStepSize {
			strat.size = -strat.xSize
		}
		strat.xSizeDiff = strat.size / strat.xMultiplier
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
			if math.Abs(strat.xSizeDiff) < strat.xStepSize {
				return
			} else if strat.xSizeDiff < 0 && strat.xPosition.GetSize() <= 0 && -strat.xSizeDiff*strat.xMultiplier*strat.xWalkedDepth.MidPrice < strat.xMinNotional {
				return
			} else if strat.xSizeDiff > 0 && strat.xPosition.GetSize() >= 0 && strat.xSizeDiff*strat.xMultiplier*strat.xWalkedDepth.MidPrice < strat.xMinNotional {
				return
			}
		}
		strat.price = math.Ceil((strat.xWalkedDepth.AskPrice+strat.orderOffset.Bot*strat.config.FokOffsetFactor)/strat.xTickSize) * strat.xTickSize
		strat.xNewOrderParam = common.NewOrderParam{
			Symbol:      strat.xSymbol,
			Side:        common.OrderSideBuy,
			Type:        common.OrderTypeLimit,
			TimeInForce: common.OrderTimeInForceFOK,
			Size:        strat.xSizeDiff,
			Price:       strat.price,
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
		strat.xyEnterSilentTime = time.Now().Add(strat.config.EnterSilent)
		strat.xOrderSilentTime = time.Now().Add(strat.config.OrderSilent)
		strat.yOrderSilentTime = time.Now()
		strat.xLastFilledBuyPrice = nil
		strat.xLastFilledSellPrice = nil
		strat.yLastFilledBuyPrice = nil
		strat.yLastFilledSellPrice = nil
		strat.hedgeYTimer.Reset(strat.config.HedgeYDelay)
		strat.hedgeCounter = strat.config.HedgeCheckCount
		logger.Debugf(
			"%s %s LONG TOP REDUCE %f > %f, %f > %f, SIZE -%f PRICE %f, X %v Y %v S %v M %f",
			strat.xSymbol, strat.ySymbol,
			strat.spread.LongLastLeave, strat.longTop,
			strat.spread.LongMedianLeave, strat.longTop,
			strat.size,
			strat.price,
			time.Now().Sub(strat.xDepthTime),
			time.Now().Sub(strat.yDepthTime),
			time.Now().Sub(strat.spread.EventTime),
			strat.xyDepthMatchRatio,
		)
	} else if !strat.config.ReduceOnly &&
		!strat.isYSpot &&
		strat.spread.ShortLastEnter > strat.shortTop &&
		strat.spread.ShortMedianEnter > strat.shortTop &&
		strat.spread.ShortLastEnter > strat.spread.ShortMedianEnter &&
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
		if strat.enterValue < strat.yMinNotional || strat.enterValue < strat.xMinNotional {
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
		strat.xSizeDiff = strat.size / strat.xMultiplier
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
			if math.Abs(strat.xSizeDiff) < strat.xStepSize {
				return
			} else if strat.xSizeDiff < 0 && strat.xPosition.GetSize() <= 0 && -strat.xSizeDiff*strat.xMultiplier*strat.xWalkedDepth.MidPrice < strat.xMinNotional {
				return
			} else if strat.xSizeDiff > 0 && strat.xPosition.GetSize() >= 0 && strat.xSizeDiff*strat.xMultiplier*strat.xWalkedDepth.MidPrice < strat.xMinNotional {
				return
			}
		}
		strat.price = math.Ceil((strat.xWalkedDepth.AskPrice+strat.orderOffset.Bot*strat.config.FokOffsetFactor)/strat.xTickSize) * strat.xTickSize
		strat.xNewOrderParam = common.NewOrderParam{
			Symbol:      strat.xSymbol,
			Side:        common.OrderSideBuy,
			Type:        common.OrderTypeLimit,
			TimeInForce: common.OrderTimeInForceFOK,
			Size:        strat.xSizeDiff,
			Price:       strat.price,
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
		strat.xyEnterSilentTime = time.Now().Add(strat.config.EnterSilent)
		strat.xOrderSilentTime = time.Now().Add(strat.config.OrderSilent)
		strat.yOrderSilentTime = time.Now()
		strat.xLastFilledBuyPrice = nil
		strat.xLastFilledSellPrice = nil
		strat.yLastFilledBuyPrice = nil
		strat.yLastFilledSellPrice = nil
		strat.hedgeYTimer.Reset(strat.config.HedgeYDelay)
		strat.hedgeCounter = strat.config.HedgeCheckCount
		logger.Debugf(
			"%s %s SHORT TOP OPEN %f > %f, %f > %f, SIZE %f, PRICE %f, X %v Y %v S %v M %f",
			strat.xSymbol, strat.ySymbol,
			strat.spread.ShortLastEnter, strat.shortTop,
			strat.spread.ShortMedianEnter, strat.shortTop,
			strat.size,
			strat.price,
			time.Now().Sub(strat.xDepthTime),
			time.Now().Sub(strat.yDepthTime),
			time.Now().Sub(strat.spread.EventTime),
			strat.xyDepthMatchRatio,
		)
	} else if !strat.config.ReduceOnly &&
		!strat.isXSpot &&
		strat.spread.LongLastEnter < strat.longBot &&
		strat.spread.LongMedianEnter < strat.longBot &&
		strat.spread.LongLastEnter < strat.spread.LongMedianEnter &&
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
		if strat.enterValue < strat.yMinNotional || strat.enterValue < strat.xMinNotional {
			if time.Now().Sub(strat.logSilentTime) > 0 {
				strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
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
		strat.xSizeDiff = strat.size / strat.xMultiplier
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
			if math.Abs(strat.xSizeDiff) < strat.xStepSize {
				return
			} else if strat.xSizeDiff < 0 && strat.xPosition.GetSize() <= 0 && -strat.xSizeDiff*strat.xMultiplier*strat.xWalkedDepth.MidPrice < strat.xMinNotional {
				return
			} else if strat.xSizeDiff > 0 && strat.xPosition.GetSize() >= 0 && strat.xSizeDiff*strat.xMultiplier*strat.xWalkedDepth.MidPrice < strat.xMinNotional {
				return
			}
		}
		strat.price = math.Floor((strat.xWalkedDepth.BidPrice+strat.orderOffset.Top*strat.config.FokOffsetFactor)/strat.xTickSize) * strat.xTickSize
		strat.xNewOrderParam = common.NewOrderParam{
			Symbol:      strat.xSymbol,
			Side:        common.OrderSideSell,
			Type:        common.OrderTypeLimit,
			TimeInForce: common.OrderTimeInForceFOK,
			Size:        strat.xSizeDiff,
			Price:       strat.price,
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
		strat.xyEnterSilentTime = time.Now().Add(strat.config.EnterSilent)
		strat.xOrderSilentTime = time.Now().Add(strat.config.OrderSilent)
		strat.yOrderSilentTime = time.Now()
		strat.xLastFilledBuyPrice = nil
		strat.xLastFilledSellPrice = nil
		strat.yLastFilledBuyPrice = nil
		strat.yLastFilledSellPrice = nil
		strat.hedgeYTimer.Reset(strat.config.HedgeYDelay)
		strat.hedgeCounter = strat.config.HedgeCheckCount
		logger.Debugf(
			"%s %s LONG BOT OPEN %f < %f, %f < %f, SIZE -%f, PRICE %f, X %v Y %v S %v M %f",
			strat.xSymbol, strat.ySymbol,
			strat.spread.LongLastEnter, strat.longBot,
			strat.spread.LongMedianEnter, strat.longBot,
			strat.size,
			strat.price,
			time.Now().Sub(strat.xDepthTime),
			time.Now().Sub(strat.yDepthTime),
			time.Now().Sub(strat.spread.EventTime),
			strat.xyDepthMatchRatio,
		)
	}
}
