package main

import (
	"context"
	"github.com/geometrybase/hft-micro/hbcrossswap"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func handlePerpHttpPositions(positions []hbcrossswap.Position) {
	for _, nextPos := range positions {
		if _, ok := kcpsSymbolsMap[nextPos.Symbol]; !ok {
			continue
		}
		var lastPosition *hbcrossswap.Position
		if p, ok := hbcrossswapPositions[nextPos.Symbol]; ok {
			p := p
			lastPosition = &p
		}
		hbcrossswapPositions[nextPos.Symbol] = nextPos
		hbcrossswapPositionsUpdateTimes[nextPos.Symbol] = time.Now()
		if lastPosition == nil ||
			lastPosition.Volume != nextPos.Volume ||
			lastPosition.CostOpen != nextPos.CostOpen ||
			lastPosition.Direction != nextPos.Direction {
			//如果SPOT变仓，立刻调SWAP，如果SWAP变仓，等ORDER SILENT TIMEOUT
			hbcrossswapOrderSilentTimes[nextPos.Symbol] = time.Now()
			logger.Debugf("SWAP HTTP POSITION %s DIRECTION %s SIZE %f COST OPEN %f", nextPos.Symbol, nextPos.Direction, nextPos.Volume, nextPos.CostOpen)
		}
	}
}

func handleSwapHttpAccount(account hbcrossswap.Account) {
	if hbcrossswapAccount == nil ||
		hbcrossswapAccount.MarginBalance != account.MarginBalance {
		logger.Debugf("SWAP HTTP USDT ACCOUNT MarginBalance %f -> %f", hbcrossswapAccount.MarginBalance, account.MarginBalance)
	}
	hbcrossswapAccount = &account
	hbcrossswapAssetUpdatedForReBalance = true
	hbcrossswapAssetUpdatedForInflux = true
	hbcrossswapAssetUpdatedForExternalInflux = true
}

func swapCreateOrder(
	ctx context.Context,
	api *hbcrossswap.API,
	timeout time.Duration,
	params hbcrossswap.NewOrderParam,
) {
	childCtx, _ := context.WithTimeout(ctx, timeout)
	_, err := api.SubmitOrder(childCtx, params)
	if err != nil {
		select {
		case <-ctx.Done():
		case hbcrossswapNewOrderErrorCh <- SwapOrderNewError{
			Error:  err,
			Params: params,
		}:
		}
		//} else if order.Status == "FILLED" ||
		//	order.Status == "CANCELED" ||
		//	order.Status == "REJECTED" ||
		//	order.Status == "EXPIRED" {
		//	hbcrossswapOrderFinishCh <- *order
	}
}
