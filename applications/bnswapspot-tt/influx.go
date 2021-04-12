package main

import (
	"github.com/geometrybase/hft-micro/influx/client"
	"github.com/geometrybase/hft-micro/logger"
	"strings"
	"time"
)

func handleSave() {

	if !bnswapAssetUpdatedForInflux || !bnspotBalanceUpdatedForInflux ||
		time.Now().Sub(bnSaveSilentTime).Seconds() < 0 {
		return
	}
	bnswapAssetUpdatedForInflux = false
	bnspotBalanceUpdatedForInflux = false

	var totalSpotBalance, totalSwapUSDTBalance, totalSwapBnBBalance *float64

	if bnspotUSDTBalance != nil {
		spotBalance := bnspotUSDTBalance.Free + bnspotUSDTBalance.Locked
		getAllBalances := true
		for _, symbol := range bnSymbols {
			balance, okBalance := bnspotBalances[symbol]
			markPrice, okMarkPrice := bnswapMarkPrices[symbol]
			if okBalance && okMarkPrice {
				spotBalance += markPrice.IndexPrice * (balance.Free + balance.Locked)
			} else {
				logger.Debugf("%s MISS BALANCE %v OR VWAP %v", symbol, okBalance, okMarkPrice)
				getAllBalances = false
				break
			}
		}
		if getAllBalances {
			totalSpotBalance = &spotBalance
			fields := make(map[string]interface{})
			fields["spotBalance"] = *totalSpotBalance
			fields["spotUsdtFreeBalance"] = bnspotUSDTBalance.Free
			fields["spotUsdtLockedBalance"] = bnspotUSDTBalance.Locked
			pt, err := client.NewPoint(
				*bnConfig.InternalInflux.Measurement,
				map[string]string{
					"type": "spotBalance",
				},
				fields,
				time.Now().UTC(),
			)
			if err != nil {
				logger.Debugf("Spot Balance NewPoint error %v", err)
			} else {
				go bnInfluxWriter.Push(pt)
			}
		}
	}

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
		if bnswapBNBAsset != nil && bnswapBNBAsset.MarginBalance != nil {
			if markPrice, ok := bnswapMarkPrices[bnBNBSymbol]; ok {
				balance := *bnswapBNBAsset.MarginBalance * markPrice.IndexPrice
				fields["swapBNBMarginBalance"] = *bnswapBNBAsset.MarginBalance
				fields["swapBNBBalance"] = balance
				totalSwapBnBBalance = &balance
			}
		}
		pt, err := client.NewPoint(
			*bnConfig.InternalInflux.Measurement,
			map[string]string{
				"type": "swapBalance",
			},
			fields,
			time.Now().UTC(),
		)
		if err != nil {
			logger.Debugf("Swap Balance NewPoint error %v", err)
		} else {
			go bnInfluxWriter.Push(pt)
		}
		totalSwapUSDTBalance = bnswapUSDTAsset.MarginBalance
	}

	for _, symbol := range bnSymbols {
		fields := make(map[string]interface{})
		if position, ok := bnswapPositions[symbol]; ok {
			fields["swapBalance"] = position.PositionAmt
			//fields["swapEntryPrice"] = position.EntryPrice
			//fields["swapEntryValue"] = position.EntryPrice * position.PositionAmt
			//if position.PositionAmt != 0 {
			//	fields["swapUnRealizedProfit"] = position.UnRealizedProfit
			//	fields["swapLiquidationPrice"] = position.LiquidationPrice
			//	fields["swapMarkPrice"] = position.MarkPrice
			//	fields["swapMaxNotionalValue"] = position.MaxNotionalValue
			//}
			//if orderBook, ok := bnswapOrderBooks[symbol]; ok {
			//	fields["swapURPnl"] = position.PositionAmt * ((orderBook.Bids[0][0]+orderBook.Asks[0][0])*0.5 - position.EntryPrice)
			//	fields["swapClose"] = (orderBook.Bids[0][0] + orderBook.Asks[0][0]) * 0.5
			//	fields["swapValue"] = (orderBook.Bids[0][0] + orderBook.Asks[0][0]) * 0.5 * position.PositionAmt
			//}
		}
		if spotBalance, ok := bnspotBalances[symbol]; ok {
			fields["spotBalance"] = spotBalance.Free + spotBalance.Locked
			if markPrice, ok := bnswapMarkPrices[symbol]; ok {
				fields["spotValue"] = markPrice.IndexPrice * (spotBalance.Free + spotBalance.Locked)
			}
		}
		if markPrice, ok := bnswapMarkPrices[symbol]; ok {
			fields["swapNextFundingRate"] = markPrice.FundingRate
		}
		if spread, ok := bnSpreads[symbol]; ok {
			fields["lastEnterSpread"] = spread.LastEnter
			fields["lastExitSpread"] = spread.LastExit
			fields["medianEnterSpread"] = spread.MedianEnter
			fields["medianExitSpread"] = spread.MedianExit

			fields["spotTakerBidVWAP"] = spread.SpotOrderBook.TakerBidVWAP
			fields["spotTakerAskVWAP"] = spread.SpotOrderBook.TakerAskVWAP
			fields["spotTakerAskFarPrice"] = spread.SpotOrderBook.TakerAskFarPrice
			fields["spotTakerBidFarPrice"] = spread.SpotOrderBook.TakerBidFarPrice

			fields["swapTakerBidVWAP"] = spread.SwapOrderBook.TakerBidVWAP
			fields["swapTakerAskVWAP"] = spread.SwapOrderBook.TakerAskVWAP

			fields["age"] = spread.Age.Seconds()
			fields["ageDiff"] = spread.AgeDiff.Seconds()
		}
		if realisedSpread, ok := bnRealisedSpread[symbol]; ok {
			fields["realisedSpread"] = realisedSpread
		}
		if quantile, ok := bnQuantiles[symbol]; ok {
			fields["quantileBot"] = quantile.Bot
			fields["quantileFarBot"] = quantile.FarBot
			fields["quantileTop"] = quantile.Top
			fields["quantileFarTop"] = quantile.FarTop
			fields["quantileMid"] = quantile.Mid
			fields["quantileMaClose"] = quantile.MaClose
		}
		pt, err := client.NewPoint(
			*bnConfig.InternalInflux.Measurement,
			map[string]string{
				"symbol": symbol,
				"type":   "singleBalance",
			},
			fields,
			time.Now().UTC(),
		)
		if err != nil {
			logger.Debugf("new position point error %v", err)
		} else {
			go bnInfluxWriter.Push(pt)
		}
	}

	if totalSpotBalance != nil && totalSwapUSDTBalance != nil && totalSwapBnBBalance != nil {
		fields := make(map[string]interface{})
		fields["totalBalance"] = *totalSpotBalance + *totalSwapUSDTBalance + *totalSwapBnBBalance
		fields["swapBalance"] = *totalSwapUSDTBalance + *totalSwapBnBBalance
		fields["spotBalance"] = *totalSpotBalance
		fields["netWorth"] = (*totalSpotBalance + *totalSwapUSDTBalance + *totalSwapBnBBalance) / *bnConfig.StartValue
		fields["startValue"] = *bnConfig.StartValue
		pt, err := client.NewPoint(
			*bnConfig.InternalInflux.Measurement,
			map[string]string{
				"type": "totalBalance",
			},
			fields,
			time.Now().UTC(),
		)
		if err != nil {
			logger.Debugf("Total Balance NewPoint error %v", err)
		} else {
			go bnInfluxWriter.Push(pt)
		}
	}
}

