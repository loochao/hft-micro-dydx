package main

import (
	"github.com/geometrybase/hft-micro/hbcrossswap"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func handleMakerHttpPositions(positions []hbcrossswap.Position) {
	for _, nextPos := range positions {
		if _, ok := mtSymbolsMap[nextPos.Symbol]; !ok {
			continue
		}
		var lastPosition *hbcrossswap.Position
		if p, ok := mPositions[nextPos.Symbol]; ok {
			p := p
			lastPosition = &p
		}
		mPositions[nextPos.Symbol] = nextPos
		mPositionsUpdateTimes[nextPos.Symbol] = time.Now()
		if lastPosition == nil ||
			lastPosition.Volume != nextPos.Volume ||
			lastPosition.CostOpen != nextPos.CostOpen {
			//如果SPOT变仓，立刻调MAKER，如果MAKER变仓，等ORDER SILENT TIMEOUT
			mOrderSilentTimes[nextPos.Symbol] = time.Now()
			logger.Debugf("MAKER HTTP POSITION %s DIRECTION %s SIZE %f COST OPEN %f", nextPos.Symbol, nextPos.Direction, nextPos.Volume, nextPos.CostOpen)
		} else if nextPos.Volume != 0 &&
			lastPosition.Direction != nextPos.Direction {
			logger.Debugf("MAKER HTTP POSITION %s DIRECTION %s SIZE %f COST OPEN %f", nextPos.Symbol, nextPos.Direction, nextPos.Volume, nextPos.CostOpen)
		}
	}
}

func handleMakerHttpAccount(account hbcrossswap.Account) {
	if mAccount == nil {
		logger.Debugf("MAKER HTTP USDT ACCOUNT MarginBalance nil -> %f", account.MarginBalance)
	//} else if mAccount.MarginBalance != account.MarginBalance {
	//	logger.Debugf("MAKER HTTP USDT ACCOUNT MarginBalance %f -> %f", mAccount.MarginBalance, account.MarginBalance)
	}
	mAccount = &account
}
