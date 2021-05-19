package main

import (
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"time"
)

func cancelAllMakerOpenOrders() {
	for symbol, order := range xOpenOrders {
		if time.Now().Sub(xCancelSilentTimes[symbol]) < 0 {
			continue
		}
		delete(xOpenOrders, symbol)
		xOrderSilentTimes[order.Symbol] = time.Now().Add(xyConfig.OrderSilent)
		xCancelSilentTimes[order.Symbol] = time.Now().Add(xyConfig.CancelSilent)
		xOrderRequestChMap[order.Symbol] <- common.OrderRequest{
			Cancel: &common.CancelOrderParam{Symbol: order.Symbol},
		}
	}
}

func updateMakerOldOrders() {
	if xAccount == nil || yAccount == nil {
		return
	}

	entryStep := (xAccount.GetFree() + yAccount.GetFree()) * xyConfig.EnterFreePct
	if entryStep < xyConfig.EnterMinimalStep {
		entryStep = xyConfig.EnterMinimalStep
	}
	entryTarget := entryStep * xyConfig.EnterTargetFactor

	for symbol, order := range xOpenOrders {
		if time.Now().Sub(xCancelSilentTimes[symbol]) < 0 {
			continue
		}
		if isOrderProfitable(order, entryTarget) {
			continue
		}
		delete(xOpenOrders, symbol)
		xOrderSilentTimes[order.Symbol] = time.Now().Add(xyConfig.OrderSilent)
		xCancelSilentTimes[order.Symbol] = time.Now().Add(xyConfig.CancelSilent)
		xOrderRequestChMap[order.Symbol] <- common.OrderRequest{
			Cancel: &common.CancelOrderParam{Symbol: order.Symbol},
		}
	}
}

func isOrderProfitable(order common.NewOrderParam, entryTarget float64) bool {
	spread, okSpread := xySpreads[order.Symbol]
	makerPosition, okMakerPosition := xPositions[order.Symbol]

	if !okSpread || !okMakerPosition {
		logger.Debugf("spread %v maker position %v not all ready %s", okSpread, okMakerPosition, order.Symbol)
		return false
	}
	if time.Now().Sub(spread.Time) > xyConfig.SpreadTimeToLive {
		logger.Debugf("spread too old %v, cancel %s", time.Now().Sub(spread.Time), order.Symbol)
		return false
	}

	makerValue := makerPosition.GetPrice() * makerPosition.GetSize()
	offset := mOrderOffsets[order.Symbol]
	delta := xyDeltas[order.Symbol]
	shortTop := delta.ShortTop + xyConfig.EnterOffsetDelta*(math.Max(makerValue, 0)/entryTarget)
	shortBot := delta.ShortBot + xyConfig.EnterOffsetDelta*(math.Max(makerValue, 0)/entryTarget)
	longBot := delta.LongBot + xyConfig.EnterOffsetDelta*(math.Min(makerValue, 0)/entryTarget)
	longTop := delta.LongTop + xyConfig.EnterOffsetDelta*(math.Min(makerValue, 0)/entryTarget)

	//检查价格有没有在OFFSET范围内，不在撤掉
	if order.Side == common.OrderSideBuy &&
		order.Price < spread.MakerDepth.MakerBid*(1.0+offset.FarBot) {
		logger.Debugf("%s BUY PRICE %f < FAR BOT %f, CANCEL",
			order.Symbol,
			order.Price,
			spread.MakerDepth.MakerBid*(1.0+offset.FarBot),
		)
		return false
	} else if order.Side == common.OrderSideBuy &&
		order.Price > spread.MakerDepth.MakerBid*(1.0+offset.NearBot) {
		logger.Debugf("%s BUY PRICE %f > NEAR BOT %f, CANCEL",
			order.Symbol,
			order.Price,
			spread.MakerDepth.MakerBid*(1.0+offset.NearBot),
		)
		return false
	} else if order.Side == common.OrderSideSell &&
		order.Price > spread.MakerDepth.MakerAsk*(1.0+offset.FarTop) {
		logger.Debugf("%s SELL PRICE %f > FAR TOP %f, CANCEL ",
			order.Symbol,
			order.Price,
			spread.MakerDepth.MakerAsk*(1.0+offset.FarTop),
		)
		return false
	} else if order.Side == common.OrderSideSell &&
		order.Price < spread.MakerDepth.MakerAsk*(1.0+offset.NearTop) {
		logger.Debugf("%s SELL PRICE %f < NEAR TOP %f, CANCEL ",
			order.Symbol,
			order.Price,
			spread.MakerDepth.MakerAsk*(1.0+offset.NearTop),
		)
		return false
	}

	if order.Side == common.OrderSideBuy &&
		!order.ReduceOnly &&
		(spread.TakerDepth.TakerBid-order.Price)/order.Price > shortTop {
		//买入开多, 是开空价差, 参考ShortTop
		return true
	} else if order.Side == common.OrderSideSell &&
		order.ReduceOnly &&
		(spread.TakerDepth.TakerAsk-order.Price)/order.Price < shortBot {
		//卖出平多, 是平空价, 参考ShortBot
		return true
	} else if order.Side == common.OrderSideSell &&
		!order.ReduceOnly &&
		(spread.TakerDepth.TakerAsk-order.Price)/order.Price < longBot {
		//卖出开空, 是开多价差, 参考LongBot
		return true
	} else if order.Side == common.OrderSideBuy &&
		order.ReduceOnly &&
		(spread.TakerDepth.TakerBid-order.Price)/order.Price > longTop {
		//买入平空, 是平多价差, 参考LongTop
		return true
	}
	if order.Side == common.OrderSideBuy {
		logger.Debugf(
			"NOT PROFITABLE %s BUY ORDER, CANCEL", order.Symbol,
		)
	} else {
		logger.Debugf(
			"NOT PROFITABLE %s SELL ORDER, CANCEL", order.Symbol,
		)
	}
	return false
}
