package main

import (
	"github.com/geometrybase/hft-micro/influx/client"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func handleSave() {

	longValue := 0.0
	shortValue := 0.0
	frRet := 0.0
	if bnRankSymbolMap != nil {
		for rank, symbol := range bnRankSymbolMap {
			fields := make(map[string]interface{})
			if position, ok := bnswapPositions[symbol]; ok {
				fields["positionAmt"] = position.PositionAmt
				fields["positionVal"] = position.PositionAmt * position.EntryPrice
				fields["positionRank"] = rank
				if position.PositionAmt > 0 {
					longValue += position.PositionAmt * position.EntryPrice
				} else {
					shortValue += position.PositionAmt * position.EntryPrice
				}
				if markPrice, ok := bnswapMarkPrices[symbol]; ok {
					frRet -= position.PositionAmt * position.EntryPrice * markPrice.FundingRate
				}
			}
			if markPrice, ok := bnswapMarkPrices[symbol]; ok {
				fields["nextFundingRate"] = markPrice.FundingRate
			}
			if realisedSpread, ok := bnRealisedPnl[symbol]; ok {
				fields["realisedSpread"] = realisedSpread
			}
			pt, err := client.NewPoint(
				*bnConfig.InternalInflux.Measurement,
				map[string]string{
					"symbol": symbol,
					"type":   "symbol",
				},
				fields,
				time.Now().UTC(),
			)
			if err != nil {
				logger.Debugf("new position point error %v", err)
			} else {
				select {
				case <-time.After(time.Millisecond):
					logger.Debugf("PUSH TO INTERNAL INFLUX WRITER TIMEOUT IN 1MS")
				case bnInternalInfluxWriter.PushCh <- pt:
				}
			}
		}
	}
	if bnswapUSDTAsset != nil && bnswapUSDTAsset.MarginBalance != nil {
		fields := make(map[string]interface{})
		fields["balance"] = *bnswapUSDTAsset.MarginBalance
		fields["walletBalance"] = *bnswapUSDTAsset.WalletBalance
		fields["crossWalletBalance"] = *bnswapUSDTAsset.CrossWalletBalance
		fields["availableBalance"] = *bnswapUSDTAsset.AvailableBalance
		fields["positionInitialMargin"] = *bnswapUSDTAsset.PositionInitialMargin
		fields["maxWithdrawAmount"] = *bnswapUSDTAsset.MaxWithdrawAmount
		fields["openOrderInitialMargin"] = *bnswapUSDTAsset.OpenOrderInitialMargin
		fields["unRealizedProfit"] = *bnswapUSDTAsset.UnrealizedProfit
		fields["initialMargin"] = *bnswapUSDTAsset.InitialMargin
		fields["maintMargin"] = *bnswapUSDTAsset.MaintMargin
		fields["netWorth"] = *bnswapUSDTAsset.MarginBalance / *bnConfig.StartValue
		fields["startValue"] = *bnConfig.StartValue
		if longValue > 0 {
			fields["longValue"] = longValue
			fields["longValue"] = longValue
			fields["nextFundingReturn"] = frRet
		}
		pt, err := client.NewPoint(
			*bnConfig.InternalInflux.Measurement,
			map[string]string{
				"type": "account",
			},
			fields,
			time.Now().UTC(),
		)
		if err != nil {
			logger.Debugf("Swap Balance NewPoint error %v", err)
		} else {
			select {
			case <-time.After(time.Millisecond):
				logger.Debugf("PUSH TO INTERNAL INFLUX WRITER TIMEOUT IN 1MS")
			case bnInternalInfluxWriter.PushCh <- pt:
			}
		}
	}

}

func handleExternalInfluxSave() {
	if bnswapUSDTAsset != nil && bnswapUSDTAsset.MarginBalance != nil {
		fields := make(map[string]interface{})
		fields["netWorth"] = *bnswapUSDTAsset.MarginBalance / *bnConfig.StartValue
		pt, err := client.NewPoint(
			*bnConfig.InternalInflux.Measurement,
			map[string]string{
				"type": "account",
			},
			fields,
			time.Now().UTC(),
		)
		if err != nil {
			logger.Debugf("Swap Balance NewPoint error %v", err)
		} else {
			select {
			case <-time.After(time.Millisecond):
				logger.Debugf("PUSH TO EXTERNAL INFLUX WRITER TIMEOUT IN 1MS")
			case bnExternalInfluxWriter.PushCh <- pt:
			}
		}
	}
}
