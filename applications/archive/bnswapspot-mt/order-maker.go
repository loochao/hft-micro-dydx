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
	dryRun bool,
	orderRequestCh chan SpotOrderRequest,
	outputOrderErrorCh chan MakerOrderNewError,
	outputNewOrderResponseCh chan bnspot.NewOrderResponse,
	cancelAllOrderResponsesCh chan []bnspot.CancelOrderResponse,
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
				orders, _, err := api.CancelAllOrder(childCtx, *request.Cancel)
				if err != nil {
					logger.Debugf("SPOT CANCEL ALL %v", err)
				} else {
					cancelAllOrderResponsesCh <- orders
				}
			} else if request.New != nil {
				childCtx, _ := context.WithTimeout(ctx, timeout)
				logger.Debugf("SPOT SUBMIT %s", request.New.ToUrlValues().Encode())
				order, _, err := api.SubmitOrder(childCtx, *request.New)
				if err != nil {
					logger.Debugf("SPOT SUBMIT ERROR %v", err)
					outputOrderErrorCh <- MakerOrderNewError{
						Error:  err,
						Params: *request.New,
					}
				} else if order.Status == bnspot.OrderStatusFilled ||
					order.Status == bnspot.OrderStatusCancelled ||
					order.Status == bnspot.OrderStatusReject ||
					order.Status == bnspot.OrderStatusExpired {
					outputNewOrderResponseCh <- *order
				}
			}
		}
	}
}

func updateMakerOldOrders() {
	for symbol, order := range bnspotOpenOrders {
		if bnspotOrderCancelCounts[symbol] > *bnConfig.OrderMaxCancelCount {
			delete(bnspotOpenOrders, symbol)
			continue
		}
		if time.Now().Sub(bnspotCancelSilentTimes[symbol]) < 0 {
			continue
		}
		if isOrderProfitable(order) {
			continue
		}
		bnspotOrderSilentTimes[order.Symbol] = time.Now().Add(*bnConfig.OrderSilent)
		bnspotCancelSilentTimes[order.Symbol] = time.Now().Add(*bnConfig.OrderCancelSilent)
		bnspotOrderCancelCounts[order.Symbol] += 1
		bnspotOrderRequestChs[order.Symbol] <- SpotOrderRequest{
			Cancel: &bnspot.CancelAllOrderParams{Symbol: order.Symbol},
		}
	}
}

func isOrderProfitable(order bnspot.NewOrderParams) bool {
	spread, ok1 := bnSpreads[order.Symbol]
	quantile, ok2 := bnQuantiles[order.Symbol]
	if !ok1 || !ok2 || time.Now().Sub(spread.SwapOrderBook.ArrivalTime) > *bnConfig.SpreadTimeToLive {
		logger.Debugf("SPREAD IS OUT OF DATE %v, CANCEL %s", time.Now().Sub(spread.SwapOrderBook.ArrivalTime), order.Symbol)
		return false
	}
	if order.Side == bnspot.OrderSideBuy &&
		order.Price < (1.0-2**bnConfig.MakerBandOffset)*spread.SpotOrderBook.BidPrice-bnspotTickSizes[order.Symbol] {
		logger.Debugf("%s BUY PRICE %f < MAKER BAND OFFSET BID PRICE %f",
			order.Symbol,
			order.Price,
			(1.0-2**bnConfig.MakerBandOffset)*spread.SpotOrderBook.BidPrice-bnspotTickSizes[order.Symbol],
		)
		return false
	} else if order.Side == bnspot.OrderSideSell &&
		order.Price > (1.0+2**bnConfig.MakerBandOffset)*spread.SpotOrderBook.AskPrice+bnspotTickSizes[order.Symbol] {
		logger.Debugf("%s SELL PRICE %f > MAKER BAND OFFSEF ASK PRICE %f",
			order.Symbol,
			order.Price,
			(1.0+2**bnConfig.MakerBandOffset)*spread.SpotOrderBook.AskPrice+bnspotTickSizes[order.Symbol],
		)
		return false
	}

	if order.Side == bnspot.OrderSideBuy &&
		(spread.SwapOrderBook.TakerBidVWAP-order.Price)/order.Price > quantile.Top-*bnConfig.MakerBandOffset {
		return true
	} else if order.Side == bnspot.OrderSideSell &&
		(spread.SwapOrderBook.TakerAskVWAP-order.Price)/order.Price < quantile.Bot+*bnConfig.MakerBandOffset {
		return true
	}
	if order.Side == bnspot.OrderSideBuy {
		logger.Debugf(
			"NOT PROFITABLE %s BUY ORDER SWAP BIDVWAP %f ORDER PRICE %f DELTA %f < TOP %f - %f",
			order.Symbol,
			spread.SwapOrderBook.TakerBidVWAP,
			order.Price,
			(spread.SwapOrderBook.TakerBidVWAP-order.Price)/order.Price,
			quantile.Top,
			*bnConfig.MakerBandOffset,
		)
	} else {
		logger.Debugf(
			"NOT PROFITABLE %s BUY ORDER SWAP ASKVWAP %f ORDER PRICE %f DELTA %f > BOT %f + %f",
			order.Symbol,
			spread.SwapOrderBook.TakerAskVWAP,
			order.Price,
			(spread.SwapOrderBook.TakerAskVWAP-order.Price)/order.Price,
			quantile.Bot,
			*bnConfig.MakerBandOffset,
		)
	}
	return false
}
