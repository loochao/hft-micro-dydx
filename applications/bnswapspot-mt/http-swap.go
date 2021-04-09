package main

import (
	"context"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func handleSwapHttpPositions(positions []bnswap.Position) {
	logger.Debugf("%v", positions)
	for _, nextPos := range positions {
		if _, ok := bnSymbolsMap[nextPos.Symbol]; !ok {
			return
		}
		if nextPos.PositionSide != "BOTH" {
			return
		}
		if nextPos.UpdateTime.Sub(bnswapLastOrderTimes[nextPos.Symbol]) < *bnConfig.PullInterval {
			return
		}
		var lastPosition *bnswap.Position
		if p, ok := bnswapPositions[nextPos.Symbol]; ok {
			p := p
			lastPosition = &p
		}
		bnswapPositions[nextPos.Symbol] = nextPos
		bnswapPositionsUpdateTimes[nextPos.Symbol] = time.Now()
		if lastPosition == nil ||
			lastPosition.PositionAmt != nextPos.PositionAmt ||
			lastPosition.EntryPrice != nextPos.EntryPrice {
			//如果SPOT变仓，立刻调SWAP，如果SWAP变仓，等ORDER SILENT TIMEOUT
			bnswapOrderSilentTimes[nextPos.Symbol] = time.Now()
			logger.Debugf("SWAP HTTP POSITION %s", nextPos.ToString())
		}
	}
}

func handleSwapHttpAccount(account bnswap.Account) {
	for _, asset := range account.Assets {
		if asset.Asset == "USDT" {
			asset := asset
			bnswapUSDTAsset = &asset
			bnswapAssetUpdatedForReBalance = true
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
	api *bnswap.API,
	timeout time.Duration,
	params bnswap.NewOrderParams,
) {
	childCtx, _ := context.WithTimeout(ctx, timeout)
	order, err := api.SubmitOrder(childCtx, params)
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
