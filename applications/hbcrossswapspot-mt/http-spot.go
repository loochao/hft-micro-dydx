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
				if  hbspotUSDTBalance.Trade != accountBalance.Balance {
					logger.Debugf("SPOT HTTP USDT TRADE CHANGE %f -> %f",hbspotUSDTBalance.Trade, accountBalance.Balance )
					hbspotUSDTBalance.Trade = accountBalance.Balance
				}
			case "frozen":
				if  hbspotUSDTBalance.Frozen != accountBalance.Balance {
					logger.Debugf("SPOT HTTP USDT FROZEN CHANGE %f -> %f",hbspotUSDTBalance.Frozen, accountBalance.Balance )
					hbspotUSDTBalance.Frozen = accountBalance.Balance
				}
			case "loan":
				if  hbspotUSDTBalance.Loan != accountBalance.Balance {
					logger.Debugf("SPOT HTTP USDT LOAN CHANGE %f -> %f",hbspotUSDTBalance.Loan, accountBalance.Balance )
					hbspotUSDTBalance.Loan = accountBalance.Balance
				}
			case "interest":
				if  hbspotUSDTBalance.Interest != accountBalance.Balance {
					logger.Debugf("SPOT HTTP USDT INTEREST CHANGE %f -> %f",hbspotUSDTBalance.Interest, accountBalance.Balance )
					hbspotUSDTBalance.Interest = accountBalance.Balance
				}
			case "lock":
				if  hbspotUSDTBalance.Lock != accountBalance.Balance {
					logger.Debugf("SPOT HTTP USDT LOCK CHANGE %f -> %f",hbspotUSDTBalance.Lock, accountBalance.Balance )
					hbspotUSDTBalance.Lock = accountBalance.Balance
				}
			case "bank":
				if  hbspotUSDTBalance.Bank != accountBalance.Balance {
					logger.Debugf("SPOT HTTP USDT BANK CHANGE %f -> %f",hbspotUSDTBalance.Bank, accountBalance.Balance )
					hbspotUSDTBalance.Bank = accountBalance.Balance
				}
			}
			hbspotBalanceUpdatedForInflux = true
			hbspotBalanceUpdatedForExternalInflux = true
			hbspotBalanceUpdatedForReBalance = true
			continue
		}
		symbol := accountBalance.Currency + "usdt"
		if _, ok := kcspSymbolsMap[symbol]; !ok {
			continue
		}
		hasBalances[symbol] = true
		if hbspotHttpBalanceUpdateSilentTimes[symbol].Sub(time.Now()) > 0 {
			continue
		}
		if _, ok := hbspotBalances[symbol]; !ok {
			hbspotBalances[symbol] = &hbspot.Balance{
				Currency: accountBalance.Currency,
				Symbol: symbol,
			}
		}
		switch accountBalance.Type {
		case "trade":
			if  hbspotBalances[symbol].Trade != accountBalance.Balance {
				logger.Debugf("SPOT HTTP %s TRADE CHANGE %f -> %f",symbol,hbspotBalances[symbol].Trade, accountBalance.Balance )
				hbspotBalances[symbol].Trade = accountBalance.Balance
			}
		case "frozen":
			if  hbspotBalances[symbol].Frozen != accountBalance.Balance {
				logger.Debugf("SPOT HTTP %s FROZEN CHANGE %f -> %f",symbol,hbspotBalances[symbol].Trade, accountBalance.Balance )
				hbspotBalances[symbol].Frozen = accountBalance.Balance
			}
		case "loan":
			if  hbspotBalances[symbol].Loan != accountBalance.Balance {
				logger.Debugf("SPOT HTTP %s LOAN CHANGE %f -> %f",symbol,hbspotBalances[symbol].Trade, accountBalance.Balance )
				hbspotBalances[symbol].Loan = accountBalance.Balance
			}
		case "interest":
			if  hbspotBalances[symbol].Interest != accountBalance.Balance {
				logger.Debugf("SPOT HTTP %s INTEREST CHANGE %f -> %f",symbol,hbspotBalances[symbol].Trade, accountBalance.Balance )
				hbspotBalances[symbol].Interest = accountBalance.Balance
			}
		case "lock":
			if  hbspotBalances[symbol].Lock != accountBalance.Balance {
				logger.Debugf("SPOT HTTP %s LOCK CHANGE %f -> %f",symbol,hbspotBalances[symbol].Trade, accountBalance.Balance )
				hbspotBalances[symbol].Lock = accountBalance.Balance
			}
		case "bank":
			if  hbspotBalances[symbol].Bank != accountBalance.Balance {
				logger.Debugf("SPOT HTTP %s BANK CHANGE %f -> %f",symbol,hbspotBalances[symbol].Trade, accountBalance.Balance )
				hbspotBalances[symbol].Bank = accountBalance.Balance
			}
		}
		hbspotBalancesUpdateTimes[symbol] = time.Now()
	}
	if !hasUSDT {
		hbspotUSDTBalance = &hbspot.Balance{
			Symbol: "usdtusdt",
			Currency: "usdt",
		}
		hbspotBalanceUpdatedForInflux = true
		hbspotBalanceUpdatedForExternalInflux = true
		hbspotBalanceUpdatedForReBalance = true
	}

	hbspotUSDTBalance.Available = hbspotUSDTBalance.Trade
	hbspotUSDTBalance.Balance = hbspotUSDTBalance.Trade + hbspotUSDTBalance.Frozen +
		hbspotUSDTBalance.Lock + hbspotUSDTBalance.Lock + hbspotUSDTBalance.Interest +
		hbspotUSDTBalance.Bank


	for _, symbol := range hbspotSymbols {
		if _, ok := hasBalances[symbol]; !ok {
			hbspotBalances[symbol] = &hbspot.Balance{
				Symbol: symbol,
				Currency: strings.Replace(symbol, "usdt", "", -1),
			}
			hbspotBalancesUpdateTimes[symbol] = time.Now()
		}
	}
	for _, symbol := range hbspotSymbols {
		hbspotBalances[symbol].Available = hbspotBalances[symbol].Trade
		hbspotBalances[symbol].Balance = hbspotBalances[symbol].Trade + hbspotBalances[symbol].Frozen +
			hbspotBalances[symbol].Lock + hbspotBalances[symbol].Lock + hbspotBalances[symbol].Interest +
			hbspotBalances[symbol].Bank
	}

}

