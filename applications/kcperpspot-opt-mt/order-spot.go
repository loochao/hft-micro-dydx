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
				childCtx, cancel := context.WithTimeout(ctx, timeout)
				_, err := api.CancelAllOrders(childCtx, *request.Cancel)
				if err != nil {
					logger.Debugf("SPOT CANCEL ALL %v", err)
				}
				cancel()
			} else if request.New != nil {
				childCtx, cancel := context.WithTimeout(ctx, timeout)
				logger.Debugf("SPOT SUBMIT %s %s %.8f %.8f", request.New.Symbol, request.New.Side, request.New.Price, request.New.Size)
				_, err := api.SubmitOrder(childCtx, *request.New)
				cancel()
				if err != nil {
					logger.Debugf("SPOT SUBMIT ERROR %s %v", request.New.Symbol, err)
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
	if kcperpUSDTAccount == nil {
		return
	}
	entryStep := (kcperpUSDTAccount.AvailableBalance + kcspotUSDTBalance.Available) * *kcConfig.EnterFreePct
	if entryStep < *kcConfig.EnterMinimalStep {
		entryStep = *kcConfig.EnterMinimalStep
	}
	entryTarget := entryStep * *kcConfig.EnterTargetFactor
	for symbol, order := range kcspotOpenOrders {
		if time.Now().Sub(kcspotCancelSilentTimes[symbol]) < 0 {
			continue
		}
		if isOrderProfitable(order, entryTarget) {
			continue
		}
		delete(kcspotOpenOrders, symbol)
		kcspotOrderSilentTimes[order.Symbol] = time.Now().Add(*kcConfig.CancelSilent)
		kcspotCancelSilentTimes[order.Symbol] = time.Now().Add(*kcConfig.CancelSilent)
		kcspotOrderCancelCounts[order.Symbol] += 1
		kcspotOrderRequestChs[order.Symbol] <- SpotOrderRequest{
			Cancel: &kcspot.CancelAllOrdersParam{Symbol: order.Symbol},
		}
	}
}

func isOrderProfitable(order kcspot.NewOrderParam, entryTarget float64) bool {
	spread, okSpread := kcSpreads[order.Symbol]
	spotBalance, okSpotBalance := kcspotBalances[order.Symbol]
	if !okSpread || !okSpotBalance || time.Now().Sub(spread.Time) > *kcConfig.SpreadTimeToLive {
		if !okSpread {
			logger.Debugf("%s SPREAD IS NOT READY, CANCEL", order.Symbol)
		} else if !okSpotBalance {
			logger.Debugf("%s BALANCE IS NOT READY, CANCEL", order.Symbol)
		} else {
			logger.Debugf("%s SPREAD IS OUT OF DATE %v, CANCEL", order.Symbol, time.Now().Sub(spread.Time))
		}
		return false
	}
	currentSpotSize := spotBalance.Available + spotBalance.Holds
	currentSpotValue := currentSpotSize * spread.MakerDepth.MidPrice
	offset := kcspotOffsets[order.Symbol]
	enterDelta := *kcConfig.EnterDelta + *kcConfig.OffsetDelta*(currentSpotValue/entryTarget)
	exitDelta := *kcConfig.ExitDelta + *kcConfig.OffsetDelta*(currentSpotValue/entryTarget)
	if order.Side == kcspot.OrderSideBuy &&
		float64(order.Price) < spread.MakerDepth.BestBidPrice*(1.0+offset.FarBot) {
		logger.Debugf("%s BUY PRICE %f < FAR BOT %f, CANCEL",
			order.Symbol,
			order.Price,
			spread.MakerDepth.BestBidPrice*(1.0+offset.FarBot),
		)
		return false
	} else if order.Side == kcspot.OrderSideBuy &&
		float64(order.Price) > spread.MakerDepth.BestBidPrice*(1.0+offset.NearBot) {
		logger.Debugf("%s BUY PRICE %f > NEAR BOT %f, CANCEL",
			order.Symbol,
			order.Price,
			spread.MakerDepth.BestBidPrice*(1.0+offset.NearBot),
		)
		return false
	} else if order.Side == kcspot.OrderSideSell &&
		float64(order.Price) > spread.MakerDepth.BestAskPrice*(1.0+offset.FarTop) {
		logger.Debugf("%s SELL PRICE %f > FAR TOP %f, CANCEL",
			order.Symbol,
			order.Price,
			spread.MakerDepth.BestAskPrice*(1.0+offset.FarTop),
		)
		return false
	} else if order.Side == kcspot.OrderSideSell &&
		float64(order.Price) < spread.MakerDepth.BestAskPrice*(1.0+offset.NearTop) {
		logger.Debugf("%s SELL PRICE %f < NEAR TOP %f, CANCEL",
			order.Symbol,
			order.Price,
			spread.MakerDepth.BestAskPrice*(1.0+offset.NearTop),
		)
		return false
	}
	if order.Side == kcspot.OrderSideBuy &&
		(spread.TakerDepth.TakerBid-float64(order.Price))/float64(order.Price) > enterDelta {
		return true
	} else if order.Side == kcspot.OrderSideSell &&
		(spread.TakerDepth.TakerAsk-float64(order.Price))/float64(order.Price) < exitDelta {
		return true
	}
	if order.Side == kcspot.OrderSideBuy {
		logger.Debugf(
			"%s NOT PROFITABLE BUY ORDER PERP TAKER BID %f ORDER PRICE %f DELTA %f < TOP %f CANCEL",
			order.Symbol,
			spread.TakerDepth.TakerBid,
			order.Price,
			(spread.TakerDepth.TakerBid-float64(order.Price))/float64(order.Price),
			enterDelta,
		)
	} else {
		logger.Debugf(
			"%s NOT PROFITABLE BUY ORDER PERP TAKER ASK %f ORDER PRICE %f DELTA %f > BOT %f CANCEL",
			order.Symbol,
			spread.TakerDepth.TakerAsk,
			order.Price,
			(spread.TakerDepth.TakerAsk-float64(order.Price))/float64(order.Price),
			exitDelta,
		)
	}
	return false
}
