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
	outputOrderErrorCh chan TakerOrderNewError,
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
			logger.Debugf("B SUBMIT %v", newOrderParam)
			_, err := api.SubmitOrder(childCtx, newOrderParam)
			if err != nil {
				logger.Debugf("B SUBMIT ERROR %v", err)
				outputOrderErrorCh <- TakerOrderNewError{
					Error:  err,
					Params: newOrderParam,
				}
			}
		}
	}
}

