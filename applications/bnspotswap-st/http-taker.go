package main

import (
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func handleTakerHttpPositions(positions []bnswap.Position) {
	for _, nextPos := range positions {
		if _, ok := mtSymbolsMap[nextPos.Symbol]; !ok {
			return
		}
		if nextPos.PositionSide != "BOTH" {
			return
		}
		if time.Now().Sub(tHttpPositionUpdateSilentTimes[nextPos.Symbol]) < 0 {
			continue
		}
		var lastPosition *bnswap.Position
		if p, ok := tPositions[nextPos.Symbol]; ok {
			p := p
			lastPosition = p
		}
		nextPos := nextPos
		tPositions[nextPos.Symbol] = &nextPos
		tPositionsUpdateTimes[nextPos.Symbol] = time.Now()
		if lastPosition == nil ||
			lastPosition.PositionAmt != nextPos.PositionAmt ||
			lastPosition.EntryPrice != nextPos.EntryPrice {
			//如果SPOT变仓，立刻调SWAP，如果SWAP变仓，等ORDER SILENT TIMEOUT
			tOrderSilentTimes[nextPos.Symbol] = time.Now()
			tEnterTimeouts[nextPos.Symbol] = time.Now()
			//logger.Debugf("TAKER HTTP POSITION %s", nextPos.ToString())
			if nextPos.PositionAmt != 0 {
				tCloseTimeouts[nextPos.Symbol] = time.Now().Add(*mtConfig.CloseTimeout)
				logger.Debugf("SET CLOSE TIMEOUT FOR %s", nextPos.Symbol)
			}

		}
	}
}

func handleTakerHttpAccount(account bnswap.Account) {
	for _, asset := range account.Assets {
		if asset.Asset == "USDT" {
			asset := asset
			if tAccount == nil {
				logger.Debugf("TAKER HTTP WB CHANGE %v -> %f", nil, *asset.WalletBalance)
				//mtLoopTimer.Reset(time.Nanosecond)
			} else if tAccount.WalletBalance != nil &&
				asset.WalletBalance != nil &&
				*tAccount.WalletBalance != *asset.WalletBalance {
				//mtLoopTimer.Reset(time.Nanosecond)
				logger.Debugf("TAKER HTTP WB CHANGE %f -> %f", *tAccount.WalletBalance, *asset.WalletBalance)
			}
			tAccount = &asset
			break
		}
		//if asset.Asset == "BNB" {
		//	asset := asset
		//	tBNBAsset = &asset
		//	continue
		//}
	}
}
