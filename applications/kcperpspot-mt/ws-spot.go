package main

import (
	"github.com/geometrybase/hft-micro/kcspot"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func handleSpotWSBalance(balance *kcspot.WsBalance) {
	if balance.Currency == "USDT" {
		if kcspotUSDTBalance == nil {
			kcspotUSDTBalance = &kcspot.Account{
				Currency:  balance.Currency,
				Available: balance.Available,
				Balance:   balance.Total,
				Holds:     balance.Hold,
			}
		} else {
			kcspotUSDTBalance.Available = balance.Available
			kcspotUSDTBalance.Balance = balance.Total
			kcspotUSDTBalance.Holds = balance.Hold
		}
		kcspotBalanceUpdatedForInflux = true
		kcspotBalanceUpdatedForExternalInflux = true
		kcspotBalanceUpdatedForReBalance = true
		//logger.Debugf("SPOT WS USDT BALANCE %v", *kcspotUSDTBalance)
		return
	}
	symbol := balance.Currency + "-USDT"
	if _, ok := kcspSymbolsMap[symbol]; !ok {
		return
	}
	var lastBalance *kcspot.Account
	if b, ok := kcspotBalances[symbol]; ok {
		b := b
		lastBalance = &b
	} else {
		kcspotBalances[symbol] = kcspot.Account{
			Currency:  balance.Currency,
			Available: balance.Available,
			Balance:   balance.Total,
			Holds:     balance.Hold,
			EventTime: balance.EventTime,
		}
	}

	if balance.EventTime.Sub(kcspotBalances[symbol].EventTime) <= 0 {
		return
	}
	account := kcspotBalances[symbol]
	account.Balance = balance.Total
	account.Available = balance.Available
	account.Holds = balance.Hold
	kcspotBalances[symbol] = account
	kcspotBalancesUpdateTimes[symbol] = time.Now()
	kcspotHttpBalanceUpdateSilentTimes[symbol] = time.Now().Add(time.Minute * 3)

	if lastBalance == nil ||
		lastBalance.Holds != kcspotBalances[symbol].Holds ||
		lastBalance.Available != kcspotBalances[symbol].Available {
		logger.Debugf("SPOT WS BALANCE CHANGED NEW %v", account)
		kcperpOrderSilentTimes[symbol] = time.Now()
		if lastBalance != nil && lastBalance.Holds+lastBalance.Available != account.Holds+account.Available {
			kcspotSilentTimes[symbol] = time.Now().Add(*kcConfig.EnterSilent)
		}
	}
}
