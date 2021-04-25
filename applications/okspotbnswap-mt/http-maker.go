package main

import (
	"github.com/geometrybase/hft-micro/hbcrossswap"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"time"
)

func handleMakerHttpPositions(positions []hbcrossswap.Position) {
	hasBuyPositions := make(map[string]bool)
	hasSellPositions := make(map[string]bool)
	for _, makerSymbol := range mSymbols {
		hasBuyPositions[makerSymbol] = 	false
		hasSellPositions[makerSymbol] = false
	}
	for _, nextPos := range positions {
		if _, ok := mtSymbolsMap[nextPos.Symbol]; !ok {
			continue
		}
		if nextPos.Direction == hbcrossswap.PositionDirectionBuy {
			hasBuyPositions[nextPos.Symbol] = true
		}else{
			hasSellPositions[nextPos.Symbol] = true
		}
		if time.Now().Sub(mHttpPositionUpdateSilentTimes[nextPos.Symbol]) < 0 {
			continue
		}
		var lastPosition *hbcrossswap.Position
		if nextPos.Direction == hbcrossswap.PositionDirectionBuy {
			if p, ok := mBalances[nextPos.Symbol]; ok {
				p := p
				lastPosition = &p
			}
			mBalances[nextPos.Symbol] = nextPos
			mBalancesUpdateTimes[nextPos.Symbol] = time.Now()
			if lastPosition == nil ||
				lastPosition.Volume != nextPos.Volume ||
				lastPosition.CostOpen != nextPos.CostOpen {
				mtLoopTimer.Reset(time.Nanosecond)
				tOrderSilentTimes[nextPos.Symbol] = time.Now()
				logger.Debugf("MAKER HTTP BUY POSITION %s SIZE %f COST OPEN %f", nextPos.Symbol, nextPos.Volume, nextPos.CostOpen)
				if lastPosition != nil && nextPos.Volume != 0 {
					logger.Debugf("MAKER ENTER SILENT %v", *mtConfig.EnterSilent)
					mSilentTimes[nextPos.Symbol] = time.Now().Add(*mtConfig.EnterSilent)
				}
			}
		} else {
			if p, ok := mSellPositions[nextPos.Symbol]; ok {
				p := p
				lastPosition = &p
			}
			mSellPositions[nextPos.Symbol] = nextPos
			mBalancesUpdateTimes[nextPos.Symbol] = time.Now()
			if lastPosition == nil ||
				lastPosition.Volume != nextPos.Volume ||
				lastPosition.CostOpen != nextPos.CostOpen {
				mtLoopTimer.Reset(time.Nanosecond)
				tOrderSilentTimes[nextPos.Symbol] = time.Now()
				logger.Debugf("MAKER HTTP SELL POSITION %s SIZE %f COST OPEN %f", nextPos.Symbol, nextPos.Volume, nextPos.CostOpen)
				if lastPosition != nil && nextPos.Volume != 0 {
					logger.Debugf("MAKER ENTER SILENT %v", *mtConfig.EnterSilent)
					mSilentTimes[nextPos.Symbol] = time.Now().Add(*mtConfig.EnterSilent)
				}
			}
		}
	}
	for takerSymbol, hasPosition := range hasBuyPositions {
		if hasPosition {
			continue
		}
		nextPos := hbcrossswap.Position{Symbol: takerSymbol, Direction: hbcrossswap.PositionDirectionBuy}
		var lastPosition *hbcrossswap.Position
		if p, ok := mBalances[nextPos.Symbol]; ok {
			p := p
			lastPosition = &p
		}
		mBalances[nextPos.Symbol] = nextPos
		mBalancesUpdateTimes[nextPos.Symbol] = time.Now()
		if lastPosition == nil ||
			lastPosition.Volume != nextPos.Volume ||
			lastPosition.CostOpen != nextPos.CostOpen {
			mtLoopTimer.Reset(time.Nanosecond)
			tOrderSilentTimes[nextPos.Symbol] = time.Now()
			logger.Debugf("MAKER HTTP BUY POSITION %s SIZE %f COST OPEN %f", nextPos.Symbol, nextPos.Volume, nextPos.CostOpen)
			if lastPosition != nil && nextPos.Volume != 0 {
				logger.Debugf("MAKER ENTER SILENT %v", *mtConfig.EnterSilent)
				mSilentTimes[nextPos.Symbol] = time.Now().Add(*mtConfig.EnterSilent)
			}
		}
	}
	for takerSymbol, hasPosition := range hasSellPositions {
		if hasPosition {
			continue
		}
		nextPos := hbcrossswap.Position{Symbol: takerSymbol, Direction: hbcrossswap.PositionDirectionSell}
		var lastPosition *hbcrossswap.Position
		if p, ok := mSellPositions[nextPos.Symbol]; ok {
			p := p
			lastPosition = &p
		}
		mSellPositions[nextPos.Symbol] = nextPos
		mBalancesUpdateTimes[nextPos.Symbol] = time.Now()
		if lastPosition == nil ||
			lastPosition.Volume != nextPos.Volume ||
			lastPosition.CostOpen != nextPos.CostOpen {
			mtLoopTimer.Reset(time.Nanosecond)
			tOrderSilentTimes[nextPos.Symbol] = time.Now()
			logger.Debugf("MAKER HTTP SELL POSITION %s SIZE %f COST OPEN %f", nextPos.Symbol, nextPos.Volume, nextPos.CostOpen)
			if lastPosition != nil && nextPos.Volume != 0 {
				logger.Debugf("MAKER ENTER SILENT %v", *mtConfig.EnterSilent)
				mSilentTimes[nextPos.Symbol] = time.Now().Add(*mtConfig.EnterSilent)
			}
		}
	}
}

func handleMakerHttpAccounts(account hbcrossswap.Account) {
	if mAccount == nil {
		mtLoopTimer.Reset(time.Nanosecond)
		logger.Debugf("MAKER HTTP USDT CHANGE MB nil -> %f", account.MarginBalance)
	} else if mAccount.MarginBalance != account.MarginBalance {
		mtLoopTimer.Reset(time.Nanosecond)
		if math.Abs(mAccount.MarginPosition-account.MarginPosition) > *mtConfig.EnterMinimalStep*0.5 {
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
