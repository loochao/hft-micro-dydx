package main

import (
	"context"
	"github.com/geometrybase/hft-micro/kcperp"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func watchPerpOrderRequest(
	ctx context.Context,
	api *kcperp.API,
	timeout time.Duration,
	dryRun bool,
	orderRequestCh chan kcperp.NewOrderParam,
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
				logger.Debugf("PERP SUBMIT ERROR %v", err)
				outputOrderErrorCh <- PerpOrderNewError{
					Error:  err,
					Params: newOrderParam,
				}
			}
			cancel()
		}
	}
}

