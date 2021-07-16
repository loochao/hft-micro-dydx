package main

import (
	"github.com/geometrybase/hft-micro/huobi-usdtfuture"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"time"
)

func handleMakerHttpPositions(positions []huobi_usdtfuture.Position) {
	hasBuyPositions := make(map[string]bool)
	hasSellPositions := make(map[string]bool)
	for _, makerSymbol := range mSymbols {
		hasBuyPositions[makerSymbol] = false
		hasSellPositions[makerSymbol] = false
	}
	for _, nextPos := range positions {
		if takerSymbol, ok := mtSymbolsMap[nextPos.Symbol]; ok {
			if nextPos.Direction == huobi_usdtfuture.PositionDirectionBuy {
				hasBuyPositions[nextPos.Symbol] = true
			} else {
				hasSellPositions[nextPos.Symbol] = true
			}
			if time.Now().Sub(mHttpPositionUpdateSilentTimes[nextPos.Symbol]) < 0 {
				continue
			}
			var lastPosition *huobi_usdtfuture.Position
			if nextPos.Direction == huobi_usdtfuture.PositionDirectionBuy {
				if p, ok := mBuyPositions[nextPos.Symbol]; ok {
					p := p
					lastPosition = &p
				}
				mBuyPositions[nextPos.Symbol] = nextPos
				mPositionsUpdateTimes[nextPos.Symbol] = time.Now()
				if lastPosition == nil ||
					lastPosition.Volume != nextPos.Volume ||
					lastPosition.CostOpen != nextPos.CostOpen {
					mtLoopTimer.Reset(time.Nanosecond)
					tOrderSilentTimes[takerSymbol] = time.Now()
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
				mPositionsUpdateTimes[nextPos.Symbol] = time.Now()
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
	}
	for makerSymbol, hasPosition := range hasBuyPositions {
		if hasPosition {
			continue
		}
		nextPos := huobi_usdtfuture.Position{Symbol: makerSymbol, Direction: huobi_usdtfuture.PositionDirectionBuy}
		var lastPosition *huobi_usdtfuture.Position
		if p, ok := mBuyPositions[nextPos.Symbol]; ok {
			p := p
			lastPosition = &p
		}
		mBuyPositions[nextPos.Symbol] = nextPos
		mPositionsUpdateTimes[nextPos.Symbol] = time.Now()
		if lastPosition == nil ||
			lastPosition.Volume != nextPos.Volume ||
			lastPosition.CostOpen != nextPos.CostOpen {
			mtLoopTimer.Reset(time.Nanosecond)
			tOrderSilentTimes[mtSymbolsMap[makerSymbol]] = time.Now()
			logger.Debugf("MAKER HTTP BUY POSITION %s SIZE %f COST OPEN %f", nextPos.Symbol, nextPos.Volume, nextPos.CostOpen)
			if lastPosition != nil && nextPos.Volume != 0 {
				logger.Debugf("MAKER ENTER SILENT %v", *mtConfig.EnterSilent)
				mSilentTimes[nextPos.Symbol] = time.Now().Add(*mtConfig.EnterSilent)
			}
		}
	}
	for makerSymbol, hasPosition := range hasSellPositions {
		if hasPosition {
			continue
		}
		nextPos := huobi_usdtfuture.Position{Symbol: makerSymbol, Direction: huobi_usdtfuture.PositionDirectionSell}
		var lastPosition *huobi_usdtfuture.Position
		if p, ok := mSellPositions[nextPos.Symbol]; ok {
			p := p
			lastPosition = &p
		}
		mSellPositions[nextPos.Symbol] = nextPos
		mPositionsUpdateTimes[nextPos.Symbol] = time.Now()
		if lastPosition == nil ||
			lastPosition.Volume != nextPos.Volume ||
			lastPosition.CostOpen != nextPos.CostOpen {
			mtLoopTimer.Reset(time.Nanosecond)
			tOrderSilentTimes[mtSymbolsMap[makerSymbol]] = time.Now()
			logger.Debugf("MAKER HTTP SELL POSITION %s SIZE %f COST OPEN %f", nextPos.Symbol, nextPos.Volume, nextPos.CostOpen)
			if lastPosition != nil && nextPos.Volume != 0 {
				logger.Debugf("MAKER ENTER SILENT %v", *mtConfig.EnterSilent)
				mSilentTimes[nextPos.Symbol] = time.Now().Add(*mtConfig.EnterSilent)
			}
		}
	}
}

func handleMakerHttpAccount(account huobi_usdtfuture.Account) {
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
