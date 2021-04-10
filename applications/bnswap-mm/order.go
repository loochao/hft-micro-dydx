package main

import (
	"context"
	"github.com/geometrybase/hft-micro/bnspot"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func watchSwapOrderRequest(
	ctx context.Context,
	api *bnswap.API,
	timeout time.Duration,
	dryRun bool,
	orderRequestCh chan SwapOrderRequest,
	outputOrderErrorCh chan SwapOrderNewError,
	outputNewOrderResponseCh chan bnswap.Order,
	cancelAllOrderResponsesCh chan bnswap.CancelAllOrderResponse,
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
				orders, err := api.CancelAllOpenOrders(childCtx, *request.Cancel)
				if err != nil {
					logger.Debugf("SWAP CANCEL ALL %v", err)
				} else {
					cancelAllOrderResponsesCh <- *orders
				}
			} else if request.New != nil {
				childCtx, _ := context.WithTimeout(ctx, timeout)
				logger.Debugf("SWAP SUBMIT %s", request.New.ToUrlValues().Encode())
				order, err := api.SubmitOrder(childCtx, *request.New)
				if err != nil {
					logger.Debugf("SWAP SUBMIT ERROR %v", err)
					outputOrderErrorCh <- SwapOrderNewError{
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

func updateSwapOldOrders() {
	for symbol, order := range bnswapOpenOrders {
		if bnswapOrderCancelCounts[symbol] > *bnConfig.OrderMaxCancelCount {
			delete(bnswapOpenOrders, symbol)
			continue
		}
		if time.Now().Sub(bnswapCancelSilentTimes[symbol]) < 0 {
			continue
		}
		if !order.ReduceOnly && !isOrderProfitable(order) {
			bnswapOrderSilentTimes[order.Symbol] = bnswapLastOrderTimes[symbol].Add(*bnConfig.OrderInterval)
			bnswapCancelSilentTimes[order.Symbol] = time.Now().Add(*bnConfig.OrderCancelSilent)
			bnswapOrderCancelCounts[order.Symbol] += 1
			bnswapOrderRequestChs[order.Symbol] <- SwapOrderRequest{
				Cancel: &bnswap.CancelAllOrderParams{Symbol: order.Symbol},
			}
			continue
		}
		if time.Now().Sub(bnswapLastOrderTimes[symbol]) < *bnConfig.OrderInterval {
			continue
		}
		bnswapOrderSilentTimes[order.Symbol] = time.Now().Add(*bnConfig.OrderSilent)
		bnswapCancelSilentTimes[order.Symbol] = time.Now().Add(*bnConfig.OrderCancelSilent)
		bnswapOrderCancelCounts[order.Symbol] += 1
		bnswapOrderRequestChs[order.Symbol] <- SwapOrderRequest{
			Cancel: &bnswap.CancelAllOrderParams{Symbol: order.Symbol},
		}
	}
}

func isOrderProfitable(order bnswap.NewOrderParams) bool {
	spread, ok1 := bnSpreads[order.Symbol]
	markPrice, ok2 := bnswapMarkPrices[order.Symbol]
	if !ok1 || !ok2 || order.ReduceOnly {
		return false
	}
	if time.Now().Sub(spread.EventTime) > *bnConfig.SpreadTimeToLive {
		logger.Debugf(
			"%s SPREAD IS OUT OF DATE %v, CANCEL",
			order.Symbol,
			time.Now().Sub(spread.EventTime),
		)
		return false
	}
	if order.Side == common.OrderSideBuy {
		if spread.MedianLong < *bnConfig.EnterMinimalSpread {
			logger.Debugf("%s LONG SPREAD %f < MINIMAL SPREAD %f, CANCEL",
				order.Symbol,
				spread.MedianLong,
				*bnConfig.EnterMinimalSpread,
			)
			return false
		} else if spread.MedianLong > *bnConfig.EnterMaximalSpread {
			logger.Debugf("%s LONG SPREAD %f > MAXIMAL SPREAD %f, CANCEL",
				order.Symbol,
				spread.MedianLong,
				*bnConfig.EnterMaximalSpread,
			)
			return false
		} else if markPrice.FundingRate < *bnConfig.MinimalLongFundingRate {
			logger.Debugf("%s LONG FUNDING RATE %f < MINIMAL FUNDING RATE %f, CANCEL",
				order.Symbol,
				spread.MedianLong,
				*bnConfig.MinimalLongFundingRate,
			)
			return false
		}
	} else if order.Side == common.OrderSideSell {
		if spread.MedianLong < *bnConfig.EnterMinimalSpread {
			logger.Debugf("%s SHORT SPREAD %f < MINIMAL SPREAD %f, CANCEL",
				order.Symbol,
				spread.MedianLong,
				*bnConfig.EnterMinimalSpread,
			)
			return false
		} else if spread.MedianLong > *bnConfig.EnterMaximalSpread {
			logger.Debugf("%s SHORT SPREAD %f > MAXIMAL SPREAD %f, CANCEL",
				order.Symbol,
				spread.MedianLong,
				*bnConfig.EnterMaximalSpread,
			)
			return false
		} else if markPrice.FundingRate > *bnConfig.MaximalShortFundingRate {
			logger.Debugf("%s SHORT FUNDING RATE %f > MAXIMAL FUNDING RATE %f, CANCEL",
				order.Symbol,
				spread.MedianLong,
				*bnConfig.MinimalLongFundingRate,
			)
			return false
		}
	}
	return true
}
