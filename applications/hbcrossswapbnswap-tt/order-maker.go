package main

import (
	"context"
	"github.com/geometrybase/hft-micro/hbcrossswap"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func watchMakerOrderRequest(
	ctx context.Context,
	api *hbcrossswap.API,
	timeout time.Duration,
	dryRun bool,
	newOrderParamCh chan hbcrossswap.NewOrderParam,
	outputOrderErrorCh chan MakerOrderNewError,
) {
	defer func(){
		logger.Debugf("LOOP END watchMakerOrderRequest %s")
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case param := <-newOrderParamCh:
			if dryRun {
				break
			}
			childCtx, _ := context.WithTimeout(ctx, timeout)
			logger.Debugf("MAKER SUBMIT %s %s %f %d", param.Symbol, param.OrderPriceType, param.Price, param.Volume)
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
