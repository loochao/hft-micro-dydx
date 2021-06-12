package main

import (
	"context"
	"github.com/geometrybase/hft-micro/kucoin-usdtfuture"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func watchMakerOrderRequest(
	ctx context.Context,
	api *kucoin_usdtfuture.API,
	timeout time.Duration,
	dryRun bool,
	orderRequestCh chan kucoin_usdtfuture.NewOrderParam,
	outputOrderErrorCh chan MakerOrderNewError,
) {
	defer func() {
		logger.Debugf("EXIT watchMakerOrderRequest")
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
			logger.Debugf("MAKER SUBMIT %s %s %f %d", param.Symbol, param.Side, param.Price, param.Size)
			_, err := api.SubmitOrder(childCtx, param)
			if err != nil {
				logger.Debugf("MAKER SUBMIT ERROR %v", err)
				outputOrderErrorCh <- MakerOrderNewError{
					Error:  err,
					Params: param,
				}
			}
		}
	}
}

