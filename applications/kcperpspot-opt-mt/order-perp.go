package main

import (
	"context"
	"github.com/geometrybase/hft-micro/kucoin-usdtfuture"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func watchPerpOrderRequest(
	ctx context.Context,
	api *kucoin_usdtfuture.API,
	timeout time.Duration,
	dryRun bool,
	orderRequestCh chan kucoin_usdtfuture.NewOrderParam,
	outputOrderErrorCh chan PerpOrderNewError,
) {
	for {
		select {
		case <-ctx.Done():
			return
		case newOrderParam := <-orderRequestCh:
			if dryRun {
				break
			}
			childCtx, cancel := context.WithTimeout(ctx, timeout)
			logger.Debugf("PERP SUBMIT %v", newOrderParam)
			_, err := api.SubmitOrder(childCtx, newOrderParam)
			if err != nil {
				logger.Debugf("PERP SUBMIT ERROR %s %v", newOrderParam.Symbol, err)
				outputOrderErrorCh <- PerpOrderNewError{
					Error:  err,
					Params: newOrderParam,
				}
			}
			cancel()
		}
	}
}
