package main

import (
	"context"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func watchOrderRequest(
	ctx context.Context,
	api *bnswap.API,
	timeout time.Duration,
	dryRun bool,
	orderRequestCh chan bnswap.NewOrderParams,
	outputOrderErrorCh chan TakerOrderNewError,
) {
	defer func() {
		logger.Debugf("EXIT watchOrderRequest")
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case param := <-orderRequestCh:
			if dryRun {
				break
			}
			childCtx, cancel := context.WithTimeout(ctx, timeout)
			_, err := api.SubmitOrder(childCtx, param)
			if err != nil {
				logger.Debugf("SUBMIT ERROR %s %s %v", param.Symbol, param.NewClientOrderId, err)
				outputOrderErrorCh <- TakerOrderNewError{
					Error:  err,
					Params: param,
				}
			}
			cancel()
		}
	}
}

