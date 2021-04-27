package main

import (
	"context"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"math"
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
				logger.Debugf("SUBMIT %s %s %f %f", request.New.Symbol, request.New.NewClientOrderId, request.New.Price, request.New.Quantity)
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
	for takerSymbol, order := range tOpenOrders {
		if tOrderCancelCounts[takerSymbol] > *mtConfig.OrderMaxCancelCount {
			delete(tOpenOrders, takerSymbol)
			tOrderCancelCounts[order.Symbol] = 0
			continue
		}
		//非挂单不用管
		if order.Type != common.OrderTypeLimit {
			continue
		}
		if time.Now().Sub(tOrderCancelSilentTimes[takerSymbol]) < 0 {
			continue
		}
		if isTakerOrderOk(*order.NewOrderParams) && time.Now().Sub(tCloseTimeouts[takerSymbol]) < 0 {
			continue
		}
		logger.Debugf("CANCEL %s", order.Symbol)
		tOrderSilentTimes[order.Symbol] = time.Now().Add(*mtConfig.OrderCancelSilent)
		tOrderCancelSilentTimes[order.Symbol] = time.Now().Add(*mtConfig.OrderCancelSilent)
		tOrderCancelCounts[order.Symbol] += 1
		tOrderRequestChs[order.Symbol] <- TakerOrderRequest{
			Cancel: &bnswap.CancelAllOrderParams{Symbol: order.Symbol},
		}
	}
}

func isTakerOrderOk(order bnswap.NewOrderParams) bool {
	spread, ok1 := mtSpreads[order.Symbol]
	takerPosition, ok2 := tPositions[order.Symbol]
	if !ok1 || !ok2 || time.Now().Sub(spread.Time) > *mtConfig.SpreadTimeToLive {
		if !ok1 || !ok2  {
			logger.Debugf("SPREAD OR POSITION IS NOT READY")
		} else {
			logger.Debugf("SPREAD IS OUT OF DATE %v, CANCEL %s", time.Now().Sub(spread.Time), order.Symbol)
		}
		return false
	}

	if tEnterTimeouts[order.Symbol].Sub(time.Now()) > 0 {
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
	} else {
		if order.Side == common.OrderSideSell {
			if tCloseTimeouts[order.Symbol].Sub(time.Now()) > 0 {
				takerPrice := (1.0 + float64(tCloseTimeouts[order.Symbol].Sub(time.Now()))/float64(*mtConfig.CloseTimeout)**mtConfig.CloseProfitPct) * takerPosition.EntryPrice
				takerPrice = math.Ceil(takerPrice/tTickSizes[order.Symbol]) * tTickSizes[order.Symbol]
				if order.Price > takerPrice*(1.0 + *mtConfig.CloseUpdateStep) {
					logger.Debugf("TAKER BUY %s %f > TARGET SELL PRICE %f",
						order.Symbol,
						order.Price,
						spread.TakerDepth.TakerFarBid,
					)
					return false
				}
			}else{
				return false
			}
		} else if order.Side == common.OrderSideBuy	 {
			if tCloseTimeouts[order.Symbol].Sub(time.Now()) > 0 {
				takerPrice := (1.0 - float64(tCloseTimeouts[order.Symbol].Sub(time.Now()))/float64(*mtConfig.CloseTimeout)**mtConfig.CloseProfitPct) * takerPosition.EntryPrice
				takerPrice = math.Floor(takerPrice/tTickSizes[order.Symbol]) * tTickSizes[order.Symbol]
				if order.Price < takerPrice*(1.0 - *mtConfig.CloseUpdateStep) {
					logger.Debugf("TAKER BUY %s %f < TARGET BUY PRICE %f",
						order.Symbol,
						order.Price,
						spread.TakerDepth.TakerFarBid,
					)
					return false
				}
			}else {
				return false
			}
		}
	}
	return true
}
