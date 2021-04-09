package main

import (
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func (st *Strategy) handleHttpPosition(positions []bnswap.Position) {
	for _, nextPos := range positions {
		symbolIndex := GetSymbolIndex(nextPos.Symbol)
		if symbolIndex == -1 {
			continue
		}
		if nextPos.PositionSide != "BOTH" {
			continue
		}
		if nextPos.UpdateTime.Sub(st.LastOrderTimes[symbolIndex]) < *st.Config.PullInterval {
			continue
		}
		st.PositionsUpdateTimes[symbolIndex] = time.Now()
		if st.Positions[symbolIndex].Symbol == "" ||
			st.Positions[symbolIndex].PositionAmt != nextPos.PositionAmt {
			logger.Debugf("HTTP POSITION CHANGED NEW %s", nextPos.ToString())
		}
		if st.Positions[symbolIndex].Symbol == "" {
			st.Positions[symbolIndex] = bnswap.Position{
				Symbol:           nextPos.Symbol,
				EntryPrice:       nextPos.EntryPrice,
				PositionAmt:      nextPos.PositionAmt,
				UnRealizedProfit: nextPos.UnRealizedProfit,
			}
		} else {

			if st.Positions[symbolIndex].EntryPrice > 0 && st.Positions[symbolIndex].PositionAmt*nextPos.PositionAmt < 0 {
				if nextPos.PositionAmt < 0 {
					st.RealisedProfits[symbolIndex] = (nextPos.EntryPrice - st.Positions[symbolIndex].EntryPrice) / st.Positions[symbolIndex].EntryPrice
					logger.Debugf("%s CLOSE LONG REALISED PROFIT PCT %f", nextPos.Symbol, st.RealisedProfits[symbolIndex])
				} else if nextPos.PositionAmt > 0 {
					st.RealisedProfits[symbolIndex] = (st.Positions[symbolIndex].EntryPrice - nextPos.EntryPrice) / st.Positions[symbolIndex].EntryPrice
					logger.Debugf("%s CLOSE SHORT REALISED PROFIT PCT %f", nextPos.Symbol, st.RealisedProfits[symbolIndex])
				}
			}

			st.Positions[symbolIndex].EntryPrice = nextPos.EntryPrice
			st.Positions[symbolIndex].PositionAmt = nextPos.PositionAmt
			st.Positions[symbolIndex].UnRealizedProfit = nextPos.UnRealizedProfit
		}
	}
}

func (st *Strategy) handleSwapHttpAccount(account bnswap.Account) {
	for _, asset := range account.Assets {
		if asset.Asset == "USDT" {
			asset := asset
			st.USDTAsset = asset
			continue
		} else if asset.Asset == "BNB" {
			asset := asset
			st.BNBAsset = asset
			continue
		}
	}
}
