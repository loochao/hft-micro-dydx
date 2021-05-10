package main

import (
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func handleSwapHttpPositions(positions []bnswap.Position) {
	for _, nextPos := range positions {
		if _, ok := bnSymbolsMap[nextPos.Symbol]; !ok {
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
		bnswapPositionsUpdateTimes[nextPos.Symbol] = time.Now()
		if lastPosition == nil ||
			lastPosition.PositionAmt != nextPos.PositionAmt ||
			lastPosition.EntryPrice != nextPos.EntryPrice {
			//logger.Debugf("%s HTTP POSITION %s", nextPos.Market,nextPos.ToString())
			if lastPosition != nil && lastPosition.PositionAmt*nextPos.PositionAmt < 0 {
				if lastPosition.PositionAmt > 0 {
					bnRealisedPnl[nextPos.Symbol] = (nextPos.EntryPrice - lastPosition.EntryPrice)/lastPosition.EntryPrice
					logger.Debugf("%s CLOSE LONG PNL %f", nextPos.Symbol, bnRealisedPnl[nextPos.Symbol])
				}else{
					bnRealisedPnl[nextPos.Symbol] = (lastPosition.EntryPrice - nextPos.EntryPrice)/lastPosition.EntryPrice
					logger.Debugf("%s CLOSE SHORT PNL %f", nextPos.Symbol, bnRealisedPnl[nextPos.Symbol])
				}
			}
		}
	}
}

func handleSwapHttpAccount(account bnswap.Account) {
	for _, asset := range account.Assets {
		if asset.Asset == "USDT" {
			asset := asset
			bnswapUSDTAsset = &asset
			continue
		}
		if asset.Asset == "BNB" {
			asset := asset
			bnswapBNBAsset = &asset
			continue
		}
	}
}

