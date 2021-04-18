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
				//} else if account.WithdrawAvailable != hbcrossswapAccount.WithdrawAvailable {
				//	logger.Debugf("SWAP WS USDT CHANGE WithdrawAvailable %f -> %f", hbcrossswapAccount.WithdrawAvailable, account.WithdrawAvailable)
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
				if nextPos.Volume != lastPos.Volume {
					logger.Debugf("SWAP WS POS %s %s %f -> %s %f", nextPos.Symbol, lastPos.Direction, lastPos.Volume, nextPos.Direction, nextPos.Volume)
				} else if nextPos.Volume != 0 && nextPos.Direction != lastPos.Direction {
					logger.Debugf("SWAP WS POS %s %s %f -> %s %f", nextPos.Symbol, lastPos.Direction, lastPos.Volume, nextPos.Direction, nextPos.Volume)
				}
				nextPos := nextPos
				hbcrossswapPositions[nextPos.Symbol] = nextPos
				hbcrossswapPositionsUpdateTimes[nextPos.Symbol] = time.Now()
				hbcrossswapHttpPositionUpdateSilentTimes[nextPos.Symbol] = time.Now().Add(*hbConfig.PullInterval*3)
			}
		}
	}
}
