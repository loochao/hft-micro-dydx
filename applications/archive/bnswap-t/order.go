package main

import (
	"context"
	"github.com/geometrybase/hft-micro/bnspot"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func startOrderRoutine(
	ctx context.Context,
	api *bnswap.API,
	timeout time.Duration,
	dryRun bool,
	inputCh chan bnswap.NewOrderParams,
	outputOrderErrorCh chan SwapOrderNewError,
	outputNewOrderResponseCh chan bnswap.Order,
) {
	for {
		select {
		case <-ctx.Done():
			return
		case request := <-inputCh:
			if dryRun {
				break
			}
			childCtx, _ := context.WithTimeout(ctx, timeout)
			//logger.Debugf("SWAP SUBMIT %s", request.ToUrlValues().Encode())
			order, err := api.SubmitOrder(childCtx, request)
			if err != nil {
				logger.Debugf("SPOT SUBMIT ERROR %v", err)
				outputOrderErrorCh <- SwapOrderNewError{
					Error:  err,
					Params: request,
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

