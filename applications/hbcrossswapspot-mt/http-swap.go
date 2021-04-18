package main

import (
	"github.com/geometrybase/hft-micro/hbcrossswap"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func handleSwapHttpPositions(positions []hbcrossswap.Position) {
	for _, nextPos := range positions {
		if _, ok := kcpsSymbolsMap[nextPos.Symbol]; !ok {
			continue
		}
		// 交易所前置机存在缓存
		// 如果有WS更新 或者 刚下过单，不使用HTTP拉过来的Position
		if hbcrossswapHttpPositionUpdateSilentTimes[nextPos.Symbol].Sub(time.Now()) > 0 {
			return
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
			lastPosition.CostOpen != nextPos.CostOpen {
			//如果SPOT变仓，立刻调SWAP，如果SWAP变仓，等ORDER SILENT TIMEOUT
			hbcrossswapOrderSilentTimes[nextPos.Symbol] = time.Now()
			logger.Debugf("SWAP HTTP POSITION %s DIRECTION %s SIZE %f COST OPEN %f", nextPos.Symbol, nextPos.Direction, nextPos.Volume, nextPos.CostOpen)
			hbLoopTimer.Reset(time.Nanosecond)
		} else if nextPos.Volume != 0 &&
			lastPosition.Direction != nextPos.Direction {
			hbLoopTimer.Reset(time.Nanosecond)
			logger.Debugf("SWAP HTTP POSITION %s DIRECTION %s SIZE %f COST OPEN %f", nextPos.Symbol, nextPos.Direction, nextPos.Volume, nextPos.CostOpen)
		}
	}
}

func handleSwapHttpAccount(account hbcrossswap.Account) {
	if hbcrossswapAccount == nil {
		logger.Debugf("SWAP HTTP USDT ACCOUNT MarginBalance nil -> %f", account.MarginBalance)
		//} else if hbcrossswapAccount.MarginBalance != account.MarginBalance{
		//	logger.Debugf("SWAP HTTP USDT ACCOUNT MarginBalance %f -> %f", hbcrossswapAccount.MarginBalance, account.MarginBalance)
	}
	hbcrossswapAccount = &account
	hbcrossswapAssetUpdatedForReBalance = true
	hbcrossswapAssetUpdatedForInflux = true
	hbcrossswapAssetUpdatedForExternalInflux = true
}
