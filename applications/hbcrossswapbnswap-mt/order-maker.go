package main

import (
	"context"
	"github.com/geometrybase/hft-micro/hbcrossswap"
	"github.com/geometrybase/hft-micro/hbspot"
	"github.com/geometrybase/hft-micro/logger"
	"strings"
	"time"
)

func watchMakerOrderRequest(
	ctx context.Context,
	api *hbcrossswap.API,
	timeout time.Duration,
	dryRun bool,
	orderRequestCh chan MakerOrderRequest,
	outputOrderErrorCh chan HOrderNewError,
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
					logger.Debugf("MAKER CANCEL ALL %v", err)
				}
			} else if request.New != nil {
				childCtx, _ := context.WithTimeout(ctx, timeout)
				logger.Debugf("MAKER SUBMIT %s %s %f %d", request.New.Symbol, request.New.OrderPriceType, request.New.Price, request.New.Volume)
				_, err := api.SubmitOrder(childCtx, *request.New)
				if err != nil {
					logger.Debugf("MAKER SUBMIT ERROR %v", err)
					outputOrderErrorCh <- HOrderNewError{
						Error:  err,
						Params: *request.New,
					}
				}
			}
		}
	}
}

func updateMakerOldOrders() {
	for symbol, order := range mOpenOrders {
		if mOrderCancelCounts[symbol] > *mtConfig.OrderMaxCancelCount {
			delete(mOpenOrders, symbol)
			continue
		}
		if time.Now().Sub(hbspotCancelSilentTimes[symbol]) < 0 {
			continue
		}
		if isOrderProfitable(order) {
			continue
		}
		tOrderSilentTimes[order.Symbol] = time.Now().Add(*mtConfig.OrderSilent)
		hbspotCancelSilentTimes[order.Symbol] = time.Now().Add(*mtConfig.OrderCancelSilent)
		mOrderCancelCounts[order.Symbol] += 1
		mOrderRequestChs[order.Symbol] <- MakerOrderRequest{
			Cancel: &hbspot.CancelAllParam{Symbol: order.Symbol},
		}
	}
}

func isOrderProfitable(order hbspot.NewOrderParam) bool {
	spread, ok1 := mtSpreads[order.Symbol]
	quantile, ok2 := mtQuantiles[order.Symbol]
	if !ok1 || !ok2 || time.Now().Sub(spread.MakerOrderBook.ParseTime) > *mtConfig.SpreadTimeToLive {
		logger.Debugf("SPREAD IS OUT OF DATE %v, CANCEL %s", time.Now().Sub(spread.MakerOrderBook.ParseTime), order.Symbol)
		return false
	}
	if strings.Contains(order.Type, hbspot.OrderSideBuy) &&
		order.OriginPrice < (1.0-2**mtConfig.MakerBandOffset)*spread.TakerOrderBook.BidPrice-tTickSizes[order.Symbol] {
		logger.Debugf("%s BUY PRICE %f < MAKER BAND OFFSET BID PRICE %f",
			order.Symbol,
			order.Price,
			(1.0-2**mtConfig.MakerBandOffset)*spread.TakerOrderBook.BidPrice-tTickSizes[order.Symbol],
		)
		return false
	} else if strings.Contains(order.Type, hbspot.OrderSideSell) &&
		order.OriginPrice > (1.0+2**mtConfig.MakerBandOffset)*spread.TakerOrderBook.AskPrice+tTickSizes[order.Symbol] {
		logger.Debugf("%s SELL PRICE %f > MAKER BAND OFFSEF ASK PRICE %f",
			order.Symbol,
			order.Price,
			(1.0+2**mtConfig.MakerBandOffset)*spread.TakerOrderBook.AskPrice+tTickSizes[order.Symbol],
		)
		return false
	}

	if strings.Contains(order.Type, hbspot.OrderSideBuy) &&
		(spread.MakerOrderBook.TakerBidVWAP-order.OriginPrice)/order.OriginPrice > quantile.ShortTop-*mtConfig.MakerBandOffset {
		return true
	} else if strings.Contains(order.Type, hbspot.OrderSideSell) &&
		(spread.MakerOrderBook.TakerAskVWAP-order.OriginPrice)/order.OriginPrice < quantile.ShortBot+*mtConfig.MakerBandOffset {
		return true
	}
	if strings.Contains(order.Type, hbspot.OrderSideBuy) {
		logger.Debugf(
			"NOT PROFITABLE %s BUY ORDER SWAP BIDVWAP %f ORDER PRICE %f DELTA %f < TOP %f - %f",
			order.Symbol,
			spread.MakerOrderBook.TakerBidVWAP,
			order.Price,
			(spread.MakerOrderBook.TakerBidVWAP-order.OriginPrice)/order.OriginPrice,
			quantile.ShortTop,
			*mtConfig.MakerBandOffset,
		)
	} else {
		logger.Debugf(
			"NOT PROFITABLE %s BUY ORDER SWAP ASKVWAP %f ORDER PRICE %f DELTA %f > BOT %f + %f",
			order.Symbol,
			spread.MakerOrderBook.TakerAskVWAP,
			order.Price,
			(spread.MakerOrderBook.TakerAskVWAP-order.OriginPrice)/order.OriginPrice,
			quantile.ShortBot,
			*mtConfig.MakerBandOffset,
		)
	}
	return false
}
