package main

import (
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func handleTakerWSAccount(data *bnswap.BalanceAndPositionUpdateEvent) {
	for _, pos := range data.Account.Positions {
		if _, ok := bnSymbolsMap[pos.Symbol]; !ok {
			continue
		}
		if pos.PositionSide != "BOTH" {
			continue
		}
		var lastPosition *bnswap.Position
		if p, ok := bnPositions[pos.Symbol]; ok {
			p := p
			lastPosition = p
		}
		if takerPosition, ok := bnPositions[pos.Symbol]; !ok {
			bnPositions[pos.Symbol] = &bnswap.Position{
				Symbol:           pos.Symbol,
				EntryPrice:       pos.EntryPrice,
				PositionAmt:      pos.PositionAmt,
				UnRealizedProfit: pos.UnRealizedProfit,
			}
		} else {
			takerPosition.EntryPrice = pos.EntryPrice
			takerPosition.PositionAmt = pos.PositionAmt
			takerPosition.UnRealizedProfit = pos.UnRealizedProfit
			bnPositions[pos.Symbol] = takerPosition
		}

		bnPositionsUpdateTimes[pos.Symbol] = time.Now()

		if lastPosition == nil ||
			lastPosition.PositionAmt != bnPositions[pos.Symbol].PositionAmt ||
			lastPosition.EntryPrice != bnPositions[pos.Symbol].EntryPrice {
			bnHttpPositionUpdateSilentTimes[pos.Symbol] = time.Now().Add(*bnConfig.HttpSilent)
			bnLoopTimer.Reset(time.Nanosecond)
			logger.Debugf("TAKER WS POSITION CHANGED NEW %s", pos.ToString())
		}
	}
	for _, balance := range data.Account.Balances {
		if balance.Asset == "USDT" {
			wb := balance.WalletBalance
			cwb := balance.CrossWalletBalance
			if bnAccount == nil {
				bnAccount = &bnswap.Asset{
					Asset:              balance.Asset,
					WalletBalance:      &wb,
					CrossWalletBalance: &cwb,
				}
				logger.Debugf("TAKER WS USDT CHANGE WB nil -> %f", wb)
				bnLoopTimer.Reset(time.Nanosecond)
			} else {
				if bnAccount.WalletBalance != nil && *bnAccount.WalletBalance != wb {
					bnLoopTimer.Reset(time.Nanosecond)
					logger.Debugf("TAKER WS USDT CHANGE WB %f -> %f", *bnAccount.WalletBalance, wb)
				}
				bnAccount.WalletBalance = &wb
				bnAccount.CrossWalletBalance = &cwb
			}
			continue
		}
		//if balance.Asset == "BNB" {
		//	wb := balance.WalletBalance
		//	cwb := balance.CrossWalletBalance
		//	if tBNBAsset == nil {
		//		tBNBAsset = &bnswap.Asset{
		//			Asset:         balance.Asset,
		//			WalletBalance: &wb,
		//			CrossWalletBalance: &cwb,
		//		}
		//	} else {
		//		tBNBAsset.WalletBalance = &wb
		//		tBNBAsset.CrossWalletBalance = &cwb
		//	}
		//	continue
		//}
	}
}
