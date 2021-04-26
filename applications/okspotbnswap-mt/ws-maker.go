package main

import (
	"github.com/geometrybase/hft-micro/logger"
	"github.com/geometrybase/hft-micro/okspot"
	"time"
)

func handleMakerWSAccount(balances []okspot.Balance) {
	for _, balance := range balances {
		if balance.Currency == "USDT" {
			balance := balance
			if mAccount != nil &&
				(balance.Balance != mAccount.Balance ||
					balance.Available != mAccount.Available ||
					balance.Hold != mAccount.Hold) {
				logger.Debugf("MAKER WS BALANCE %s", balance.ToString())
			}
			mAccount = &balance
			continue
		}
		makerSymbol := balance.Currency + "-USDT"
		if takerSymbol, ok := mtSymbolsMap[makerSymbol]; ok {
			var lastBalance *okspot.Balance
			if b, ok := mBalances[makerSymbol]; ok {
				b := b
				lastBalance = &b
			}
			mBalances[makerSymbol] = balance
			mBalancesUpdateTimes[makerSymbol] = time.Now()
			logger.Debugf("MAKER %s B%f A%f H%f", balance.Currency, balance.Balance, balance.Available, balance.Hold)
			if lastBalance == nil ||
				lastBalance.Balance != balance.Balance {
				logger.Debugf("MAKER WS BALANCE %s", balance.ToString())
				//如果SPOT变仓，立刻调SWAP，如果SWAP变仓，等ORDER SILENT TIMEOUT
				tOrderSilentTimes[takerSymbol] = time.Now()
				mHttpPositionUpdateSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.HttpSilent)
				if lastBalance != nil && lastBalance.Balance != balance.Balance {
					mSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.EnterSilent)
					mtLoopTimer.Reset(time.Nanosecond)
				}
			}
		}
	}
}
