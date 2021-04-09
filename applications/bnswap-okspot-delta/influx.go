package main

import (
	"github.com/geometrybase/hft/influx/client"
	"github.com/geometrybase/hft/logger"
	"time"
)

func handleInternalSave() {

	if !bnswapAssetUpdatedForInflux || !okspotBalanceUpdatedForInflux ||
		time.Now().Sub(bnSaveSilentTime).Seconds() < 0 {
		return
	}
	bnswapAssetUpdatedForInflux = false
	okspotBalanceUpdatedForInflux = false

	var totalSpotBalance, totalSwapUSDTBalance, totalSwapBnBBalance *float64

	if okspotUSDTBalance != nil {
		spotBalance := okspotUSDTBalance.Balance
		getAllBalances := true
		for _, symbol := range boSymbols {
			balance, okBalance := okspotBalances[symbol]
			markPrice, okMarkPrice := bnswapMarkPrices[symbol]
			if okBalance && okMarkPrice {
				spotBalance += markPrice.IndexPrice * balance.Balance
			} else {
				logger.Debugf("%s MISS BALANCE %v OR MARK PRICE %v", symbol, okBalance, okMarkPrice)
				getAllBalances = false
				break
			}
		}
		if getAllBalances {
			totalSpotBalance = &spotBalance
			fields := make(map[string]interface{})
			fields["spotBalance"] = *totalSpotBalance
			fields["spotUsdtFreeBalance"] = okspotUSDTBalance.Available
			fields["spotUsdtLockedBalance"] = okspotUSDTBalance.Hold
			pt, err := client.NewPoint(
				*boConfig.InternalInflux.Measurement,
				map[string]string{
					"type": "spotBalance",
				},
				fields,
				time.Now().UTC(),
			)
			if err != nil {
				logger.Debugf("Spot Balance NewPoint error %v", err)
			} else {
				go boInfluxWriter.Push(pt)
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
			*boConfig.InternalInflux.Measurement,
			map[string]string{
				"type": "swapBalance",
			},
			fields,
			time.Now().UTC(),
		)
		if err != nil {
			logger.Debugf("Swap Balance NewPoint error %v", err)
		} else {
			go boInfluxWriter.Push(pt)
		}
		totalSwapUSDTBalance = bnswapUSDTAsset.MarginBalance
	}

	for _, symbol := range boSymbols {
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
		if spotBalance, ok := okspotBalances[symbol]; ok {
			fields["spotBalance"] = spotBalance.Balance
			//fields["spotBalanceFree"] = spotBalance.Free
			//fields["spotBalanceLocked"] = spotBalance.Locked
			if midVwap, ok := bnMidVwaps[symbol]; ok {
				fields["midVwap"] = midVwap
				//fields["spotAskVwap"] = okspotAskVwaps[symbol]
				//fields["spotBidVwap"] = okspotBidVwaps[symbol]
				//fields["swapAskVwap"] = bnswapAskVwaps[symbol]
				//fields["swapBidVwap"] = bnswapBidVwaps[symbol]
				//fields["spotAskFarPrice"] = okspotAskFarPrices[symbol]
				//fields["spotBidFarPrice"] = okspotBidFarPrices[symbol]
				//fields["swapAskFarPrice"] = bnswapAskFarPrices[symbol]
				//fields["swapBidFarPrice"] = bnswapBidFarPrices[symbol]
				fields["spotValue"] = midVwap * spotBalance.Balance
			}
		}
		if markPrice, ok := bnswapMarkPrices[symbol]; ok {
			fields["swapNextFundingRate"] = markPrice.FundingRate
			//fields["swapMarkPrice"] = markPrice.MarkPrice
			//fields["swapIndexPrice"] = markPrice.IndexPrice
		}
		if lastEnterDelta, ok := boLastEnterDeltas[symbol]; ok {
			fields["lastEnterDelta"] = lastEnterDelta
			fields["lastExitDelta"] = boLastExitDeltas[symbol]
			fields["medianEnterDelta"] = boMedianEnterDeltas[symbol]
			fields["medianExitDelta"] = boMedianExitDeltas[symbol]
			//fields["enterDeltaWindowLen"] = len(boEnterDeltaWindows[symbol])
			//fields["exitDeltaWindowLen"] = len(boExitDeltaWindows[symbol])
			fields["arrivalTimeDiff"] = boArrivalTimes[symbol][len(boArrivalTimes[symbol])-1].Sub(boArrivalTimes[symbol][0]).Seconds()
		}
		if realisedDelta, ok := bnRealisedDelta[symbol]; ok {
			fields["realisedDelta"] = realisedDelta
		}
		if timeDelta, ok := boSwapSpotTimeDeltas[symbol]; ok {
			fields["timeDelta"] = timeDelta
		}
		if timeOffset, ok := boSystemTimeDeltas[symbol]; ok {
			fields["timeOffset"] = timeOffset
		}
		if quantile, ok := bnQuantiles[symbol]; ok {
			fields["quantileBot"] = quantile.Bot
			fields["quantileTop"] = quantile.Top
			fields["quantileMid"] = quantile.Mid
			fields["quantileMaClose"] = quantile.MaClose
		}
		pt, err := client.NewPoint(
			*boConfig.InternalInflux.Measurement,
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
			go boInfluxWriter.Push(pt)
		}
	}

	if totalSpotBalance != nil && totalSwapUSDTBalance != nil && totalSwapBnBBalance != nil {
		fields := make(map[string]interface{})
		fields["totalBalance"] = *totalSpotBalance + *totalSwapUSDTBalance + *totalSwapBnBBalance
		fields["swapBalance"] = *totalSwapUSDTBalance + *totalSwapBnBBalance
		fields["spotBalance"] = *totalSpotBalance
		fields["netWorth"] = (*totalSpotBalance + *totalSwapUSDTBalance + *totalSwapBnBBalance) / *boConfig.StartValue
		fields["startValue"] = *boConfig.StartValue
		//fields["entryValue"] = *boConfig.EntryStep
		//fields["topQuantile"] = *boConfig.Quantile
		//fields["botQuantile"] = *boConfig.BotQuantile
		//fields["resetUnrealisedTriggerPct"] = *boConfig.ResetUnrealisedTriggerPct
		//fields["resetUnrealisedPnlInterval"] = boConfig.ResetUnrealisedPnlInterval.Seconds()
		pt, err := client.NewPoint(
			*boConfig.InternalInflux.Measurement,
			map[string]string{
				"type": "totalBalance",
			},
			fields,
			time.Now().UTC(),
		)
		if err != nil {
			logger.Debugf("Total Balance NewPoint error %v", err)
		} else {
			go boInfluxWriter.Push(pt)
		}
	}
}

func handleExternalSave() {
	if !bnswapAssetUpdatedForExternalInflux ||
		!okspotBalanceUpdatedForExternalInflux ||
		time.Now().Sub(bnSaveSilentTime).Seconds() < 0 {
		return
	}
	bnswapAssetUpdatedForExternalInflux = false
	okspotBalanceUpdatedForExternalInflux = false

	var totalSpotBalance, totalSwapUSDTBalance, totalSwapBnBBalance *float64

	if okspotUSDTBalance != nil {
		spotBalance := okspotUSDTBalance.Balance
		getAllBalances := true
		for _, symbol := range boSymbols {
			balance, okBalance := okspotBalances[symbol]
			markPrice, okMarkPrice := bnswapMarkPrices[symbol]
			if okBalance && okMarkPrice {
				spotBalance += markPrice.IndexPrice * balance.Balance
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
		if markPrice, ok := bnswapMarkPrices[bnBNBSymbol]; ok {
			balance := *bnswapBNBAsset.MarginBalance * markPrice.IndexPrice
			totalSwapBnBBalance = &balance
		}
	}

	if bnswapUSDTAsset != nil && bnswapUSDTAsset.MarginBalance != nil {
		totalSwapUSDTBalance = bnswapUSDTAsset.MarginBalance
	}

	if totalSpotBalance != nil && totalSwapUSDTBalance != nil && totalSwapBnBBalance != nil {
		fields := make(map[string]interface{})
		fields["netWorth"] = (*totalSpotBalance + *totalSwapUSDTBalance + *totalSwapBnBBalance) / *boConfig.StartValue
		if len(fields) > 0 {
			pt, err := client.NewPoint(
				*boConfig.ExternalInflux.Measurement,
				map[string]string{
					"name": *boConfig.Name,
				},
				fields,
				time.Now().UTC(),
			)
			if err != nil {
				logger.Debugf("Margin NewPoint error %v", err)
			} else {
				go boExternalInfluxWriter.Push(pt)
			}
		}
	} else {
		logger.Debugf("SPOT %v SWAP USDT %v  SWAP BNB %v", totalSpotBalance, totalSwapUSDTBalance, totalSwapBnBBalance)
	}
}
