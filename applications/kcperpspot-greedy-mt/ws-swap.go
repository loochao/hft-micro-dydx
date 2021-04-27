package main

import (
	"github.com/geometrybase/hft-micro/kcperp"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func handlePerpWSBalance(wsBalance *kcperp.WsBalanceEvent) {
	if kcperpUSDTAccount != nil {
		if wsBalance.Currency != nil && kcperpUSDTAccount.Currency == *wsBalance.Currency {
			if wsBalance.AvailableBalance != nil && kcperpUSDTAccount.AvailableBalance != *wsBalance.AvailableBalance {
				//logger.Debugf(
				//	"PERP WS AvailableBalance %f -> %f",
				//	kcperpUSDTAccount.AvailableBalance,
				//	*wsBalance.AvailableBalance,
				//)
				kcperpUSDTAccount.AvailableBalance = *wsBalance.AvailableBalance
				kcLoopTimer.Reset(time.Nanosecond)
			}
			if wsBalance.OrderMargin != nil {
				//logger.Debugf(
				//	"PERP WS OrderMargin %f -> %f",
				//	kcperpUSDTAccount.OrderMargin,
				//	*wsBalance.OrderMargin,
				//)
				kcperpUSDTAccount.OrderMargin = *wsBalance.OrderMargin
			}
			if wsBalance.HoldBalance != nil {
				//logger.Debugf(
				//	"PERP WS FrozenFunds %f -> %f",
				//	kcperpUSDTAccount.FrozenFunds,
				//	*wsBalance.HoldBalance,
				//)
				kcperpUSDTAccount.FrozenFunds = *wsBalance.HoldBalance
			}
		}
	}
}

func handlePerpWSPosition(nextPos *kcperp.WSPosition) {
	if _, ok := kcpsSymbolsMap[nextPos.Symbol]; ok {
		if lastPos, ok := kcperpPositions[nextPos.Symbol]; ok && nextPos.EventTime.Sub(lastPos.EventTime) > 0 {
			if nextPos.CurrentQty != nil {
				lastPos.CurrentQty = *nextPos.CurrentQty
				logger.Debugf("PERP WS POS %s NEW QTY %v", nextPos.Symbol, lastPos.CurrentQty)
				kcperpHttpPositionUpdateSilentTimes[lastPos.Symbol] = time.Now().Add(*kcConfig.HttpSilent)
				kcLoopTimer.Reset(time.Nanosecond)
			}
			if nextPos.AvgEntryPrice != nil {
				lastPos.AvgEntryPrice = *nextPos.AvgEntryPrice
			}
			if nextPos.UnrealisedPnl != nil {
				lastPos.UnrealisedPnl = *nextPos.UnrealisedPnl
			}
			kcperpPositions[nextPos.Symbol] = lastPos
			kcperpPositionsUpdateTimes[nextPos.Symbol] = time.Now()
		}
	}
}
