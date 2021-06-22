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
	symbol string,
	dryRun bool,
	orderRequestCh chan bnswap.NewOrderParams,
	outputOrderCh chan bnswap.Order,
	outputOrderErrorCh chan TakerOrderNewError,
) {
	logger.Debugf("START watchTakerOrderRequest %s", symbol)
	defer logger.Debugf("EXIT watchTakerOrderRequest %s", symbol)
	for {
		select {
		case <-ctx.Done():
			return
		case newOrderParam := <-orderRequestCh:
			if dryRun {
				break
			}
			childCtx, _ := context.WithTimeout(ctx, timeout)
			order, err := api.SubmitOrder(childCtx, newOrderParam)
			if err != nil {
				logger.Debugf("api.SubmitOrder(childCtx, newOrderParam) error %v %v", err, newOrderParam)
				select {
				case outputOrderErrorCh <- TakerOrderNewError{
					Error:  err,
					Params: newOrderParam,
				}:
				default:
					logger.Debugf("outputOrderErrorCh <- TakerOrderNewError failed, ch len %d", len(outputOrderErrorCh))
				}
			} else {
				select {
				case outputOrderCh <- *order:
				default:
					logger.Debugf("outputOrderCh <- *order failed ch len %d", len(orderRequestCh))
				}
			}
		}
	}
}
