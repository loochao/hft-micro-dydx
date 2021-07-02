package main

import (
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"time"
)

func (strat *XYStrategy) updateXYOrder() {
	if strat.xSystemStatus != common.SystemStatusReady ||
		strat.ySystemStatus != common.SystemStatusReady {
		if time.Now().Sub(strat.logSilentTime) > 0 {
			strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
			logger.Debugf("updateXYOrder xSystemStatus %v ySystemStatus %v", strat.xSystemStatus, strat.ySystemStatus)
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
		time.Now().Sub(strat.spread.EventTime) > strat.config.SpreadTimeToEnter {
		if time.Now().Sub(strat.spread.EventTime) > strat.config.SpreadTimeToCancel {
			strat.tryCancelXOpenOrder("spread time out")
			strat.tryCancelYOpenOrder("spread time out")
		}
		return
	}

	strat.xSize = strat.xPosition.GetSize() * strat.xMultiplier
	strat.ySize = strat.yPosition.GetSize() * strat.yMultiplier
	strat.xValue = strat.xSize * strat.xWalkedDepth.MidPrice
	strat.yValue = strat.ySize * strat.yWalkedDepth.MidPrice
	strat.xAbsValue = math.Abs(strat.xValue)
	strat.yAbsValue = math.Abs(strat.yValue)

	strat.midPrice = (strat.xWalkedDepth.MidPrice + strat.yWalkedDepth.MidPrice) * 0.5

	if time.Now().Sub(strat.xOrderSilentTime) < 0 {
		return
	}

	if time.Now().Sub(strat.xCancelSilentTime) < 0 {
		return
	}

	if time.Now().Sub(strat.yOrderSilentTime) < 0 {
		return
	}

	if time.Now().Sub(strat.yCancelSilentTime) < 0 {
		return
	}

	if strat.config.EnterDepthMatchRatio > strat.xyDepthMatchRatio {
		strat.xOrderSilentTime = time.Now().Add(strat.config.EnterSilent)
		strat.tryCancelXOpenOrder("small match ratio")
		logger.Debugf("%s match ratio %f < %f, silent %v", strat.xSymbol, strat.xyDepthMatchRatio, strat.config.EnterDepthMatchRatio, strat.config.EnterSilent)
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

	if strat.spread.XYLastEnter > strat.config.EnterDelta &&
		strat.spread.XYMedianEnter > strat.config.EnterDelta &&
		strat.spread.XYLastEnter > strat.spread.XYMedianEnter &&
		strat.xSize < strat.xStepSize && strat.ySize < strat.yStepSize {

		strat.enterValue = strat.enterStep
		if strat.enterValue > strat.maxOrderValue {
			strat.enterValue = strat.maxOrderValue
		}
		strat.size = strat.enterValue / strat.midPrice
		strat.size = math.Round(strat.size/strat.xyMergedSpotStepSize) * strat.xyMergedSpotStepSize
		strat.enterValue = strat.size * strat.midPrice
		if strat.enterValue > strat.usdAvailable {
			strat.xOrderSilentTime = time.Now().Add(strat.config.ErrorSilent)
			if time.Now().Sub(strat.logSilentTime) > 0 {
				strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
				logger.Debugf(
					"%s FAILED XY OPEN, ENTRY VALUE %f MORE THAN usdAvailable %f, %f > %f, %f > %f, SIZE %f",
					strat.xSymbol,
					strat.enterValue,
					strat.usdAvailable,
					strat.spread.XYLastEnter, strat.config.EnterDelta,
					strat.spread.XYMedianEnter, strat.config.EnterDelta,
					strat.size,
				)
			}
			return
		}
		strat.size = math.Floor(strat.size/strat.xMultiplier/strat.xStepSize) * strat.xStepSize
		if strat.size <= 0 || strat.enterValue < 1.2*strat.yMinNotional || strat.enterValue < 1.2*strat.xMinNotional {
			strat.xOrderSilentTime = time.Now().Add(strat.config.EnterSilent)
			if time.Now().Sub(strat.logSilentTime) > 0 {
				strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
				logger.Debugf(
					"%s FAILED XY OPEN, ORDER VALUE %f TOO SMALL, %f > %f, %f > %f, SIZE %f",
					strat.xSymbol,
					strat.enterValue,
					strat.spread.XYLastEnter, strat.config.EnterDelta,
					strat.spread.XYMedianEnter, strat.config.EnterDelta,
					strat.size,
				)
			}
			return
		}
		strat.price = math.Floor(strat.xWalkedDepth.MidPrice*(1.0+strat.xOrderOffset.Bot)/strat.xTickSize) * strat.xTickSize
		if strat.price > strat.xWalkedDepth.BestAskPrice-strat.xTickSize {
			strat.price = strat.xWalkedDepth.BestAskPrice - strat.xTickSize
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
		strat.xOpenOrderCheckTimer.Reset(strat.config.OrderCheckInterval)
		strat.tradeDir = 1
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
		strat.yOrderSilentTime = time.Now()
		logger.Debugf(
			"%s XY OPEN %f > %f, %f > %f, SIZE %f PRICE %f, X %v Y %v M %f",
			strat.xSymbol,
			strat.spread.XYLastEnter, strat.config.EnterDelta,
			strat.spread.XYMedianEnter, strat.config.EnterDelta,
			strat.size,
			strat.price,
			time.Now().Sub(strat.xDepthTime),
			time.Now().Sub(strat.yDepthTime),
			strat.xyDepthMatchRatio,
		)
	} else if strat.spread.YXLastEnter < -strat.config.EnterDelta &&
		strat.spread.YXMedianEnter < -strat.config.EnterDelta &&
		strat.spread.YXLastEnter < strat.spread.YXMedianEnter &&
		strat.xSize < strat.xStepSize && strat.ySize < strat.yStepSize {

		strat.enterValue = strat.enterStep
		if strat.enterValue > strat.maxOrderValue {
			strat.enterValue = strat.maxOrderValue
		}
		strat.size = strat.enterValue / strat.midPrice
		strat.size = math.Round(strat.size/strat.xyMergedSpotStepSize) * strat.xyMergedSpotStepSize
		strat.enterValue = strat.size * strat.midPrice
		if strat.enterValue > strat.usdAvailable {
			strat.yOrderSilentTime = time.Now().Add(strat.config.ErrorSilent)
			if time.Now().Sub(strat.logSilentTime) > strat.config.LogInterval {
				strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
				logger.Debugf(
					"%s FAILED YX OPEN, ENTRY VALUE %f MORE THAN usdAvailable %f, %f < %f, %f < %f, SIZE %f",
					strat.ySymbol,
					strat.enterValue,
					strat.usdAvailable,
					strat.spread.YXLastEnter, -strat.config.EnterDelta,
					strat.spread.YXMedianEnter, -strat.config.EnterDelta,
					strat.size,
				)
			}
			return
		}
		strat.size = math.Floor(strat.size/strat.yMultiplier/strat.xStepSize) * strat.xStepSize
		if strat.size <= 0 || strat.enterValue < 1.2*strat.yMinNotional || strat.enterValue < 1.2*strat.xMinNotional {
			strat.yOrderSilentTime = time.Now().Add(strat.config.ErrorSilent)
			if time.Now().Sub(strat.logSilentTime) > 0 {
				strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
				logger.Debugf(
					"%s FAILED YX OPEN, ORDER VALUE %f TOO SMALL, %f < %f, %f < %f, SIZE %f",
					strat.ySymbol,
					strat.enterValue,
					strat.spread.YXLastEnter, -strat.config.EnterDelta,
					strat.spread.YXMedianEnter, -strat.config.EnterDelta,
					strat.size,
				)
			}
			return
		}
		strat.price = math.Floor(strat.yWalkedDepth.MidPrice*(1.0+strat.yOrderOffset.Bot)/strat.yTickSize) * strat.yTickSize
		if strat.price > strat.yWalkedDepth.BestBidPrice-strat.yTickSize {
			strat.price = strat.yWalkedDepth.BestBidPrice - strat.yTickSize
		}
		strat.yNewOrderParam = common.NewOrderParam{
			Symbol:      strat.ySymbol,
			Side:        common.OrderSideBuy,
			Type:        common.OrderTypeLimit,
			Price:       strat.price,
			TimeInForce: common.OrderTimeInForceGTC,
			Size:        strat.size,
			PostOnly:    true,
			ReduceOnly:  false,
			ClientID:    strat.yExchange.GenerateClientID(),
		}
		strat.yOpenOrder = &strat.yNewOrderParam
		strat.yOpenOrderCheckTimer.Reset(strat.config.OrderCheckInterval)
		strat.tradeDir = -1
		if !strat.config.DryRun {
			//logger.Debugf("sending strat.xOrderRequestCh <- common.OrderRequest %s", strat.xSymbol)
			select {
			case strat.yOrderRequestCh <- common.OrderRequest{
				New: &strat.yNewOrderParam,
			}:
				//logger.Debugf("sent strat.xOrderRequestCh <- common.OrderRequest %s", strat.xSymbol)
			}
		}
		strat.xLastFilledBuyPrice = nil
		strat.xLastFilledSellPrice = nil
		strat.yLastFilledBuyPrice = nil
		strat.yLastFilledSellPrice = nil
		strat.yOrderSilentTime = time.Now().Add(strat.config.OrderSilent)
		strat.xOrderSilentTime = time.Now()
		logger.Debugf(
			"%s YX OPEN %f < %f, %f < %f, SIZE %f PRICE %f, X %v Y %v M %f",
			strat.xSymbol,
			strat.spread.YXLastEnter, -strat.config.EnterDelta,
			strat.spread.YXMedianEnter, -strat.config.EnterDelta,
			strat.size,
			strat.price,
			time.Now().Sub(strat.xDepthTime),
			time.Now().Sub(strat.yDepthTime),
			strat.xyDepthMatchRatio,
		)

	}
}

func (strat *XYStrategy) isXOpenOrderOk() bool {
	if time.Now().Sub(strat.spread.EventTime) > strat.config.SpreadTimeToCancel {
		logger.Debugf("%s CANCEL, SPREAD IS OUT OF DATE", strat.xSymbol)
		return false
	}
	if strat.xOpenOrder.Side != common.OrderSideBuy {
		return false
	}
	//检查价格有没有在OFFSET范围内，不在撤掉
	if strat.xOpenOrder.Price < strat.xWalkedDepth.MidPrice*(1.0+strat.xOrderOffset.FarBot)-strat.xTickSize {
		logger.Debugf("%s X CANCEL, BUY PRICE %f < FAR BOT %f",
			strat.xSymbol,
			strat.xOpenOrder.Price,
			strat.xWalkedDepth.MidPrice*(1.0+strat.xOrderOffset.FarBot)-strat.xTickSize,
		)
		return false
	} else if strat.xOpenOrder.Price > strat.xWalkedDepth.MidPrice*(1.0+strat.xOrderOffset.NearBot)+strat.xTickSize {
		logger.Debugf("%s X CANCEL, BUY PRICE %f > NEAR BOT %f",
			strat.xSymbol,
			strat.xOpenOrder.Price,
			strat.xWalkedDepth.MidPrice*(1.0+strat.xOrderOffset.NearBot)+strat.xTickSize,
		)
		return false
	}

	if (strat.yWalkedDepth.BidPrice-strat.xOpenOrder.Price)/strat.xOpenOrder.Price >= strat.config.EnterDelta*(1.0-strat.config.CancelOffsetFactor) {
		//XY OPEN
		return true
	} else {
		logger.Debugf(
			"%s CANCEL, XY OPEN %f < %f", strat.xSymbol,
			(strat.yWalkedDepth.BidPrice-strat.xOpenOrder.Price)/strat.xOpenOrder.Price >= strat.config.EnterDelta*(1.0-strat.config.CancelOffsetFactor),
		)
		return false
	}
}

func (strat *XYStrategy) isYOpenOrderOk() bool {
	if time.Now().Sub(strat.spread.EventTime) > strat.config.SpreadTimeToCancel {
		logger.Debugf("%s CANCEL, SPREAD IS OUT OF DATE", strat.xSymbol)
		return false
	}
	if strat.yOpenOrder.Side != common.OrderSideBuy {
		return false
	}
	//检查价格有没有在OFFSET范围内，不在撤掉
	if strat.yOpenOrder.Price < strat.yWalkedDepth.MidPrice*(1.0+strat.yOrderOffset.FarBot)-strat.yTickSize {
		logger.Debugf("%s Y CANCEL, BUY PRICE %f < FAR BOT %f",
			strat.ySymbol,
			strat.yOpenOrder.Price,
			strat.yWalkedDepth.MidPrice*(1.0+strat.yOrderOffset.FarBot)-strat.yTickSize,
		)
		return false
	} else if strat.yOpenOrder.Price > strat.yWalkedDepth.MidPrice*(1.0+strat.yOrderOffset.NearBot)+strat.yTickSize {
		logger.Debugf("%s X CANCEL, BUY PRICE %f > NEAR BOT %f",
			strat.ySymbol,
			strat.yOpenOrder.Price,
			strat.yWalkedDepth.MidPrice*(1.0+strat.yOrderOffset.NearBot)+strat.yTickSize,
		)
		return false
	}

	if (strat.yWalkedDepth.BidPrice-strat.xOpenOrder.Price)/strat.xOpenOrder.Price >= strat.config.EnterDelta*(1.0-strat.config.CancelOffsetFactor) {
		//XY OPEN
		return true
	} else {
		logger.Debugf(
			"%s CANCEL, XY OPEN %f < %f", strat.xSymbol,
			(strat.yWalkedDepth.BidPrice-strat.xOpenOrder.Price)/strat.xOpenOrder.Price >= strat.config.EnterDelta*(1.0-strat.config.CancelOffsetFactor),
		)
		return false
	}
}

func (strat *XYStrategy) tryCancelXOpenOrder(reason string) {
	if strat.xOpenOrder == nil {
		return
	}
	if time.Now().Sub(strat.xCancelSilentTime) < 0 {
		return
	}
	strat.xCancelSilentTime = time.Now().Add(strat.config.CancelSilent)
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

func (strat *XYStrategy) tryCancelYOpenOrder(reason string) {
	if strat.yOpenOrder == nil {
		return
	}
	if time.Now().Sub(strat.yCancelSilentTime) < 0 {
		return
	}
	strat.yCancelSilentTime = time.Now().Add(strat.config.CancelSilent)
	if !strat.config.DryRun {
		//logger.Debugf("sending cancel strat.xOrderRequestCh <- common.OrderRequest %s %s", strat.xSymbol, reason)
		select {
		case strat.yOrderRequestCh <- common.OrderRequest{
			Cancel: &strat.yCancelOrderParam,
		}:
			//logger.Debugf("sent cancel strat.xOrderRequestCh <- common.OrderRequest %s %s", strat.xSymbol, reason)
		}
	}
	strat.yOpenOrder = nil
}
