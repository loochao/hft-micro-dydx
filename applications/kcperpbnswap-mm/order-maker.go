package main

import (
	"context"
	"github.com/geometrybase/hft-micro/kucoin-usdtfuture"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func watchMakerOrderRequest(
	ctx context.Context,
	api *kucoin_usdtfuture.API,
	timeout time.Duration,
	dryRun bool,
	orderRequestCh chan MakerOrderRequest,
	outputOrderRespCh chan MakerOpenOrder,
	outputOrderErrorCh chan MakerOrderNewError,
) {
	defer func() {
		logger.Debugf("EXIT watchMakerOrderRequest")
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case request := <-orderRequestCh:
			if dryRun {
				break
			}
			if request.New != nil {
				childCtx, _ := context.WithTimeout(ctx, timeout)
				//logger.Debugf("MAKER SUBMIT %s %s %f %d", request.New.Market, request.New.Side, request.New.Price, request.New.Size)
				resp, err := api.SubmitOrder(childCtx, *request.New)
				if err != nil {
					logger.Debugf("MAKER SUBMIT ERROR %v", err)
					outputOrderErrorCh <- MakerOrderNewError{
						Error:  err,
						Params: *request.New,
					}
				} else {
					outputOrderRespCh <- MakerOpenOrder{
						NewOrderParam:   request.New,
						ResponseOrderID: resp.OrderId,
						Symbol:          request.New.Symbol,
					}
				}
			} else if request.Cancel != nil {
				childCtx, _ := context.WithTimeout(ctx, timeout)
				//logger.Debugf("MAKER CANCEL ALL %s", request.Cancel.Market)
				resp, err := api.CancelAllOrders(childCtx, *request.Cancel)
				if err != nil {
					logger.Debugf("MAKER SUBMIT ERROR %v", err)
				} else {
					for _, s := range resp.CancelledOrderIds {
						outputOrderRespCh <- MakerOpenOrder{
							NewOrderParam:   nil,
							ResponseOrderID: s,
							Symbol:          request.Cancel.Symbol,
						}
					}
				}
			}
		}
	}
}

func cancelAllMakerOpenOrders() {
	for symbol, order := range mOpenOrders {
		if mOrderCancelCounts[symbol] > *mtConfig.OrderMaxCancelCount {
			delete(mOpenOrders, symbol)
			continue
		}
		if time.Now().Sub(mOrderCancelSilentTimes[symbol]) < 0 {
			continue
		}
		mOrderSilentTimes[order.Symbol] = time.Now().Add(*mtConfig.OrderCancelSilent)
		mOrderCancelSilentTimes[order.Symbol] = time.Now().Add(*mtConfig.OrderCancelSilent)
		mOrderCancelCounts[order.Symbol] += 1
		mOrderRequestChs[order.Symbol] <- MakerOrderRequest{
			Cancel: &kucoin_usdtfuture.CancelAllOrdersParam{Symbol: order.Symbol},
		}
	}
}

func updateMakerOldOrders() {
	for symbol, order := range mOpenOrders {
		if mOrderCancelCounts[symbol] > *mtConfig.OrderMaxCancelCount {
			delete(mOpenOrders, symbol)
			mOrderCancelCounts[order.Symbol] = 0
			continue
		}
		if time.Now().Sub(mOrderCancelSilentTimes[symbol]) < 0 {
			continue
		}
		if isOrderProfitable(*order.NewOrderParam) {
			continue
		}
		mOrderSilentTimes[order.Symbol] = time.Now().Add(*mtConfig.OrderCancelSilent)
		mOrderCancelSilentTimes[order.Symbol] = time.Now().Add(*mtConfig.OrderCancelSilent)
		mOrderCancelCounts[order.Symbol] += 1
		mOrderRequestChs[order.Symbol] <- MakerOrderRequest{
			Cancel: &kucoin_usdtfuture.CancelAllOrdersParam{Symbol: order.Symbol},
		}
	}
}

func isOrderProfitable(order kucoin_usdtfuture.NewOrderParam) bool {
	spread, ok1 := mtSpreads[order.Symbol]
	quantile, ok2 := mtQuantiles[order.Symbol]
	if !ok1 || !ok2 || time.Now().Sub(spread.Time) > *mtConfig.SpreadTimeToLive {
		if !ok1 {
			logger.Debugf("%s SPREAD IS NOT READY, CANCEL", order.Symbol)
		} else if !ok2 {
			logger.Debugf("%s QUANTILE IS NOT READY, CANCEL", order.Symbol)
		} else {
			logger.Debugf("%s SPREAD IS OUT OF DATE %v, CANCEL", order.Symbol, time.Now().Sub(spread.Time))
		}
		return false
	}

	if order.Side == kucoin_usdtfuture.OrderSideBuy &&
		float64(order.Price) < spread.MakerDepth.BestBidPrice {
		logger.Debugf("BUY %s %f < BEST BID %f",
			order.Symbol,
			order.Price,
			spread.MakerDepth.BestBidPrice,
		)
		return false
	} else if order.Side == kucoin_usdtfuture.OrderSideSell &&
		float64(order.Price) > spread.MakerDepth.BestAskPrice {
		logger.Debugf("SELL %s %f > BEST ASK %f",
			order.Symbol,
			order.Price,
			spread.MakerDepth.BestAskPrice,
		)
		return false
	}

	if order.Side == kucoin_usdtfuture.OrderSideBuy &&
		!order.ReduceOnly &&
		(spread.TakerDepth.TakerBid-float64(order.Price))/float64(order.Price) > quantile.ShortTop-*mtConfig.MakerOrderOffset {
		//买入开多, 是开空价差, 参考ShortTop
		return true
	} else if order.Side == kucoin_usdtfuture.OrderSideSell &&
		order.ReduceOnly &&
		(spread.TakerDepth.TakerAsk-float64(order.Price))/float64(order.Price) < quantile.ShortBot+*mtConfig.MakerOrderOffset {
		//卖出平多, 是平空价, 参考ShortBot
		return true
	} else if order.Side == kucoin_usdtfuture.OrderSideSell &&
		!order.ReduceOnly &&
		(spread.TakerDepth.TakerAsk-float64(order.Price))/float64(order.Price) < quantile.LongBot+*mtConfig.MakerOrderOffset {
		//卖出开空, 是开多价差, 参考LongBot
		return true
	} else if order.Side == kucoin_usdtfuture.OrderSideBuy &&
		order.ReduceOnly &&
		(spread.TakerDepth.TakerBid-float64(order.Price))/float64(order.Price) > quantile.LongTop-*mtConfig.MakerOrderOffset {
		//买入平空, 是平多价差, 参考LongTop
		return true
	}
	if order.Side == kucoin_usdtfuture.OrderSideBuy {
		logger.Debugf(
			"NOT PROFITABLE %s SPREAD %f BUY ORDER, CANCEL", (spread.TakerDepth.TakerBid-float64(order.Price))/float64(order.Price), order.Symbol,
		)
	} else {
		logger.Debugf(
			"NOT PROFITABLE %s SPREAD %f SELL ORDER, CANCEL", (spread.TakerDepth.TakerAsk-float64(order.Price))/float64(order.Price), order.Symbol,
		)
	}
	return false
}
