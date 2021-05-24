package main

import (
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func handleSwapHttpAccount(account bnswap.Account) {
	for _, asset := range account.Assets {
		if asset.Asset == "USDT" {
			bnswapAssetUpdatedForReBalance = true
			bnswapAssetUpdatedForInflux = true
			bnswapAssetUpdatedForExternalInflux = true
			if bnswapUSDTAsset != nil &&
				bnswapUSDTAsset.EventTime.Sub(asset.EventTime) > 0 {
				logger.Debugf("%v is older than USDT %v", asset, bnswapUSDTAsset.EventTime)
				continue
			}
			asset := asset
			bnswapUSDTAsset = &asset
			bnLoopTimer.Reset(time.Nanosecond)
			continue
		}
		if asset.Asset == "BNB" {
			if bnswapBNBAsset != nil &&
				bnswapBNBAsset.EventTime.Sub(asset.EventTime) > 0 {
				logger.Debugf("%v is older than USDT %v", asset, bnswapBNBAsset.EventTime)
				continue
			}
			asset := asset
			bnswapBNBAsset = &asset
			continue
		}
	}
	logger.Debugf("%d", len(account.Positions))
	for _, nextPos := range account.Positions {
		logger.Debugf("%s %v", nextPos.Symbol, nextPos.EventTime)
		if _, ok := bnspotOffsets[nextPos.Symbol]; !ok {
			continue
		}
		if nextPos.PositionSide != "BOTH" {
			continue
		}
		//if bnswapHttpPositionUpdateSilentTimes[nextPos.Symbol].Sub(nextPos.ParseTime) > 0 {
		//	continue
		//}
		bnswapPositions[nextPos.Symbol] = nextPos
		bnswapPositionsUpdateTimes[nextPos.Symbol] = time.Now()

		var lastPosition *bnswap.Position
		if currentPosition, ok := bnswapPositions[nextPos.Symbol]; ok {
			if currentPosition.EventTime.Sub(nextPos.EventTime) > 0 {
				logger.Debugf("%s nextPos EventTime is older %v < %v", nextPos.EventTime, currentPosition.EventTime)
				continue
			}
			lastPosition = &bnswap.Position{}
			*lastPosition = currentPosition
		}
		if lastPosition == nil ||
			lastPosition.PositionAmt != nextPos.PositionAmt ||
			lastPosition.EntryPrice != nextPos.EntryPrice {
			//如果SPOT变仓，立刻调SWAP，如果SWAP变仓，等ORDER SILENT TIMEOUT
			bnLoopTimer.Reset(time.Nanosecond)
			//bnswapOrderSilentTimes[nextPos.Symbol] = time.Now()
			logger.Debugf("SWAP HTTP POSITION %s", nextPos.ToString())
			if lastPosition != nil {
				logger.Debugf("SWAP %s POS OLD TIME %v NEW TIME %v", nextPos.Symbol, lastPosition.EventTime, nextPos.EventTime)
			}
		}
	}
}

