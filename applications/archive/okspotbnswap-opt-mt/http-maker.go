package main

import (
	"github.com/geometrybase/hft-micro/logger"
	"github.com/geometrybase/hft-micro/okex-usdtspot"
	"strings"
	"time"
)

func handleMakerHttpBalances(balances []okex_usdtspot.Balance) {
	hasBalances := make(map[string]bool)
	for _, balance := range balances {
		if balance.Currency == "USDT" {
			balance := balance
			if mAccount == nil ||
				(balance.Balance != mAccount.Balance ||
					balance.Available != mAccount.Available ||
					balance.Hold != mAccount.Hold) {
				logger.Debugf("MAKER HTTP BALANCE %s", balance.ToString())
			}
			mAccount = &balance
			continue
		}
		makerSymbol := balance.Currency + "-USDT"
		if takerSymbol, ok := mtSymbolsMap[makerSymbol]; ok {
			//不能因为Silent而把Balance置成0
			hasBalances[makerSymbol] = true

			if time.Now().Sub(mHttpPositionUpdateSilentTimes[makerSymbol]) < 0 {
				continue
			}
			mBalancesUpdateTimes[makerSymbol] = time.Now()
			var lastBalance *okex_usdtspot.Balance
			if b, ok := mBalances[makerSymbol]; ok {
				b := b
				lastBalance = &b
			}
			mBalances[makerSymbol] = balance
			if lastBalance == nil ||
				lastBalance.Balance != balance.Balance {
				logger.Debugf("MAKER HTTP BALANCE %s", balance.ToString())
				//如果MAKER变仓，立刻调SWAP，如果SWAP变仓，等ORDER SILENT TIMEOUT
				tOrderSilentTimes[takerSymbol] = time.Now()
				mtLoopTimer.Reset(time.Nanosecond)
				if lastBalance != nil && lastBalance.Balance != balance.Balance && balance.Balance != 0 {
					logger.Debugf("ENTER SILENT %s", makerSymbol)
					mSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.EnterSilent)
					mtLoopTimer.Reset(time.Nanosecond)
				}
			}
		}
	}
	//假如没有返回，则仓位是零, 也需要更新mBalancesUpdateTimes
	for makerSymbol, takerSymbol := range mtSymbolsMap {
		if _, ok := hasBalances[makerSymbol]; !ok {
			var lastBalance *okex_usdtspot.Balance
			if b, ok := mBalances[makerSymbol]; ok {
				b := b
				lastBalance = &b
			}
			balance := okex_usdtspot.Balance{
				Currency:  strings.Replace(makerSymbol, "-USDT", "", -1),
				Available: 0.0,
				Hold:      0.0,
				Balance:   0.0,
			}
			mBalancesUpdateTimes[makerSymbol] = time.Now()
			mBalances[makerSymbol] = balance
			if lastBalance == nil ||
				lastBalance.Balance != balance.Balance {
				logger.Debugf("MAKER HTTP BALANCE %s", balance.ToString())
				//如果MAKER变仓，立刻调SWAP，如果SWAP变仓，等ORDER SILENT TIMEOUT
				tOrderSilentTimes[takerSymbol] = time.Now()
				mtLoopTimer.Reset(time.Nanosecond)
			}
		}
	}
}
