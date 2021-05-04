package main

import (
	"context"
	"github.com/geometrybase/hft-micro/bnspot"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func watchSpotOrderRequest(
	ctx context.Context,
	api *bnspot.API,
	timeout time.Duration,
	symbol string,
	dryRun bool,
	orderRequestCh chan SpotOrderRequest,
	outputOrderErrorCh chan MakerOrderNewError,
	outputNewOrderResponseCh chan bnspot.NewOrderResponse,
	cancelAllOrderResponsesCh chan []bnspot.CancelOrderResponse,
) {
	logger.Debugf("START watchSpotOrderRequest %s", symbol)
	defer logger.Debugf("EXIT watchSpotOrderRequest %s", symbol)
	for {
		select {
		case <-ctx.Done():
			return
		case request := <-orderRequestCh:
			if dryRun {
				break
			}
			if request.Cancel != nil {
				childCtx, _ := context.WithTimeout(ctx, timeout)
				orders, _, err := api.CancelAllOrder(childCtx, *request.Cancel)
				if err != nil {
					logger.Debugf("api.CancelAllOrder(childCtx, *request.Cancel) error %v", err)
				} else {
					select {
					case cancelAllOrderResponsesCh <- orders:
					default:
						logger.Debugf("cancelAllOrderResponsesCh <- orders failed, ch len %d", len(cancelAllOrderResponsesCh))
					}
				}
			} else if request.New != nil {
				childCtx, _ := context.WithTimeout(ctx, timeout)
				order, _, err := api.SubmitOrder(childCtx, *request.New)
				if err != nil {
					logger.Debugf("api.SubmitOrder(childCtx, *request.New) error %v", err)
					select {
					case outputOrderErrorCh <- MakerOrderNewError{
						Error:  err,
						Params: *request.New,
					}:
					default:
						logger.Debugf("outputOrderErrorCh <- MakerOrderNewError failed, ch len %d", len(outputOrderErrorCh))
					}
				} else {
					select {
					case outputNewOrderResponseCh <- *order:
					default:
						logger.Debugf("outputNewOrderResponseCh <- *order failed, ch len %d", len(outputNewOrderResponseCh))
					}
				}
			}
		}
	}
}

func updateMakerOldOrders() {
	for symbol, order := range bnspotOpenOrders {
		//if bnspotOrderCancelCounts[symbol] > *bnConfig.OrderMaxCancelCount {
		//	delete(bnspotOpenOrders, symbol)
		//	continue
		//}
		if time.Now().Sub(bnspotCancelSilentTimes[symbol]) < 0 {
			continue
		}
		if isOrderOK(order) {
			continue
		}
		bnspotCancelSilentTimes[order.Symbol] = time.Now().Add(*bnConfig.OrderCancelSilent)
		//bnspotOrderCancelCounts[order.Symbol] += 1
		bnspotOrderSilentTimes[order.Symbol] = time.Now().Add(*bnConfig.OrderCancelSilent)
		delete(bnspotOpenOrders, symbol)
		bnspotOrderRequestChs[order.Symbol] <- SpotOrderRequest{
			Cancel: &bnspot.CancelAllOrderParams{Symbol: order.Symbol},
		}
	}
}

func isOrderOK(order bnspot.NewOrderParams) bool {
	spread, ok1 := bnSpreads[order.Symbol]
	quantile, ok2 := bnQuantiles[order.Symbol]
	offset := bnspotOffsets[order.Symbol]
	if !ok1 {
		logger.Debugf("%s spread is not ready", order.Symbol)
		return false
	}
	if !ok2 {
		logger.Debugf("%s quantile is not ready", order.Symbol)
	}
	if time.Now().Sub(spread.Time) > *bnConfig.SpreadTimeToLive {
		logger.Debugf("%s spread is out of date %v > %v", order.Symbol, time.Now().Sub(spread.Time), *bnConfig.SpreadTimeToLive)
		return false
	}
	if order.Side == bnspot.OrderSideBuy &&
		order.Price < spread.MakerDepth.MakerBid*(1.0+offset.FarBot) {
		logger.Debugf("%s buy price %f < far bot %f",
			order.Symbol,
			order.Price,
			spread.MakerDepth.MakerBid*(1.0+offset.FarBot),
		)
		return false
	} else if order.Side == bnspot.OrderSideBuy &&
		order.Price > spread.MakerDepth.MakerBid*(1.0+offset.NearBot) {
		logger.Debugf("%s buy price %f > near bot %f",
			order.Symbol,
			order.Price,
			spread.MakerDepth.MakerBid*(1.0+offset.NearBot),
		)
		return false
	} else if order.Side == bnspot.OrderSideSell &&
		order.Price > spread.MakerDepth.MakerAsk*(1.0+offset.FarTop) {
		logger.Debugf("%s sell price %f > far top %f",
			order.Symbol,
			order.Price,
			spread.MakerDepth.MakerAsk*(1.0+offset.FarTop),
		)
		return false
	} else if order.Side == bnspot.OrderSideSell &&
		order.Price < spread.MakerDepth.MakerAsk*(1.0+offset.NearTop) {
		logger.Debugf("%s sell price %f < near top %f",
			order.Symbol,
			order.Price,
			spread.MakerDepth.MakerAsk*(1.0+offset.NearTop),
		)
		return false
	}

	if order.Side == bnspot.OrderSideBuy &&
		(spread.TakerDepth.TakerBid-order.Price)/order.Price > quantile.Top {
		return true
	} else if order.Side == bnspot.OrderSideSell &&
		(spread.TakerDepth.TakerAsk-order.Price)/order.Price < quantile.Bot{
		return true
	}
	if order.Side == bnspot.OrderSideBuy {
		logger.Debugf(
			"NOT PROFITABLE %s BUY ORDER SWAP BID %f ORDER PRICE %f DELTA %f < TOP %f",
			order.Symbol,
			spread.TakerDepth.TakerBid,
			order.Price,
			(spread.TakerDepth.TakerBid-order.Price)/order.Price,
			quantile.Top,
		)
	} else {
		logger.Debugf(
			"NOT PROFITABLE %s BUY ORDER SWAP ASK %f ORDER PRICE %f DELTA %f > BOT %f",
			order.Symbol,
			spread.TakerDepth.TakerAsk,
			order.Price,
			(spread.TakerDepth.TakerAsk-order.Price)/order.Price,
			quantile.Bot,
		)
	}
	return false
}
