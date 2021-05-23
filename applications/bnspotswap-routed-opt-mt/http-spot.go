package main

import (
	"context"
	"fmt"
	"github.com/geometrybase/hft-micro/bnspot"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"math/rand"
	"strings"
	"time"
)

func handleSpotHttpAccount(account bnspot.Account) {
	hasUSDT := false
	hasSpotBalances := make(map[string]bool)
	for _, balance := range account.Balances {
		if balance.Asset == "USDT" {
			hasUSDT = true
			balance := balance
			if bnspotUSDTBalance == nil || bnspotUSDTBalance.Free != balance.Free {
				logger.Debugf("SPOT HTTP BALANCE %s", balance.ToString())
			}
			bnspotUSDTBalance = &balance
			bnspotBalanceUpdatedForInflux = true
			bnspotBalanceUpdatedForExternalInflux = true
			bnspotBalanceUpdatedForReBalance = true
			continue
		}
		symbol := balance.Asset + "USDT"
		if _, ok := bnspotOffsets[symbol]; !ok {
			continue
		}
		hasSpotBalances[symbol] = true
		if bnspotHttpBalanceUpdateSilentTimes[symbol].Sub(time.Now()) > 0 {
			continue
		}
		var lastBalance *bnspot.Balance
		if b, ok := bnspotBalances[symbol]; ok {
			b := b
			lastBalance = &b
		}
		bnspotBalances[symbol] = balance
		bnspotBalancesUpdateTimes[symbol] = time.Now()

		if lastBalance == nil ||
			lastBalance.Free+lastBalance.Locked != balance.Free+balance.Locked {
			logger.Debugf("SPOT HTTP BALANCE %s", balance.ToString())
			//如果SPOT变仓，立刻调SWAP，如果SWAP变仓，等ORDER SILENT TIMEOUT
			if symbol == bnBNBSymbol {
				bnswapOrderSilentTimes[symbol] = time.Now().Add(*bnConfig.PullInterval * 3)
				//} else {
				//	bnswapOrderSilentTimes[symbol] = time.Now()
			}
			if lastBalance != nil && lastBalance.Free+lastBalance.Locked != balance.Free+balance.Locked {
				bnspotSilentTimes[symbol] = time.Now().Add(*bnConfig.EnterSilent)
			}
			bnLoopTimer.Reset(time.Nanosecond)
		}
	}
	if !hasUSDT {
		balance := bnspot.Balance{
			Free:   0,
			Locked: 0,
			Asset:  "USDT",
		}
		if bnspotUSDTBalance == nil || bnspotUSDTBalance.Free != balance.Free {
			logger.Debugf("SPOT HTTP BALANCE %s", balance.ToString())
		}
		bnspotUSDTBalance = &balance
		bnspotBalanceUpdatedForInflux = true
		bnspotBalanceUpdatedForExternalInflux = true
		bnspotBalanceUpdatedForReBalance = true
	}

	for _, symbol := range bnSymbols {
		if _, ok := hasSpotBalances[symbol]; !ok {
			balance := bnspot.Balance{
				Asset:  strings.Replace(symbol, "USDT", "", -1),
				Free:   0.0,
				Locked: 0.0,
			}
			lastBalance, hasLast := bnspotBalances[symbol]
			if !hasLast ||
				lastBalance.Free+lastBalance.Locked != balance.Free+balance.Locked {
				logger.Debugf("SPOT HTTP BALANCE %s", balance.ToString())
				//如果SPOT变仓，立刻调SWAP，如果SWAP变仓，等ORDER SILENT TIMEOUT
				if symbol == bnBNBSymbol {
					bnswapOrderSilentTimes[symbol] = time.Now().Add(*bnConfig.PullInterval * 3)
				} else {
					bnswapOrderSilentTimes[symbol] = time.Now()
				}
				if hasLast {
					bnspotSilentTimes[symbol] = time.Now().Add(*bnConfig.EnterSilent)
				}
			}
			bnspotBalances[symbol] = balance
			bnspotBalancesUpdateTimes[symbol] = time.Now()
		}
	}
}

func reBalanceUSDT(
	ctx context.Context,
	api *bnspot.API,
	timeout time.Duration,
	change float64,
) {
	tType := bnspot.TransferSpotToUSDTFuture
	if change > 0 {
		change = math.Floor(change)
		logger.Debugf("REBALANCE SPOT TO SWAP %f", change)
	} else if change < 0 {
		tType = bnspot.TransferUSDTFutureToSpot
		change = math.Floor(-change)
		logger.Debugf("REBALANCE SWAP TO SPOT %f", change)
	}
	if change != 0 {
		childCtx, _ := context.WithTimeout(ctx, timeout)
		resp, _, err := api.NewFutureAccountTransfer(childCtx, bnspot.FutureAccountTransferParams{
			Asset:  "USDT",
			Type:   tType,
			Amount: change,
		})
		if err != nil {
			logger.Debugf("NewFutureAccountTransfer error %v", err)
		} else {
			logger.Debugf("%v", *resp)
		}
	}
}

