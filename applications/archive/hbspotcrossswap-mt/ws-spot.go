package main

import (
	"github.com/geometrybase/hft-micro/hbspot"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func handleSpotWSBalance(balance *hbspot.WSBalance) {
	if balance.Currency == "usdt" {
		if hbspotUSDTBalance != nil {
			if balance.Available != nil && *balance.Available != hbspotUSDTBalance.Available {
				//logger.Debugf("SPOT WS USDT AVAILABLE CHANGED %f -> %f", hbspotUSDTBalance.Available, *balance.Available)
				hbspotUSDTBalance.Available = *balance.Available
			}
			if balance.Balance != nil && *balance.Balance != hbspotUSDTBalance.Balance {
				logger.Debugf("SPOT WS USDT BALANCE CHANGED %f -> %f", hbspotUSDTBalance.Balance, *balance.Balance)
				hbspotUSDTBalance.Balance = *balance.Balance
				hbLoopTimer.Reset(time.Nanosecond)
			}
		}
		hbspotBalanceUpdatedForInflux = true
		hbspotBalanceUpdatedForExternalInflux = true
		return
	}
	spotSymbol := balance.Currency + "usdt"
	if swapSymbol, ok := hbSpotSwapSymbolsMap[spotSymbol]; ok {
		hasLast := true
		if _, ok := hbspotBalances[spotSymbol]; !ok {
			hasLast = false
			hbspotBalances[spotSymbol] = &hbspot.Balance{}
		}
		if balance.Available != nil && *balance.Available != hbspotBalances[spotSymbol].Available {
			//logger.Debugf("SPOT WS %s AVAILABLE CHANGED %f -> %f", spotSymbol, hbspotBalances[spotSymbol].Available, *balance.Available)
			nb := hbspotBalances[spotSymbol]
			nb.Available = *balance.Available
			hbspotBalances[spotSymbol] = nb
		}
		if balance.Balance != nil && *balance.Balance != hbspotBalances[spotSymbol].Balance {
			logger.Debugf("SPOT WS %s BALANCE CHANGED %f -> %f", spotSymbol, hbspotBalances[spotSymbol].Balance, *balance.Balance)
			nb := hbspotBalances[spotSymbol]
			nb.Balance = *balance.Balance
			hbspotBalances[spotSymbol] = nb
			hbspotHttpBalanceUpdateSilentTimes[spotSymbol] = time.Now().Add(*hbConfig.HttpSilent)
			hbLoopTimer.Reset(time.Nanosecond)
			hbcrossswapOrderSilentTimes[swapSymbol] = time.Now().Add(time.Nanosecond)
			if hasLast {
				logger.Debugf("SPOT ENTER SILENT %s %v", spotSymbol, *hbConfig.EnterSilent)
				hbspotSilentTimes[spotSymbol] = time.Now().Add(*hbConfig.EnterSilent)
			}
		}
		hbspotBalancesUpdateTimes[spotSymbol] = time.Now()
	}
}
