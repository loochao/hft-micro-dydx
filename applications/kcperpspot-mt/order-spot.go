package main

import (
	"context"
	"github.com/geometrybase/hft-micro/kcspot"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func watchSpotOrderRequest(
	ctx context.Context,
	api *kcspot.API,
	timeout time.Duration,
	dryRun bool,
	orderRequestCh chan SpotOrderRequest,
	outputOrderErrorCh chan SpotOrderNewError,
) {
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
				_, err := api.CancelAllOrders(childCtx, *request.Cancel)
				if err != nil {
					logger.Debugf("SPOT CANCEL ALL %v", err)
				}
			} else if request.New != nil {
				childCtx, _ := context.WithTimeout(ctx, timeout)
				logger.Debugf("SPOT SUBMIT %v", request.New)
				_,  err := api.SubmitOrder(childCtx, *request.New)
				if err != nil {
					logger.Debugf("SPOT SUBMIT ERROR %v", err)
					outputOrderErrorCh <- SpotOrderNewError{
						Error:  err,
						Params: *request.New,
					}
				//} else if order.Status == kcspot.OrderStatusFilled ||
				//	order.Status == kcspot.OrderStatusCancelled ||
				//	order.Status == kcspot.OrderStatusReject ||
				//	order.Status == kcspot.OrderStatusExpired {
				//	outputNewOrderResponseCh <- *order
				}
			}
		}
	}
}

func updateSpotOldOrders() {
	for symbol, order := range kcspotOpenOrders {
		if kcspotOrderCancelCounts[symbol] > *kcConfig.OrderMaxCancelCount {
			delete(kcspotOpenOrders, symbol)
			continue
		}
		if time.Now().Sub(kcspotCancelSilentTimes[symbol]) < 0 {
			continue
		}
		if isOrderProfitable(order) {
			continue
		}
		kcspotOrderSilentTimes[order.Symbol] = time.Now().Add(*kcConfig.OrderSilent)
		kcspotCancelSilentTimes[order.Symbol] = time.Now().Add(*kcConfig.OrderCancelSilent)
		kcspotOrderCancelCounts[order.Symbol] += 1
		kcspotOrderRequestChs[order.Symbol] <- SpotOrderRequest{
			Cancel: &kcspot.CancelAllOrdersParam{Symbol: order.Symbol},
		}
	}
}

func isOrderProfitable(order kcspot.NewOrderParam) bool {
	spread, ok1 := kcSpreads[order.Symbol]
	quantile, ok2 := kcQuantiles[order.Symbol]
	if !ok1 || !ok2 || time.Now().Sub(spread.PerpOrderBook.ParseTime) > *kcConfig.SpreadTimeToLive {
		logger.Debugf("SPREAD IS OUT OF DATE %v, CANCEL %s", time.Now().Sub(spread.PerpOrderBook.ParseTime), order.Symbol)
		return false
	}
	if order.Side == kcspot.OrderSideBuy &&
		order.Price < (1.0-2**kcConfig.MakerBandOffset)*spread.SpotOrderBook.BidPrice-kcspotTickSizes[order.Symbol] {
		logger.Debugf("%s BUY PRICE %f < MAKER BAND OFFSET BID PRICE %f",
			order.Symbol,
			order.Price,
			(1.0-2**kcConfig.MakerBandOffset)*spread.SpotOrderBook.BidPrice-kcspotTickSizes[order.Symbol],
		)
		return false
	} else if order.Side == kcspot.OrderSideSell &&
		order.Price > (1.0+2**kcConfig.MakerBandOffset)*spread.SpotOrderBook.AskPrice+kcspotTickSizes[order.Symbol] {
		logger.Debugf("%s SELL PRICE %f > MAKER BAND OFFSEF ASK PRICE %f",
			order.Symbol,
			order.Price,
			(1.0+2**kcConfig.MakerBandOffset)*spread.SpotOrderBook.AskPrice+kcspotTickSizes[order.Symbol],
		)
		return false
	}

	if order.Side == kcspot.OrderSideBuy &&
		(spread.PerpOrderBook.TakerBidVWAP-order.Price)/order.Price > quantile.Top-*kcConfig.MakerBandOffset {
		return true
	} else if order.Side == kcspot.OrderSideSell &&
		(spread.PerpOrderBook.TakerAskVWAP-order.Price)/order.Price < quantile.Bot+*kcConfig.MakerBandOffset {
		return true
	}
	if order.Side == kcspot.OrderSideBuy {
		logger.Debugf(
			"NOT PROFITABLE %s BUY ORDER PERP BIDVWAP %f ORDER PRICE %f DELTA %f < TOP %f - %f",
			order.Symbol,
			spread.PerpOrderBook.TakerBidVWAP,
			order.Price,
			(spread.PerpOrderBook.TakerBidVWAP-order.Price)/order.Price,
			quantile.Top,
			*kcConfig.MakerBandOffset,
		)
	} else {
		logger.Debugf(
			"NOT PROFITABLE %s BUY ORDER PERP ASKVWAP %f ORDER PRICE %f DELTA %f > BOT %f + %f",
			order.Symbol,
			spread.PerpOrderBook.TakerAskVWAP,
			order.Price,
			(spread.PerpOrderBook.TakerAskVWAP-order.Price)/order.Price,
			quantile.Bot,
			*kcConfig.MakerBandOffset,
		)
	}
	return false
}
