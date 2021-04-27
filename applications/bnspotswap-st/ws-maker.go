package main

import (
	"github.com/geometrybase/hft-micro/bnspot"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func handleMakerWSAccount(wsBalance *bnspot.WsBalanceEvent) {

	if mAccount != nil {
		if wsBalance.Currency != nil && mAccount.Currency == *wsBalance.Currency {
			if wsBalance.AvailableBalance != nil && mAccount.AvailableBalance != *wsBalance.AvailableBalance {
				//logger.Debugf(
				//	"PERP WS AvailableBalance %f -> %f",
				//	bnspotUSDTAccount.AvailableBalance,
				//	*wsBalance.AvailableBalance,
				//)
				mAccount.AvailableBalance = *wsBalance.AvailableBalance
				mtLoopTimer.Reset(time.Nanosecond)
			}
			if wsBalance.OrderMargin != nil {
				//logger.Debugf(
				//	"PERP WS OrderMargin %f -> %f",
				//	bnspotUSDTAccount.OrderMargin,
				//	*wsBalance.OrderMargin,
				//)
				mAccount.OrderMargin = *wsBalance.OrderMargin
			}
			if wsBalance.HoldBalance != nil {
				//logger.Debugf(
				//	"PERP WS FrozenFunds %f -> %f",
				//	bnspotUSDTAccount.FrozenFunds,
				//	*wsBalance.HoldBalance,
				//)
				mAccount.FrozenFunds = *wsBalance.HoldBalance
			}
		}
	}
}

func handleMakerWSPosition(nextPos *bnspot.WSPosition) {
	if takerSymbol, ok := mtSymbolsMap[nextPos.Symbol]; ok {
		if lastPos, ok := mPositions[nextPos.Symbol]; ok && nextPos.EventTime.Sub(lastPos.EventTime) > 0 {
			if nextPos.CurrentQty != nil {
				if lastPos.CurrentQty != *nextPos.CurrentQty {
					logger.Debugf("MAKER WS POS %s %f -> %f", nextPos.Symbol, lastPos.CurrentQty, *nextPos.CurrentQty)
					if *nextPos.CurrentQty != 0 {
						logger.Debugf("MAKER ENTER SILENT %v", *mtConfig.EnterSilent)
						mSilentTimes[nextPos.Symbol] = time.Now().Add(*mtConfig.EnterSilent)
					}
					tOrderSilentTimes[takerSymbol] = time.Now()
					mtLoopTimer.Reset(time.Nanosecond)
				}
				lastPos.CurrentQty = *nextPos.CurrentQty
				mHttpPositionUpdateSilentTimes[lastPos.Symbol] = time.Now().Add(*mtConfig.HttpSilent)
				mtLoopTimer.Reset(time.Nanosecond)
			}
			if nextPos.AvgEntryPrice != nil {
				lastPos.AvgEntryPrice = *nextPos.AvgEntryPrice
			}
			if nextPos.UnrealisedPnl != nil {
				lastPos.UnrealisedPnl = *nextPos.UnrealisedPnl
			}
			mPositions[nextPos.Symbol] = lastPos
			mPositionsUpdateTimes[nextPos.Symbol] = time.Now()
		}
	}
}
