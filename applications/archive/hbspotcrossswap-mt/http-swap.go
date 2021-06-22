package main

import (
	"github.com/geometrybase/hft-micro/hbcrossswap"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func handleSwapHttpPositions(positions []hbcrossswap.Position) {
	hasPositions := make(map[string]bool)
	for _, makerSymbol := range hbcrossswapSymbols {
		hasPositions[makerSymbol] = false
	}
	for _, nextPos := range positions {
		if _, ok := hbSwapSpotSymbolsMap[nextPos.Symbol]; !ok {
			continue
		}
		if nextPos.Direction != hbcrossswap.PositionDirectionSell {
			continue
		}
		hasPositions[nextPos.Symbol] = true

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
			logger.Debugf("SWAP HTTP SELL POSITION %s SIZE %f COST OPEN %f", nextPos.Symbol, nextPos.Volume, nextPos.CostOpen)
			hbLoopTimer.Reset(time.Nanosecond)
		}
	}
	for symbol, has := range hasPositions {
		if has {
			continue
		}
		nextPos := hbcrossswap.Position{
			Symbol: symbol,
			Direction: hbcrossswap.PositionDirectionSell,
		}
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
			logger.Debugf("SWAP HTTP SELL POSITION %s SIZE %f COST OPEN %f", nextPos.Symbol, nextPos.Volume, nextPos.CostOpen)
			hbLoopTimer.Reset(time.Nanosecond)
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
	hbcrossswapAssetUpdatedForInflux = true
	hbcrossswapAssetUpdatedForExternalInflux = true
}
