package main

import (
	"github.com/geometrybase/hft-micro/influx/client"
	"github.com/geometrybase/hft-micro/logger"
	"strings"
	"time"
)

func handleSave() {

	if !kcperpAssetUpdatedForInflux || !kcspotBalanceUpdatedForInflux ||
		time.Now().Sub(kcSaveSilentTime).Seconds() < 0 {
		return
	}
	kcperpAssetUpdatedForInflux = false
	kcspotBalanceUpdatedForInflux = false

	var totalSpotBalance, totalPerpUSDTBalance *float64

	if kcspotUSDTBalance != nil {
		spotBalance := kcspotUSDTBalance.Available + kcspotUSDTBalance.Holds
		getAllBalances := true
		for _, spotSymbol := range kcspotSymbols {
			balance, okBalance := kcspotBalances[spotSymbol]
			spread, okSpread := kcSpreads[spotSymbol]
			if okBalance && okSpread {
				spotBalance += spread.SpotOrderBook.TakerBidVWAP * (balance.Available + balance.Holds)
			} else {
				logger.Debugf("%s MISS BALANCE %v OR TAKER VWAP %v", spotSymbol, okBalance, spread.SpotOrderBook.TakerBidVWAP)
				getAllBalances = false
				break
			}
		}
		if getAllBalances {
			totalSpotBalance = &spotBalance
			fields := make(map[string]interface{})
			fields["spotBalance"] = *totalSpotBalance
			fields["spotUsdtAvailable"] = kcspotUSDTBalance.Available
			fields["spotUsdtHolds"] = kcspotUSDTBalance.Holds
			pt, err := client.NewPoint(
				*kcConfig.InternalInflux.Measurement,
				map[string]string{
					"type": "spotBalance",
				},
				fields,
				time.Now().UTC(),
			)
			if err != nil {
				logger.Debugf("Spot Balance NewPoint error %v", err)
			} else {
				go kcInfluxWriter.Push(pt)
			}
		}
	}

	if kcperpUSDTAccount != nil {
		fields := make(map[string]interface{})
		fields["perpMarginBalance"] = kcperpUSDTAccount.MarginBalance
		fields["perpAvailableBalance"] = kcperpUSDTAccount.AvailableBalance
		fields["perpUnrealisedPNL"] = kcperpUSDTAccount.UnrealisedPNL
		fields["perpPositionMargin"] = kcperpUSDTAccount.PositionMargin
		pt, err := client.NewPoint(
			*kcConfig.InternalInflux.Measurement,
			map[string]string{
				"type": "perpBalance",
			},
			fields,
			time.Now().UTC(),
		)
		if err != nil {
			logger.Debugf("Perp Balance NewPoint error %v", err)
		} else {
			go kcInfluxWriter.Push(pt)
		}
		tp := kcperpUSDTAccount.MarginBalance + kcperpUSDTAccount.UnrealisedPNL
		totalPerpUSDTBalance = &tp
	}

	for _, perpSymbol := range kcperpSymbols {
		spotSymbol := kcpsSymbolsMap[perpSymbol]
		fields := make(map[string]interface{})
		if position, ok := kcperpPositions[perpSymbol]; ok {
			fields["perpCurrentQty"] = position.CurrentQty
			if markPrice, ok := kcperpMarkPrices[perpSymbol]; ok {
				fields["perpValue"] = position.CurrentQty * kcperpMultipliers[perpSymbol] * markPrice.IndexPrice
			}
		}
		if spotBalance, ok := kcspotBalances[spotSymbol]; ok {
			fields["spotBalance"] = spotBalance.Available + spotBalance.Holds
			if markPrice, ok := kcperpMarkPrices[perpSymbol]; ok {
				fields["spotValue"] = markPrice.IndexPrice * (spotBalance.Available + spotBalance.Holds)
			}
		}
		if fr, ok := kcperpFundingRates[perpSymbol]; ok {
			fields["perpNextFundingRate"] = fr.Value
			fields["perpPredictedFundingRate"] = fr.PredictedValue
		}
		if spread, ok := kcSpreads[spotSymbol]; ok {
			fields["lastEnterSpread"] = spread.LastEnter
			fields["lastExitSpread"] = spread.LastExit
			fields["medianEnterSpread"] = spread.MedianEnter
			fields["medianExitSpread"] = spread.MedianExit

			fields["spotTakerBidVWAP"] = spread.SpotOrderBook.TakerBidVWAP
			fields["spotMakerBidVWAP"] = spread.SpotOrderBook.MakerBidVWAP
			fields["spotTakerAskVWAP"] = spread.SpotOrderBook.TakerAskVWAP
			fields["spotMakerAskVWAP"] = spread.SpotOrderBook.MakerAskVWAP
			fields["spotTakerAskFarPrice"] = spread.SpotOrderBook.TakerAskFarPrice
			fields["spotTakerBidFarPrice"] = spread.SpotOrderBook.TakerBidFarPrice
			fields["spotTakerAskFarPrice5"] = (1.0 + *kcConfig.MakerBandOffset) * spread.SpotOrderBook.AskPrice
			fields["spotTakerBidFarPrice5"] = (1.0 - *kcConfig.MakerBandOffset) * spread.SpotOrderBook.BidPrice
			if order, ok := kcspotOpenOrders[spotSymbol]; ok {
				fields["spotOpenOrderPrice"] = order.Price
			}

			fields["perpTakerBidVWAP"] = spread.PerpOrderBook.TakerBidVWAP
			fields["perpMakerBidVWAP"] = spread.PerpOrderBook.MakerBidVWAP
			fields["perpTakerAskVWAP"] = spread.PerpOrderBook.TakerAskVWAP
			fields["perpMakerAskVWAP"] = spread.PerpOrderBook.MakerAskVWAP

			fields["age"] = spread.Age.Seconds()
			fields["ageDiff"] = spread.AgeDiff.Seconds()
		}
		if realisedSpread, ok := kcRealisedSpread[spotSymbol]; ok {
			fields["realisedSpread"] = realisedSpread
		}
		if quantile, ok := kcQuantiles[spotSymbol]; ok {
			fields["quantileBot"] = quantile.Bot
			fields["quantileTop"] = quantile.Top
			fields["quantileMid"] = quantile.Mid
			fields["quantileMaClose"] = quantile.MaClose
		}
		pt, err := client.NewPoint(
			*kcConfig.InternalInflux.Measurement,
			map[string]string{
				"perpSymbol": perpSymbol,
				"spotSymbol": spotSymbol,
				"type":       "singleBalance",
			},
			fields,
			time.Now().UTC(),
		)
		if err != nil {
			logger.Debugf("new position point error %v", err)
		} else {
			go kcInfluxWriter.Push(pt)
		}
	}

	if totalSpotBalance != nil && totalPerpUSDTBalance != nil {
		netWorth := (*totalSpotBalance + *totalPerpUSDTBalance) / *kcConfig.StartValue
		fields := make(map[string]interface{})
		fields["totalBalance"] = *totalSpotBalance + *totalPerpUSDTBalance
		fields["perpBalance"] = *totalPerpUSDTBalance
		fields["spotBalance"] = *totalSpotBalance
		fields["netWorth"] = (*totalSpotBalance + *totalPerpUSDTBalance) / *kcConfig.StartValue
		fields["startValue"] = *kcConfig.StartValue
		fields["netWorth"] = netWorth
		for name, start := range kcConfig.StartValues {
			if start > 0 {
				fields["refStartValue_"+strings.ToLower(name)] = start
				fields["currentValue_"+strings.ToLower(name)] = netWorth * start
			}
		}
		pt, err := client.NewPoint(
			*kcConfig.InternalInflux.Measurement,
			map[string]string{
				"type": "totalBalance",
			},
			fields,
			time.Now().UTC(),
		)
		if err != nil {
			logger.Debugf("Total Balance NewPoint error %v", err)
		} else {
			go kcInfluxWriter.Push(pt)
		}
	}
}

