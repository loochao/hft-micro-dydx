package main

import (
	"github.com/geometrybase/hft-micro/kcperp"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func handlePerpHttpPositions(positions []kcperp.Position) {
	for _, nextPos := range positions {
		if _, ok := kcpsSymbolsMap[nextPos.Symbol]; !ok {
			continue
		}
		if time.Now().Sub(kcperpHttpPositionUpdateSilentTimes[nextPos.Symbol]) < 0 {
			continue
		}
		var lastPosition *kcperp.Position
		if p, ok := kcperpPositions[nextPos.Symbol]; ok {
			p := p
			lastPosition = &p
		}
		kcperpPositions[nextPos.Symbol] = nextPos
		kcperpPositionsUpdateTimes[nextPos.Symbol] = time.Now()
		if lastPosition == nil ||
			lastPosition.CurrentQty != nextPos.CurrentQty ||
			lastPosition.AvgEntryPrice != nextPos.AvgEntryPrice {
			//如果SPOT变仓，立刻调PERP，如果PERP变仓，等ORDER SILENT TIMEOUT
			kcperpOrderSilentTimes[nextPos.Symbol] = time.Now()
			kcLoopTimer.Reset(time.Nanosecond)
			logger.Debugf("PERP HTTP POSITION SIZE %f PRICE %f", nextPos.CurrentQty, nextPos.AvgEntryPrice)
		}
	}
}

func handlePerpHttpAccount(account kcperp.Account) {
	if account.Currency == "USDT" {
		if kcperpUSDTAccount == nil ||
			kcperpUSDTAccount.AvailableBalance != account.AvailableBalance {
			logger.Debugf("PERP HTTP USDT ACCOUNT AvailableBalance %f MarginBalance %f", account.AvailableBalance, account.MarginBalance)
			kcLoopTimer.Reset(time.Nanosecond)
		}
		kcperpUSDTAccount = &account
	}
}