func reBalanceBnB(
	ctx context.Context,
	api *bnspot.API,
	timeout time.Duration,
	spotFree float64,
	swapFree float64,
	change float64,
) {
	tType := bnspot.TransferSpotToUSDTFuture
	if change > 0 {
		if change > spotFree {
			change = spotFree
		}
		change = math.Floor(change/0.01) * 0.01
	} else if change < 0 {
		//不能把SWAP的钱转空，至少要剩有个保险金额
		if -change > swapFree {
			change = -swapFree
		}
		tType = bnspot.TransferUSDTFutureToSpot
		change = math.Floor(-change/0.01) * 0.01
	}
	if change != 0 {
		logger.Debugf("BNB CHANGE %f", change)
		childCtx, _ := context.WithTimeout(ctx, timeout)
		resp, _, err := api.NewFutureAccountTransfer(childCtx, bnspot.FutureAccountTransferParams{
			Asset:  "BNB",
			Type:   tType,
			Amount: change,
		})
		if err != nil {
			logger.Debugf("NewFutureAccountTransfer error %v", err)
		} else {
			logger.Debugf("NewFutureAccountTransfer success %d %f %v", tType, change, resp.TranId)
		}
	}
}

func handleReBalanceBnb() {
	if time.Now().Sub(bnspotOrderSilentTimes[bnBNBSymbol]) < 0 {
		return
	}
	if time.Now().Sub(bnspotBalancesUpdateTimes[bnBNBSymbol]) > *bnConfig.BalancePositionMaxAge {
		return
	}
	bnbBalance, ok1 := bnspotBalances[bnBNBSymbol]
	bnbPremiumIndex, ok2 := bnswapPremiumIndexes[bnBNBSymbol]
	if ok1 && ok2 && bnswapBNBAsset != nil && bnswapBNBAsset.MarginBalance != nil && bnspotUSDTBalance != nil {
		currentSize := bnbBalance.Free + *bnswapBNBAsset.MarginBalance
		if currentSize < *bnConfig.BnbMinSize {
			size := *bnConfig.BnbMinSize - currentSize
			size = math.Ceil(size/bnspotStepSizes[bnBNBSymbol]) * bnspotStepSizes[bnBNBSymbol]
			price := bnbPremiumIndex.IndexPrice
			price = math.Ceil(price/bnspotTickSizes[bnBNBSymbol]) * bnspotTickSizes[bnBNBSymbol]
			if size*price < bnspotMinNotional[bnBNBSymbol] {
				size = math.Ceil(bnspotMinNotional[bnBNBSymbol]/price/bnspotStepSizes[bnBNBSymbol]) * bnspotStepSizes[bnBNBSymbol]
			}
			if price*size < bnspotUSDTBalance.Free && price*size > bnspot.MinNotionals[bnBNBSymbol] {
				logger.Debugf("CHANGE BNB SIZE %f PRICE %f", size, price)
				bnspotOrderSilentTimes[bnBNBSymbol] = time.Now().Add(*bnConfig.OrderSilent)
				bnspotBalancesUpdateTimes[bnBNBSymbol] = time.Unix(0, 0)
				bnspotHttpBalanceUpdateSilentTimes[bnBNBSymbol] = time.Now().Add(*bnConfig.HttpSilent)
				bnspotOrderRequestChs[bnBNBSymbol] <- SpotOrderRequest{
					New: &bnspot.NewOrderParams{
						Symbol:           bnBNBSymbol,
						Side:             bnspot.OrderSideBuy,
						Type:             bnspot.OrderTypeMarket,
						Quantity:         size,
						NewClientOrderID: fmt.Sprintf("%d%04d", time.Now().Unix(), rand.Intn(10000)),
					},
				}
			}
		} else {
			bnspotOrderSilentTimes[bnBNBSymbol] = time.Now().Add(*bnConfig.PullInterval * 3)
			go reBalanceBnB(bnGlobalCtx, bnspotAPI, *bnConfig.OrderTimeout, bnbBalance.Free, *bnswapBNBAsset.MarginBalance, currentSize*0.5-*bnswapBNBAsset.MarginBalance)
		}
	}
}
