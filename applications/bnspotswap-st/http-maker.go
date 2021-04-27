package main

import (
	"github.com/geometrybase/hft-micro/bnspot"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func handleMakerHttpPositions(positions []bnspot.Position) {
	hasBuyPositions := make(map[string]bool)
	hasSellPositions := make(map[string]bool)
	for _, makerSymbol := range mSymbols {
		hasBuyPositions[makerSymbol] = false
		hasSellPositions[makerSymbol] = false
	}
	for _, nextPos := range positions {
		if takerSymbol, ok := mtSymbolsMap[nextPos.Symbol]; ok {
			if time.Now().Sub(mHttpPositionUpdateSilentTimes[nextPos.Symbol]) < 0 {
				continue
			}
			var lastPosition *bnspot.Position
			if p, ok := mPositions[nextPos.Symbol]; ok {
				p := p
				lastPosition = &p
			}
			mPositions[nextPos.Symbol] = nextPos
			mPositionsUpdateTimes[nextPos.Symbol] = time.Now()
			if lastPosition == nil ||
				lastPosition.CurrentQty != nextPos.CurrentQty ||
				lastPosition.AvgEntryPrice != nextPos.AvgEntryPrice {
				tOrderSilentTimes[takerSymbol] = time.Now()
				mtLoopTimer.Reset(time.Nanosecond)
				logger.Debugf("MAKER HTTP POSITION %s SIZE %f PRICE %f", nextPos.Symbol, nextPos.CurrentQty, nextPos.AvgEntryPrice)
				if lastPosition != nil && nextPos.CurrentQty != 0 {
					logger.Debugf("MAKER ENTER SILENT %v", *mtConfig.EnterSilent)
					mSilentTimes[nextPos.Symbol] = time.Now().Add(*mtConfig.EnterSilent)
				}
			}
		}
	}
}

func handleMakerHttpAccount(account bnspot.Account) {
	if account.Currency == "USDT" {
		if mAccount == nil ||
			mAccount.AvailableBalance != account.AvailableBalance {
			logger.Debugf("MAKER HTTP USDT ACCOUNT AvailableBalance %f MarginBalance %f", account.AvailableBalance, account.MarginBalance)
			mtLoopTimer.Reset(time.Nanosecond)
		}
		mAccount = &account
	}
}
