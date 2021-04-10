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
			logger.Debugf("%s HTTP POSITION %s", nextPos.Symbol,nextPos.ToString())
			//如果开仓立即挂平仓单, 如果平仓至少等一个OrderInterval
			if nextPos.PositionAmt != 0 {
				bnswapOrderSilentTimes[nextPos.Symbol] = time.Now()
			}else {
				bnswapOrderSilentTimes[nextPos.Symbol] = bnswapLastOrderTimes[nextPos.Symbol].Add(*bnConfig.OrderInterval)
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

