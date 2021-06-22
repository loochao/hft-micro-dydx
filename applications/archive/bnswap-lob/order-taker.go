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
				//logger.Debugf("SUBMIT %s %s %f %f", request.New.Market, request.New.NewClientOrderId, request.New.Price, request.New.Quantity)
				_, err := api.SubmitOrder(childCtx, *request.New)
				if err != nil {
					logger.Debugf("SUBMIT ERROR %s %s %v", request.New.Symbol, request.New.NewClientOrderId, err)
					outputOrderErrorCh <- TakerOrderNewError{
						Error:  err,
						Params: *request.New,
					}
				}
			} else if request.Cancel != nil {
				childCtx, _ := context.WithTimeout(ctx, timeout)
				_, err := api.CancelAllOpenOrders(childCtx, *request.Cancel)
				if err != nil {
					logger.Debugf("CANCEL ERROR %s %v", request.Cancel.Symbol, err)
				}
			}
		}
	}
}

func updateTakerOldOrders() {
	for takerSymbol, order := range swapOpenOrders {
		if swapOrderCancelCounts[takerSymbol] > *mtConfig.OrderMaxCancelCount {
			delete(swapOpenOrders, takerSymbol)
			swapOrderCancelCounts[order.Symbol] = 0
			continue
		}
		//非挂单不用管
		if order.Type != common.OrderTypeLimit {
			continue
		}
		if time.Now().Sub(swapOrderCancelSilentTimes[takerSymbol]) < 0 {
			continue
		}
		if isTakerOrderOk(*order.NewOrderParams) {
			continue
		}
		tOrderSilentTimes[order.Symbol] = time.Now().Add(*mtConfig.OrderCancelSilent)
		swapOrderCancelSilentTimes[order.Symbol] = time.Now().Add(*mtConfig.OrderCancelSilent)
		swapOrderCancelCounts[order.Symbol] += 1
		swapOrderRequestChs[order.Symbol] <- TakerOrderRequest{
			Cancel: &bnswap.CancelAllOrderParams{Symbol: order.Symbol},
		}
	}
}

func isTakerOrderOk(order bnswap.NewOrderParams) bool {
	takerDepth, ok := swapWalkedDepths[order.Symbol]
	if !ok || time.Now().Sub(takerDepth.Time) > *mtConfig.DepthTimeToLive {
		if !ok {
			logger.Debugf("SPREAD IS NOT READY")
		} else {
			logger.Debugf("SPREAD IS OUT OF DATE %v, CANCEL %s", time.Now().Sub(takerDepth.Time), order.Symbol)
		}
		return false
	}

	if order.Side == common.OrderSideSell &&
		order.Price > takerDepth.AskPrice {
		return false
	} else if order.Side == common.OrderSideBuy &&
		order.Price < takerDepth.BidPrice {
		return false
	}
	return true
}
