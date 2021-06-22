package main

import (
	"context"
	"github.com/geometrybase/hft-micro/bnspot"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func watchSwapOrderRequest(
	ctx context.Context,
	api *bnswap.API,
	timeout time.Duration,
	dryRun bool,
	orderRequestCh chan bnswap.NewOrderParams,
	outputOrderErrorCh chan SwapOrderNewError,
	outputNewOrderResponseCh chan bnswap.Order,
) {
	for {
		select {
		case <-ctx.Done():
			return
		case params := <-orderRequestCh:
			if dryRun {
				break
			}
			childCtx, _ := context.WithTimeout(ctx, timeout)
			order, err := api.SubmitOrder(childCtx, params)
			if err != nil {
				logger.Debugf("SWAP SUBMIT ERROR %v, %s", err, params.ToUrlValues().Encode())
				outputOrderErrorCh <- SwapOrderNewError{
					Error:  err,
					Params: params,
				}
			} else if order.Status == bnspot.OrderStatusFilled ||
				order.Status == bnspot.OrderStatusCancelled ||
				order.Status == bnspot.OrderStatusReject ||
				order.Status == bnspot.OrderStatusExpired {
				outputNewOrderResponseCh <- *order
			}
		}
	}
}

