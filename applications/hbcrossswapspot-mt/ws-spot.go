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
			}
		}
		hbspotBalanceUpdatedForInflux = true
		hbspotBalanceUpdatedForExternalInflux = true
		hbspotBalanceUpdatedForReBalance = true
		return
	}
	symbol := balance.Currency + "usdt"
	if _, ok := kcspSymbolsMap[symbol]; !ok {
		return
	}
	hasLast := true
	if _, ok :=  hbspotBalances[symbol]; !ok {
		hasLast = false
		hbspotBalances[symbol] = &hbspot.Balance{}
	}
	if balance.Available != nil && *balance.Available != hbspotBalances[symbol].Available {
		//logger.Debugf("SPOT WS %s AVAILABLE CHANGED %f -> %f", symbol, hbspotBalances[symbol].Available, *balance.Available)
		nb := hbspotBalances[symbol]
		nb.Available = *balance.Available
		hbspotBalances[symbol] = nb
	}
	if balance.Balance != nil && *balance.Balance != hbspotBalances[symbol].Balance {
		logger.Debugf("SPOT WS %s BALANCE CHANGED %f -> %f", symbol, hbspotBalances[symbol].Balance, *balance.Balance)
		nb := hbspotBalances[symbol]
		nb.Balance = *balance.Balance
		hbspotBalances[symbol] = nb
		hbcrossswapOrderSilentTimes[symbol] = time.Now().Add(time.Millisecond*10)
		hbspotHttpBalanceUpdateSilentTimes[symbol] = time.Now().Add(time.Minute * 5)
		if hasLast {
			hbspotSilentTimes[symbol] = time.Now().Add(*hbConfig.EnterSilent)
		}
	}
	hbspotBalancesUpdateTimes[symbol] = time.Now()
}
