package main

import (
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func handleWSAccountEvent(data *bnswap.BalanceAndPositionUpdateEvent) {
	logger.Debugf("SWAP WS %v", data.EventTime)
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
				UnrealizedProfit: nextPos.UnRealizedProfit,
				PositionSide:     "BOTH",
				EventTime:        data.EventTime,
				ParseTime:        data.ParseTime,
			}
		} else {
			if currentPosition.EventTime.Sub(nextPos.EventTime) > 0 {
				logger.Debugf("%s nextPos EventTime is older %v < %v", nextPos.EventTime, currentPosition.EventTime)
				continue
			}
			lastPosition = &bnswap.Position{}
			*lastPosition = currentPosition
			currentPosition.EventTime = nextPos.EventTime
			currentPosition.ParseTime = nextPos.ParseTime
			currentPosition.EntryPrice = nextPos.EntryPrice
			currentPosition.PositionAmt = nextPos.PositionAmt
			currentPosition.UnrealizedProfit = nextPos.UnRealizedProfit
			bnswapPositions[nextPos.Symbol] = currentPosition
		}

		bnswapPositionsUpdateTimes[nextPos.Symbol] = time.Now()

		if lastPosition == nil ||
			lastPosition.PositionAmt != bnswapPositions[nextPos.Symbol].PositionAmt ||
			lastPosition.EntryPrice != bnswapPositions[nextPos.Symbol].EntryPrice {
			logger.Debugf("WS POSITION CHANGED NEW %s", nextPos.ToString())
			if lastPosition != nil {
				logger.Debugf("SWAP %s POS OLD TIME %v NEW TIME %v", nextPos.Symbol, lastPosition.EventTime, nextPos.EventTime)
			}
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
					EventTime:          balance.EventTime,
					ParseTime:          balance.ParseTime,
				}
			} else if balance.EventTime.Sub(bnswapUSDTAsset.EventTime) >= 0 {
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
					EventTime:          balance.EventTime,
					ParseTime:          balance.ParseTime,
				}
			} else if balance.EventTime.Sub(bnswapBNBAsset.EventTime) >= 0 {
				bnswapBNBAsset.WalletBalance = &wb
				bnswapBNBAsset.CrossWalletBalance = &cwb
			}
			continue
		}
	}
}
