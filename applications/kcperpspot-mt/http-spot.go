package main

import (
	"github.com/geometrybase/hft-micro/kcspot"
	"github.com/geometrybase/hft-micro/logger"
	"strings"
	"time"
)

func handleSpotHttpAccount(accounts []kcspot.Account) {
	hasUSDT := false
	hasAccounts := make(map[string]bool)
	for _, account := range accounts {
		if account.Currency == "USDT" {
			if account.Type == kcspot.AccountTypeTrade {
				hasUSDT = true
				balance := account
				if kcspotUSDTBalance == nil || kcspotUSDTBalance.Available != balance.Available {
					logger.Debugf("SPOT HTTP USDT BALANCE CHANGE %v", balance)
					kcLoopTimer.Reset(time.Nanosecond)
				}
				kcspotUSDTBalance = &balance
			}
			continue
		}
		symbol := account.Currency + "-USDT"
		if _, ok := kcspSymbolsMap[symbol]; !ok {
			continue
		}
		hasAccounts[symbol] = true
		if kcspotHttpBalanceUpdateSilentTimes[symbol].Sub(time.Now()) > 0 {
			continue
		}
		//if account.EventTime.Sub(kcspotLastOrderTimes[symbol]).Seconds() < 0.0 {
		//	continue
		//}

		var lastAccount *kcspot.Account
		if b, ok := kcspotBalances[symbol]; ok {
			b := b
			lastAccount = &b
		}

		kcspotBalances[symbol] = account
		kcspotBalancesUpdateTimes[symbol] = time.Now()

		if lastAccount == nil ||
			lastAccount.Holds != kcspotBalances[symbol].Holds ||
			lastAccount.Available != kcspotBalances[symbol].Available {
			logger.Debugf("SPOT HTTP BALANCE %v", account)
			//如果SPOT变仓，立刻调PERP，如果PERP变仓，等ORDER SILENT TIMEOUT
			kcperpOrderSilentTimes[symbol] = time.Now()
			if lastAccount != nil && lastAccount.Available+lastAccount.Holds != kcspotBalances[symbol].Available+kcspotBalances[symbol].Holds {
				kcspotSilentTimes[symbol] = time.Now().Add(*kcConfig.EnterSilent)
			}
		}
	}
	if !hasUSDT {
		balance := kcspot.Account{
			Balance:   0,
			Available: 0,
			Holds:     0,
			Currency:  "USDT",
		}
		if kcspotUSDTBalance == nil || kcspotUSDTBalance.Balance != balance.Balance {
			logger.Debugf("SPOT HTTP BALANCE %v", balance)
		}
		kcspotUSDTBalance = &balance
	}

	for _, symbol := range kcspotSymbols {
		if _, ok := hasAccounts[symbol]; !ok {
			account := kcspot.Account{
				Currency:  strings.Replace(symbol, "-USDT", "", -1),
				Balance:   0,
				Available: 0,
				Holds:     0,
			}
			lastBalance, hasLast := kcspotBalances[symbol]
			if !hasLast ||
				lastBalance.Balance != account.Balance {
				logger.Debugf("SPOT HTTP BALANCE CHANGE %v", account)
				//如果SPOT变仓，立刻调PERP，如果PERP变仓，等ORDER SILENT TIMEOUT
				kcperpOrderSilentTimes[symbol] = time.Now()
				kcLoopTimer.Reset(time.Nanosecond)
				if hasLast {
					kcspotSilentTimes[symbol] = time.Now().Add(*kcConfig.EnterSilent)
				}
			}
			kcspotBalances[symbol] = account
			kcspotBalancesUpdateTimes[symbol] = time.Now()
		}
	}
}

