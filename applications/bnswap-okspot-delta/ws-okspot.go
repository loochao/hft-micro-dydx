package main

import (
	"github.com/geometrybase/hft/logger"
	"github.com/geometrybase/hft/okspot"
	"time"
)

func handleSpotWSOrder(orders []okspot.WSOrder) {
	for _, wsOrder := range orders {
		if wsOrder.State == okspot.OrderStateCanceled ||
			wsOrder.State == okspot.OrderStateFailed ||
			wsOrder.State == okspot.OrderStateFullyFilled {
			order := wsOrder
			okspotOrderFinishCh <- order
		}
	}
}

func handleSpotWSBalances(balances []okspot.Balance) {
	for _, balance := range balances {
		if balance.Currency == "USDT" {
			balance := balance
			if okspotUSDTBalance != nil &&
				(balance.Balance != okspotUSDTBalance.Balance ||
					balance.Available != okspotUSDTBalance.Available ||
					balance.Hold != okspotUSDTBalance.Hold) {
				logger.Debugf("SPOT WS BALANCE %s", balance.ToString())
			}
			okspotUSDTBalance = &balance
			okspotBalanceUpdatedForInflux = true
			okspotBalanceUpdatedForExternalInflux = true
			continue
		}
		symbol := balance.Currency + "USDT"
		if _, ok := boSymbolsMap[symbol]; !ok {
			continue
		}
		var lastBalance *okspot.Balance
		if b, ok := okspotBalances[symbol]; ok {
			b := b
			lastBalance = &b
		}

		okspotBalances[symbol] = balance
		okspotBalancesUpdated[symbol] = true

		if lastBalance == nil ||
			lastBalance.Balance != okspotBalances[symbol].Balance {
			logger.Debugf("SPOT WS BALANCE %s", balance.ToString())
			//如果SPOT变仓，立刻调SWAP，如果SWAP变仓，等ORDER SILENT TIMEOUT
			bnswapOrderSilentTimes[symbol] = time.Now()
			if lastBalance != nil {
				if lastBalance.Balance < balance.Balance {
					//加仓可减仓
					okspotEnterSilentTimes[symbol] = time.Now().Add(*boConfig.EnterSilent)
					okspotExitSilentTimes[symbol] = time.Now()
				} else {
					//减仓可加仓
					okspotEnterSilentTimes[symbol] = time.Now()
					okspotExitSilentTimes[symbol] = time.Now().Add(*boConfig.ExitSilent)
				}
			}
		}
	}
}
