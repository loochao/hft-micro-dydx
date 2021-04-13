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
			hasUSDT = true
			balance := account
			if kcspotUSDTBalance == nil || kcspotUSDTBalance.Available != balance.Available {
				logger.Debugf("SPOT HTTP USDT BALANCE CHANGE %v", balance)
			}
			kcspotUSDTBalance = &balance
			kcspotBalanceUpdatedForInflux = true
			kcspotBalanceUpdatedForExternalInflux = true
			kcspotBalanceUpdatedForReBalance = true
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
			Balance: 0,
			Available: 0,
			Holds: 0,
			Currency:  "USDT",
		}
		if kcspotUSDTBalance == nil || kcspotUSDTBalance.Balance != balance.Balance {
			logger.Debugf("SPOT HTTP BALANCE %v", balance)
		}
		kcspotUSDTBalance = &balance
		kcspotBalanceUpdatedForInflux = true
		kcspotBalanceUpdatedForExternalInflux = true
		kcspotBalanceUpdatedForReBalance = true
	}

	for _, symbol := range kcspotSymbols {
		if _, ok := hasAccounts[symbol]; !ok {
			account := kcspot.Account{
				Currency:  strings.Replace(symbol, "-USDT", "", -1),
				Balance: 0,
				Available: 0,
				Holds: 0,
			}
			lastBalance, hasLast := kcspotBalances[symbol]
			if !hasLast ||
				lastBalance.Balance != account.Balance {
				logger.Debugf("SPOT HTTP BALANCE CHANGE %v", account)
				//如果SPOT变仓，立刻调PERP，如果PERP变仓，等ORDER SILENT TIMEOUT
				kcperpOrderSilentTimes[symbol] = time.Now()
				if hasLast {
					kcspotSilentTimes[symbol] = time.Now().Add(*kcConfig.EnterSilent)
				}
			}
			kcspotBalances[symbol] = account
			kcspotBalancesUpdateTimes[symbol] = time.Now()
		}
	}
}

//func reBalanceUSDT(
//	ctx context.Context,
//	api *kcspot.API,
//	timeout time.Duration,
//	change float64,
//) {
//	tType := kcspot.TransferSpotToUSDTFuture
//	if change > 0 {
//		change = math.Floor(change)
//		logger.Debugf("REBALANCE SPOT TO PERP %f", change)
//	} else if change < 0 {
//		tType = kcspot.TransferUSDTFutureToSpot
//		change = math.Floor(-change)
//		logger.Debugf("REBALANCE PERP TO SPOT %f", change)
//	}
//	if change != 0 {
//		childCtx, _ := context.WithTimeout(ctx, timeout)
//		_, _, err := api.NewFutureAccountTransfer(childCtx, kcspot.FutureAccountTransferParams{
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
//	api *kcspot.API,
//	timeout time.Duration,
//	spotFree float64,
//	swapFree float64,
//	change float64,
//) {
//	tType := kcspot.TransferSpotToUSDTFuture
//	if change > 0 {
//		if change > spotFree {
//			change = spotFree
//		}
//		change = math.Floor(change/0.01) * 0.01
//	} else if change < 0 {
//		//不能把PERP的钱转空，至少要剩有个保险金额
//		if -change > swapFree {
//			change = -swapFree
//		}
//		tType = kcspot.TransferUSDTFutureToSpot
//		change = math.Floor(-change/0.01) * 0.01
//	}
//	if change != 0 {
//		logger.Debugf("BNB CHANGE %f", change)
//		childCtx, _ := context.WithTimeout(ctx, timeout)
//		resp, _, err := api.NewFutureAccountTransfer(childCtx, kcspot.FutureAccountTransferParams{
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
//	if time.Now().Sub(kcspotOrderSilentTimes[bnBNBSymbol]) < 0 {
//		return
//	}
//	if time.Now().Sub(kcspotBalancesUpdateTimes[bnBNBSymbol]) > *kcConfig.BalancePositionMaxAge {
//		return
//	}
//	bnbBalance, ok1 := kcspotBalances[bnBNBSymbol]
//	bnbMarkPrice, ok2 := kcperpMarkPrices[bnBNBSymbol]
//	if ok1 && ok2 && kcperpBNBAsset != nil && kcperpBNBAsset.MarginBalance != nil && kcspotUSDTBalance != nil {
//		currentSize := bnbBalance.Free + *kcperpBNBAsset.MarginBalance
//		if currentSize < *kcConfig.BnbMinSize {
//			size := *kcConfig.BnbMinSize - currentSize
//			size = math.Ceil(size/kcspotStepSizes[bnBNBSymbol]) * kcspotStepSizes[bnBNBSymbol]
//			price := bnbMarkPrice.IndexPrice * (1.0 + *kcConfig.EnterSlippage)
//			price = math.Ceil(price/kcspotTickSizes[bnBNBSymbol])*kcspotTickSizes[bnBNBSymbol]
//			if size*price < kcspotMinNotional[bnBNBSymbol] {
//				size = math.Ceil(kcspotMinNotional[bnBNBSymbol]/price/kcspotStepSizes[bnBNBSymbol]) * kcspotStepSizes[bnBNBSymbol]
//			}
//			if price*size < kcspotUSDTBalance.Free {
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
//				kcspotOrderSilentTimes[bnBNBSymbol] = time.Now().Add(*kcConfig.OrderSilent)
//				kcspotBalancesUpdateTimes[bnBNBSymbol] = time.Unix(0, 0)
//				kcspotLastOrderTimes[bnBNBSymbol] = time.Now()
//				kcspotOrderRequestChs[bnBNBSymbol] <- SpotOrderRequest{
//					New: &kcspot.NewOrderParams{
//						Symbol:           bnBNBSymbol,
//						Side:             kcspot.OrderSideBuy,
//						Type:             kcspot.OrderTypeLimit,
//						TimeInForce:      "FOK",
//						Price:            price,
//						Quantity:         size,
//						NewClientOrderID: clOrdID,
//					},
//				}
//			}
//		} else {
//			kcspotOrderSilentTimes[bnBNBSymbol] = time.Now().Add(*kcConfig.PullInterval * 3)
//			go reBalanceBnB(kcGlobalCtx, kcspotAPI, *kcConfig.OrderTimeout, bnbBalance.Free, *kcperpBNBAsset.MarginBalance, currentSize*0.5-*kcperpBNBAsset.MarginBalance)
//		}
//	}
//}
