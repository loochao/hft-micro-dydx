package main

import (
	"github.com/geometrybase/hft-micro/hbcrossswap"
	"github.com/geometrybase/hft-micro/influx/client"
	"github.com/geometrybase/hft-micro/logger"
	"strings"
	"time"
)

func handleSave() {

	if tAccount != nil &&
		tAccount.MarginBalance != nil &&
		mAccount != nil {
		totalBalance := *tAccount.MarginBalance + mAccount.MarginBalance
		netWorth := totalBalance / *mtConfig.StartValue
		fields := make(map[string]interface{})
		fields["totalBalance"] = totalBalance
		fields["takerBalance"] = *tAccount.MarginBalance
		fields["makerBalance"] = mAccount.MarginBalance
		fields["netWorth"] = netWorth
		fields["startValue"] = *mtConfig.StartValue
		fields["netWorth"] = netWorth
		if tAccount.AvailableBalance != nil {
			fields["takerAvailable"] = *tAccount.AvailableBalance
		}
		if tAccount.UnrealizedProfit != nil {
			fields["takerUnrealizedProfit"] = *tAccount.UnrealizedProfit
		}
		fields["makerAvailable"] = mAccount.WithdrawAvailable
		fields["makerUnrealizedProfit"] = mAccount.ProfitUnreal
		pt, err := client.NewPoint(
			*mtConfig.InternalInflux.Measurement,
			map[string]string{
				"type": "balance",
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

	for _, makerSymbol := range mSymbols {
		takerSymbol := mtSymbolsMap[makerSymbol]
		fields := make(map[string]interface{})
		if position, ok := mPositions[makerSymbol]; ok {
			if position.Direction == hbcrossswap.OrderDirectionBuy {
				fields["makerSize"] = position.Volume * mContractSizes[makerSymbol]
			} else {
				fields["makerSize"] = -position.Volume * mContractSizes[makerSymbol]
			}
			if spread, ok := mtSpreads[makerSymbol]; ok {
				if position.Direction == hbcrossswap.OrderDirectionBuy {
					fields["makerValue"] = position.Volume * mContractSizes[makerSymbol] * spread.MakerOrderBook.BidVWAP
				} else {
					fields["makerValue"] = -position.Volume * mContractSizes[makerSymbol] * spread.MakerOrderBook.AskVWAP
				}
			}
		}
		if takerPosition, ok := tPositions[takerSymbol]; ok {
			fields["takerSize"] = takerPosition.PositionAmt
			if spread, ok := mtSpreads[makerSymbol]; ok {
				fields["takerValue"] = spread.TakerOrderBook.BidVWAP * takerPosition.PositionAmt
			}
		}
		if fr, ok := mFundingRates[makerSymbol]; ok {
			fields["makerFundingRate"] = fr.FundingRate
		}
		if pi, ok := tPremiumIndexes[takerSymbol]; ok {
			fields["takerFundingRate"] = pi.FundingRate
		}
		if fr, ok := mtFundingRates[makerSymbol]; ok {
			fields["fundingRate"] = fr
		}
		if spread, ok := mtSpreads[makerSymbol]; ok {

			fields["spreadShortLastEnter"] = spread.ShortLastEnter
			fields["spreadShortLastExit"] = spread.ShortLastExit
			fields["spreadShortMedianEnter"] = spread.ShortMedianEnter
			fields["spreadShortMedianExit"] = spread.ShortMedianExit

			fields["spreadLongLastEnter"] = spread.LongLastEnter
			fields["spreadLongLastExit"] = spread.LongLastExit
			fields["spreadLongMedianEnter"] = spread.LongMedianEnter
			fields["spreadLongMedianExit"] = spread.LongMedianExit

			fields["takerBidVWAP"] = spread.TakerOrderBook.BidVWAP
			fields["takerAskVWAP"] = spread.TakerOrderBook.AskVWAP
			fields["makerBidVWAP"] = spread.MakerOrderBook.BidVWAP
			fields["makerAskVWAP"] = spread.MakerOrderBook.AskVWAP
			fields["age"] = spread.Age.Seconds()
			fields["ageDiff"] = spread.AgeDiff.Seconds()
		}
		if realisedSpread, ok := mtRealisedSpread[makerSymbol]; ok {
			fields["realisedSpread"] = realisedSpread
		}
		if quantile, ok := mtQuantiles[makerSymbol]; ok {
			fields["quantileShortBot"] = quantile.ShortBot
			fields["quantileShortTop"] = quantile.ShortTop
			fields["quantileLongBot"] = quantile.LongBot
			fields["quantileLongTop"] = quantile.LongTop
			fields["quantileMid"] = quantile.Mid
			fields["quantileMaClose"] = quantile.MaClose
		}
		pt, err := client.NewPoint(
			*mtConfig.InternalInflux.Measurement,
			map[string]string{
				"takerSymbol": takerSymbol,
				"makerSymbol": makerSymbol,
				"type":        "symbol",
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
}

func handleExternalInfluxSave() {
	if tAccount != nil &&
		tAccount.MarginBalance != nil &&
		mAccount != nil {
		totalBalance := *tAccount.MarginBalance + mAccount.MarginBalance
		netWorth := totalBalance / *mtConfig.StartValue
		fields := make(map[string]interface{})
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


