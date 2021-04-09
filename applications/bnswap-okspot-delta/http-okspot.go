package main

import (
	"context"
	"github.com/geometrybase/hft/common"
	"github.com/geometrybase/hft/logger"
	"github.com/geometrybase/hft/okspot"
	"time"
)

//import (
//	"context"
//	"github.com/geometrybase/hft/logger"
//	"github.com/geometrybase/hft/okspot"
//)

func handleSpotHttpAccount(balances []okspot.Balance) {
	for _, balance := range balances {
		if balance.Currency == "USDT" {
			balance := balance
			if okspotUSDTBalance == nil ||
				(balance.Balance != okspotUSDTBalance.Balance ||
					balance.Available != okspotUSDTBalance.Available ||
					balance.Hold != okspotUSDTBalance.Hold) {
				logger.Debugf("SPOT HTTP BALANCE %s", balance.ToString())
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

		if lastBalance == nil ||
			lastBalance.Available != okspotBalances[symbol].Available ||
			lastBalance.Hold != okspotBalances[symbol].Hold {
			logger.Debugf("SPOT HTTP BALANCE %s", balance.ToString())
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
	for _, symbol := range boSymbols {
		if _, ok := okspotBalances[symbol]; !ok {
			okspotBalances[symbol] = okspot.Balance{
				Currency:  okspot.USDTSymbolToCurrency(symbol),
				Available: 0.0,
				Hold:      0.0,
				Balance:   0.0,
			}
			logger.Debugf("SPOT HTTP BALANCE %s EMPTY", okspot.USDTSymbolToCurrency(symbol))
		}
		okspotBalancesUpdated[symbol] = true
	}
}

func createSpotOrder(
	ctx context.Context,
	credentials *okspot.Credentials,
	api *okspot.API,
	timeout time.Duration,
	params okspot.NewOrderParams,
) {
	childCtx, _ := context.WithTimeout(ctx, timeout)
	orderStr, _ := common.JSONEncode(params)
	logger.Debugf("SPOT SUBMIT %s", orderStr)
	orderRes, err := api.SubmitOrder(childCtx, *credentials, params)
	if err != nil {
		logger.Debugf("SPOT SUBMIT ERROR %v", err)
		select {
		case <-ctx.Done():
		case okspotOrderNewErrorCh <- SpotOrderNewError{
			Error:  err,
			Params: params,
		}:
		}
		if orderRes.Result {
			logger.Debugf("SPOT ORDER SUBMIT SUCCESS")
		} else {
			logger.Debugf("SPOT ORDER ERROR %s %s", orderRes.ErrorCode, orderRes.ErrorMessage)
		}
	}
}

//func reBalanceUSDT(
//	ctx context.Context,
//	credentials *okspot.Credentials,
//	api *okspot.API,
//	timeout time.Duration,
//	spotFree float64,
//	swapFree float64,
//	insuranceFund float64,
//	change float64,
//) {
//	tType := okspot.TransferSpotToUSDTFuture
//	if change > 0 {
//		if change > spotFree {
//			change = spotFree
//		}
//		change = math.Floor(change)
//	} else if change < 0 {
//		//不能把SWAP的钱转空，至少要剩有个保险金额
//		if -change > swapFree-insuranceFund {
//			change = 0
//			if swapFree-insuranceFund > 0 {
//				change = -(swapFree - insuranceFund)
//			}
//		}
//		tType = okspot.TransferUSDTFutureToSpot
//		change = math.Floor(-change)
//	}
//	if change != 0 {
//		childCtx, _ := context.WithTimeout(ctx, timeout)
//		_, err := api.NewFutureAccountTransfer(childCtx, credentials, okspot.FutureAccountTransferParams{
//			Asset:  "USDT",
//			Type:   tType,
//			Amount: change,
//		})
//		if err != nil {
//			logger.Debugf("NewFutureAccountTransfer error %v", err)
//		}
//	}
//}

func getOkOrderLimits(ctx context.Context, api *okspot.API, symbolsMap map[string]bool) (
	matchedSymbols []string, tickSizes map[string]float64, stepSizes map[string]float64, minSizes map[string]float64, err error,
) {
	var instruments []okspot.Instrument
	instruments, err = api.GetInstruments(ctx)
	if err != nil {
		return
	}
	tickSizes = make(map[string]float64)
	stepSizes = make(map[string]float64)
	minSizes = make(map[string]float64)
	matchedSymbols = make([]string, 0)
	for _, instrument := range instruments {
		if len(instrument.InstrumentId) < 5 {
			continue
		}
		if instrument.InstrumentId[len(instrument.InstrumentId)-5:] != "-USDT" {
			continue
		}
		symbol := okspot.InstrumentIdToSymbol(instrument.InstrumentId)
		if _, ok := symbolsMap[symbol]; !ok {
			continue
		}
		matchedSymbols = append(matchedSymbols, symbol)
		tickSizes[symbol] = instrument.TickSize
		stepSizes[symbol] = instrument.SizeIncrement
		minSizes[symbol] = instrument.MinSize
	}
	logger.Debugf("SPOT TICK SIZES %v", tickSizes)
	logger.Debugf("SPOT STEP SIZES %v", stepSizes)
	logger.Debugf("SPOT MIN SIZES %v", minSizes)
	return
}
