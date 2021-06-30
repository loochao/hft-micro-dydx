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
		strat.fundingRateSettleSilent {
		if time.Now().Sub(strat.spread.EventTime) > strat.config.SpreadTimeToLive {
			strat.tryCancelXOpenOrder("spread time out")
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

	strat.shortTop = strat.config.ShortEnterDelta + strat.config.EnterOffsetDelta*strat.offsetFactor - *strat.xyFundingRate*strat.config.FrOffsetFactor
	strat.shortBot = strat.config.ShortExitDelta + strat.config.ExitOffsetDelta*(strat.offsetFactor-strat.offsetStep) - *strat.xyFundingRate*strat.config.FrOffsetFactor
	strat.longBot = strat.config.LongEnterDelta - strat.config.EnterOffsetDelta*strat.offsetFactor - *strat.xyFundingRate*strat.config.FrOffsetFactor
	strat.longTop = strat.config.LongExitDelta - strat.config.ExitOffsetDelta*(strat.offsetFactor-strat.offsetStep) - *strat.xyFundingRate*strat.config.FrOffsetFactor

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
		strat.spread.ShortLastLeave < strat.spread.ShortMedianLeave &&
		*strat.xyFundingRate < strat.config.MinimalKeepFundingRate &&
		strat.xSize >= strat.xStepSize*strat.xMultiplier {
		strat.enterValue = math.Min(4*strat.enterStep, math.Min(strat.xAbsValue, strat.yAbsValue))
		if *strat.xyFundingRate > strat.config.MinimalKeepFundingRate*0.5 {
			strat.enterValue = math.Min(2*strat.enterStep, math.Min(strat.xAbsValue, strat.yAbsValue))
		}
		if strat.enterValue > strat.maxOrderValue {
			strat.enterValue = strat.maxOrderValue
		}
		strat.size = strat.enterValue / strat.midPrice
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
			strat.price = math.Ceil(strat.xMidPrice*(1.0+strat.orderOffset.Top)/strat.xTickSize) * strat.xTickSize
			if strat.price < strat.xTicker.GetBidPrice()+strat.xTickSize {
				strat.price = strat.xTicker.GetBidPrice() + strat.xTickSize
			}
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

		strat.enterValue = math.Min(4*strat.enterStep, math.Min(strat.xAbsValue, strat.yAbsValue))
		if *strat.xyFundingRate < -strat.config.MinimalKeepFundingRate*0.5 {
			strat.enterValue = math.Min(2*strat.enterStep, math.Min(strat.xAbsValue, strat.yAbsValue))
		}
		if strat.enterValue > strat.maxOrderValue {
			strat.enterValue = strat.maxOrderValue
		}
		strat.size = strat.enterValue / strat.midPrice
		strat.size = math.Round(strat.size/strat.xyMergedSpotStepSize) * strat.xyMergedSpotStepSize
		strat.enterValue = strat.size * strat.midPrice
		if strat.xAbsValue-strat.enterValue < strat.xyMergedSpotStepSize*1.005 ||
			strat.yAbsValue-strat.enterValue < strat.xyMergedSpotStepSize*1.005 ||
			strat.size > -strat.xSize {
			strat.size = -strat.xSize
		}
		strat.size = math.Floor(strat.size/strat.xMultiplier/strat.xStepSize) * strat.xStepSize
		if strat.size > 0 && (!strat.isXSpot || strat.enterValue > 1.2*strat.xMinNotional) {
			strat.price = math.Floor(strat.xMidPrice*(1.0+strat.orderOffset.Bot)/strat.xTickSize) * strat.xTickSize
			if strat.price > strat.xTicker.GetAskPrice()-strat.xTickSize {
				strat.price = strat.xTicker.GetAskPrice() - strat.xTickSize
			}
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
					strat.size,
				)
			}
			return
		}
		strat.size = math.Floor(strat.size/strat.xMultiplier/strat.xStepSize) * strat.xStepSize
		if strat.size <= 0 || strat.enterValue < 1.2*strat.yMinNotional || strat.enterValue < 1.2*strat.xMinNotional {
			strat.xOrderSilentTime = time.Now().Add(strat.config.XErrorSilent)
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
		strat.price = math.Floor(strat.xMidPrice*(1.0+strat.orderOffset.Bot)/strat.xTickSize) * strat.xTickSize
		if strat.price > strat.xTicker.GetAskPrice()-strat.xTickSize {
			strat.price = strat.xTicker.GetAskPrice() - strat.xTickSize
		}
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
		if strat.enterValue > strat.usdAvailable {
			strat.xOrderSilentTime = time.Now().Add(strat.config.XErrorSilent)
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
			strat.xOrderSilentTime = time.Now().Add(strat.config.XErrorSilent)
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
		strat.price = math.Ceil(strat.xMidPrice*(1.0+strat.orderOffset.Top)/strat.xTickSize) * strat.xTickSize
		if strat.price < strat.xTicker.GetBidPrice()+strat.xTickSize {
			strat.price = strat.xTicker.GetBidPrice() + strat.xTickSize
		}
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

func (strat *XYStrategy) isXOpenOrderOk() bool {
	if time.Now().Sub(strat.spread.EventTime) > strat.config.SpreadTimeToLive {
		logger.Debugf("%s SPREAD IS OUT OF DATE, CANCEL", strat.xSymbol)
		return false
	}
	//检查价格有没有在OFFSET范围内，不在撤掉
	if strat.xOpenOrder.Side == common.OrderSideBuy &&
		strat.xOpenOrder.Price < strat.xMidPrice*(1.0+strat.orderOffset.FarBot)-strat.xTickSize {
		logger.Debugf("%s BUY PRICE %f < FAR BOT %f, CANCEL",
			strat.xSymbol,
			strat.xOpenOrder.Price,
			strat.xMidPrice*(1.0+strat.orderOffset.FarBot),
		)
		return false
	} else if strat.xOpenOrder.Side == common.OrderSideBuy &&
		strat.xOpenOrder.Price > strat.xMidPrice*(1.0+strat.orderOffset.NearBot)+strat.xTickSize {
		logger.Debugf("%s BUY PRICE %f > NEAR BOT %f, CANCEL",
			strat.xSymbol,
			strat.xOpenOrder.Price,
			strat.xMidPrice*(1.0+strat.orderOffset.NearBot),
		)
		return false
	} else if strat.xOpenOrder.Side == common.OrderSideSell &&
		strat.xOpenOrder.Price > strat.xMidPrice*(1.0+strat.orderOffset.FarTop)+strat.xTickSize {
		logger.Debugf("%s SELL PRICE %f > FAR TOP %f, CANCEL ",
			strat.xSymbol,
			strat.xOpenOrder.Price,
			strat.xMidPrice*(1.0+strat.orderOffset.FarTop),
		)
		return false
	} else if strat.xOpenOrder.Side == common.OrderSideSell &&
		strat.xOpenOrder.Price < strat.xMidPrice*(1.0+strat.orderOffset.NearTop)-strat.xTickSize {
		logger.Debugf("%s SELL PRICE %f < NEAR TOP %f, CANCEL ",
			strat.xSymbol,
			strat.xOpenOrder.Price,
			strat.xMidPrice*(1.0+strat.orderOffset.NearTop),
		)
		return false
	}

	if strat.xOpenOrder.Side == common.OrderSideBuy &&
		!strat.xOpenOrder.ReduceOnly &&
		(strat.yTicker.GetBidPrice()-strat.xOpenOrder.Price)/strat.xOpenOrder.Price > strat.shortTop-(strat.config.ShortEnterDelta-strat.config.ShortExitDelta)*strat.config.CancelOffsetFactor {
		//买入开多, 是开空价差, 参考ShortTop
		return true
	} else if strat.xOpenOrder.Side == common.OrderSideSell &&
		strat.xOpenOrder.ReduceOnly &&
		(strat.yTicker.GetAskPrice()-strat.xOpenOrder.Price)/strat.xOpenOrder.Price < strat.shortBot+(strat.config.ShortEnterDelta-strat.config.ShortExitDelta)*strat.config.CancelOffsetFactor {
		//卖出平多, 是平空价, 参考ShortBot
		return true
	} else if strat.xOpenOrder.Side == common.OrderSideSell &&
		!strat.xOpenOrder.ReduceOnly &&
		(strat.yTicker.GetAskPrice()-strat.xOpenOrder.Price)/strat.xOpenOrder.Price < strat.longBot+(strat.config.LongExitDelta-strat.config.LongEnterDelta)*strat.config.CancelOffsetFactor {
		//卖出开空, 是开多价差, 参考LongBot
		return true
	} else if strat.xOpenOrder.Side == common.OrderSideBuy &&
		strat.xOpenOrder.ReduceOnly &&
		(strat.yTicker.GetBidPrice()-strat.xOpenOrder.Price)/strat.xOpenOrder.Price > strat.longTop-(strat.config.LongExitDelta-strat.config.LongEnterDelta)*strat.config.CancelOffsetFactor {
		//买入平空, 是平多价差, 参考LongTop
		return true
	}
	if strat.xOpenOrder.Side == common.OrderSideBuy {
		if strat.xOpenOrder.ReduceOnly {
			logger.Debugf(
				"NOT PROFITABLE %s BUY ORDER, CANCEL, LONG TOP REDUCE SPREAD %f < %f  X %f %f Y %f %f", strat.xSymbol,
				(strat.yTicker.GetBidPrice()-strat.xOpenOrder.Price)/strat.xOpenOrder.Price,
				strat.longTop-(strat.config.LongExitDelta-strat.config.LongEnterDelta)*strat.config.CancelOffsetFactor,
				strat.xTicker.GetBidPrice(),
				strat.xTicker.GetAskPrice(),
				strat.yTicker.GetBidPrice(),
				strat.yTicker.GetAskPrice(),
			)
		} else {
			logger.Debugf(
				"NOT PROFITABLE %s BUY ORDER, CANCEL, SHORT TOP OPEN SPREAD %f > %f  X %f %f Y %f %f", strat.xSymbol,
				(strat.yTicker.GetBidPrice()-strat.xOpenOrder.Price)/strat.xOpenOrder.Price,
				strat.shortTop-(strat.config.ShortEnterDelta-strat.config.ShortExitDelta)*strat.config.CancelOffsetFactor,
				strat.xTicker.GetBidPrice(),
				strat.xTicker.GetAskPrice(),
				strat.yTicker.GetBidPrice(),
				strat.yTicker.GetAskPrice(),
			)
		}
	} else {
		if strat.xOpenOrder.ReduceOnly {
			logger.Debugf(
				"NOT PROFITABLE %s BUY ORDER, CANCEL, SHORT BOT REDUCE SPREAD %f > %f  X %f %f Y %f %f", strat.xSymbol,
				(strat.yTicker.GetAskPrice()-strat.xOpenOrder.Price)/strat.xOpenOrder.Price,
				strat.shortBot+(strat.config.ShortEnterDelta-strat.config.ShortExitDelta)*strat.config.CancelOffsetFactor,
				strat.xTicker.GetBidPrice(),
				strat.xTicker.GetAskPrice(),
				strat.yTicker.GetBidPrice(),
				strat.yTicker.GetAskPrice(),
			)
		} else {
			logger.Debugf(
				"NOT PROFITABLE %s BUY ORDER, CANCEL, LONG BOT OPEN SPREAD %f > %f  X %f %f Y %f %f", strat.xSymbol,
				(strat.yTicker.GetAskPrice()-strat.xOpenOrder.Price)/strat.xOpenOrder.Price,
				strat.longBot+(strat.config.LongExitDelta-strat.config.LongEnterDelta)*strat.config.CancelOffsetFactor,
				strat.xTicker.GetBidPrice(),
				strat.xTicker.GetAskPrice(),
				strat.yTicker.GetBidPrice(),
				strat.yTicker.GetAskPrice(),
			)
		}
	}
	return false
}
