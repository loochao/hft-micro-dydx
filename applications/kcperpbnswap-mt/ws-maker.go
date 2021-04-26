package main

import (
	"github.com/geometrybase/hft-micro/kcperp"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"time"
)

func handleMakerWSAccount(wsBalance *kcperp.WSAccounts) {
	for _, account := range wsBalance.Accounts {
		if account.MarginAsset == "USDT" {
			account := account
			if mAccount == nil {
				logger.Debugf("MAKER WS USDT CHANGE WA nil -> %f MB nil -> %f", account.WithdrawAvailable, account.MarginBalance)
				mtLoopTimer.Reset(time.Nanosecond)
			} else if mAccount.MarginBalance != account.MarginBalance {
				mtLoopTimer.Reset(time.Nanosecond)
				if math.Abs(mAccount.MarginPosition-account.MarginPosition) > *mtConfig.EnterMinimalStep*0.5 {
					logger.Debugf("MAKER WS USDT CHANGE WA %f -> %f MB %f -> %f ",
						mAccount.WithdrawAvailable,
						account.WithdrawAvailable,
						mAccount.MarginBalance,
						account.MarginBalance,
					)
				}
			}
			mAccount = &account
			return
		}
	}
}

func handleMakerWSPosition(wsPositions *kcperp.WSPositions) {
	for _, nextPos := range wsPositions.Positions {
		if takerSymbol, ok := mtSymbolsMap[nextPos.Symbol]; ok {
			if nextPos.Direction == kcperp.PositionDirectionBuy {
				if lastPos, ok := mPositions[nextPos.Symbol]; ok {
					mHttpPositionUpdateSilentTimes[nextPos.Symbol] = time.Now().Add(*mtConfig.HttpSilent)
					if nextPos.Volume != lastPos.Volume {
						logger.Debugf("MAKER WS BUY POS %s %f -> %f", nextPos.Symbol, lastPos.Volume, nextPos.Volume)
						if nextPos.Volume != 0 {
							logger.Debugf("MAKER ENTER SILENT %v", *mtConfig.EnterSilent)
							mSilentTimes[nextPos.Symbol] = time.Now().Add(*mtConfig.EnterSilent)
						}
						tOrderSilentTimes[takerSymbol] = time.Now()
						mtLoopTimer.Reset(time.Nanosecond)
					}
					nextPos := nextPos
					mPositions[nextPos.Symbol] = nextPos
					mPositionsUpdateTimes[nextPos.Symbol] = time.Now()
				}
			} else {
				if lastPos, ok := mSellPositions[nextPos.Symbol]; ok {
					mHttpPositionUpdateSilentTimes[nextPos.Symbol] = time.Now().Add(*mtConfig.HttpSilent)
					if nextPos.Volume != lastPos.Volume {
						logger.Debugf("MAKER WS SELL POS %s %f -> %f", nextPos.Symbol, lastPos.Volume, nextPos.Volume)
						if nextPos.Volume != 0 {
							logger.Debugf("MAKER ENTER SILENT %v", *mtConfig.EnterSilent)
							mSilentTimes[nextPos.Symbol] = time.Now().Add(*mtConfig.EnterSilent)
						}
						tOrderSilentTimes[takerSymbol] = time.Now()
						mtLoopTimer.Reset(time.Nanosecond)
					}
					nextPos := nextPos
					mSellPositions[nextPos.Symbol] = nextPos
					mPositionsUpdateTimes[nextPos.Symbol] = time.Now()
				}
			}
		}
	}
}
