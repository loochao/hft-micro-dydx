package main

import (
	"context"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func watchTakerOrderRequest(
	ctx context.Context,
	api *bnswap.API,
	timeout time.Duration,
	dryRun bool,
	orderRequestCh chan bnswap.NewOrderParams,
	outputOrderCh chan bnswap.Order,
	outputOrderErrorCh chan TakerOrderNewError,
) {
	defer func() {
		logger.Debugf("EXIT watchTakerOrderRequest")
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case newOrderParam := <-orderRequestCh:
			if dryRun {
				break
			}
			childCtx, _ := context.WithTimeout(ctx, timeout)
			logger.Debugf("TAKER SUBMIT %v", newOrderParam)
			order, err := api.SubmitOrder(childCtx, newOrderParam)
			if err != nil {
				logger.Debugf("TAKER SUBMIT ERROR %v %v", err, newOrderParam)
				outputOrderErrorCh <- TakerOrderNewError{
					Error:  err,
					Params: newOrderParam,
				}
			} else if order.Status == "FILLED" ||
				order.Status == "CANCELED" ||
				order.Status == "REJECTED" ||
				order.Status == "EXPIRED" {
				bnswapOrderFinishCh <- *order
				select {
				case <-ctx.Done():
				case <-time.After(time.Second):
					logger.Debugf("SEND ORDER RESP OUT TIMEOUT IN 1S")
				case outputOrderCh <- *order:
				}
			}
		}
	}
}
