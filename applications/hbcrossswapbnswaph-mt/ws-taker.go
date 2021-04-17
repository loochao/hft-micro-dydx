package main

import (
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func handleTakerWSAccount(data *bnswap.BalanceAndPositionUpdateEvent) {
	for _, pos := range data.Account.Positions {
		if _, ok := tmSymbolsMap[pos.Symbol]; !ok {
			continue
		}
		if pos.PositionSide != "BOTH" {
			continue
		}
		var lastPosition *bnswap.Position
		if p, ok := tPositions[pos.Symbol]; ok {
			p := p
			lastPosition = p
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
			logger.Debugf("WS POSITION CHANGED NEW %s", pos.ToString())
		}
	}
	for _, balance := range data.Account.Balances {
		if balance.Asset == "USDT" {
			wb := balance.WalletBalance
			cwb := balance.CrossWalletBalance
			if tUSDTAsset == nil {
				tUSDTAsset = &bnswap.Asset{
					Asset:         balance.Asset,
					WalletBalance: &wb,
					CrossWalletBalance: &cwb,
				}
			} else {
				tUSDTAsset.WalletBalance = &wb
				tUSDTAsset.CrossWalletBalance = &cwb
			}
			continue
		}
		if balance.Asset == "BNB" {
			wb := balance.WalletBalance
			cwb := balance.CrossWalletBalance
			if tBNBAsset == nil {
				tBNBAsset = &bnswap.Asset{
					Asset:         balance.Asset,
					WalletBalance: &wb,
					CrossWalletBalance: &cwb,
				}
			} else {
				tBNBAsset.WalletBalance = &wb
				tBNBAsset.CrossWalletBalance = &cwb
			}
			continue
		}
	}
}

