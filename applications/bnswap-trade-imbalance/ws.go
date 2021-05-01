package main

import (
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func handleTakerWSAccount(data *bnswap.BalanceAndPositionUpdateEvent) {
	for _, pos := range data.Account.Positions {
		if _, ok := swapSymbolsMap[pos.Symbol]; !ok {
			logger.Debugf("not in tm")
			continue
		}
		if pos.PositionSide != "BOTH" {
			logger.Debugf("not both")
			continue
		}
		var lastPosition *bnswap.Position
		if p, ok := swapPositions[pos.Symbol]; ok {
			lastPosition = &bnswap.Position{}
			*lastPosition = *p
		}
		if takerPosition, ok := swapPositions[pos.Symbol]; !ok {
			swapPositions[pos.Symbol] = &bnswap.Position{
				Symbol:           pos.Symbol,
				EntryPrice:       pos.EntryPrice,
				PositionAmt:      pos.PositionAmt,
				UnRealizedProfit: pos.UnRealizedProfit,
			}
		} else {
			takerPosition.EntryPrice = pos.EntryPrice
			takerPosition.PositionAmt = pos.PositionAmt
			takerPosition.UnRealizedProfit = pos.UnRealizedProfit
			swapPositions[pos.Symbol] = takerPosition
		}

		swapPositionsUpdateTimes[pos.Symbol] = time.Now()

		if lastPosition == nil ||
			lastPosition.PositionAmt != swapPositions[pos.Symbol].PositionAmt ||
			lastPosition.EntryPrice != swapPositions[pos.Symbol].EntryPrice {
			swapHttpPositionUpdateSilentTimes[pos.Symbol] = time.Now().Add(*mtConfig.HttpSilent)
			if lastPosition != nil {
				logger.Debugf("%s WS POS %f -> %f", pos.Symbol, lastPosition.PositionAmt, pos.PositionAmt)
			} else {
				logger.Debugf("%s WS POS nil -> %f", pos.Symbol, pos.PositionAmt)
			}
		}
	}
	for _, balance := range data.Account.Balances {
		if balance.Asset == "USDT" {
			wb := balance.WalletBalance
			cwb := balance.CrossWalletBalance
			if swapAccount == nil {
				swapAccount = &bnswap.Asset{
					Asset:              balance.Asset,
					WalletBalance:      &wb,
					CrossWalletBalance: &cwb,
				}
				//logger.Debugf("WS USDT WB nil -> %f", wb)
				//swapLoopTimer.Reset(time.Nanosecond)
			} else {
				if swapAccount.WalletBalance != nil && *swapAccount.WalletBalance != wb {
					//swapLoopTimer.Reset(time.Nanosecond)
					//logger.Debugf("WS USDT WB %f -> %f", *swapAccount.WalletBalance, wb)
				}
				swapAccount.WalletBalance = &wb
				swapAccount.CrossWalletBalance = &cwb
			}
			continue
		}
	}
}
