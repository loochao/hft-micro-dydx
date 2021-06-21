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
		time.Now().Sub(strat.spread.Time) > strat.config.SpreadTimeToLive ||
		!strat.tradable {
		if time.Now().Sub(strat.spread.Time) > strat.config.SpreadTimeToLive {
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
		strat.xSize >= strat.xStepSize {

		strat.enterValue = math.Min(4*strat.enterStep, math.Min(strat.xAbsValue, strat.yAbsValue))
		if *strat.xyFundingRate > strat.config.MinimalKeepFundingRate*0.5 {
			strat.enterValue = math.Min(2*strat.enterStep, math.Min(strat.xAbsValue, strat.yAbsValue))
		}
		strat.size = strat.enterValue / strat.midPrice
		strat.size = math.Round(strat.size/strat.xyMergedSpotStepSize) * strat.xyMergedSpotStepSize
		strat.enterValue = strat.size * strat.midPrice

		if strat.xAbsValue-strat.enterValue < strat.xyMergedSpotStepSize || strat.yAbsValue-strat.enterValue < strat.xyMergedSpotStepSize {
			//两种情况都把x全平，间接y全平
			strat.size = strat.xSize
		}

		strat.size = math.Round(strat.size / strat.xMultiplier)
		if strat.size > 0 {
			strat.price = math.Ceil(strat.xWalkedDepth.MidPrice/strat.xTickSize) * strat.xTickSize
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
		strat.xSize <= -strat.xStepSize {

		strat.enterValue = math.Min(4*strat.enterStep, math.Min(strat.xAbsValue, strat.yAbsValue))
		if *strat.xyFundingRate < -strat.config.MinimalKeepFundingRate*0.5 {
			strat.enterValue = math.Min(2*strat.enterStep, math.Min(strat.xAbsValue, strat.yAbsValue))
		}
		strat.size = strat.enterValue / strat.midPrice
		strat.size = math.Round(strat.size/strat.xyMergedSpotStepSize) * strat.xyMergedSpotStepSize
		strat.enterValue = strat.size * strat.midPrice
		if strat.xAbsValue-strat.enterValue < strat.xyMergedSpotStepSize || strat.yAbsValue-strat.enterValue < strat.xyMergedSpotStepSize {
			strat.size = -strat.xSize
		}
		strat.size = math.Round(strat.size / strat.xMultiplier)
		if strat.size > 0 {
			strat.price = math.Floor(strat.xWalkedDepth.MidPrice/strat.xTickSize) * strat.xTickSize
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
		strat.size = math.Round(strat.size / strat.xMultiplier)
		if strat.size <= 0 || strat.enterValue < strat.yMinNotional || strat.enterValue < strat.xMinNotional {
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
		strat.price = math.Floor(strat.xWalkedDepth.MidPrice/strat.xTickSize) * strat.xTickSize
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
		strat.size = math.Round(strat.size / strat.xMultiplier)
		if strat.size <= 0 || strat.enterValue < strat.yMinNotional || strat.enterValue < strat.xMinNotional {
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

		strat.price = math.Ceil(strat.xWalkedDepth.MidPrice/strat.xTickSize) * strat.xTickSize
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
		strat.xOpenOrder.Price < strat.xWalkedDepth.BidPrice - strat.xTickSize {
		logger.Debugf("%s BUY PRICE %f < FAR BOT %f, CANCEL",
			strat.xSymbol,
			strat.xOpenOrder.Price,
			strat.xWalkedDepth.BidPrice - strat.xTickSize,
		)
		return false
	//} else if strat.xOpenOrder.Side == common.OrderSideBuy &&
	//	strat.xOpenOrder.Price > strat.xWalkedDepth.BidPrice*(1.0+strat.orderOffset.NearBot) {
	//	logger.Debugf("%s BUY PRICE %f > NEAR BOT %f, CANCEL",
	//		strat.xSymbol,
	//		strat.xOpenOrder.Price,
	//		strat.xWalkedDepth.BidPrice*(1.0+strat.orderOffset.NearBot),
	//	)
	//	return false
	} else if strat.xOpenOrder.Side == common.OrderSideSell &&
		strat.xOpenOrder.Price > strat.xWalkedDepth.AskPrice + strat.xTickSize {
		logger.Debugf("%s SELL PRICE %f > FAR TOP %f, CANCEL ",
			strat.xSymbol,
			strat.xOpenOrder.Price,
			strat.xWalkedDepth.AskPrice + strat.xTickSize,
		)
		return false
	//} else if strat.xOpenOrder.Side == common.OrderSideSell &&
	//	strat.xOpenOrder.Price < strat.xWalkedDepth.AskPrice*(1.0+strat.orderOffset.NearTop) {
	//	logger.Debugf("%s SELL PRICE %f < NEAR TOP %f, CANCEL ",
	//		strat.xSymbol,
	//		strat.xOpenOrder.Price,
	//		strat.xWalkedDepth.AskPrice*(1.0+strat.orderOffset.NearTop),
	//	)
	//	return false
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

