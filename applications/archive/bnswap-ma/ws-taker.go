package main

import (
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"time"
)

func handleTakerWSAccount(data *bnswap.BalanceAndPositionUpdateEvent) {
	for _, pos := range data.Account.Positions {
		if _, ok := mtSymbolsMap[pos.Symbol]; !ok {
			logger.Debugf("not in tm")
			continue
		}
		if pos.PositionSide != "BOTH" {
			logger.Debugf("not both")
			continue
		}
		var lastPosition *bnswap.Position
		if p, ok := tPositions[pos.Symbol]; ok {
			lastPosition = &bnswap.Position{}
			*lastPosition = *p
		}
		if takerPosition, ok := tPositions[pos.Symbol]; !ok {
			tPositions[pos.Symbol] = &bnswap.Position{
				Symbol:           pos.Symbol,
				EntryPrice:       pos.EntryPrice,
				PositionAmt:      pos.PositionAmt,
				UnRealizedProfit: pos.UnRealizedProfit,
			}
		} else {
			takerPosition.EntryPrice = pos.EntryPrice
			takerPosition.PositionAmt = pos.PositionAmt
			takerPosition.UnRealizedProfit = pos.UnRealizedProfit
			tPositions[pos.Symbol] = takerPosition
		}

		tPositionsUpdateTimes[pos.Symbol] = time.Now()

		if lastPosition == nil ||
			lastPosition.PositionAmt != tPositions[pos.Symbol].PositionAmt ||
			lastPosition.EntryPrice != tPositions[pos.Symbol].EntryPrice {
			tHttpPositionUpdateSilentTimes[pos.Symbol] = time.Now().Add(*mtConfig.HttpSilent)
			if lastPosition != nil && lastPosition.PositionAmt*pos.PositionAmt >= 0 {
				if math.Abs(lastPosition.PositionAmt) < math.Abs(pos.PositionAmt) {
					logger.Debugf("WS POS ENTER SILENT %s %v",pos.Symbol, *mtConfig.EnterSilent)
				}
			}
			if pos.PositionAmt != 0 {
				tCloseTimeouts[pos.Symbol] = time.Now().Add(*mtConfig.CloseTimeout)
				logger.Debugf("WS POS SET TIMEOUT FOR %s", pos.Symbol)
			}
			//mtLoopTimer.Reset(time.Nanosecond)
			//logger.Debugf("WS POS %s %f %f", pos.Market, pos.PositionAmt, pos.EntryPrice)
		}
	}
	for _, balance := range data.Account.Balances {
		if balance.Asset == "USDT" {
			wb := balance.WalletBalance
			cwb := balance.CrossWalletBalance
			if tAccount == nil {
				tAccount = &bnswap.Asset{
					Asset:              balance.Asset,
					WalletBalance:      &wb,
					CrossWalletBalance: &cwb,
				}
				//logger.Debugf("WS USDT WB nil -> %f", wb)
				//mtLoopTimer.Reset(time.Nanosecond)
			} else {
				if tAccount.WalletBalance != nil && *tAccount.WalletBalance != wb {
					//mtLoopTimer.Reset(time.Nanosecond)
					//logger.Debugf("WS USDT WB %f -> %f", *tAccount.WalletBalance, wb)
				}
				tAccount.WalletBalance = &wb
				tAccount.CrossWalletBalance = &cwb
			}
			continue
		}
	}
}
