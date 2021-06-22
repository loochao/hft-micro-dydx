package main

import (
	"github.com/geometrybase/hft-micro/influx/client"
	"github.com/geometrybase/hft-micro/logger"
	"strings"
	"time"
)

func handleSave() {

	if bnAccount != nil &&
		bnAccount.MarginBalance != nil {
		netWorth := *bnAccount.MarginBalance / *bnConfig.StartValue
		fields := make(map[string]interface{})
		fields["marginBalance"] = *bnAccount.MarginBalance
		fields["netWorth"] = netWorth
		fields["startValue"] = *bnConfig.StartValue
		fields["netWorth"] = netWorth
		if bnAccount.AvailableBalance != nil {
			fields["availableBalance"] = *bnAccount.AvailableBalance
		}
		if bnAccount.UnrealizedProfit != nil {
			fields["unrealizedProfit"] = *bnAccount.UnrealizedProfit
		}
		if bnTimeEmaDelta != nil {
			fields["timeEmaDelta"] = *bnTimeEmaDelta
			fields["systemOverHeated"] = bnSystemOverHeated
			fields["enterThreshold"] = *bnConfig.EnterThreshold
			fields["leaveThreshold"] = *bnConfig.LeaveThreshold
		}
		pt, err := client.NewPoint(
			*bnConfig.InternalInflux.Measurement,
			map[string]string{
				"type": "balance",
			},
			fields,
			time.Now().UTC(),
		)
		if err != nil {
			logger.Debugf("Spot Balance NewPoint error %v", err)
		} else {
			go bnInternalInfluxWriter.PushPoint(pt)
		}
	}

	for _, symbol := range bnSymbols[:*bnConfig.TradeSymbolIndex] {
		fields := make(map[string]interface{})
		if takerPosition, ok := bnPositions[symbol]; ok {
			fields["positionAmt"] = takerPosition.PositionAmt
		}
		if bid, ok := bnBidPrices[symbol]; ok {
			fields["bidPrice"] = bid.Price
		}
		if realisedSpread, ok := bnRealisedProfitPcts[symbol]; ok {
			fields["realisedSpread"] = realisedSpread
		}
		if quantile, ok := bnQuantiles[symbol]; ok {
			fields["quantileTop"] = quantile.Top
			fields["quantileBot"] = quantile.Bot
			fields["quantileMid"] = quantile.Mid
			fields["quantileDir"] = quantile.Dir
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
			logger.Debugf("new buyPosition point error %v", err)
		} else {
			go bnInternalInfluxWriter.PushPoint(pt)
		}
	}
}

func handleExternalInfluxSave() {
	if bnAccount != nil &&
		bnAccount.MarginBalance != nil {
		netWorth :=  *bnAccount.MarginBalance / *bnConfig.StartValue
		fields := make(map[string]interface{})
		fields["netWorth"] = netWorth
		for name, start := range bnConfig.StartValues {
			if start > 0 {
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
				go bnExternalInfluxWriter.PushPoint(pt)
			}
		}
	}
}

