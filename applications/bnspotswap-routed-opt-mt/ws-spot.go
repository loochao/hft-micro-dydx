package main

import (
	"github.com/geometrybase/hft-micro/bnspot"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func handleSpotWSOutboundAccountPosition(account *bnspot.AccountUpdateEvent) {
	logger.Debugf("SPOT WS %v", account.EventTime)
	for _, wsBalance := range account.Balances {

		if wsBalance.Asset == "USDT" {
			if bnspotUSDTBalance == nil ||
				wsBalance.EventTime.Sub(bnspotUSDTBalance.EventTime) >= 0 {
				balance := wsBalance.ToBalance()
				bnspotUSDTBalance = &balance
			}
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

		if lastBalance != nil &&
			lastBalance.EventTime.Sub(wsBalance.EventTime) > 0 {
			logger.Debugf("%v is older than %s %v", wsBalance, lastBalance.Asset, lastBalance.EventTime)
			continue
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
