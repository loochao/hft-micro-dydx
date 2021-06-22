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
	orderRequestCh chan OrderRequest,
	outputOrderErrorCh chan OrderNewError,
) {
	defer func() {
		logger.Debugf("EXIT watchTakerOrderRequest")
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case req := <-orderRequestCh:
			if dryRun {
				break
			}
			if req.New != nil {
				childCtx, _ := context.WithTimeout(ctx, timeout)
				logger.Debugf("BNSWAP SUBMIT %v", *req.New)
				_, err := api.SubmitOrder(childCtx, *req.New)
				if err != nil {
					logger.Debugf("BNSWAP SUBMIT ERROR %v %v", err, *req.New)
					outputOrderErrorCh <- OrderNewError{
						Error:  err,
						Params: *req.New,
					}
				}
			} else if req.Cancel != nil{
				childCtx, _ := context.WithTimeout(ctx, timeout)
				logger.Debugf("BNSWAP CANCEL %v", *req.Cancel)
				_, err := api.CancelAllOpenOrders(childCtx, *req.Cancel)
				if err != nil {
					logger.Debugf("BNSWAP CANCEL ERROR %v %v", err, *req.New)
				}
			}
		}
	}
}
