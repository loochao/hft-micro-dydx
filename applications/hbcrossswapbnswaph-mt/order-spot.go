package main

import (
	"context"
	"github.com/geometrybase/hft-micro/hbspot"
	"github.com/geometrybase/hft-micro/logger"
	"strings"
	"time"
)

func watchSpotOrderRequest(
	ctx context.Context,
	api *hbspot.API,
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
				logger.Debugf("SPOT SUBMIT %s %s %s %s", request.New.Symbol, request.New.Type, request.New.Price, request.New.Amount)
				_, err := api.SubmitOrder(childCtx, *request.New)
				if err != nil {
					logger.Debugf("SPOT SUBMIT ERROR %v", err)
					outputOrderErrorCh <- SpotOrderNewError{
						Error:  err,
						Params: *request.New,
					}
				}
			}
		}
	}
}

func updateSpotOldOrders() {
	for symbol, order := range hbspotOpenOrders {
		if hOrderCancelCounts[symbol] > *hbConfig.OrderMaxCancelCount {
			delete(hbspotOpenOrders, symbol)
			continue
		}
		if time.Now().Sub(hbspotCancelSilentTimes[symbol]) < 0 {
			continue
		}
		if isOrderProfitable(order) {
			continue
		}
		bOrderSilentTimes[order.Symbol] = time.Now().Add(*hbConfig.OrderSilent)
		hbspotCancelSilentTimes[order.Symbol] = time.Now().Add(*hbConfig.OrderCancelSilent)
		hOrderCancelCounts[order.Symbol] += 1
		hbspotOrderRequestChs[order.Symbol] <- SpotOrderRequest{
			Cancel: &hbspot.CancelAllParam{Symbol: order.Symbol},
		}
	}
}

func isOrderProfitable(order hbspot.NewOrderParam) bool {
	spread, ok1 := hbSpreads[order.Symbol]
	quantile, ok2 := kcQuantiles[order.Symbol]
	if !ok1 || !ok2 || time.Now().Sub(spread.PerpOrderBook.ParseTime) > *hbConfig.SpreadTimeToLive {
		logger.Debugf("SPREAD IS OUT OF DATE %v, CANCEL %s", time.Now().Sub(spread.PerpOrderBook.ParseTime), order.Symbol)
		return false
	}
	if strings.Contains(order.Type, hbspot.OrderSideBuy) &&
		order.OriginPrice < (1.0-2**hbConfig.MakerBandOffset)*spread.SpotOrderBook.BidPrice-hbspotTickSizes[order.Symbol] {
		logger.Debugf("%s BUY PRICE %f < MAKER BAND OFFSET BID PRICE %f",
			order.Symbol,
			order.Price,
			(1.0-2**hbConfig.MakerBandOffset)*spread.SpotOrderBook.BidPrice-hbspotTickSizes[order.Symbol],
		)
		return false
	} else if strings.Contains(order.Type, hbspot.OrderSideSell) &&
		order.OriginPrice > (1.0+2**hbConfig.MakerBandOffset)*spread.SpotOrderBook.AskPrice+hbspotTickSizes[order.Symbol] {
		logger.Debugf("%s SELL PRICE %f > MAKER BAND OFFSEF ASK PRICE %f",
			order.Symbol,
			order.Price,
			(1.0+2**hbConfig.MakerBandOffset)*spread.SpotOrderBook.AskPrice+hbspotTickSizes[order.Symbol],
		)
		return false
	}

	if strings.Contains(order.Type, hbspot.OrderSideBuy) &&
		(spread.PerpOrderBook.TakerBidVWAP-order.OriginPrice)/order.OriginPrice > quantile.Top-*hbConfig.MakerBandOffset {
		return true
	} else if strings.Contains(order.Type, hbspot.OrderSideSell) &&
		(spread.PerpOrderBook.TakerAskVWAP-order.OriginPrice)/order.OriginPrice < quantile.Bot+*hbConfig.MakerBandOffset {
		return true
	}
	if strings.Contains(order.Type, hbspot.OrderSideBuy) {
		logger.Debugf(
			"NOT PROFITABLE %s BUY ORDER SWAP BIDVWAP %f ORDER PRICE %f DELTA %f < TOP %f - %f",
			order.Symbol,
			spread.PerpOrderBook.TakerBidVWAP,
			order.Price,
			(spread.PerpOrderBook.TakerBidVWAP-order.OriginPrice)/order.OriginPrice,
			quantile.Top,
			*hbConfig.MakerBandOffset,
		)
	} else {
		logger.Debugf(
			"NOT PROFITABLE %s BUY ORDER SWAP ASKVWAP %f ORDER PRICE %f DELTA %f > BOT %f + %f",
			order.Symbol,
			spread.PerpOrderBook.TakerAskVWAP,
			order.Price,
			(spread.PerpOrderBook.TakerAskVWAP-order.OriginPrice)/order.OriginPrice,
			quantile.Bot,
			*hbConfig.MakerBandOffset,
		)
	}
	return false
}
