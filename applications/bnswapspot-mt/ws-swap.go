package main

import (
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func handleWSAccountEvent(data *bnswap.BalanceAndPositionUpdateEvent) {
	for _, pos := range data.Account.Positions {
		if _, ok := bnSymbolsMap[pos.Symbol]; !ok {
			continue
		}
		if pos.PositionSide != "BOTH" {
			continue
		}
		var lastPosition *bnswap.Position
		if p, ok := bnswapPositions[pos.Symbol]; ok {
			p := p
			lastPosition = &p
		}
		if futuresPosition, ok := bnswapPositions[pos.Symbol]; !ok {
			bnswapPositions[pos.Symbol] = bnswap.Position{
				Symbol:           pos.Symbol,
				EntryPrice:       pos.EntryPrice,
				PositionAmt:      pos.PositionAmt,
				UnRealizedProfit: pos.UnRealizedProfit,
			}
		} else {
			futuresPosition.EntryPrice = pos.EntryPrice
			futuresPosition.PositionAmt = pos.PositionAmt
			futuresPosition.UnRealizedProfit = pos.UnRealizedProfit
			bnswapPositions[pos.Symbol] = futuresPosition
		}

		bnswapPositionsUpdateTimes[pos.Symbol] = time.Now()

		if lastPosition == nil ||
			lastPosition.PositionAmt != bnswapPositions[pos.Symbol].PositionAmt ||
			lastPosition.EntryPrice != bnswapPositions[pos.Symbol].EntryPrice {
			//bnswapOrderSilentTimes[pos.Symbol] = time.Now()
			logger.Debugf("WS POSITION CHANGED NEW %s", pos.ToString())
		}
	}
	for _, balance := range data.Account.Balances {
		if balance.Asset == "USDT" {
			wb := balance.WalletBalance
			cwb := balance.CrossWalletBalance
			if bnswapUSDTAsset == nil {
				bnswapUSDTAsset = &bnswap.Asset{
					Asset:         balance.Asset,
					WalletBalance: &wb,
					CrossWalletBalance: &cwb,
				}
			} else {
				bnswapUSDTAsset.WalletBalance = &wb
				bnswapUSDTAsset.CrossWalletBalance = &cwb
			}
			continue
		}
		if balance.Asset == "BNB" {
			wb := balance.WalletBalance
			cwb := balance.CrossWalletBalance
			if bnswapBNBAsset == nil {
				bnswapBNBAsset = &bnswap.Asset{
					Asset:         balance.Asset,
					WalletBalance: &wb,
					CrossWalletBalance: &cwb,
				}
			} else {
				bnswapBNBAsset.WalletBalance = &wb
				bnswapBNBAsset.CrossWalletBalance = &cwb
			}
			continue
		}
	}
}

func handleWSOrder(wsOrder *bnswap.WSOrder) {
	if wsOrder.Status == "FILLED" ||
		wsOrder.Status == "CANCELED" ||
		wsOrder.Status == "REJECTED" ||
		wsOrder.Status == "EXPIRED" {
		bnswapOrderFinishCh <- *wsOrder.ToOrder()
	}
}
