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
		if time.Now().Sub(mCancelSilentTimes[symbol]) < 0 {
			continue
		}
		delete(mOpenOrders, symbol)
		mOrderSilentTimes[order.Symbol] = time.Now().Add(*mtConfig.OrderSilent)
		mCancelSilentTimes[order.Symbol] = time.Now().Add(*mtConfig.CancelSilent)
		mOrderRequestChs[order.Symbol] <- MakerOrderRequest{
			Cancel: &okspot.CancelOrderParam{
				Symbol:    order.Symbol,
				ClientOid: order.ClientOID,
			},
		}
	}
}

func updateMakerOldOrders() {
	if mAccount == nil || tAccount == nil || tAccount.AvailableBalance == nil {
		return
	}
	entryStep := (mAccount.Available + *tAccount.AvailableBalance) * *mtConfig.EnterFreePct
	if entryStep < *mtConfig.EnterMinimalStep {
		entryStep = *mtConfig.EnterMinimalStep
	}
	entryTarget := entryStep * *mtConfig.EnterTargetFactor

	for symbol, order := range mOpenOrders {
		if time.Now().Sub(mCancelSilentTimes[symbol]) < 0 {
			continue
		}
		if isOrderProfitable(*order.NewOrderParam, entryTarget) {
			continue
		}
		delete(mOpenOrders, symbol)
		mOrderSilentTimes[order.Symbol] = time.Now().Add(*mtConfig.CancelSilent)
		mCancelSilentTimes[order.Symbol] = time.Now().Add(*mtConfig.CancelSilent)
		mOrderRequestChs[order.Symbol] <- MakerOrderRequest{
			Cancel: &okspot.CancelOrderParam{
				Symbol:    order.Symbol,
				ClientOid: order.ClientOID,
			},
		}
	}
}

func isOrderProfitable(order okspot.NewOrderParam, entryTarget float64) bool {
	spread, okSpread := mtSpreads[order.Symbol]
	makerBalance, okMakerBalance := mBalances[order.Symbol]
	if !okSpread || !okMakerBalance || time.Now().Sub(spread.Time) > *mtConfig.SpreadTimeToLive {
		logger.Debugf("SPREAD OR BALANCE NOT READY, CANCEL %s", order.Symbol)
		return false
	}
	currentSpotSize := makerBalance.Balance
	currentSpotValue := currentSpotSize * spread.MakerDepth.MidPrice
	makerOffset := mOrderOffsets[order.Symbol]
	enterDelta := *mtConfig.EnterDelta + *mtConfig.OffsetDelta*(currentSpotValue/entryTarget)
	exitDelta := *mtConfig.ExitDelta + *mtConfig.OffsetDelta*(currentSpotValue/entryTarget)

	//检查价格有没有挂太远，太远撤掉
	if order.Side == okspot.OrderSideBuy &&
		*order.Price < spread.MakerDepth.MidPrice*(1.0+makerOffset.FarBot) {
		logger.Debugf("%s BUY PRICE %f < FAR BOT %f, CANCEL",
			order.Symbol,
			*order.Price,
			spread.MakerDepth.MidPrice*(1.0+makerOffset.FarBot),
		)
		return false
	} else if order.Side == okspot.OrderSideBuy &&
		*order.Price > spread.MakerDepth.MidPrice*(1.0+makerOffset.NearBot) {
		logger.Debugf("%s BUY PRICE %f > NEAR BOT %f, CANCEL",
			order.Symbol,
			*order.Price,
			spread.MakerDepth.MidPrice*(1.0+makerOffset.NearBot),
		)
		return false
	} else if order.Side == okspot.OrderSideSell &&
		*order.Price > spread.MakerDepth.MidPrice*(1.0 + makerOffset.FarTop) {
		logger.Debugf("%s SELL PRICE %f > FAR TOP %f, CANCEL",
			order.Symbol,
			*order.Price,
			spread.MakerDepth.MidPrice*(1.0 + makerOffset.FarTop) ,
		)
		return false
	} else if order.Side == okspot.OrderSideSell &&
		*order.Price < spread.MakerDepth.MidPrice*(1.0 + makerOffset.NearTop) {
		logger.Debugf("%s SELL PRICE %f < NEAR TOP %f, CANCEL",
			order.Symbol,
			*order.Price,
			spread.MakerDepth.MidPrice*(1.0 + makerOffset.NearTop),
		)
		return false
	}

	if order.Side == okspot.OrderSideBuy &&
		(spread.TakerDepth.TakerBid-*order.Price) / *order.Price > enterDelta {
		//买入开多, 是开空价差, 参考ShortTop
		return true
	} else if order.Side == okspot.OrderSideSell &&
		(spread.TakerDepth.TakerAsk-*order.Price) / *order.Price < exitDelta {
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
