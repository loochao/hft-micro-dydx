package main

import (
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func cancelAllMakerOpenOrders() {
	for symbol, order := range mOpenOrders {
		if time.Now().Sub(mCancelSilentTimes[symbol]) < 0 {
			continue
		}
		delete(mOpenOrders, symbol)
		mOrderSilentTimes[order.Symbol] = time.Now().Add(mConfig.OrderSilent)
		mCancelSilentTimes[order.Symbol] = time.Now().Add(mConfig.CancelSilent)
		mOrderRequestChMap[order.Symbol] <- common.OrderRequest{
			Cancel: &common.CancelOrderParam{Symbol: order.Symbol},
		}
	}
}

func updateMakerOldOrders() {
	for symbol, order := range mOpenOrders {
		if time.Now().Sub(mCancelSilentTimes[symbol]) < 0 {
			continue
		}
		if isOrderProfitable(order) {
			continue
		}
		delete(mOpenOrders, symbol)
		mOrderSilentTimes[order.Symbol] = time.Now().Add(mConfig.OrderSilent)
		mCancelSilentTimes[order.Symbol] = time.Now().Add(mConfig.CancelSilent)
		select {
		case mOrderRequestChMap[order.Symbol] <- common.OrderRequest{
			Cancel: &common.CancelOrderParam{Symbol: order.Symbol},
		}:
		default:
			logger.Debugf(" mOrderRequestChMap[order.Symbol] <- common.OrderRequest failed, %s ch len %d", symbol, len(mOrderRequestChMap[order.Symbol]))
		}
	}
}

func isOrderProfitable(order common.NewOrderParam) bool {
	depth, okDepth := mWalkedDepths[order.Symbol]

	if !okDepth {
		//logger.Debugf("depth for %s not all ready, cancel", order.Symbol)
		return false
	}
	if time.Now().Sub(depth.Time) > mConfig.DepthTimeToLive {
		//logger.Debugf("depth for %s too old %v, cancel", time.Now().Sub(depth.Time), order.Symbol)
		return false
	}

	offset := mOrderOffsets[order.Symbol]

	//检查价格有没有在OFFSET范围内，不在撤掉
	if order.Side == common.OrderSideBuy &&
		order.Price < depth.MakerBid*(1.0+offset.FarBot) {
		//logger.Debugf("%s BUY PRICE %f < FAR BOT %f, CANCEL",
		//	order.Symbol,
		//	order.Price,
		//	depth.MakerBid*(1.0+offset.FarBot),
		//)
		return false
	} else if order.Side == common.OrderSideBuy &&
		order.Price > depth.MakerBid*(1.0+offset.NearBot) {
		//logger.Debugf("%s BUY PRICE %f > NEAR BOT %f, CANCEL",
		//	order.Symbol,
		//	order.Price,
		//	depth.MakerBid*(1.0+offset.NearBot),
		//)
		return false
	} else if order.Side == common.OrderSideSell &&
		order.Price > depth.MakerAsk*(1.0+offset.FarTop) {
		//logger.Debugf("%s SELL PRICE %f > FAR TOP %f, CANCEL ",
		//	order.Symbol,
		//	order.Price,
		//	depth.MakerAsk*(1.0+offset.FarTop),
		//)
		return false
	} else if order.Side == common.OrderSideSell &&
		order.Price < depth.MakerAsk*(1.0+offset.NearTop) {
		//logger.Debugf("%s SELL PRICE %f < NEAR TOP %f, CANCEL ",
		//	order.Symbol,
		//	order.Price,
		//	depth.MakerAsk*(1.0+offset.NearTop),
		//)
		return false
	}
	return true
}
