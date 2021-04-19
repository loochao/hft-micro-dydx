package main

import (
	"github.com/geometrybase/hft-micro/hbcrossswap"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"time"
)

func handleWSAccount(wsBalance *hbcrossswap.WSAccounts) {
	for _, account := range wsBalance.Accounts {
		if account.MarginAsset == "USDT" {
			account := account
			if hbcrossswapAccount == nil {
				logger.Debugf("SWAP WS USDT CHANGE MP nil -> %f MB nil -> %f", account.MarginPosition, account.MarginBalance)
				hbLoopTimer.Reset(time.Nanosecond)
			} else if hbcrossswapAccount.MarginBalance != account.MarginBalance {
				hbLoopTimer.Reset(time.Nanosecond)
				if math.Abs(hbcrossswapAccount.MarginPosition-account.MarginPosition) > *hbConfig.EnterMinimalStep*0.5 {
					logger.Debugf("SWAP WS USDT CHANGE MP %f -> %f MB %f -> %f ",
						hbcrossswapAccount.MarginPosition,
						account.MarginPosition,
						hbcrossswapAccount.MarginBalance,
						account.MarginBalance,
					)
				}
			}
			hbcrossswapAccount = &account
			return
		}
	}
}

func handleWSPosition(wsPositions *hbcrossswap.WSPositions) {
	for _, nextPos := range wsPositions.Positions {
		if nextPos.Direction != hbcrossswap.PositionDirectionSell {
			continue
		}
		if _, ok := kcpsSymbolsMap[nextPos.Symbol]; ok {
			if lastPos, ok := hbcrossswapPositions[nextPos.Symbol]; ok {
				if nextPos.Volume != lastPos.Volume {
					logger.Debugf("SWAP WS SELL POS %s %f -> %f", nextPos.Symbol, lastPos.Volume, nextPos.Volume)
				}
				hbLoopTimer.Reset(time.Nanosecond)
				nextPos := nextPos
				hbcrossswapPositions[nextPos.Symbol] = nextPos
				hbcrossswapPositionsUpdateTimes[nextPos.Symbol] = time.Now()
				hbcrossswapHttpPositionUpdateSilentTimes[nextPos.Symbol] = time.Now().Add(*hbConfig.PullInterval * 3)
			}
		}
	}
}
