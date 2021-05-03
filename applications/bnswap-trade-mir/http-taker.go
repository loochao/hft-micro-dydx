package main

import (
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func handleTakerHttpPositions(positions []bnswap.Position) {
	for _, nextPos := range positions {
		if _, ok := swapSymbolsMap[nextPos.Symbol]; !ok {
			return
		}
		if nextPos.PositionSide != "BOTH" {
			return
		}
		if time.Now().Sub(swapHttpPositionUpdateSilentTimes[nextPos.Symbol]) < 0 {
			continue
		}
		var lastPosition *bnswap.Position
		if p, ok := swapPositions[nextPos.Symbol]; ok {
			p := p
			lastPosition = p
		}
		nextPos := nextPos
		swapPositions[nextPos.Symbol] = &nextPos
		swapPositionsUpdateTimes[nextPos.Symbol] = time.Now()
		if lastPosition == nil ||
			lastPosition.PositionAmt != nextPos.PositionAmt ||
			lastPosition.EntryPrice != nextPos.EntryPrice {
			//如果SPOT变仓，立刻调SWAP，如果SWAP变仓，等ORDER SILENT TIMEOUT
			swapOrderSilentTimes[nextPos.Symbol] = time.Now()
		}
	}
}

func handleTakerHttpAccount(account bnswap.Account) {
	for _, asset := range account.Assets {
		if asset.Asset == "USDT" {
			asset := asset
			if swapAccount == nil {
				logger.Debugf("TAKER HTTP WB CHANGE %v -> %f", nil, *asset.WalletBalance)
				//swapLoopTimer.Reset(time.Nanosecond)
			} else if swapAccount.WalletBalance != nil &&
				asset.WalletBalance != nil &&
				*swapAccount.WalletBalance != *asset.WalletBalance {
				//swapLoopTimer.Reset(time.Nanosecond)
				logger.Debugf("TAKER HTTP WB CHANGE %f -> %f", *swapAccount.WalletBalance, *asset.WalletBalance)
			}
			swapAccount = &asset
			break
		}
		//if asset.Asset == "BNB" {
		//	asset := asset
		//	tBNBAsset = &asset
		//	continue
		//}
	}
}
