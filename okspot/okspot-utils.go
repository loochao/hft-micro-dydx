package okspot

import (
	"context"
	"fmt"
	"github.com/geometrybase/hft-micro/logger"
	"strings"
	"time"
)

func WatchBalancesFromHttp(
	ctx context.Context, api *API,
	credentials *Credentials,
	interval time.Duration,
	output chan []Balance,
) {
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			subCtx, _ := context.WithTimeout(ctx, time.Minute)
			balances, err := api.GetAccounts(subCtx, *credentials)
			if err != nil {
				logger.Debugf("WatchAccountFromHttp GetAccount error %v", err)
			} else {
				output <- balances
			}
			timer.Reset(time.Now().Truncate(interval).Add(interval).Sub(time.Now()))
		}
	}
}

func SymbolToInstrumentId(symbol string) string {
	return strings.Replace(symbol, "USDT", "-USDT", -1)
}

func InstrumentIdToSymbol(instrumentId string) string {
	return strings.Replace(instrumentId, "-USDT", "USDT", -1)
}

func USDTSymbolToCurrency(symbol string) string {
	return strings.Replace(symbol, "USDT", "", -1)
}

func CurrencyToUSDTSymbol(currency string) string {
	return fmt.Sprintf("%sUSDT", currency)
}
