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
				logger.Debugf("SPOT WS USDT AVAILABLE CHANGED %f -> ", hbspotUSDTBalance.Available, *balance.Available)
				hbspotUSDTBalance.Available = *balance.Available
			}
			if balance.Balance != nil && *balance.Balance != hbspotUSDTBalance.Balance {
				logger.Debugf("SPOT WS USDT BALANCE CHANGED %f -> ", hbspotUSDTBalance.Balance, *balance.Balance)
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
	if balance.Available != nil && *balance.Available != hbspotBalances[symbol].Available {
		logger.Debugf("SPOT WS %s AVAILABLE CHANGED %f -> ", symbol, hbspotUSDTBalance.Available, *balance.Available)
		hbspotBalances[symbol].Available = *balance.Available
	}
	if balance.Balance != nil && *balance.Balance != hbspotBalances[symbol].Balance {
		logger.Debugf("SPOT WS %s AVAILABLE CHANGED %f -> ", symbol, hbspotUSDTBalance.Balance, *balance.Balance)
		hbspotBalances[symbol].Balance = *balance.Balance
		hbcrossswapOrderSilentTimes[symbol] = time.Now()
	}
	hbspotBalancesUpdateTimes[symbol] = time.Now()
	hbspotHttpBalanceUpdateSilentTimes[symbol] = time.Now().Add(time.Minute * 3)
}