//func reBalanceUSDT(
//	ctx context.Context,
//	api *hbspot.API,
//	timeout time.Duration,
//	change float64,
//) {
//	tType := hbspot.TransferSpotToUSDTFuture
//	if change > 0 {
//		change = math.Floor(change)
//		logger.Debugf("REBALANCE SPOT TO SWAP %f", change)
//	} else if change < 0 {
//		tType = hbspot.TransferUSDTFutureToSpot
//		change = math.Floor(-change)
//		logger.Debugf("REBALANCE SWAP TO SPOT %f", change)
//	}
//	if change != 0 {
//		childCtx, _ := context.WithTimeout(ctx, timeout)
//		_, _, err := api.NewFutureAccountTransfer(childCtx, hbspot.FutureAccountTransferParams{
//			Asset:  "USDT",
//			Type:   tType,
//			Amount: change,
//		})
//		if err != nil {
//			logger.Debugf("NewFutureAccountTransfer error %v", err)
//		}
//	}
//}
//
//func reBalanceBnB(
//	ctx context.Context,
//	api *hbspot.API,
//	timeout time.Duration,
//	spotFree float64,
//	swapFree float64,
//	change float64,
//) {
//	tType := hbspot.TransferSpotToUSDTFuture
//	if change > 0 {
//		if change > spotFree {
//			change = spotFree
//		}
//		change = math.Floor(change/0.01) * 0.01
//	} else if change < 0 {
//		//不能把SWAP的钱转空，至少要剩有个保险金额
//		if -change > swapFree {
//			change = -swapFree
//		}
//		tType = hbspot.TransferUSDTFutureToSpot
//		change = math.Floor(-change/0.01) * 0.01
//	}
//	if change != 0 {
//		logger.Debugf("BNB CHANGE %f", change)
//		childCtx, _ := context.WithTimeout(ctx, timeout)
//		resp, _, err := api.NewFutureAccountTransfer(childCtx, hbspot.FutureAccountTransferParams{
//			Asset:  "BNB",
//			Type:   tType,
//			Amount: change,
//		})
//		if err != nil {
//			logger.Debugf("NewFutureAccountTransfer error %v", err)
//		} else {
//			logger.Debugf("%v", resp)
//		}
//	}
//}
//
//func handleReBalanceBnb() {
//	if time.Now().Sub(hbspotOrderSilentTimes[bnBNBSymbol]) < 0 {
//		return
//	}
//	if time.Now().Sub(hbspotBalancesUpdateTimes[bnBNBSymbol]) > *hbConfig.BalancePositionMaxAge {
//		return
//	}
//	bnbBalance, ok1 := hbspotBalances[bnBNBSymbol]
//	bnbMarkPrice, ok2 := hbcrossswapMarkPrices[bnBNBSymbol]
//	if ok1 && ok2 && hbcrossswapBNBAsset != nil && hbcrossswapBNBAsset.MarginBalance != nil && hbspotUSDTBalance != nil {
//		currentSize := bnbBalance.Free + *hbcrossswapBNBAsset.MarginBalance
//		if currentSize < *hbConfig.BnbMinSize {
//			size := *hbConfig.BnbMinSize - currentSize
//			size = math.Ceil(size/hbspotStepSizes[bnBNBSymbol]) * hbspotStepSizes[bnBNBSymbol]
//			price := bnbMarkPrice.IndexPrice * (1.0 + *hbConfig.EnterSlippage)
//			price = math.Ceil(price/hbspotTickSizes[bnBNBSymbol])*hbspotTickSizes[bnBNBSymbol]
//			if size*price < hbspotMinNotional[bnBNBSymbol] {
//				size = math.Ceil(hbspotMinNotional[bnBNBSymbol]/price/hbspotStepSizes[bnBNBSymbol]) * hbspotStepSizes[bnBNBSymbol]
//			}
//			if price*size < hbspotUSDTBalance.Free {
//				logger.Debugf("CHANGE BNB SIZE %f PRICE %f", size, price)
//				id, _ := common.GenerateShortId()
//				clOrdID := fmt.Sprintf(
//					"%sBNBBURN",
//					id,
//				)
//				clOrdID = strings.ReplaceAll(clOrdID, ".", "_")
//				if len(clOrdID) > 36 {
//					clOrdID = clOrdID[:36]
//				}
//				hbspotOrderSilentTimes[bnBNBSymbol] = time.Now().Add(*hbConfig.OrderSilent)
//				hbspotBalancesUpdateTimes[bnBNBSymbol] = time.Unix(0, 0)
//				hbspotLastOrderTimes[bnBNBSymbol] = time.Now()
//				hbspotOrderRequestChs[bnBNBSymbol] <- SpotOrderRequest{
//					New: &hbspot.NewOrderParams{
//						Symbol:           bnBNBSymbol,
//						Side:             hbspot.OrderSideBuy,
//						Type:             hbspot.OrderTypeLimit,
//						TimeInForce:      "FOK",
//						Price:            price,
//						Quantity:         size,
//						NewClientOrderID: clOrdID,
//					},
//				}
//			}
//		} else {
//			hbspotOrderSilentTimes[bnBNBSymbol] = time.Now().Add(*hbConfig.PullInterval * 3)
//			go reBalanceBnB(hbGlobalCtx, hbspotAPI, *hbConfig.OrderTimeout, bnbBalance.Free, *hbcrossswapBNBAsset.MarginBalance, currentSize*0.5-*hbcrossswapBNBAsset.MarginBalance)
//		}
//	}
//}
