package main

import (
	"github.com/geometrybase/hft-micro/hbcrossswap"
	"github.com/geometrybase/hft-micro/influx/client"
	"github.com/geometrybase/hft-micro/logger"
	"strings"
	"time"
)

func handleSave() {

	if !bAssetUpdatedForInflux || !hAccountUpdatedForInflux ||
		time.Now().Sub(hbSaveSilentTime).Seconds() < 0 {
		return
	}
	bAssetUpdatedForInflux = false
	hAccountUpdatedForInflux = false

	var totalSpotBalance, totalPerpUSDTBalance *float64

	if hbspotUSDTBalance != nil {
		spotBalance := hbspotUSDTBalance.Balance
		getAllBalances := true
		for _, spotSymbol := range mSymbols {
			balance, okBalance := tPositions[spotSymbol]
			spread, okSpread := mtSpreads[spotSymbol]
			if okBalance && okSpread {
				spotBalance += spread.TakerOrderBook.TakerBidVWAP *balance.Balance
			} else {
				logger.Debugf("%s MISS BALANCE %v OR TAKER VWAP %v", spotSymbol, okBalance, spread.TakerOrderBook.TakerBidVWAP)
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
				*mtConfig.InternalInflux.Measurement,
				map[string]string{
					"type": "spotBalance",
				},
				fields,
				time.Now().UTC(),
			)
			if err != nil {
				logger.Debugf("Spot Balance NewPoint error %v", err)
			} else {
				go mtInfluxWriter.Push(pt)
			}
		}
	}

	if mAccount != nil {
		fields := make(map[string]interface{})
		fields["swapMarginBalance"] = mAccount.MarginBalance
		fields["swapWithdrawAvailable"] = mAccount.WithdrawAvailable
		fields["swapProfitUnreal"] = mAccount.ProfitUnreal
		fields["swapMarginPosition"] = mAccount.MarginPosition
		pt, err := client.NewPoint(
			*mtConfig.InternalInflux.Measurement,
			map[string]string{
				"type": "swapBalance",
			},
			fields,
			time.Now().UTC(),
		)
		if err != nil {
			logger.Debugf("Perp Balance NewPoint error %v", err)
		} else {
			go mtInfluxWriter.Push(pt)
		}
		tp := mAccount.MarginBalance
		totalPerpUSDTBalance = &tp
	}

	for _, swapSymbol := range tSymbols {
		spotSymbol := mtSymbolsMap[swapSymbol]
		fields := make(map[string]interface{})
		if position, ok := mPositions[swapSymbol]; ok {
			if position.Direction == hbcrossswap.OrderDirectionBuy {
				fields["swapSize"] = position.Volume* mContractSizes[swapSymbol]
			}else{
				fields["swapSize"] = -position.Volume* mContractSizes[swapSymbol]
			}
			if spread, ok := mtSpreads[spotSymbol]; ok {
				if position.Direction == hbcrossswap.OrderDirectionBuy {
					fields["swapValue"] = position.Volume* mContractSizes[swapSymbol]*spread.MakerOrderBook.TakerBidVWAP
				}else{
					fields["swapValue"] = -position.Volume* mContractSizes[swapSymbol]*spread.MakerOrderBook.TakerAskVWAP
				}
			}
		}
		if spotBalance, ok := tPositions[spotSymbol]; ok {
			fields["spotBalance"] = spotBalance.Balance
			if spread, ok := mtSpreads[spotSymbol]; ok {
				fields["spotValue"] = spread.TakerOrderBook.TakerBidVWAP * spotBalance.Balance
			}
		}
		if fr, ok := mFundingRates[swapSymbol]; ok {
			fields["swapNextFundingRate"] = fr.FundingRate
			fields["swapEstimatedRate"] = fr.EstimatedRate
		}
		if spread, ok := mtSpreads[spotSymbol]; ok {
			fields["lastEnterSpread"] = spread.ShortLastEnter
			fields["lastExitSpread"] = spread.ShortLastExit
			fields["medianEnterSpread"] = spread.ShortMedianEnter
			fields["medianExitSpread"] = spread.ShortMedianExit

			fields["spotTakerBidVWAP"] = spread.TakerOrderBook.TakerBidVWAP
			fields["spotMakerBidVWAP"] = spread.TakerOrderBook.MakerBidVWAP
			fields["spotTakerAskVWAP"] = spread.TakerOrderBook.TakerAskVWAP
			fields["spotMakerAskVWAP"] = spread.TakerOrderBook.MakerAskVWAP
			fields["spotTakerAskFarPrice"] = spread.TakerOrderBook.TakerAskFarPrice
			fields["spotTakerBidFarPrice"] = spread.TakerOrderBook.TakerBidFarPrice
			fields["spotTakerAskFarPrice5"] = (1.0 + *mtConfig.MakerBandOffset) * spread.TakerOrderBook.AskPrice
			fields["spotTakerBidFarPrice5"] = (1.0 - *mtConfig.MakerBandOffset) * spread.TakerOrderBook.BidPrice
			if order, ok := mOpenOrders[spotSymbol]; ok {
				fields["spotOpenOrderPrice"] = order.Price
			}

			fields["swapTakerBidVWAP"] = spread.MakerOrderBook.TakerBidVWAP
			fields["swapMakerBidVWAP"] = spread.MakerOrderBook.MakerBidVWAP
			fields["swapTakerAskVWAP"] = spread.MakerOrderBook.TakerAskVWAP
			fields["swapMakerAskVWAP"] = spread.MakerOrderBook.MakerAskVWAP

			fields["age"] = spread.Age.Seconds()
			fields["ageDiff"] = spread.AgeDiff.Seconds()
		}
		if realisedSpread, ok := hbRealisedSpread[spotSymbol]; ok {
			fields["realisedSpread"] = realisedSpread
		}
		if quantile, ok := mtQuantiles[spotSymbol]; ok {
			fields["quantileBot"] = quantile.ShortBot
			fields["quantileTop"] = quantile.ShortTop
			fields["quantileMid"] = quantile.Mid
			fields["quantileMaClose"] = quantile.MaClose
		}
		pt, err := client.NewPoint(
			*mtConfig.InternalInflux.Measurement,
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
			go mtInfluxWriter.Push(pt)
		}
	}

	if totalSpotBalance != nil && totalPerpUSDTBalance != nil {
		netWorth := (*totalSpotBalance + *totalPerpUSDTBalance) / *mtConfig.StartValue
		fields := make(map[string]interface{})
		fields["totalBalance"] = *totalSpotBalance + *totalPerpUSDTBalance
		fields["swapBalance"] = *totalPerpUSDTBalance
		fields["spotBalance"] = *totalSpotBalance
		fields["netWorth"] = (*totalSpotBalance + *totalPerpUSDTBalance) / *mtConfig.StartValue
		fields["startValue"] = *mtConfig.StartValue
		fields["netWorth"] = netWorth
		for name, start := range mtConfig.StartValues {
			if start > 0 {
				fields["refStartValue_"+strings.ToLower(name)] = start
				fields["currentValue_"+strings.ToLower(name)] = netWorth * start
			}
		}
		pt, err := client.NewPoint(
			*mtConfig.InternalInflux.Measurement,
			map[string]string{
				"type": "totalBalance",
			},
			fields,
			time.Now().UTC(),
		)
		if err != nil {
			logger.Debugf("Total Balance NewPoint error %v", err)
		} else {
			go mtInfluxWriter.Push(pt)
		}
	}
}

