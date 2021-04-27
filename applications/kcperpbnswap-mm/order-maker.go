package main

import (
	"context"
	"github.com/geometrybase/hft-micro/kcperp"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func watchMakerOrderRequest(
	ctx context.Context,
	api *kcperp.API,
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
				//logger.Debugf("MAKER SUBMIT %s %s %f %d", request.New.Symbol, request.New.Side, request.New.Price, request.New.Size)
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
				//logger.Debugf("MAKER CANCEL ALL %s", request.Cancel.Symbol)
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
			Cancel: &kcperp.CancelAllOrdersParam{Symbol: order.Symbol},
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
			Cancel: &kcperp.CancelAllOrdersParam{Symbol: order.Symbol},
		}
	}
}

func isOrderProfitable(order kcperp.NewOrderParam) bool {
	spread, ok1 := mtSpreads[order.Symbol]
	if !ok1 || time.Now().Sub(spread.Time) > *mtConfig.SpreadTimeToLive {
		if !ok1 {
			logger.Debugf("SPREAD IS NOT READY")
		} else {
			logger.Debugf("SPREAD IS OUT OF DATE %v, CANCEL %s", time.Now().Sub(spread.Time), order.Symbol)
		}
		return false
	}

	if order.Side == kcperp.OrderSideBuy &&
		float64(order.Price) < spread.MakerDepth.BestBidPrice {
		logger.Debugf("BUY %s %f < BEST BID %f",
			order.Symbol,
			order.Price,
			spread.MakerDepth.BestBidPrice,
		)
		return false
	} else if order.Side == kcperp.OrderSideSell &&
		float64(order.Price) > spread.MakerDepth.BestAskPrice {
		logger.Debugf("SELL %s %f > BEST ASK %f",
			order.Symbol,
			order.Price,
			spread.MakerDepth.BestAskPrice,
		)
		return false
	}

	return true

	////检查价格有没有挂太远，太远撤掉
	//if order.Side == kcperp.OrderSideBuy &&
	//	float64(order.Price) < (1.0-4**mtConfig.MakerOrderOffset)*spread.MakerDepth.TakerFarBid {
	//	logger.Debugf("%s BUY PRICE %f < MAKER BID MINIMAL PRICE %f",
	//		order.Symbol,
	//		order.Price,
	//		(1.0-2**mtConfig.MakerOrderOffset)*spread.MakerDepth.TakerFarBid,
	//	)
	//	return false
	//} else if order.Side == kcperp.OrderSideSell &&
	//	float64(order.Price) > (1.0+4**mtConfig.MakerOrderOffset)*spread.MakerDepth.TakerFarAsk {
	//	logger.Debugf("%s SELL PRICE %f > MAKER ASK MAXIMAL PRICE %f",
	//		order.Symbol,
	//		order.Price,
	//		(1.0+2**mtConfig.MakerOrderOffset)*spread.MakerDepth.TakerFarAsk,
	//	)
	//	return false
	//}
	//
	//if order.Side == kcperp.OrderSideBuy &&
	//	!order.ReduceOnly &&
	//	(spread.TakerDepth.TakerBid-float64(order.Price))/float64(order.Price) > quantile.ShortTop-*mtConfig.MakerOrderOffset {
	//	//买入开多, 是开空价差, 参考ShortTop
	//	return true
	//} else if order.Side == kcperp.OrderSideSell &&
	//	order.ReduceOnly &&
	//	(spread.TakerDepth.TakerAsk-float64(order.Price))/float64(order.Price) < quantile.ShortBot+*mtConfig.MakerOrderOffset {
	//	//卖出平多, 是平空价, 参考ShortBot
	//	return true
	//} else if order.Side == kcperp.OrderSideSell &&
	//	!order.ReduceOnly &&
	//	(spread.TakerDepth.TakerAsk-float64(order.Price))/float64(order.Price) < quantile.LongBot+*mtConfig.MakerOrderOffset {
	//	//卖出开空, 是开多价差, 参考LongBot
	//	return true
	//} else if order.Side == kcperp.OrderSideBuy &&
	//	order.ReduceOnly &&
	//	(spread.TakerDepth.TakerBid-float64(order.Price))/float64(order.Price) > quantile.LongTop-*mtConfig.MakerOrderOffset {
	//	//买入平空, 是平多价差, 参考LongTop
	//	return true
	//}
	//if order.Side == kcperp.OrderSideBuy {
	//	logger.Debugf(
	//		"NOT PROFITABLE %s BUY ORDER, CANCEL", order.Symbol,
	//	)
	//} else {
	//	logger.Debugf(
	//		"NOT PROFITABLE %s SELL ORDER, CANCEL", order.Symbol,
	//	)
	//}
	//return false
}
