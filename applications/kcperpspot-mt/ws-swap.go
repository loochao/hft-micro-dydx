package main

import (
	"github.com/geometrybase/hft-micro/kcperp"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func handleWSBalance(wsBalance *kcperp.WsBalanceEvent) {
	if kcperpUSDTAccount != nil {
		if wsBalance.Currency != nil && kcperpUSDTAccount.Currency == *wsBalance.Currency {
			if wsBalance.AvailableBalance != nil {
				logger.Debugf(
					"PERP WS AvailableBalance %f -> %f",
					kcperpUSDTAccount.AvailableBalance,
					*wsBalance.AvailableBalance,
				)
				kcperpUSDTAccount.AvailableBalance = *wsBalance.AvailableBalance
			}
			if wsBalance.OrderMargin != nil {
				logger.Debugf(
					"PERP WS AvailableBalance %f -> %f",
					kcperpUSDTAccount.OrderMargin,
					*wsBalance.OrderMargin,
				)
				kcperpUSDTAccount.OrderMargin = *wsBalance.OrderMargin
			}
			if wsBalance.HoldBalance != nil {
				logger.Debugf(
					"PERP WS AvailableBalance %f -> %f",
					kcperpUSDTAccount.FrozenFunds,
					*wsBalance.HoldBalance,
				)
				kcperpUSDTAccount.FrozenFunds = *wsBalance.HoldBalance
			}
		}
	}
}

func handleWSPosition(nextPos *kcperp.WSPosition) {
	if _, ok := kcpsSymbolsMap[nextPos.Symbol]; ok {
		if lastPos, ok := kcperpPositions[nextPos.Symbol]; ok && nextPos.EventTime.Sub(lastPos.EventTime) > 0 {
			if nextPos.CurrentQty != nil {
				lastPos.CurrentQty = *nextPos.CurrentQty
				logger.Debugf("PERP WS POS NEW QTY %v", lastPos.CurrentQty)
				if nextPos.AvgEntryPrice != nil {
					lastPos.AvgEntryPrice = *nextPos.AvgEntryPrice
				}
			}
			kcperpPositionsUpdateTimes[nextPos.Symbol] = time.Now()
		}
	}
}
