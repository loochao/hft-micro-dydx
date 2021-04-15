package main

import (
	"github.com/geometrybase/hft-micro/hbcrossswap"
	"github.com/geometrybase/hft-micro/influx/client"
	"github.com/geometrybase/hft-micro/logger"
	"strings"
	"time"
)

func handleSave() {

	if !hbcrossswapAssetUpdatedForInflux || !hbspotBalanceUpdatedForInflux ||
		time.Now().Sub(kcSaveSilentTime).Seconds() < 0 {
		return
	}
	hbcrossswapAssetUpdatedForInflux = false
	hbspotBalanceUpdatedForInflux = false

	var totalSpotBalance, totalPerpUSDTBalance *float64

	if hbspotUSDTBalance != nil {
		spotBalance := hbspotUSDTBalance.Balance
		getAllBalances := true
		for _, spotSymbol := range hbspotSymbols {
			balance, okBalance := hbspotBalances[spotSymbol]
			spread, okSpread := hbSpreads[spotSymbol]
			if okBalance && okSpread {
				spotBalance += spread.SpotOrderBook.TakerBidVWAP *balance.Balance
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
			fields["spotUsdtAvailable"] = hbspotUSDTBalance.Available
			fields["spotUsdtFrozen"] = hbspotUSDTBalance.Frozen
			pt, err := client.NewPoint(
				*hbConfig.InternalInflux.Measurement,
				map[string]string{
					"type": "spotBalance",
				},
				fields,
				time.Now().UTC(),
			)
			if err != nil {
				logger.Debugf("Spot Balance NewPoint error %v", err)
			} else {
				go hbInfluxWriter.Push(pt)
			}
		}
	}

	if hbcrossswapAccount != nil {
		fields := make(map[string]interface{})
		fields["swapMarginBalance"] = hbcrossswapAccount.MarginBalance
		fields["swapWithdrawAvailable"] = hbcrossswapAccount.WithdrawAvailable
		fields["swapProfitUnreal"] = hbcrossswapAccount.ProfitUnreal
		fields["swapProfitUnreal"] = hbcrossswapAccount.MarginPosition
		pt, err := client.NewPoint(
			*hbConfig.InternalInflux.Measurement,
			map[string]string{
				"type": "swapBalance",
			},
			fields,
			time.Now().UTC(),
		)
		if err != nil {
			logger.Debugf("Perp Balance NewPoint error %v", err)
		} else {
			go hbInfluxWriter.Push(pt)
		}
		tp := hbcrossswapAccount.MarginBalance + hbcrossswapAccount.ProfitUnreal
		totalPerpUSDTBalance = &tp
	}

	for _, swapSymbol := range hbcrossswapSymbols {
		spotSymbol := kcpsSymbolsMap[swapSymbol]
		fields := make(map[string]interface{})
		if position, ok := hbcrossswapPositions[swapSymbol]; ok {
			if position.Direction == hbcrossswap.OrderDirectionBuy {
				fields["swapSize"] = position.Volume*hbcrossswapContractSizes[swapSymbol]
			}else{
				fields["swapSize"] = -position.Volume*hbcrossswapContractSizes[swapSymbol]
			}
			if spread, ok := hbSpreads[spotSymbol]; ok {
				if position.Direction == hbcrossswap.OrderDirectionBuy {
					fields["swapValue"] = position.Volume*hbcrossswapContractSizes[swapSymbol]*spread.PerpOrderBook.TakerBidVWAP
				}else{
					fields["swapSize"] = -position.Volume*hbcrossswapContractSizes[swapSymbol]*spread.PerpOrderBook.TakerAskVWAP
				}
			}
		}
		if spotBalance, ok := hbspotBalances[spotSymbol]; ok {
			fields["spotBalance"] = spotBalance.Balance
			if spread, ok := hbSpreads[spotSymbol]; ok {
				fields["spotValue"] = spread.SpotOrderBook.TakerBidVWAP * spotBalance.Balance
			}
		}
		if fr, ok := hbcrossswapFundingRates[swapSymbol]; ok {
			fields["swapNextFundingRate"] = fr.FundingRate
			fields["swapEstimatedRate"] = fr.EstimatedRate
		}
		if spread, ok := hbSpreads[spotSymbol]; ok {
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
			fields["spotTakerAskFarPrice5"] = (1.0 + *hbConfig.MakerBandOffset) * spread.SpotOrderBook.AskPrice
			fields["spotTakerBidFarPrice5"] = (1.0 - *hbConfig.MakerBandOffset) * spread.SpotOrderBook.BidPrice
			if order, ok := hbspotOpenOrders[spotSymbol]; ok {
				fields["spotOpenOrderPrice"] = order.Price
			}

			fields["swapTakerBidVWAP"] = spread.PerpOrderBook.TakerBidVWAP
			fields["swapMakerBidVWAP"] = spread.PerpOrderBook.MakerBidVWAP
			fields["swapTakerAskVWAP"] = spread.PerpOrderBook.TakerAskVWAP
			fields["swapMakerAskVWAP"] = spread.PerpOrderBook.MakerAskVWAP

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
			*hbConfig.InternalInflux.Measurement,
			map[string]string{
				"swapSymbol": swapSymbol,
				"spotSymbol": spotSymbol,
				"type":       "singleBalance",
			},
			fields,
			time.Now().UTC(),
		)
		if err != nil {
			logger.Debugf("new position point error %v", err)
		} else {
			go hbInfluxWriter.Push(pt)
		}
	}

	if totalSpotBalance != nil && totalPerpUSDTBalance != nil {
		netWorth := (*totalSpotBalance + *totalPerpUSDTBalance) / *hbConfig.StartValue
		fields := make(map[string]interface{})
		fields["totalBalance"] = *totalSpotBalance + *totalPerpUSDTBalance
		fields["swapBalance"] = *totalPerpUSDTBalance
		fields["spotBalance"] = *totalSpotBalance
		fields["netWorth"] = (*totalSpotBalance + *totalPerpUSDTBalance) / *hbConfig.StartValue
		fields["startValue"] = *hbConfig.StartValue
		fields["netWorth"] = netWorth
		for name, start := range hbConfig.StartValues {
			if start > 0 {
				fields["refStartValue_"+strings.ToLower(name)] = start
				fields["currentValue_"+strings.ToLower(name)] = netWorth * start
			}
		}
		pt, err := client.NewPoint(
			*hbConfig.InternalInflux.Measurement,
			map[string]string{
				"type": "totalBalance",
			},
			fields,
			time.Now().UTC(),
		)
		if err != nil {
			logger.Debugf("Total Balance NewPoint error %v", err)
		} else {
			go hbInfluxWriter.Push(pt)
		}
	}
}

func handleExternalInfluxSave() {
	if !hbcrossswapAssetUpdatedForExternalInflux ||
		!hbspotBalanceUpdatedForExternalInflux ||
		time.Now().Sub(kcSaveSilentTime).Seconds() < 0 {
		return
	}
	hbcrossswapAssetUpdatedForExternalInflux = false
	hbspotBalanceUpdatedForExternalInflux = false

	var totalSpotBalance, totalPerpUSDTBalance *float64

	if hbspotUSDTBalance != nil {
		spotBalance := hbspotUSDTBalance.Balance
		getAllBalances := true
		for _, spotSymbol := range hbspotSymbols {
			balance, okBalance := hbspotBalances[spotSymbol]
			spread, okSpread := hbSpreads[spotSymbol]
			if okBalance && okSpread {
				spotBalance += spread.SpotOrderBook.TakerBidVWAP * balance.Balance
			} else {
				getAllBalances = false
				break
			}
		}
		if getAllBalances {
			totalSpotBalance = &spotBalance
		}
	}

	if hbcrossswapAccount != nil {
		tp := hbcrossswapAccount.MarginBalance + hbcrossswapAccount.ProfitUnreal
		totalPerpUSDTBalance = &tp
	}

	if totalSpotBalance != nil && totalPerpUSDTBalance != nil {
		fields := make(map[string]interface{})
		netWorth := (*totalSpotBalance + *totalPerpUSDTBalance) / *hbConfig.StartValue
		fields["netWorth"] = netWorth
		for name, start := range hbConfig.StartValues {
			if start > 0 {
				fields["currentValue_"+strings.ToLower(name)] = netWorth * start
			}
		}
		if len(fields) > 0 {
			pt, err := client.NewPoint(
				*hbConfig.ExternalInflux.Measurement,
				map[string]string{
					"name": *hbConfig.Name,
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
