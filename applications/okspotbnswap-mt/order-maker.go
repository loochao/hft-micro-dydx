package main

import (
	"context"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/geometrybase/hft-micro/okspot"
	"time"
)

func watchMakerOrderRequest(
	ctx context.Context,
	api *okspot.API,
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
				//logger.Debugf("MAKER SUBMIT %s %s %f %f", request.New.Symbol, request.New.Type, *(request.New.Price), *(request.New.Size))
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
				//logger.Debugf("MAKER CANCEL %s %s", request.Cancel.Symbol, request.Cancel.ClientOid)
				resp, err := api.CancelOrders(childCtx, *request.Cancel)
				if err != nil {
					logger.Debugf("MAKER SUBMIT ERROR %v", err)
				} else {
					outputOrderRespCh <- MakerOpenOrder{
						NewOrderParam:   nil,
						ResponseOrderID: resp.OrderId,
						Symbol:          request.Cancel.Symbol,
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
		mOrderSilentTimes[order.Symbol] = time.Now().Add(*mtConfig.OrderSilent)
		mOrderCancelSilentTimes[order.Symbol] = time.Now().Add(*mtConfig.OrderCancelSilent)
		mOrderCancelCounts[order.Symbol] += 1
		mOrderRequestChs[order.Symbol] <- MakerOrderRequest{
			Cancel: &okspot.CancelOrderParam{
				Symbol:    order.Symbol,
				ClientOid: order.ClientOID,
			},
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
		mOrderSilentTimes[order.Symbol] = time.Now().Add(*mtConfig.OrderSilent)
		mOrderCancelSilentTimes[order.Symbol] = time.Now().Add(*mtConfig.OrderCancelSilent)
		mOrderCancelCounts[order.Symbol] += 1
		mOrderRequestChs[order.Symbol] <- MakerOrderRequest{
			Cancel: &okspot.CancelOrderParam{
				Symbol:    order.Symbol,
				ClientOid: order.ClientOID,
			},
		}
	}
}

func isOrderProfitable(order okspot.NewOrderParam) bool {
	spread, ok1 := mtSpreads[order.Symbol]
	quantile, ok2 := mtQuantiles[order.Symbol]
	if !ok1 || !ok2 || time.Now().Sub(spread.Time) > *mtConfig.SpreadTimeToLive {
		logger.Debugf("SPREAD IS OUT OF DATE %v, CANCEL %s", time.Now().Sub(spread.Time), order.Symbol)
		return false
	}

	//检查价格有没有挂太远，太远撤掉
	if order.Side == okspot.OrderSideBuy &&
		*order.Price < (1.0-4**mtConfig.MakerOrderOffset)*spread.MakerDepth.TakerFarBid {
		logger.Debugf("%s BUY PRICE %f < MAKER BID MINIMAL PRICE %f",
			order.Symbol,
			*order.Price,
			(1.0-2**mtConfig.MakerOrderOffset)*spread.MakerDepth.TakerFarBid,
		)
		return false
	} else if order.Side == okspot.OrderSideSell &&
		*order.Price > (1.0+4**mtConfig.MakerOrderOffset)*spread.MakerDepth.TakerFarAsk {
		logger.Debugf("%s SELL PRICE %f > MAKER ASK MAXIMAL PRICE %f",
			order.Symbol,
			*order.Price,
			(1.0+2**mtConfig.MakerOrderOffset)*spread.MakerDepth.TakerFarAsk,
		)
		return false
	}

	if order.Side == okspot.OrderSideBuy &&
		(spread.TakerDepth.TakerBid-*order.Price) / *order.Price > quantile.ShortTop-*mtConfig.MakerOrderOffset {
		//买入开多, 是开空价差, 参考ShortTop
		return true
	} else if order.Side == okspot.OrderSideSell &&
		(spread.TakerDepth.TakerAsk-*order.Price) / *order.Price < quantile.ShortBot+*mtConfig.MakerOrderOffset {
		//卖出平多, 是平空价, 参考ShortBot
		return true
	}
	if order.Side == okspot.OrderSideBuy {
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
