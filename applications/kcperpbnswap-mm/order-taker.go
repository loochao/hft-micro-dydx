package main

import (
	"context"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func watchTakerOrderRequest(
	ctx context.Context,
	api *bnswap.API,
	timeout time.Duration,
	dryRun bool,
	orderRequestCh chan TakerOrderRequest,
	outputOrderRespCh chan TakerOpenOrder,
	outputOrderErrorCh chan TakerOrderNewError,
) {
	defer func() {
		logger.Debugf("EXIT watchTakerOrderRequest")
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
				//logger.Debugf("TAKER SUBMIT %v", request.New.ToString())
				_, err := api.SubmitOrder(childCtx, *request.New)
				if err != nil {
					logger.Debugf("TAKER SUBMIT ERROR %v %v", err, *request.New)
					outputOrderErrorCh <- TakerOrderNewError{
						Error:  err,
						Params: *request.New,
					}
				} else {
					outputOrderRespCh <- TakerOpenOrder{
						NewOrderParams: request.New,
						Symbol:         request.New.Symbol,
					}
				}
			} else if request.Cancel != nil {
				childCtx, _ := context.WithTimeout(ctx, timeout)
				//logger.Debugf("TAKER CANCEL %v", *request.Cancel)
				_, err := api.CancelAllOpenOrders(childCtx, *request.Cancel)
				if err != nil {
					logger.Debugf("TAKER CANCEL ERROR %v %v", err, *request.Cancel)
				} else {
					outputOrderRespCh <- TakerOpenOrder{
						NewOrderParams: nil,
						Symbol:         request.Cancel.Symbol,
					}
				}
			}
		}
	}
}

func updateTakerOldOrders() {
	for takerSymbol, order := range tOpenOrders {
		if tOrderCancelCounts[takerSymbol] > *mtConfig.OrderMaxCancelCount {
			delete(tOpenOrders, takerSymbol)
			tOrderCancelCounts[order.Symbol] = 0
			continue
		}
		if time.Now().Sub(tOrderCancelSilentTimes[takerSymbol]) < 0 {
			continue
		}
		if isTakerOrderOk(*order.NewOrderParams) && time.Now().Sub(mtLimitHedgeTimeouts[takerSymbol]) < 0{
			continue
		}
		tOrderSilentTimes[order.Symbol] = time.Now().Add(*mtConfig.OrderCancelSilent)
		tOrderCancelSilentTimes[order.Symbol] = time.Now().Add(*mtConfig.OrderCancelSilent)
		tOrderCancelCounts[order.Symbol] += 1
		tOrderRequestChs[order.Symbol] <- TakerOrderRequest{
			Cancel: &bnswap.CancelAllOrderParams{Symbol: order.Symbol},
		}
	}
}

func isTakerOrderOk(order bnswap.NewOrderParams) bool {
	spread, ok := mtSpreads[tmSymbolsMap[order.Symbol]]
	if !ok || time.Now().Sub(spread.Time) > *mtConfig.SpreadTimeToLive {
		if !ok {
			logger.Debugf("SPREAD IS NOT READY")
		}else{
			logger.Debugf("SPREAD IS OUT OF DATE %v, CANCEL %s", time.Now().Sub(spread.Time), order.Symbol)
		}
		return false
	}
	//检查价格有没有挂太远，太远撤掉
	if order.Side == common.OrderSideBuy &&
		order.Price < spread.TakerDepth.BestBidPrice {
		logger.Debugf("TAKER BUY %s %f < BEST BID %f",
			order.Symbol,
			order.Price,
			spread.TakerDepth.BestBidPrice,
		)
		return false
	} else if order.Side == common.OrderSideSell &&
		order.Price > spread.TakerDepth.TakerFarAsk {
		logger.Debugf("TAKER SELL %s %f > BEST ASK %f",
			order.Symbol,
			order.Price,
			spread.TakerDepth.TakerFarBid,
		)
		return false
	}
	return true
}
