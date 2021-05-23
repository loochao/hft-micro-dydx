package main

import (
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func handleWSAccountEvent(data *bnswap.BalanceAndPositionUpdateEvent) {
	for _, nextPos := range data.Account.Positions {
		if _, ok := bnspotOffsets[nextPos.Symbol]; !ok {
			continue
		}
		if nextPos.PositionSide != "BOTH" {
			continue
		}
		var lastPosition *bnswap.Position
		if currentPosition, ok := bnswapPositions[nextPos.Symbol]; !ok {
			bnswapPositions[nextPos.Symbol] = bnswap.Position{
				Symbol:           nextPos.Symbol,
				EntryPrice:       nextPos.EntryPrice,
				PositionAmt:      nextPos.PositionAmt,
				UnRealizedProfit: nextPos.UnRealizedProfit,
				PositionSide:     "BOTH",
			}
		} else {
			if currentPosition.EventTime.Sub(nextPos.EventTime) >= 0 {
				logger.Debugf("%s nextPos EventTime is older %v < %v", nextPos.EventTime, currentPosition.EventTime)
				continue
			}
			lastPosition = &currentPosition
			currentPosition.EntryPrice = nextPos.EntryPrice
			currentPosition.PositionAmt = nextPos.PositionAmt
			currentPosition.UnRealizedProfit = nextPos.UnRealizedProfit
			bnswapPositions[nextPos.Symbol] = currentPosition
		}

		bnswapPositionsUpdateTimes[nextPos.Symbol] = time.Now()

		if lastPosition == nil ||
			lastPosition.PositionAmt != bnswapPositions[nextPos.Symbol].PositionAmt ||
			lastPosition.EntryPrice != bnswapPositions[nextPos.Symbol].EntryPrice {
			logger.Debugf("WS POSITION CHANGED NEW %s", nextPos.ToString())
			bnswapHttpPositionUpdateSilentTimes[nextPos.Symbol] = time.Now().Add(*bnConfig.HttpSilent)
			bnLoopTimer.Reset(time.Nanosecond)
		}
	}
	for _, balance := range data.Account.Balances {
		if balance.Asset == "USDT" {
			wb := balance.WalletBalance
			cwb := balance.CrossWalletBalance
			if bnswapUSDTAsset == nil {
				bnswapUSDTAsset = &bnswap.Asset{
					Asset:              balance.Asset,
					WalletBalance:      &wb,
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
					Asset:              balance.Asset,
					WalletBalance:      &wb,
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
