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
	defer func() {
		logger.Debugf("EXIT watchTakerOrderRequest")
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case param := <-orderRequestCh:
			if dryRun {
				break
			}
			childCtx, _ := context.WithTimeout(ctx, timeout)
			_, err := api.SubmitOrder(childCtx, param)
			if err != nil {
				logger.Debugf("SUBMIT ERROR %s %s %v", param.Symbol, param.NewClientOrderId, err)
				outputOrderErrorCh <- TakerOrderNewError{
					Error:  err,
					Params: param,
				}
			}
		}
	}
}

