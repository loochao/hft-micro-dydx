package main

import (
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func (st *Strategy) handleWSAccountEvent(data *bnswap.BalanceAndPositionUpdateEvent) {
	for _, pos := range data.Account.Positions {
		symbolIndex := GetSymbolIndex(pos.Symbol)
		if symbolIndex == -1 {
			continue
		}
		if pos.PositionSide != "BOTH" {
			continue
		}
		if st.Positions[symbolIndex].Symbol == "" ||
			st.Positions[symbolIndex].PositionAmt != pos.PositionAmt {
			logger.Debugf("WS POSITION CHANGED NEW %s", pos.ToString())
		}
		if st.Positions[symbolIndex].Symbol == "" {
			st.Positions[symbolIndex] = bnswap.Position{
				Symbol:           pos.Symbol,
				EntryPrice:       pos.EntryPrice,
				PositionAmt:      pos.PositionAmt,
				UnRealizedProfit: pos.UnRealizedProfit,
			}
		} else {
			if st.Positions[symbolIndex].EntryPrice > 0 && st.Positions[symbolIndex].PositionAmt*pos.PositionAmt < 0{
				if pos.PositionAmt < 0 {
					st.RealisedProfits[symbolIndex] = (pos.EntryPrice - st.Positions[symbolIndex].EntryPrice) / st.Positions[symbolIndex].EntryPrice
					logger.Debugf("%s CLOSE LONG REALISED PROFIT PCT %f", pos.Symbol, st.RealisedProfits[symbolIndex])
				}else if pos.PositionAmt > 0 {
					st.RealisedProfits[symbolIndex] = (st.Positions[symbolIndex].EntryPrice - pos.EntryPrice) / st.Positions[symbolIndex].EntryPrice
					logger.Debugf("%s CLOSE SHORT REALISED PROFIT PCT %f", pos.Symbol, st.RealisedProfits[symbolIndex])
				}
			}
			st.Positions[symbolIndex].EntryPrice = pos.EntryPrice
			st.Positions[symbolIndex].PositionAmt = pos.PositionAmt
			st.Positions[symbolIndex].UnRealizedProfit = pos.UnRealizedProfit
		}
		st.PositionsUpdateTimes[symbolIndex] = time.Now()
	}
	for _, balance := range data.Account.Balances {
		if balance.Asset == "USDT" {
			wb := balance.WalletBalance
			cwb := balance.CrossWalletBalance
			st.USDTAsset.Asset = balance.Asset
			st.USDTAsset.WalletBalance = &wb
			st.USDTAsset.CrossWalletBalance = &cwb
			continue
		} else if balance.Asset == "BNB" {
			wb := balance.WalletBalance
			cwb := balance.CrossWalletBalance
			st.BNBAsset.Asset = balance.Asset
			st.BNBAsset.WalletBalance = &wb
			st.BNBAsset.CrossWalletBalance = &cwb
			continue
		}
	}
}


func (st *Strategy) handleWSOrder(wsOrder *bnswap.WSOrder) {
	if wsOrder.Status == "FILLED" ||
		wsOrder.Status == "CANCELED" ||
		wsOrder.Status == "REJECTED" ||
		wsOrder.Status == "EXPIRED" {
		st.OrderFinishCh <- *wsOrder.ToOrder()
	}
}
