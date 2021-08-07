package main

import (
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"time"
)

func (strat *XYStrategy) updateXOrder() {

	if strat.xSystemStatus != common.SystemStatusReady ||
		strat.ySystemStatus != common.SystemStatusReady {
		if time.Now().Sub(strat.logSilentTime) > 0 {
			strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
			logger.Debugf("updateXOrder %s xSystemStatus %v ySystemStatus %v", strat.xSymbol, strat.xSystemStatus, strat.ySystemStatus)
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
		time.Now().Sub(strat.spread.EventTime) > strat.config.SpreadTimeToEnter ||
		strat.fundingRateSettleSilent {
		if time.Now().Sub(strat.yTickerTime) > strat.config.YTickerTimeToCancel {
			strat.tryCancelXOpenOrder("ticker time out")
		} else if strat.fundingRateSettleSilent {
			strat.tryCancelXOpenOrder("funding rate silent")
		}
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

	if time.Now().Sub(strat.xCancelSilentTime) < 0 {
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
		goto hedgeSmall
	}

	if strat.xOpenOrder != nil {
		if !strat.isXOpenOrderOk() {
			strat.tryCancelXOpenOrder("open order not ok")
		}
		//只要有OpenOrder, 说明X还在挂单，这个时候不用X对Y
		strat.lastXActiveTime = time.Now()
		return
	}

	if strat.spread.ShortMedianLeave <= strat.shortBot &&
		strat.spread.ShortLastLeave <= strat.spread.ShortMedianLeave &&
		*strat.xyFundingRate < strat.config.MinimalKeepFundingRate &&
		strat.xSize >= strat.xStepSize*strat.xMultiplier {
		strat.enterValue = math.Min(math.Max(4*strat.enterStep, strat.xAbsValue*0.5), math.Min(strat.xAbsValue, strat.yAbsValue))
		if *strat.xyFundingRate > strat.config.MinimalKeepFundingRate*0.5 {
			strat.enterValue = math.Min(math.Max(2*strat.enterStep, strat.xAbsValue*0.25), math.Min(strat.xAbsValue, strat.yAbsValue))
		}
		if strat.enterValue > strat.maxOrderValue {
			strat.enterValue = strat.maxOrderValue
		}
		strat.xSizeDiff = strat.enterValue / strat.midPrice
		strat.xSizeDiff = math.Round(strat.xSizeDiff/strat.xyMergedSpotStepSize) * strat.xyMergedSpotStepSize
		strat.enterValue = strat.xSizeDiff * strat.midPrice
		if strat.xAbsValue-strat.enterValue < strat.xyMergedSpotStepSize*1.005 ||
			strat.yAbsValue-strat.enterValue < strat.xyMergedSpotStepSize*1.005 ||
			strat.xSizeDiff > strat.xSize {
			//两种情况都把x全平，间接y全平
			strat.xSizeDiff = strat.xSize
		}
		strat.xSizeDiff = math.Floor(strat.xSizeDiff/strat.xMultiplier/strat.xStepSize) * strat.xStepSize
		if strat.xSizeDiff > 0 && (!strat.isXSpot || strat.enterValue > 1.2*strat.xMinNotional) {
			strat.xPrice = math.Ceil(strat.xMidPrice*(1.0+strat.orderOffset.Top)/strat.xTickSize) * strat.xTickSize
			if strat.xPrice < strat.xTicker.GetBidPrice()+strat.xTickSize {
				strat.xPrice = strat.xTicker.GetBidPrice() + strat.xTickSize
			}
			strat.xNewOrderParam = common.NewOrderParam{
				Symbol:      strat.xSymbol,
				Side:        common.OrderSideSell,
				Type:        common.OrderTypeLimit,
				Price:       strat.xPrice,
				TimeInForce: common.OrderTimeInForceGTC,
				Size:        strat.xSizeDiff,
				PostOnly:    true,
				ReduceOnly:  true,
				ClientID:    strat.xExchange.GenerateClientID(),
			}
			strat.xOpenOrder = &strat.xNewOrderParam
			strat.xOpenOrderCheckTimer.Reset(strat.config.XOrderCheckInterval)
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
			strat.xOrderSilentTime = time.Now().Add(strat.config.XOrderSilent)
			strat.lastXActiveTime = strat.xOrderSilentTime
			logger.Debugf(
				"%s %s SHORT BOT REDUCE %f < %f, %f < %f QM %f, ES %f EV %f,SIZE %f PRICE %f, X %v Y %v",
				strat.config.Name,
				strat.xSymbol,
				strat.spread.ShortLastLeave, strat.shortBot,
				strat.spread.ShortMedianLeave, strat.shortBot,
				strat.enterStep,
				strat.enterValue,
				*strat.quantileMiddle,
				strat.xSizeDiff,
				strat.xPrice,
				time.Now().Sub(strat.xTickerTime),
				time.Now().Sub(strat.yTickerTime),
			)
		}
	} else if strat.spread.LongMedianLeave >= strat.longTop &&
		strat.spread.LongLastLeave >= strat.spread.LongMedianLeave &&
		*strat.xyFundingRate > -strat.config.MinimalKeepFundingRate &&
		strat.xSize <= -strat.xStepSize*strat.xMultiplier {

		strat.enterValue = math.Min(math.Max(4*strat.enterStep, strat.xAbsValue*0.5), math.Min(strat.xAbsValue, strat.yAbsValue))
		if *strat.xyFundingRate < -strat.config.MinimalKeepFundingRate*0.5 {
			strat.enterValue = math.Min(math.Max(2*strat.enterStep, strat.xAbsValue*0.25), math.Min(strat.xAbsValue, strat.yAbsValue))
		}
		if strat.enterValue > strat.maxOrderValue {
			strat.enterValue = strat.maxOrderValue
		}
		strat.xSizeDiff = strat.enterValue / strat.midPrice
		strat.xSizeDiff = math.Round(strat.xSizeDiff/strat.xyMergedSpotStepSize) * strat.xyMergedSpotStepSize
		strat.enterValue = strat.xSizeDiff * strat.midPrice
		if strat.xAbsValue-strat.enterValue < strat.xyMergedSpotStepSize*1.005 ||
			strat.yAbsValue-strat.enterValue < strat.xyMergedSpotStepSize*1.005 ||
			strat.xSizeDiff > -strat.xSize {
			strat.xSizeDiff = -strat.xSize
		}
		strat.xSizeDiff = math.Floor(strat.xSizeDiff/strat.xMultiplier/strat.xStepSize) * strat.xStepSize
		if strat.xSizeDiff > 0 && (!strat.isXSpot || strat.enterValue > 1.2*strat.xMinNotional) {
			strat.xPrice = math.Floor(strat.xMidPrice*(1.0+strat.orderOffset.Bot)/strat.xTickSize) * strat.xTickSize
			if strat.xPrice > strat.xTicker.GetAskPrice()-strat.xTickSize {
				strat.xPrice = strat.xTicker.GetAskPrice() - strat.xTickSize
			}
			strat.xNewOrderParam = common.NewOrderParam{
				Symbol:      strat.xSymbol,
				Side:        common.OrderSideBuy,
				Type:        common.OrderTypeLimit,
				Price:       strat.xPrice,
				TimeInForce: common.OrderTimeInForceGTC,
				Size:        strat.xSizeDiff,
				PostOnly:    true,
				ReduceOnly:  true,
				ClientID:    strat.xExchange.GenerateClientID(),
			}
			strat.xOpenOrder = &strat.xNewOrderParam
			strat.xOpenOrderCheckTimer.Reset(strat.config.XOrderCheckInterval)
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
			strat.xOrderSilentTime = time.Now().Add(strat.config.XOrderSilent)
			strat.lastXActiveTime = strat.xOrderSilentTime
			logger.Debugf(
				"%s %s LONG TOP REDUCE %f > %f, %f > %f, QM %f, SIZE %f PRICE %f, X %v Y %v",
				strat.config.Name,
				strat.xSymbol,
				strat.spread.LongLastLeave, strat.longTop,
				strat.spread.LongMedianLeave, strat.longTop,
				strat.enterStep,
				strat.enterValue,
				*strat.quantileMiddle,
				strat.xSizeDiff,
				strat.xPrice,
				time.Now().Sub(strat.xTickerTime),
				time.Now().Sub(strat.yTickerTime),
			)
		}
	} else if !strat.config.ReduceOnly &&
		!strat.isYSpot &&
		strat.spread.ShortMedianEnter >= strat.shortTop &&
		strat.spread.ShortLastEnter >= strat.spread.ShortMedianEnter &&
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
		strat.xSizeDiff = strat.enterValue / strat.midPrice
		strat.xSizeDiff = math.Round(strat.xSizeDiff/strat.xyMergedSpotStepSize) * strat.xyMergedSpotStepSize
		strat.enterValue = strat.xSizeDiff * strat.midPrice
		if strat.enterValue > strat.usdAvailable {
			strat.xOrderSilentTime = time.Now().Add(strat.config.XErrorSilent)
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
			goto hedgeSmall
		}
		strat.xSizeDiff = math.Floor(strat.xSizeDiff/strat.xMultiplier/strat.xStepSize) * strat.xStepSize
		if strat.xSizeDiff <= 0 || strat.enterValue < 1.2*strat.yMinNotional || strat.enterValue < 1.2*strat.xMinNotional {
			strat.xOrderSilentTime = time.Now().Add(strat.config.XErrorSilent)
			if time.Now().Sub(strat.logSilentTime) > 0 {
				strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
				logger.Debugf(
					"%s %s %s FAILED SHORT TOP OPEN, ORDER VALUE %f TOO SMALL, %f > %f, %f > %f, SIZE %f",
					strat.config.Name,
					strat.xSymbol, strat.ySymbol,
					strat.enterValue,
					strat.spread.ShortLastEnter, strat.shortTop,
					strat.spread.ShortMedianEnter, strat.shortTop,
					strat.xSizeDiff,
				)
			}
			goto hedgeSmall
		}
		strat.xPrice = math.Floor(strat.xMidPrice*(1.0+strat.orderOffset.Bot)/strat.xTickSize) * strat.xTickSize
		if strat.xPrice > strat.xTicker.GetAskPrice()-strat.xTickSize {
			strat.xPrice = strat.xTicker.GetAskPrice() - strat.xTickSize
		}
		strat.xNewOrderParam = common.NewOrderParam{
			Symbol:      strat.xSymbol,
			Side:        common.OrderSideBuy,
			Type:        common.OrderTypeLimit,
			Price:       strat.xPrice,
			TimeInForce: common.OrderTimeInForceGTC,
			Size:        strat.xSizeDiff,
			PostOnly:    true,
			ReduceOnly:  false,
			ClientID:    strat.xExchange.GenerateClientID(),
		}
		strat.xOpenOrder = &strat.xNewOrderParam
		strat.xOpenOrderCheckTimer.Reset(strat.config.XOrderCheckInterval)
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
		strat.xOrderSilentTime = time.Now().Add(strat.config.XOrderSilent)
		strat.lastXActiveTime = strat.xOrderSilentTime
		logger.Debugf(
			"%s %s SHORT TOP OPEN %f > %f, %f > %f,  ES %f EV %f, QM %f, SIZE %f PRICE %f, X %v Y %v",
			strat.config.Name,
			strat.xSymbol,
			strat.spread.ShortLastEnter, strat.shortTop,
			strat.spread.ShortMedianEnter, strat.shortTop,
			strat.enterStep,
			strat.enterValue,
			*strat.quantileMiddle,
			strat.xSizeDiff,
			strat.xPrice,
			time.Now().Sub(strat.xTickerTime),
			time.Now().Sub(strat.yTickerTime),
		)
	} else if !strat.config.ReduceOnly &&
		!strat.isXSpot &&
		strat.spread.LongMedianEnter <= strat.longBot &&
		strat.spread.LongLastEnter <= strat.spread.LongMedianEnter &&
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
		strat.xSizeDiff = strat.enterValue / strat.midPrice
		strat.xSizeDiff = math.Round(strat.xSizeDiff/strat.xyMergedSpotStepSize) * strat.xyMergedSpotStepSize
		strat.enterValue = strat.xSizeDiff * strat.midPrice
		if strat.enterValue > strat.usdAvailable {
			strat.xOrderSilentTime = time.Now().Add(strat.config.XErrorSilent)
			if time.Now().Sub(strat.logSilentTime) > strat.config.LogInterval {
				strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
				logger.Debugf(
					"%s %s %s FAILED LONG BOT OPEN, ENTRY VALUE %f MORE THAN usdAvailable %f, %f < %f, %f < %f, SIZE %f",
					strat.config.Name,
					strat.xSymbol,
					strat.ySymbol,
					strat.enterValue,
					strat.usdAvailable,
					strat.spread.LongLastEnter, strat.longBot,
					strat.spread.LongMedianEnter, strat.longBot,
					strat.xSizeDiff,
				)
			}
			goto hedgeSmall
		}
		strat.xSizeDiff = math.Floor(strat.xSizeDiff/strat.xMultiplier/strat.xStepSize) * strat.xStepSize
		if strat.xSizeDiff <= 0 || strat.enterValue < 1.2*strat.yMinNotional || strat.enterValue < 1.2*strat.xMinNotional {
			strat.xOrderSilentTime = time.Now().Add(strat.config.XErrorSilent)
			if time.Now().Sub(strat.logSilentTime) > 0 {
				strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
				logger.Debugf(
					"%s %s %s FAILED LONG BOT OPEN, ORDER VALUE %f TOO SMALL, %f < %f, %f < %f, SIZE %f",
					strat.config.Name,
					strat.xSymbol, strat.ySymbol,
					strat.enterValue,
					strat.spread.LongLastEnter, strat.longBot,
					strat.spread.LongMedianEnter, strat.longBot,
					strat.xSizeDiff,
				)
			}
			goto hedgeSmall
		}
		strat.xPrice = math.Ceil(strat.xMidPrice*(1.0+strat.orderOffset.Top)/strat.xTickSize) * strat.xTickSize
		if strat.xPrice < strat.xTicker.GetBidPrice()+strat.xTickSize {
			strat.xPrice = strat.xTicker.GetBidPrice() + strat.xTickSize
		}
		strat.xNewOrderParam = common.NewOrderParam{
			Symbol:      strat.xSymbol,
			Side:        common.OrderSideSell,
			Type:        common.OrderTypeLimit,
			Price:       strat.xPrice,
			TimeInForce: common.OrderTimeInForceGTC,
			Size:        strat.xSizeDiff,
			PostOnly:    true,
			ReduceOnly:  false,
			ClientID:    strat.xExchange.GenerateClientID(),
		}
		strat.xOpenOrder = &strat.xNewOrderParam
		strat.xOpenOrderCheckTimer.Reset(strat.config.XOrderCheckInterval)
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
		strat.xOrderSilentTime = time.Now().Add(strat.config.XOrderSilent)
		strat.lastXActiveTime = strat.xOrderSilentTime
		logger.Debugf(
			"%s %s LONG BOT OPEN %f < %f, %f < %f, ES %f EV %f, QM %f, SIZE %f PRICE %f, X %v Y %v",
			strat.config.Name,
			strat.xSymbol,
			strat.spread.LongLastEnter, strat.longBot,
			strat.spread.LongMedianEnter, strat.longBot,
			strat.enterStep,
			strat.enterValue,
			*strat.quantileMiddle,
			strat.xSizeDiff,
			strat.xPrice,
			time.Now().Sub(strat.xTickerTime),
			time.Now().Sub(strat.yTickerTime),
		)
	}

	//如果lastEnterTime没有更新，说明没有信号触发，就需要检查对冲的情况
hedgeSmall:
	if time.Now().Sub(strat.lastXActiveTime) > strat.config.HedgeXTimeout {
		//如果已经没有信号对冲，重新检查x y的仓位，对冲较小的
		if math.Abs(strat.xPosition.GetSize()*strat.xMultiplier) < math.Abs(strat.yPosition.GetSize()*strat.yMultiplier) {
			//X的size比Y小，不用操作X
			return
		}
		strat.xSizeDiff = -strat.yPosition.GetSize()*strat.yMultiplier/strat.xMultiplier - strat.xPosition.GetSize()
		strat.xSizeDiff = math.Round(strat.xSizeDiff/strat.xStepSize) * strat.xStepSize

		//如y下单也加上控制，以限下单太大，造成市场冲击
		if strat.xSizeDiff*strat.xMultiplier < -strat.maxOrderValue/strat.xTicker.GetBidPrice() {
			strat.xSizeDiff = -strat.maxOrderValue / strat.xTicker.GetBidPrice() / strat.xMultiplier
		} else if strat.xSizeDiff*strat.xMultiplier > strat.maxOrderValue/strat.xTicker.GetAskPrice() {
			strat.xSizeDiff = strat.maxOrderValue / strat.xTicker.GetAskPrice() / strat.xMultiplier
		}

		strat.xSizeDiff = math.Floor(strat.xSizeDiff/strat.yStepSize) * strat.yStepSize

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
		} else {
			strat.orderSide = common.OrderSideBuy
		}
		strat.xNewOrderParam = common.NewOrderParam{
			Symbol:     strat.xSymbol,
			Side:       strat.orderSide,
			Type:       common.OrderTypeMarket,
			Size:       strat.xSizeDiff,
			PostOnly:   false,
			ReduceOnly: true,
			ClientID:   strat.xExchange.GenerateClientID(),
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
		strat.xPositionUpdateTime = time.Time{}
		logger.Debugf(
			"%s %s %s REVERSE HEDGE X BY Y, SIZE X %f Y %f, ORDER SIDE %s SIZE %f",
			strat.config.Name,
			strat.xSymbol, strat.ySymbol,
			strat.xPosition.GetSize()*strat.xMultiplier,
			strat.yPosition.GetSize()*strat.yMultiplier,
			strat.orderSide,
			strat.xSizeDiff,
		)
	}

}

func (strat *XYStrategy) isXOpenOrderOk() bool {
	if time.Now().Sub(strat.yTickerTime) > strat.config.YTickerTimeToCancel {
		logger.Debugf("%s %s Y TICKER IS OUT OF DATE IN %v, CANCEL", strat.config.Name, strat.xSymbol, strat.config.YTickerTimeToCancel)
		return false
	}
	//检查价格有没有在OFFSET范围内，不在撤掉
	if strat.xOpenOrder.Side == common.OrderSideBuy &&
		strat.xOpenOrder.Price < strat.xMidPrice*(1.0+strat.orderOffset.FarBot)-strat.xTickSize {
		logger.Debugf("%s %s BUY PRICE %f < FAR BOT %f, CANCEL",
			strat.config.Name,
			strat.xSymbol,
			strat.xOpenOrder.Price,
			strat.xMidPrice*(1.0+strat.orderOffset.FarBot),
		)
		return false
	} else if strat.xOpenOrder.Side == common.OrderSideBuy &&
		strat.xOpenOrder.Price > strat.xMidPrice*(1.0+strat.orderOffset.NearBot)+strat.xTickSize {
		logger.Debugf("%s %s BUY PRICE %f > NEAR BOT %f, CANCEL",
			strat.config.Name,
			strat.xSymbol,
			strat.xOpenOrder.Price,
			strat.xMidPrice*(1.0+strat.orderOffset.NearBot),
		)
		return false
	} else if strat.xOpenOrder.Side == common.OrderSideSell &&
		strat.xOpenOrder.Price > strat.xMidPrice*(1.0+strat.orderOffset.FarTop)+strat.xTickSize {
		logger.Debugf("%s %s SELL PRICE %f > FAR TOP %f, CANCEL ",
			strat.config.Name,
			strat.xSymbol,
			strat.xOpenOrder.Price,
			strat.xMidPrice*(1.0+strat.orderOffset.FarTop),
		)
		return false
	} else if strat.xOpenOrder.Side == common.OrderSideSell &&
		strat.xOpenOrder.Price < strat.xMidPrice*(1.0+strat.orderOffset.NearTop)-strat.xTickSize {
		logger.Debugf("%s %s SELL PRICE %f < NEAR TOP %f, CANCEL ",
			strat.config.Name,
			strat.xSymbol,
			strat.xOpenOrder.Price,
			strat.xMidPrice*(1.0+strat.orderOffset.NearTop),
		)
		return false
	}

	if strat.xOpenOrder.Side == common.OrderSideBuy &&
		!strat.xOpenOrder.ReduceOnly &&
		(strat.yTicker.GetBidPrice()-strat.xOpenOrder.Price)/strat.xOpenOrder.Price > strat.shortTop {
		//买入开多, 是开空价差, 参考ShortTop
		return true
	} else if strat.xOpenOrder.Side == common.OrderSideSell &&
		strat.xOpenOrder.ReduceOnly &&
		(strat.yTicker.GetAskPrice()-strat.xOpenOrder.Price)/strat.xOpenOrder.Price < strat.shortBot {
		//卖出平多, 是平空价, 参考ShortBot
		return true
	} else if strat.xOpenOrder.Side == common.OrderSideSell &&
		!strat.xOpenOrder.ReduceOnly &&
		(strat.yTicker.GetAskPrice()-strat.xOpenOrder.Price)/strat.xOpenOrder.Price < strat.longBot {
		//卖出开空, 是开多价差, 参考LongBot
		return true
	} else if strat.xOpenOrder.Side == common.OrderSideBuy &&
		strat.xOpenOrder.ReduceOnly &&
		(strat.yTicker.GetBidPrice()-strat.xOpenOrder.Price)/strat.xOpenOrder.Price > strat.longTop {
		//买入平空, 是平多价差, 参考LongTop
		return true
	}
	if strat.xOpenOrder.Side == common.OrderSideBuy {
		if strat.xOpenOrder.ReduceOnly {
			logger.Debugf(
				"%s NOT PROFITABLE %s BUY ORDER, CANCEL, LONG TOP REDUCE SPREAD %f < %f  X %f %f Y %f %f", strat.xSymbol,
				strat.config.Name,
				(strat.yTicker.GetBidPrice()-strat.xOpenOrder.Price)/strat.xOpenOrder.Price,
				strat.longTop,
				strat.xTicker.GetBidPrice(),
				strat.xTicker.GetAskPrice(),
				strat.yTicker.GetBidPrice(),
				strat.yTicker.GetAskPrice(),
			)
		} else {
			logger.Debugf(
				"%s NOT PROFITABLE %s BUY ORDER, CANCEL, SHORT TOP OPEN SPREAD %f < %f  X %f %f Y %f %f", strat.xSymbol,
				strat.config.Name,
				(strat.yTicker.GetBidPrice()-strat.xOpenOrder.Price)/strat.xOpenOrder.Price,
				strat.shortTop,
				strat.xTicker.GetBidPrice(),
				strat.xTicker.GetAskPrice(),
				strat.yTicker.GetBidPrice(),
				strat.yTicker.GetAskPrice(),
			)
		}
	} else {
		if strat.xOpenOrder.ReduceOnly {
			logger.Debugf(
				"%s NOT PROFITABLE %s BUY ORDER, CANCEL, SHORT BOT REDUCE SPREAD %f > %f  X %f %f Y %f %f", strat.xSymbol,
				strat.config.Name,
				(strat.yTicker.GetAskPrice()-strat.xOpenOrder.Price)/strat.xOpenOrder.Price,
				strat.shortBot,
				strat.xTicker.GetBidPrice(),
				strat.xTicker.GetAskPrice(),
				strat.yTicker.GetBidPrice(),
				strat.yTicker.GetAskPrice(),
			)
		} else {
			logger.Debugf(
				"%s NOT PROFITABLE %s BUY ORDER, CANCEL, LONG BOT OPEN SPREAD %f > %f  X %f %f Y %f %f", strat.xSymbol,
				strat.config.Name,
				(strat.yTicker.GetAskPrice()-strat.xOpenOrder.Price)/strat.xOpenOrder.Price,
				strat.longBot,
				strat.xTicker.GetBidPrice(),
				strat.xTicker.GetAskPrice(),
				strat.yTicker.GetBidPrice(),
				strat.yTicker.GetAskPrice(),
			)
		}
	}
	return false
}
