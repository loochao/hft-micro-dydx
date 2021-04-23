package main

import (
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func handleTakerHttpPositions(positions []bnswap.Position) {
	for _, nextPos := range positions {
		if _, ok := bnSymbolsMap[nextPos.Symbol]; !ok {
			return
		}
		if nextPos.PositionSide != "BOTH" {
			return
		}
		if time.Now().Sub(bnHttpPositionUpdateSilentTimes[nextPos.Symbol]) < 0 {
			continue
		}
		var lastPosition *bnswap.Position
		if p, ok := bnPositions[nextPos.Symbol]; ok {
			p := p
			lastPosition = p
		}
		nextPos := nextPos
		bnPositions[nextPos.Symbol] = &nextPos
		bnPositionsUpdateTimes[nextPos.Symbol] = time.Now()
		if lastPosition == nil ||
			lastPosition.PositionAmt != nextPos.PositionAmt ||
			lastPosition.EntryPrice != nextPos.EntryPrice {
			//如果SPOT变仓，立刻调SWAP，如果SWAP变仓，等ORDER SILENT TIMEOUT
			bnOrderSilentTimes[nextPos.Symbol] = time.Now()
			logger.Debugf("BNSWAP HTTP POSITION %s", nextPos.ToString())
		}
	}
}

func handleTakerHttpAccount(account bnswap.Account) {
	for _, asset := range account.Assets {
		if asset.Asset == "USDT" {
			asset := asset
			if bnAccount == nil {
				logger.Debugf("BNSWAP HTTP WB CHANGE %v -> %f", nil, *asset.WalletBalance)
				bnLoopTimer.Reset(time.Nanosecond)
			} else if bnAccount.WalletBalance != nil &&
				asset.WalletBalance != nil &&
				*bnAccount.WalletBalance != *asset.WalletBalance {
				bnLoopTimer.Reset(time.Nanosecond)
				logger.Debugf("BNSWAP HTTP WB CHANGE %f -> %f", *bnAccount.WalletBalance, *asset.WalletBalance)
			}
			bnAccount = &asset
			break
		}
		//if asset.Asset == "BNB" {
		//	asset := asset
		//	tBNBAsset = &asset
		//	continue
		//}
	}
}
