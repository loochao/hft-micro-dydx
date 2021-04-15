package main

import (
	"github.com/geometrybase/hft-micro/hbcrossswap"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func handleWSAccount(wsBalance *hbcrossswap.WSAccounts) {
	for _, account := range wsBalance.Accounts {
		if account.MarginAsset == "USDT" {
			account := account
			if hbcrossswapAccount == nil {
				logger.Debugf("SWAP WS USDT CHANGE MARGIN BALANCE %f", account.MarginBalance)
			} else if account.MarginBalance != hbcrossswapAccount.MarginBalance {
				logger.Debugf("SWAP WS USDT CHANGE MARGIN BALANCE %f -> %f", hbcrossswapAccount.MarginBalance, account.MarginBalance)
			}
			hbcrossswapAccount = &account
			return
		}
	}
}

func handleWSPosition(wsPositions *hbcrossswap.WSPositions) {
	for _, nextPos := range wsPositions.Positions {
		if _, ok := kcpsSymbolsMap[nextPos.Symbol]; ok {
			if lastPos, ok := hbcrossswapPositions[nextPos.Symbol]; ok {
				if nextPos.Volume != lastPos.Volume || nextPos.Direction != lastPos.Direction {
					logger.Debugf("SWAP WS POS %s %s %d -> %s %d", nextPos.Symbol, lastPos.Direction, lastPos.Volume, nextPos.Direction, nextPos.Volume)
				}
				hbcrossswapPositions[nextPos.Symbol] = lastPos
				hbcrossswapPositionsUpdateTimes[nextPos.Symbol] = time.Now()
			}
		}
	}
}
