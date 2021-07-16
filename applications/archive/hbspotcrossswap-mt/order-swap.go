package main

import (
	"context"
	"github.com/geometrybase/hft-micro/huobi-usdtfuture"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func watchSwapOrderRequest(
	ctx context.Context,
	api *huobi_usdtfuture.API,
	timeout time.Duration,
	dryRun bool,
	orderRequestCh chan huobi_usdtfuture.NewOrderParam,
	outputOrderErrorCh chan SwapOrderNewError,
) {
	for {
		select {
		case <-ctx.Done():
			return
		case newOrderParam := <-orderRequestCh:
			if dryRun {
				break
			}
			childCtx, _ := context.WithTimeout(ctx, timeout)
			logger.Debugf("SWAP SUBMIT %v", newOrderParam)
			_, err := api.SubmitOrder(childCtx, newOrderParam)
			if err != nil {
				logger.Debugf("SWAP SUBMIT ERROR %v %v", err, newOrderParam)
				outputOrderErrorCh <- SwapOrderNewError{
					Error:  err,
					Params: newOrderParam,
				}
			}
		}
	}
}

