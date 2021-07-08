package main

import (
	"github.com/geometrybase/hft-micro/kucoin-usdtspot"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"time"
)

func handleSpotWSBalance(balance *kucoin_usdtspot.WsBalance) {
	if balance.Currency == "USDT" {
		if kcspotUSDTBalance == nil {
			kcspotUSDTBalance = &kucoin_usdtspot.Account{
				Currency:  balance.Currency,
				Available: balance.Available,
				Balance:   balance.Total,
				Holds:     balance.Hold,
			}
			kcLoopTimer.Reset(time.Nanosecond)
		} else {
			kcspotUSDTBalance.Available = balance.Available
			kcspotUSDTBalance.Balance = balance.Total
			kcspotUSDTBalance.Holds = balance.Hold
			kcLoopTimer.Reset(time.Nanosecond)
		}
		//logger.Debugf("SPOT WS USDT BALANCE %v", *kcspotUSDTBalance)
		return
	}
	symbol := balance.Currency + "-USDT"
	if _, ok := kcspSymbolsMap[symbol]; !ok {
		return
	}
	var lastBalance *kucoin_usdtspot.Account
	if b, ok := kcspotBalances[symbol]; ok {
		b := b
		lastBalance = &b
	} else {
		kcspotBalances[symbol] = kucoin_usdtspot.Account{
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

	if lastBalance == nil ||
		math.Abs(lastBalance.Holds+lastBalance.Available-kcspotBalances[symbol].Available-kcspotBalances[symbol].Holds) > 0.000001 {
		if lastBalance == nil {
			logger.Debugf("SPOT WS BALANCE CHANGED %s Available nil -> %f Holds nil -> %f", account.Currency, account.Available, account.Holds)
		} else {
			logger.Debugf("SPOT WS BALANCE CHANGED %s Available %f -> %f Holds %f -> %f", account.Currency, lastBalance.Available, account.Available, lastBalance.Holds, account.Holds)
		}
		kcspotHttpBalanceUpdateSilentTimes[symbol] = time.Now().Add(*kcConfig.HttpSilent)
		kcperpOrderSilentTimes[symbol] = time.Now()
		kcLoopTimer.Reset(time.Nanosecond)
		if lastBalance != nil && lastBalance.Holds+lastBalance.Available != account.Holds+account.Available {
			kcspotSilentTimes[symbol] = time.Now().Add(*kcConfig.EnterSilent)
		}
	}
}
