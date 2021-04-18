package main

import (
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func handleTakerHttpPositions(positions []bnswap.Position) {
	for _, nextPos := range positions {
		if _, ok := tmSymbolsMap[nextPos.Symbol]; !ok {
			return
		}
		if nextPos.PositionSide != "BOTH" {
			return
		}
		if nextPos.UpdateTime.Sub(tLastOrderTimes[nextPos.Symbol]) < *mtConfig.PullInterval {
			return
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
			logger.Debugf("MAKER HTTP POSITION %s", nextPos.ToString())
		}
	}
}

func handleTakerHttpAccount(account bnswap.Account) {
	for _, asset := range account.Assets {
		if asset.Asset == "USDT" {
			asset := asset
			if tAccount == nil {
				logger.Debugf("TAKER HTTP AVAILABLE BALANCE %v -> %f", nil, *asset.AvailableBalance)
				mtLoopTimer.Reset(time.Nanosecond)
			} else if tAccount.AvailableBalance != nil &&
				asset.AvailableBalance != nil &&
				*tAccount.AvailableBalance != *asset.AvailableBalance {
				mtLoopTimer.Reset(time.Nanosecond)
				logger.Debugf("TAKER HTTP AVAILABLE BALANCE %f -> %f", *tAccount.AvailableBalance, *asset.AvailableBalance)
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
