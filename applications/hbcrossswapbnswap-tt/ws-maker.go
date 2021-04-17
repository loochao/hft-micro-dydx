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
			if mAccount == nil {
				logger.Debugf("SWAP WS USDT CHANGE MARGIN BALANCE %f", account.MarginBalance)
				//} else if account.WithdrawAvailable != mAccount.WithdrawAvailable {
				//	logger.Debugf("SWAP WS USDT CHANGE WithdrawAvailable %f -> %f", mAccount.WithdrawAvailable, account.WithdrawAvailable)
			}
			mAccount = &account
			return
		}
	}
}

func handleWSPosition(wsPositions *hbcrossswap.WSPositions) {
	for _, nextPos := range wsPositions.Positions {
		if _, ok := mtSymbolsMap[nextPos.Symbol]; ok {
			if lastPos, ok := mPositions[nextPos.Symbol]; ok {
				if nextPos.Volume != lastPos.Volume {
					logger.Debugf("SWAP WS POS %s %s %f -> %s %f", nextPos.Symbol, lastPos.Direction, lastPos.Volume, nextPos.Direction, nextPos.Volume)
				} else if nextPos.Volume != 0 && nextPos.Direction != lastPos.Direction {
					logger.Debugf("SWAP WS POS %s %s %f -> %s %f", nextPos.Symbol, lastPos.Direction, lastPos.Volume, nextPos.Direction, nextPos.Volume)
				}
				nextPos := nextPos
				mPositions[nextPos.Symbol] = nextPos
				mPositionsUpdateTimes[nextPos.Symbol] = time.Now()
			}
		}
	}
}