func handleExternalInfluxSave() {
	if !hbcrossswapAssetUpdatedForExternalInflux ||
		!hbspotBalanceUpdatedForExternalInflux ||
		time.Now().Sub(hbSaveSilentTime).Seconds() < 0 {
		return
	}
	hbcrossswapAssetUpdatedForExternalInflux = false
	hbspotBalanceUpdatedForExternalInflux = false

	var totalSpotBalance, totalPerpUSDTBalance *float64

	if hbspotUSDTBalance != nil {
		spotBalance := hbspotUSDTBalance.Balance
		getAllBalances := true
		for _, spotSymbol := range mSymbols {
			balance, okBalance := tPositions[spotSymbol]
			spread, okSpread := mtSpreads[spotSymbol]
			if okBalance && okSpread {
				spotBalance += spread.TakerOrderBook.TakerBidVWAP * balance.Balance
			} else {
				getAllBalances = false
				break
			}
		}
		if getAllBalances {
			totalSpotBalance = &spotBalance
		}
	}

	if mAccount != nil {
		tp := mAccount.MarginBalance
		totalPerpUSDTBalance = &tp
	}

	if totalSpotBalance != nil && totalPerpUSDTBalance != nil {
		fields := make(map[string]interface{})
		netWorth := (*totalSpotBalance + *totalPerpUSDTBalance) / *mtConfig.StartValue
		fields["netWorth"] = netWorth
		for name, start := range mtConfig.StartValues {
			if start > 0 {
				fields["currentValue_"+strings.ToLower(name)] = netWorth * start
			}
		}
		if len(fields) > 0 {
			pt, err := client.NewPoint(
				*mtConfig.ExternalInflux.Measurement,
				map[string]string{
					"name": *mtConfig.Name,
				},
				fields,
				time.Now().UTC(),
			)
			if err != nil {
				logger.Debugf("Margin NewPoint error %v", err)
			} else {
				go mtExternalInfluxWriter.Push(pt)
			}
		}
	}
}