func handleExternalInfluxSave() {
	if !kcperpAssetUpdatedForExternalInflux ||
		!kcspotBalanceUpdatedForExternalInflux ||
		time.Now().Sub(kcSaveSilentTime).Seconds() < 0 {
		return
	}
	kcperpAssetUpdatedForExternalInflux = false
	kcspotBalanceUpdatedForExternalInflux = false

	var totalSpotBalance, totalPerpUSDTBalance *float64

	if kcspotUSDTBalance != nil {
		spotBalance := kcspotUSDTBalance.Available + kcspotUSDTBalance.Holds
		getAllBalances := true
		for _, spotSymbol := range kcspotSymbols {
			balance, okBalance := kcspotBalances[spotSymbol]
			spread, okSpread := kcSpreads[spotSymbol]
			if okBalance && okSpread {
				spotBalance += spread.SpotOrderBook.TakerBidVWAP * (balance.Available + balance.Holds)
			} else {
				getAllBalances = false
				break
			}
		}
		if getAllBalances {
			totalSpotBalance = &spotBalance
		}
	}

	if kcperpUSDTAccount != nil {
		tp := kcperpUSDTAccount.MarginBalance + kcperpUSDTAccount.UnrealisedPNL
		totalPerpUSDTBalance = &tp
	}

	if totalSpotBalance != nil && totalPerpUSDTBalance != nil {
		fields := make(map[string]interface{})
		netWorth := (*totalSpotBalance + *totalPerpUSDTBalance) / *kcConfig.StartValue
		fields["netWorth"] = netWorth
		for name, start := range kcConfig.StartValues {
			if start > 0 {
				fields["currentValue_"+strings.ToLower(name)] = netWorth * start
			}
		}
		if len(fields) > 0 {
			pt, err := client.NewPoint(
				*kcConfig.ExternalInflux.Measurement,
				map[string]string{
					"name": *kcConfig.Name,
				},
				fields,
				time.Now().UTC(),
			)
			if err != nil {
				logger.Debugf("Margin NewPoint error %v", err)
			} else {
				go kcExternalInfluxWriter.Push(pt)
			}
		}
	}
}