func handleExternalInfluxSave() {
	if !bnswapAssetUpdatedForExternalInflux ||
		!bnspotBalanceUpdatedForExternalInflux ||
		time.Now().Sub(bnSaveSilentTime).Seconds() < 0 {
		return
	}
	bnswapAssetUpdatedForExternalInflux = false
	bnspotBalanceUpdatedForExternalInflux = false

	var totalSpotBalance, totalSwapUSDTBalance, totalSwapBnBBalance *float64

	if bnspotUSDTBalance != nil {
		spotBalance := bnspotUSDTBalance.Free + bnspotUSDTBalance.Locked
		getAllBalances := true
		for _, symbol := range bnSymbols {
			balance, okBalance := bnspotBalances[symbol]
			markPrice, okMP := bnswapMarkPrices[symbol]
			if okBalance && okMP {
				spotBalance += markPrice.IndexPrice * (balance.Free + balance.Locked)
			} else {
				getAllBalances = false
				break
			}
		}
		if getAllBalances {
			totalSpotBalance = &spotBalance
		}
	}

	if bnswapBNBAsset != nil && bnswapBNBAsset.MarginBalance != nil {
		if spread, ok := bnSpreads[bnBNBSymbol]; ok {
			balance := *bnswapBNBAsset.MarginBalance * (spread.SpotOrderBook.BidPrice + spread.SpotOrderBook.AskPrice) * 0.5
			totalSwapBnBBalance = &balance
		}
	}

	if bnswapUSDTAsset != nil && bnswapUSDTAsset.MarginBalance != nil {
		totalSwapUSDTBalance = bnswapUSDTAsset.MarginBalance
	}

	if totalSpotBalance != nil && totalSwapUSDTBalance != nil && totalSwapBnBBalance != nil {
		netWorth := (*totalSpotBalance + *totalSwapUSDTBalance + *totalSwapBnBBalance) / *bnConfig.StartValue
		fields := make(map[string]interface{})
		fields["netWorth"] = netWorth
		for name, start := range bnConfig.StartValues {
			if start > 0 {
				fields["refStartValue_"+strings.ToLower(name)] = start
				fields["currentValue_"+strings.ToLower(name)] = netWorth * start
			}
		}
		if len(fields) > 0 {
			pt, err := client.NewPoint(
				*bnConfig.ExternalInflux.Measurement,
				map[string]string{
					"name": *bnConfig.Name,
				},
				fields,
				time.Now().UTC(),
			)
			if err != nil {
				logger.Debugf("Margin NewPoint error %v", err)
			} else {
				go bnExternalInfluxWriter.Push(pt)
			}
		}
	}
}
