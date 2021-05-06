package main

import (
	"context"
	"github.com/geometrybase/hft-micro/kcperp"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"time"
)

func watchMakerOrderRequest(
	ctx context.Context,
	api *kcperp.API,
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
				logger.Debugf("MAKER SUBMIT %s %s %f %d", request.New.Symbol, request.New.Side, request.New.Price, request.New.Size)
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
				logger.Debugf("MAKER CANCEL ALL %s", request.Cancel.Symbol)
				resp, err := api.CancelAllOrders(childCtx, *request.Cancel)
				if err != nil {
					logger.Debugf("MAKER SUBMIT ERROR %v", err)
				} else {
					for _, s := range resp.CancelledOrderIds {
						outputOrderRespCh <- MakerOpenOrder{
							NewOrderParam:   nil,
							ResponseOrderID: s,
							Symbol:          request.Cancel.Symbol,
						}
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
		mOrderCancelCounts[order.Symbol] += 1
		mOrderRequestChs[order.Symbol] <- MakerOrderRequest{
			Cancel: &kcperp.CancelAllOrdersParam{Symbol: order.Symbol},
		}
	}
}

func updateMakerOldOrders() {
	if mAccount == nil || tAccount == nil || tAccount.AvailableBalance == nil {
		return
	}

	entryStep := (mAccount.AvailableBalance + *tAccount.AvailableBalance) * *mtConfig.EnterFreePct
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
		mOrderSilentTimes[order.Symbol] = time.Now().Add(*mtConfig.OrderSilent)
		mCancelSilentTimes[order.Symbol] = time.Now().Add(*mtConfig.CancelSilent)
		mOrderCancelCounts[order.Symbol] += 1
		mOrderRequestChs[order.Symbol] <- MakerOrderRequest{
			Cancel: &kcperp.CancelAllOrdersParam{Symbol: order.Symbol},
		}
	}
}

func isOrderProfitable(order kcperp.NewOrderParam, entryTarget float64) bool {
	spread, okSpread := mtSpreads[order.Symbol]
	makerPosition, okMakerPosition := mPositions[order.Symbol]

	if !okSpread || !okMakerPosition || time.Now().Sub(spread.Time) > *mtConfig.SpreadTimeToLive {
		logger.Debugf("SPREAD OR MAKER POSITION NOT READY, CANCEL %s", order.Symbol)
		return false
	}

	makerValue := makerPosition.AvgEntryPrice * makerPosition.CurrentQty * mMultipliers[order.Symbol]
	offset := mOrderOffsets[order.Symbol]
	shortTop := *mtConfig.EnterDelta + *mtConfig.OffsetDelta*(math.Max(makerValue, 0)/entryTarget)
	shortBot := *mtConfig.ExitDelta + *mtConfig.OffsetDelta*(math.Max(makerValue, 0)/entryTarget)
	longBot := -*mtConfig.EnterDelta + *mtConfig.OffsetDelta*(math.Min(makerValue, 0)/entryTarget)
	longTop := -*mtConfig.ExitDelta + *mtConfig.OffsetDelta*(math.Min(makerValue, 0)/entryTarget)

	//检查价格有没有在OFFSET范围内，不在撤掉
	if order.Side == kcperp.OrderSideBuy &&
		float64(order.Price) < spread.MakerDepth.MakerBid*(1.0+offset.FarBot) {
		logger.Debugf("%s BUY PRICE %f < FAR BOT %f, CANCEL",
			order.Symbol,
			order.Price,
			spread.MakerDepth.MakerBid*(1.0+offset.FarBot),
		)
		return false
	} else if order.Side == kcperp.OrderSideBuy &&
		float64(order.Price) > spread.MakerDepth.MakerBid*(1.0+offset.NearBot) {
		logger.Debugf("%s BUY PRICE %f > NEAR BOT %f, CANCEL",
			order.Symbol,
			order.Price,
			spread.MakerDepth.MakerBid*(1.0+offset.NearBot),
		)
		return false
	} else if order.Side == kcperp.OrderSideSell &&
		float64(order.Price) > spread.MakerDepth.MakerAsk*(1.0+offset.FarTop) {
		logger.Debugf("%s SELL PRICE %f > FAR TOP %f, CANCEL ",
			order.Symbol,
			order.Price,
			spread.MakerDepth.MakerAsk*(1.0+offset.FarTop),
		)
		return false
	} else if order.Side == kcperp.OrderSideSell &&
		float64(order.Price) < spread.MakerDepth.MakerAsk*(1.0+offset.NearTop) {
		logger.Debugf("%s SELL PRICE %f < NEAR TOP %f, CANCEL ",
			order.Symbol,
			order.Price,
			spread.MakerDepth.MakerAsk*(1.0+offset.NearTop),
		)
		return false
	}

	if order.Side == kcperp.OrderSideBuy &&
		!order.ReduceOnly &&
		(spread.TakerDepth.TakerBid-float64(order.Price))/float64(order.Price) > shortTop {
		//买入开多, 是开空价差, 参考ShortTop
		return true
	} else if order.Side == kcperp.OrderSideSell &&
		order.ReduceOnly &&
		(spread.TakerDepth.TakerAsk-float64(order.Price))/float64(order.Price) < shortBot{
		//卖出平多, 是平空价, 参考ShortBot
		return true
	} else if order.Side == kcperp.OrderSideSell &&
		!order.ReduceOnly &&
		(spread.TakerDepth.TakerAsk-float64(order.Price))/float64(order.Price) < longBot{
		//卖出开空, 是开多价差, 参考LongBot
		return true
	} else if order.Side == kcperp.OrderSideBuy &&
		order.ReduceOnly &&
		(spread.TakerDepth.TakerBid-float64(order.Price))/float64(order.Price) > longTop {
		//买入平空, 是平多价差, 参考LongTop
		return true
	}
	if order.Side == kcperp.OrderSideBuy {
		logger.Debugf(
			"NOT PROFITABLE %s BUY ORDER, CANCEL", order.Symbol,
		)
	} else {
		logger.Debugf(
			"NOT PROFITABLE %s SELL ORDER, CANCEL", order.Symbol,
		)
	}
	return true
}
