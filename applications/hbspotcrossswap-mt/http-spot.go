package main

import (
	"github.com/geometrybase/hft-micro/hbspot"
	"github.com/geometrybase/hft-micro/logger"
	"strings"
	"time"
)

func handleSpotHttpAccount(account hbspot.Account) {
	hasUSDT := false
	hasBalances := make(map[string]bool)
	if hbspotUSDTBalance == nil {
		hbspotUSDTBalance = &hbspot.Balance{
			Currency: "usdt",
			Symbol:   "usdtusdt",
		}
	}
	for _, accountBalance := range account.Balances {
		if accountBalance.Currency == "usdt" {
			hasUSDT = true
			switch accountBalance.Type {
			case "trade":
				if hbspotUSDTBalance.Trade != accountBalance.Balance {
					//logger.Debugf("SPOT HTTP USDT TRADE CHANGE %f -> %f", hbspotUSDTBalance.Trade, accountBalance.Balance)
					hbspotUSDTBalance.Trade = accountBalance.Balance
				}
			case "frozen":
				if hbspotUSDTBalance.Frozen != accountBalance.Balance {
					//logger.Debugf("SPOT HTTP USDT FROZEN CHANGE %f -> %f", hbspotUSDTBalance.Frozen, accountBalance.Balance)
					hbspotUSDTBalance.Frozen = accountBalance.Balance
				}
			}
			hbspotBalanceUpdatedForInflux = true
			hbspotBalanceUpdatedForExternalInflux = true
			continue
		}
		symbol := accountBalance.Currency + "usdt"
		if _, ok := hbSpotSwapSymbolsMap[symbol]; !ok {
			continue
		}
		hasBalances[symbol] = true
		//if hbspotHttpBalanceUpdateSilentTimes[symbol].Sub(time.Now()) > 0 {
		//	continue
		//}
		if _, ok := hbspotBalances[symbol]; !ok {
			hbspotBalances[symbol] = &hbspot.Balance{
				Currency: accountBalance.Currency,
				Symbol:   symbol,
			}
		}
		nb := hbspotBalances[symbol]
		switch accountBalance.Type {
		case "trade":
			if nb.Trade != accountBalance.Balance {
				//logger.Debugf("SPOT HTTP %s TRADE CHANGE %f -> %f", symbol, nb.Trade, accountBalance.Balance)
				nb.Trade = accountBalance.Balance
			}
		case "frozen":
			if nb.Frozen != accountBalance.Balance {
				//logger.Debugf("SPOT HTTP %s FROZEN CHANGE %f -> %f", symbol, nb.Trade, accountBalance.Balance)
				nb.Frozen = accountBalance.Balance
			}
		default:
		}
		hbspotBalances[symbol] = nb
		hbspotBalancesUpdateTimes[symbol] = time.Now()
	}

	if !hasUSDT {
		hbspotUSDTBalance = &hbspot.Balance{
			Symbol:   "usdtusdt",
			Currency: "usdt",
		}
		hbspotBalanceUpdatedForInflux = true
		hbspotBalanceUpdatedForExternalInflux = true
	}

	if hbspotUSDTBalance.Available != hbspotUSDTBalance.Trade {
		//logger.Debugf("SPOT HTTP USDT Available %f -> %f", hbspotUSDTBalance.Available, hbspotUSDTBalance.Trade)
		hbspotUSDTBalance.Available = hbspotUSDTBalance.Trade
	}
	if hbspotUSDTBalance.Balance != hbspotUSDTBalance.Trade + hbspotUSDTBalance.Frozen{
		logger.Debugf("SPOT HTTP USDT Balance %f -> %f", hbspotUSDTBalance.Balance, hbspotUSDTBalance.Trade + hbspotUSDTBalance.Frozen)
		hbspotUSDTBalance.Balance = hbspotUSDTBalance.Trade + hbspotUSDTBalance.Frozen
	}

	for _, symbol := range hbspotSymbols {
		if _, ok := hasBalances[symbol]; !ok {
			hbspotBalances[symbol] = &hbspot.Balance{
				Symbol:   symbol,
				Currency: strings.Replace(symbol, "usdt", "", -1),
			}
			hbspotBalancesUpdateTimes[symbol] = time.Now()
		}
	}
	for _, symbol := range hbspotSymbols {
		if hbspotHttpBalanceUpdateSilentTimes[symbol].Sub(time.Now()) > 0 {
			continue
		}
		nb := hbspotBalances[symbol]
		if nb.Available != nb.Trade {
			//logger.Debugf("SPOT HTTP %s Available %f -> %f", symbol, nb.Available, nb.Trade)
			nb.Available = nb.Trade
		}
		if nb.Balance != nb.Trade+nb.Frozen {
			logger.Debugf("SPOT HTTP %s Balance %f -> %f", symbol, nb.Available, nb.Trade+nb.Frozen)
			nb.Balance = nb.Trade + nb.Frozen
			hbLoopTimer.Reset(time.Nanosecond)
		}
		hbspotBalances[symbol] = nb
	}

}

