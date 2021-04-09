package main

import (
	"context"
	"github.com/geometrybase/hft/bnswap"
	"github.com/geometrybase/hft/logger"
	"time"
)

func handleSwapHttpPosition(nextPos bnswap.Position) {
	if _, ok := boSymbolsMap[nextPos.Symbol]; !ok {
		return
	}
	if nextPos.PositionSide != "BOTH" {
		return
	}
	var lastPosition *bnswap.Position
	if p, ok := bnswapPositions[nextPos.Symbol]; ok {
		p := p
		lastPosition = &p
	}
	bnswapPositions[nextPos.Symbol] = nextPos
	bnswapPositionsUpdated[nextPos.Symbol] = true
	if lastPosition == nil ||
		lastPosition.PositionAmt != nextPos.PositionAmt ||
		lastPosition.EntryPrice != nextPos.EntryPrice {
		//如果SPOT变仓，立刻调SWAP，如果SWAP变仓，等ORDER SILENT TIMEOUT
		bnswapOrderSilentTimes[nextPos.Symbol] = time.Now()
		logger.Debugf("SWAP HTTP POSITION %s", nextPos.ToString())
	}
}

func handleSwapHttpAccount(account bnswap.Account) {
	for _, asset := range account.Assets {
		if asset.Asset == "USDT" {
			asset := asset
			bnswapUSDTAsset = &asset
			bnswapAssetUpdatedForInflux = true
			bnswapAssetUpdatedForExternalInflux = true
			continue
		}
		if asset.Asset == "BNB" {
			asset := asset
			bnswapBNBAsset = &asset
			continue
		}
	}
}

func swapCreateOrder(
	ctx context.Context,
	credentials *bnswap.Credentials,
	api *bnswap.API,
	timeout time.Duration,
	params bnswap.NewOrderParams,
) {
	childCtx, _ := context.WithTimeout(ctx, timeout)
	order, err := api.SubmitOrder(childCtx, credentials, &params)
	if err != nil {
		logger.Debugf("SUBMIT ERROR %s  %v ", params.ToString(), err)
		select {
		case <-ctx.Done():
		case bnswapOrderNewErrorCh <- SwapOrderNewError{
			Error:  err,
			Params: params,
		}:
		}
	} else if order.Status == "FILLED" ||
		order.Status == "CANCELED" ||
		order.Status == "REJECTED" ||
		order.Status == "EXPIRED" {
		bnswapOrderFinishCh <- *order
	}
}

