package main

import (
	"github.com/geometrybase/hft-micro/hbcrossswap"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"time"
)

func handleMakerHttpPositions(positions []hbcrossswap.Position) {
	for _, nextPos := range positions {
		if _, ok := mtSymbolsMap[nextPos.Symbol]; !ok {
			continue
		}
		if time.Now().Sub(mHttpPositionUpdateSilentTimes[nextPos.Symbol]) < 0 {
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
			mtLoopTimer.Reset(time.Nanosecond)
			tOrderSilentTimes[nextPos.Symbol] = time.Now()
			logger.Debugf("MAKER HTTP POSITION %s DIRECTION %s SIZE %f COST OPEN %f", nextPos.Symbol, nextPos.Direction, nextPos.Volume, nextPos.CostOpen)
			if lastPosition != nil && nextPos.Volume != 0 {
				logger.Debugf("MAKER ENTER SILENT %v", *mtConfig.EnterSilent)
				mSilentTimes[nextPos.Symbol] = time.Now().Add(*mtConfig.EnterSilent)
			}
		} else if nextPos.Volume != 0 &&
			lastPosition.Direction != nextPos.Direction {
			mtLoopTimer.Reset(time.Nanosecond)
			tOrderSilentTimes[nextPos.Symbol] = time.Now()
			if nextPos.Volume != 0 {
				logger.Debugf("MAKER ENTER SILENT %v", *mtConfig.EnterSilent)
				mSilentTimes[nextPos.Symbol] = time.Now().Add(*mtConfig.EnterSilent)
			}
			logger.Debugf("MAKER HTTP POSITION %s DIRECTION %s SIZE %f COST OPEN %f", nextPos.Symbol, nextPos.Direction, nextPos.Volume, nextPos.CostOpen)
		}
	}
}

func handleMakerHttpAccount(account hbcrossswap.Account) {
	if mAccount == nil {
		mtLoopTimer.Reset(time.Nanosecond)
		logger.Debugf("MAKER HTTP USDT CHANGE MB nil -> %f", account.MarginBalance)
	} else if mAccount.MarginBalance != account.MarginBalance {
		mtLoopTimer.Reset(time.Nanosecond)
		if math.Abs(mAccount.MarginPosition - account.MarginPosition) > *mtConfig.EnterMinimalStep*0.5 {
			logger.Debugf("MAKER HTTP USDT CHANGE WA %f -> %f MB %f -> %f ",
				mAccount.WithdrawAvailable,
				account.WithdrawAvailable,
				mAccount.MarginBalance,
				account.MarginBalance,
			)
		}
	}
	mAccount = &account
}
