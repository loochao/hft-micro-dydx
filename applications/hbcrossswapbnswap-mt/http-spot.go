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
			hAccountUpdatedForInflux = true
			hbspotBalanceUpdatedForExternalInflux = true
			hbspotBalanceUpdatedForReBalance = true
			continue
		}
		symbol := accountBalance.Currency + "usdt"
		if _, ok := tmSymbolsMap[symbol]; !ok {
			continue
		}
		hasBalances[symbol] = true
		//if hHttpPositionUpdateSilentTimes[symbol].Sub(time.Now()) > 0 {
		//	continue
		//}
		if _, ok := tPositions[symbol]; !ok {
			tPositions[symbol] = &hbspot.Balance{
				Currency: accountBalance.Currency,
				Symbol:   symbol,
			}
		}
		nb := tPositions[symbol]
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
		tPositions[symbol] = nb
		mPositionsUpdateTimes[symbol] = time.Now()
	}

	if !hasUSDT {
		hbspotUSDTBalance = &hbspot.Balance{
			Symbol:   "usdtusdt",
			Currency: "usdt",
		}
		hAccountUpdatedForInflux = true
		hbspotBalanceUpdatedForExternalInflux = true
		hbspotBalanceUpdatedForReBalance = true
	}

	if hbspotUSDTBalance.Available != hbspotUSDTBalance.Trade {
		//logger.Debugf("SPOT HTTP USDT Available %f -> %f", hbspotUSDTBalance.Available, hbspotUSDTBalance.Trade)
		hbspotUSDTBalance.Available = hbspotUSDTBalance.Trade
	}
	if hbspotUSDTBalance.Balance != hbspotUSDTBalance.Trade + hbspotUSDTBalance.Frozen{
		logger.Debugf("SPOT HTTP USDT Balance %f -> %f", hbspotUSDTBalance.Balance, hbspotUSDTBalance.Trade + hbspotUSDTBalance.Frozen)
		hbspotUSDTBalance.Balance = hbspotUSDTBalance.Trade + hbspotUSDTBalance.Frozen
	}

	for _, symbol := range mSymbols {
		if _, ok := hasBalances[symbol]; !ok {
			tPositions[symbol] = &hbspot.Balance{
				Symbol:   symbol,
				Currency: strings.Replace(symbol, "usdt", "", -1),
			}
			mPositionsUpdateTimes[symbol] = time.Now()
		}
	}
	for _, symbol := range mSymbols {
		if hHttpPositionUpdateSilentTimes[symbol].Sub(time.Now()) > 0 {
			continue
		}
		nb := tPositions[symbol]
		if nb.Available != nb.Trade {
			//logger.Debugf("SPOT HTTP %s Available %f -> %f", symbol, nb.Available, nb.Trade)
			nb.Available = nb.Trade
		}
		if nb.Balance != nb.Trade+nb.Frozen {
			logger.Debugf("SPOT HTTP %s Balance %f -> %f", symbol, nb.Available, nb.Trade+nb.Frozen)
			nb.Balance = nb.Trade + nb.Frozen
		}
		tPositions[symbol] = nb
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
//	if time.Now().Sub(tOrderSilentTimes[bnBNBSymbol]) < 0 {
//		return
//	}
//	if time.Now().Sub(mPositionsUpdateTimes[bnBNBSymbol]) > *mtConfig.BalancePositionMaxAge {
//		return
//	}
//	bnbBalance, ok1 := tPositions[bnBNBSymbol]
//	bnbMarkPrice, ok2 := hbcrossswapMarkPrices[bnBNBSymbol]
//	if ok1 && ok2 && hbcrossswapBNBAsset != nil && hbcrossswapBNBAsset.MarginBalance != nil && hbspotUSDTBalance != nil {
//		currentSize := bnbBalance.Free + *hbcrossswapBNBAsset.MarginBalance
//		if currentSize < *mtConfig.BnbMinSize {
//			size := *mtConfig.BnbMinSize - currentSize
//			size = math.Ceil(size/tStepSizes[bnBNBSymbol]) * tStepSizes[bnBNBSymbol]
//			price := bnbMarkPrice.IndexPrice * (1.0 + *mtConfig.EnterSlippage)
//			price = math.Ceil(price/tTickSizes[bnBNBSymbol])*tTickSizes[bnBNBSymbol]
//			if size*price < tMinNotional[bnBNBSymbol] {
//				size = math.Ceil(tMinNotional[bnBNBSymbol]/price/tStepSizes[bnBNBSymbol]) * tStepSizes[bnBNBSymbol]
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
//				tOrderSilentTimes[bnBNBSymbol] = time.Now().Add(*mtConfig.OrderSilent)
//				mPositionsUpdateTimes[bnBNBSymbol] = time.Unix(0, 0)
//				mLastOrderTimes[bnBNBSymbol] = time.Now()
//				mOrderRequestChs[bnBNBSymbol] <- MakerOrderRequest{
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
//			tOrderSilentTimes[bnBNBSymbol] = time.Now().Add(*mtConfig.PullInterval * 3)
//			go reBalanceBnB(mtGlobalCtx, tAPI, *mtConfig.OrderTimeout, bnbBalance.Free, *hbcrossswapBNBAsset.MarginBalance, currentSize*0.5-*hbcrossswapBNBAsset.MarginBalance)
//		}
//	}
//}
