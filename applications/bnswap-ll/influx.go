package main

import (
	"github.com/geometrybase/hft-micro/influx/client"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func handleSave() {

	if bnswapUSDTAsset != nil && bnswapUSDTAsset.MarginBalance != nil {
		fields := make(map[string]interface{})
		fields["swapBalance"] = *bnswapUSDTAsset.MarginBalance
		fields["swapWalletBalance"] = *bnswapUSDTAsset.WalletBalance
		fields["swapCrossWalletBalance"] = *bnswapUSDTAsset.CrossWalletBalance
		fields["swapAvailableBalance"] = *bnswapUSDTAsset.AvailableBalance
		fields["swapPositionInitialMargin"] = *bnswapUSDTAsset.PositionInitialMargin
		fields["swapMaxWithdrawAmount"] = *bnswapUSDTAsset.MaxWithdrawAmount
		fields["swapOpenOrderInitialMargin"] = *bnswapUSDTAsset.OpenOrderInitialMargin
		fields["swapUnRealizedProfit"] = *bnswapUSDTAsset.UnrealizedProfit
		fields["swapInitialMargin"] = *bnswapUSDTAsset.InitialMargin
		fields["swapMaintMargin"] = *bnswapUSDTAsset.MaintMargin
		fields["netWorth"] = *bnswapUSDTAsset.MarginBalance / *bnConfig.StartValue
		fields["startValue"] = *bnConfig.StartValue
		if bnSignal != nil {
			fields["direction"] = bnSignal.Direction
			fields["slowSellPrice"] = bnSignal.SlowSellPrice
			fields["slowBuyPrice"] = bnSignal.SlowBuyPrice
			fields["fastSellPrice"] = bnSignal.FastSellPrice
			fields["fastBuyPrice"] = bnSignal.FastBuyPrice
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

	for _, symbol := range bnSymbols {
		fields := make(map[string]interface{})
		if position, ok := bnswapPositions[symbol]; ok {
			fields["swapBalance"] = position.PositionAmt
		}
		if markPrice, ok := bnswapMarkPrices[symbol]; ok {
			fields["swapNextFundingRate"] = markPrice.FundingRate
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
