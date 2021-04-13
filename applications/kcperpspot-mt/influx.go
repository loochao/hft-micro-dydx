package main

//func handleSave() {
//
//	if !kcperpAssetUpdatedForInflux || !kcspotBalanceUpdatedForInflux ||
//		time.Now().Sub(bnSaveSilentTime).Seconds() < 0 {
//		return
//	}
//	kcperpAssetUpdatedForInflux = false
//	kcspotBalanceUpdatedForInflux = false
//
//	var totalSpotBalance, totalPerpUSDTBalance, totalPerpBnBBalance *float64
//
//	if kcspotUSDTBalance != nil {
//		spotBalance := kcspotUSDTBalance.Free + kcspotUSDTBalance.Locked
//		getAllBalances := true
//		for _, symbol := range kcspotSymbols {
//			balance, okBalance := kcspotBalances[symbol]
//			markPrice, okMarkPrice := kcperpMarkPrices[symbol]
//			if okBalance && okMarkPrice {
//				spotBalance += markPrice.IndexPrice * (balance.Free + balance.Locked)
//			} else {
//				logger.Debugf("%s MISS BALANCE %v OR VWAP %v", symbol, okBalance, okMarkPrice)
//				getAllBalances = false
//				break
//			}
//		}
//		if getAllBalances {
//			totalSpotBalance = &spotBalance
//			fields := make(map[string]interface{})
//			fields["spotBalance"] = *totalSpotBalance
//			fields["spotUsdtFreeBalance"] = kcspotUSDTBalance.Free
//			fields["spotUsdtLockedBalance"] = kcspotUSDTBalance.Locked
//			pt, err := client.NewPoint(
//				*kcConfig.InternalInflux.Measurement,
//				map[string]string{
//					"type": "spotBalance",
//				},
//				fields,
//				time.Now().UTC(),
//			)
//			if err != nil {
//				logger.Debugf("Spot Balance NewPoint error %v", err)
//			} else {
//				go bnInfluxWriter.Push(pt)
//			}
//		}
//	}
//
//	if kcperpUSDTAccount != nil && kcperpUSDTAccount.MarginBalance != nil {
//		fields := make(map[string]interface{})
//		fields["swapBalance"] = *kcperpUSDTAccount.MarginBalance
//		fields["swapWalletBalance"] = *kcperpUSDTAccount.WalletBalance
//		fields["swapCrossWalletBalance"] = *kcperpUSDTAccount.CrossWalletBalance
//		fields["swapAvailableBalance"] = *kcperpUSDTAccount.AvailableBalance
//		fields["swapPositionInitialMargin"] = *kcperpUSDTAccount.PositionInitialMargin
//		fields["swapMaxWithdrawAmount"] = *kcperpUSDTAccount.MaxWithdrawAmount
//		fields["swapOpenOrderInitialMargin"] = *kcperpUSDTAccount.OpenOrderInitialMargin
//		fields["swapUnRealizedProfit"] = *kcperpUSDTAccount.UnrealizedProfit
//		fields["swapInitialMargin"] = *kcperpUSDTAccount.InitialMargin
//		fields["swapMaintMargin"] = *kcperpUSDTAccount.MaintMargin
//		if kcperpBNBAsset != nil && kcperpBNBAsset.MarginBalance != nil {
//			if markPrice, ok := kcperpMarkPrices[bnBNBSymbol]; ok {
//				balance := *kcperpBNBAsset.MarginBalance * markPrice.IndexPrice
//				fields["swapBNBMarginBalance"] = *kcperpBNBAsset.MarginBalance
//				fields["swapBNBBalance"] = balance
//				totalPerpBnBBalance = &balance
//			}
//		}
//		pt, err := client.NewPoint(
//			*kcConfig.InternalInflux.Measurement,
//			map[string]string{
//				"type": "swapBalance",
//			},
//			fields,
//			time.Now().UTC(),
//		)
//		if err != nil {
//			logger.Debugf("Perp Balance NewPoint error %v", err)
//		} else {
//			go bnInfluxWriter.Push(pt)
//		}
//		totalPerpUSDTBalance = kcperpUSDTAccount.MarginBalance
//	}
//
//	for _, symbol := range kcspotSymbols {
//		fields := make(map[string]interface{})
//		if position, ok := kcperpPositions[symbol]; ok {
//			fields["swapBalance"] = position.PositionAmt
//			//fields["swapEntryPrice"] = position.EntryPrice
//			//fields["swapEntryValue"] = position.EntryPrice * position.PositionAmt
//			//if position.PositionAmt != 0 {
//			//	fields["swapUnRealizedProfit"] = position.UnRealizedProfit
//			//	fields["swapLiquidationPrice"] = position.LiquidationPrice
//			//	fields["swapMarkPrice"] = position.MarkPrice
//			//	fields["swapMaxNotionalValue"] = position.MaxNotionalValue
//			//}
//			//if orderBook, ok := kcperpOrderBooks[symbol]; ok {
//			//	fields["swapURPnl"] = position.PositionAmt * ((orderBook.Bids[0][0]+orderBook.Asks[0][0])*0.5 - position.EntryPrice)
//			//	fields["swapClose"] = (orderBook.Bids[0][0] + orderBook.Asks[0][0]) * 0.5
//			//	fields["swapValue"] = (orderBook.Bids[0][0] + orderBook.Asks[0][0]) * 0.5 * position.PositionAmt
//			//}
//		}
//		if spotBalance, ok := kcspotBalances[symbol]; ok {
//			fields["spotBalance"] = spotBalance.Free + spotBalance.Locked
//			if markPrice, ok := kcperpMarkPrices[symbol]; ok {
//				fields["spotValue"] = markPrice.IndexPrice * (spotBalance.Free + spotBalance.Locked)
//			}
//		}
//		if markPrice, ok := kcperpMarkPrices[symbol]; ok {
//			fields["swapNextFundingRate"] = markPrice.FundingRate
//		}
//		if spread, ok := bnSpreads[symbol]; ok {
//			fields["lastEnterSpread"] = spread.LastEnter
//			fields["lastExitSpread"] = spread.LastExit
//			fields["medianEnterSpread"] = spread.MedianEnter
//			fields["medianExitSpread"] = spread.MedianExit
//
//			fields["spotTakerBidVWAP"] = spread.SpotOrderBook.TakerBidVWAP
//			fields["spotMakerBidVWAP"] = spread.SpotOrderBook.MakerBidVWAP
//			fields["spotTakerAskVWAP"] = spread.SpotOrderBook.TakerAskVWAP
//			fields["spotMakerAskVWAP"] = spread.SpotOrderBook.MakerAskVWAP
//			fields["spotTakerAskFarPrice"] = spread.SpotOrderBook.TakerAskFarPrice
//			fields["spotTakerBidFarPrice"] = spread.SpotOrderBook.TakerBidFarPrice
//			fields["spotTakerAskFarPrice5"] = (1.0 + *kcConfig.MakerBandOffset) * spread.SpotOrderBook.AskPrice
//			fields["spotTakerBidFarPrice5"] = (1.0 - *kcConfig.MakerBandOffset) * spread.SpotOrderBook.BidPrice
//			if order, ok := kcspotOpenOrders[symbol]; ok {
//				fields["spotOpenOrderPrice"] = order.Price
//			}
//
//			fields["swapTakerBidVWAP"] = spread.PerpOrderBook.TakerBidVWAP
//			fields["swapMakerBidVWAP"] = spread.PerpOrderBook.MakerBidVWAP
//			fields["swapTakerAskVWAP"] = spread.PerpOrderBook.TakerAskVWAP
//			fields["swapMakerAskVWAP"] = spread.PerpOrderBook.MakerAskVWAP
//
//			fields["age"] = spread.Age.Seconds()
//			fields["ageDiff"] = spread.AgeDiff.Seconds()
//		}
//		if realisedSpread, ok := bnRealisedSpread[symbol]; ok {
//			fields["realisedSpread"] = realisedSpread
//		}
//		if quantile, ok := bnQuantiles[symbol]; ok {
//			fields["quantileBot"] = quantile.Bot
//			fields["quantileFarBot"] = quantile.FarBot
//			fields["quantileTop"] = quantile.Top
//			fields["quantileFarTop"] = quantile.FarTop
//			fields["quantileMid"] = quantile.Mid
//			fields["quantileMaClose"] = quantile.MaClose
//		}
//		pt, err := client.NewPoint(
//			*kcConfig.InternalInflux.Measurement,
//			map[string]string{
//				"symbol": symbol,
//				"type":   "singleBalance",
//			},
//			fields,
//			time.Now().UTC(),
//		)
//		if err != nil {
//			logger.Debugf("new position point error %v", err)
//		} else {
//			go bnInfluxWriter.Push(pt)
//		}
//	}
//
//	if totalSpotBalance != nil && totalPerpUSDTBalance != nil && totalPerpBnBBalance != nil {
//		netWorth := (*totalSpotBalance + *totalPerpUSDTBalance + *totalPerpBnBBalance) / *kcConfig.StartValue
//		fields := make(map[string]interface{})
//		fields["totalBalance"] = *totalSpotBalance + *totalPerpUSDTBalance + *totalPerpBnBBalance
//		fields["swapBalance"] = *totalPerpUSDTBalance + *totalPerpBnBBalance
//		fields["spotBalance"] = *totalSpotBalance
//		fields["netWorth"] = (*totalSpotBalance + *totalPerpUSDTBalance + *totalPerpBnBBalance) / *kcConfig.StartValue
//		fields["startValue"] = *kcConfig.StartValue
//		fields["netWorth"] = netWorth
//		for name, start := range kcConfig.StartValues {
//			if start > 0 {
//				fields["refStartValue_"+strings.ToLower(name)] = start
//				fields["currentValue_"+strings.ToLower(name)] = netWorth * start
//			}
//		}
//		pt, err := client.NewPoint(
//			*kcConfig.InternalInflux.Measurement,
//			map[string]string{
//				"type": "totalBalance",
//			},
//			fields,
//			time.Now().UTC(),
//		)
//		if err != nil {
//			logger.Debugf("Total Balance NewPoint error %v", err)
//		} else {
//			go bnInfluxWriter.Push(pt)
//		}
//	}
//}
//
//func handleExternalInfluxSave() {
//	if !kcperpAssetUpdatedForExternalInflux ||
//		!kcspotBalanceUpdatedForExternalInflux ||
//		time.Now().Sub(bnSaveSilentTime).Seconds() < 0 {
//		return
//	}
//	kcperpAssetUpdatedForExternalInflux = false
//	kcspotBalanceUpdatedForExternalInflux = false
//
//	var totalSpotBalance, totalPerpUSDTBalance, totalPerpBnBBalance *float64
//
//	if kcspotUSDTBalance != nil {
//		spotBalance := kcspotUSDTBalance.Free + kcspotUSDTBalance.Locked
//		getAllBalances := true
//		for _, symbol := range kcspotSymbols {
//			balance, okBalance := kcspotBalances[symbol]
//			markPrice, okMP := kcperpMarkPrices[symbol]
//			if okBalance && okMP {
//				spotBalance += markPrice.IndexPrice * (balance.Free + balance.Locked)
//			} else {
//				getAllBalances = false
//				break
//			}
//		}
//		if getAllBalances {
//			totalSpotBalance = &spotBalance
//		}
//	}
//
//	if kcperpBNBAsset != nil && kcperpBNBAsset.MarginBalance != nil {
//		if spread, ok := bnSpreads[bnBNBSymbol]; ok {
//			balance := *kcperpBNBAsset.MarginBalance * (spread.SpotOrderBook.BidPrice + spread.SpotOrderBook.AskPrice) * 0.5
//			totalPerpBnBBalance = &balance
//		}
//	}
//
//	if kcperpUSDTAccount != nil && kcperpUSDTAccount.MarginBalance != nil {
//		totalPerpUSDTBalance = kcperpUSDTAccount.MarginBalance
//	}
//
//	if totalSpotBalance != nil && totalPerpUSDTBalance != nil && totalPerpBnBBalance != nil {
//		fields := make(map[string]interface{})
//		netWorth := (*totalSpotBalance + *totalPerpUSDTBalance + *totalPerpBnBBalance) / *kcConfig.StartValue
//		fields["netWorth"] = netWorth
//		for name, start := range kcConfig.StartValues {
//			if start > 0 {
//				fields["currentValue_"+strings.ToLower(name)] = netWorth * start
//			}
//		}
//		if len(fields) > 0 {
//			pt, err := client.NewPoint(
//				*kcConfig.ExternalInflux.Measurement,
//				map[string]string{
//					"name": *kcConfig.Name,
//				},
//				fields,
//				time.Now().UTC(),
//			)
//			if err != nil {
//				logger.Debugf("Margin NewPoint error %v", err)
//			} else {
//				go bnExternalInfluxWriter.Push(pt)
//			}
//		}
//	}
//}
