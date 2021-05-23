package main

import (
	"github.com/geometrybase/hft-micro/bnspot"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func handleSpotWSOutboundAccountPosition(account *bnspot.AccountUpdateEvent) {
	for _, wsBalance := range account.Balances {

		if wsBalance.Asset == "USDT" {
			balance := wsBalance.ToBalance()
			//if bnspotUSDTBalance != nil &&
			//	(bnspotUSDTBalance.Free != balance.Free ||
			//		bnspotUSDTBalance.Locked != balance.Locked) {
			//	logger.Debugf(
			//		"USDT CHANGE Free %f->%f Locked %f->%f",
			//		bnspotUSDTBalance.Free, balance.Free,
			//		bnspotUSDTBalance.Locked, balance.Locked,
			//	)
			//}
			bnspotUSDTBalance = &balance
			bnspotBalanceUpdatedForInflux = true
			bnspotBalanceUpdatedForExternalInflux = true
			bnspotBalanceUpdatedForReBalance = true
			continue
		}

		symbol := wsBalance.Asset + "USDT"
		if _, ok := bnspotOffsets[symbol]; !ok {
			continue
		}
		var lastBalance *bnspot.Balance
		if b, ok := bnspotBalances[symbol]; ok {
			b := b
			lastBalance = &b
		}

		bnspotBalances[symbol] = wsBalance.ToBalance()
		bnspotBalancesUpdateTimes[symbol] = time.Now()
		bnspotHttpBalanceUpdateSilentTimes[symbol] = time.Now().Add(*bnConfig.HttpSilent)

		if lastBalance == nil ||
			lastBalance.Free+lastBalance.Locked != bnspotBalances[symbol].Free+bnspotBalances[symbol].Locked {
			logger.Debugf("SPOT WS BALANCE CHANGED NEW %s", wsBalance.ToString())
			//如果SPOT变仓，立刻调SWAP，如果SWAP变仓，等ORDER SILENT TIMEOUT
			if symbol == bnBNBSymbol {
				bnswapOrderSilentTimes[symbol] = time.Now().Add(*bnConfig.PullInterval * 3)
			} else {
				bnswapOrderSilentTimes[symbol] = time.Now()
			}
			if lastBalance != nil && lastBalance.Free+lastBalance.Locked != bnspotBalances[symbol].Free+bnspotBalances[symbol].Locked {
				bnspotSilentTimes[symbol] = time.Now().Add(*bnConfig.EnterSilent)
			}
			bnLoopTimer.Reset(time.Nanosecond)
		}
	}
}
