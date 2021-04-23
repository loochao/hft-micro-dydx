package main

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/influx/client"
	"github.com/geometrybase/hft-micro/logger"
	"strings"
	"time"
)

func handleSave() {

	var totalSpotBalance, totalPerpUSDTBalance *float64

	if kcspotUSDTBalance != nil {
		spotBalance := kcspotUSDTBalance.Available + kcspotUSDTBalance.Holds
		getAllBalances := true
		for _, spotSymbol := range kcspotSymbols {
			balance, okBalance := kcspotBalances[spotSymbol]
			spread, okSpread := kcSpreads[spotSymbol]
			if okBalance && okSpread {
				spotBalance += spread.MakerDepth.TakerBid * (balance.Available + balance.Holds)
			} else {
				logger.Debugf("%s MISS BALANCE %v OR TAKER VWAP %v", spotSymbol, okBalance, spread.MakerDepth.TakerBid)
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
				go kcInternalInfluxWriter.Push(pt)
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
			go kcInternalInfluxWriter.Push(pt)
		}
		tp := kcperpUSDTAccount.MarginBalance + kcperpUSDTAccount.UnrealisedPNL
		totalPerpUSDTBalance = &tp
	}

	for _, perpSymbol := range kcperpSymbols {
		spotSymbol := kcpsSymbolsMap[perpSymbol]
		fields := make(map[string]interface{})
		if position, ok := kcperpPositions[perpSymbol]; ok {
			fields["perpCurrentQty"] = position.CurrentQty
			if spread, ok := kcSpreads[spotSymbol]; ok {
				fields["perpValue"] = position.CurrentQty * kcperpMultipliers[perpSymbol] * spread.TakerDepth.TakerBid
			}
		}
		if spotBalance, ok := kcspotBalances[spotSymbol]; ok {
			fields["spotBalance"] = spotBalance.Available + spotBalance.Holds
			if spread, ok := kcSpreads[spotSymbol]; ok {
				fields["spotValue"] = spread.MakerDepth.MakerAsk * (spotBalance.Available + spotBalance.Holds)
			}
		}
		if fr, ok := kcperpFundingRates[perpSymbol]; ok {
			fields["perpNextFundingRate"] = fr.Value
			fields["perpPredictedFundingRate"] = fr.PredictedValue
		}
		if spread, ok := kcSpreads[spotSymbol]; ok {
			fields["shortLastEnter"] = spread.ShortLastEnter
			fields["shortLastLeave"] = spread.ShortLastLeave
			fields["shortMedianEnter"] = spread.ShortMedianEnter
			fields["shortMedianLeave"] = spread.ShortMedianLeave

			fields["spotTakerBid"] = spread.MakerDepth.TakerBid
			fields["spotMakerBid"] = spread.MakerDepth.MakerBid
			fields["spotTakerAsk"] = spread.MakerDepth.TakerAsk
			fields["spotMakerAsk"] = spread.MakerDepth.MakerAsk
			fields["spotTakerFarAsk"] = spread.MakerDepth.TakerFarAsk
			fields["spotTakerFarBid"] = spread.MakerDepth.TakerFarBid
			if order, ok := kcspotOpenOrders[spotSymbol]; ok {
				fields["spotOpenOrderPrice"] = order.Price
			}
			fields["perpTakerBid"] = spread.TakerDepth.TakerBid
			fields["perpMakerBid"] = spread.TakerDepth.MakerBid
			fields["perpTakerAsk"] = spread.TakerDepth.TakerAsk
			fields["perpMakerAsk"] = spread.TakerDepth.MakerAsk

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
			go kcInternalInfluxWriter.Push(pt)
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
			go kcInternalInfluxWriter.Push(pt)
		}
	}
}

func handleExternalInfluxSave() {

	var totalSpotBalance, totalPerpUSDTBalance *float64

	if kcspotUSDTBalance != nil {
		spotBalance := kcspotUSDTBalance.Available + kcspotUSDTBalance.Holds
		getAllBalances := true
		for _, spotSymbol := range kcspotSymbols {
			balance, okBalance := kcspotBalances[spotSymbol]
			spread, okSpread := kcSpreads[spotSymbol]
			if okBalance && okSpread {
				spotBalance += spread.MakerDepth.TakerBid * (balance.Available + balance.Holds)
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

func reportsSaveLoop(
	ctx context.Context,
	influxWriter *common.InfluxWriter,
	influxConfig InfluxConfig,
	depthReportCh chan common.DepthReport,
	spreadReportCh chan common.SpreadReport,
) {
	depthReports := make(map[string]common.DepthReport)
	spreadReports := make(map[string]common.SpreadReport)
	saveTimer := time.NewTimer(*influxConfig.SaveInterval)
	for {
		select {
		case <-ctx.Done():
			return
		case spreadReport := <-spreadReportCh:
			spreadReports[spreadReport.MakerSymbol] = spreadReport
			break
		case depthReport := <-depthReportCh:
			depthReports[depthReport.Exchange] = depthReport
			break
		case <-saveTimer.C:
			for exchange, report := range depthReports {
				fields := make(map[string]interface{})
				fields["avgLen"] = report.AvgLen
				fields["dropRatio"] = report.DropRatio
				fields["bias"] = report.Bias
				fields["decay"] = report.Decay
				fields["emaTimeDelta"] = report.EmaTimeDelta
				if len(fields) > 0 {
					pt, err := client.NewPoint(
						*influxConfig.Measurement,
						map[string]string{
							"exchange": exchange,
							"type":     "depth-report",
						},
						fields,
						time.Now().UTC(),
					)
					if err != nil {
						logger.Debugf("DepthReport NewPoint error %v", err)
					} else {
						select {
						case influxWriter.PushCh <- pt:
						default:
						}
					}
				}
			}
			for makerSymbol, report := range spreadReports {
				fields := make(map[string]interface{})
				fields["matchRatio"] = report.MatchRatio
				fields["maxAgeDiff"] = float64(report.MaxAgeDiff)
				fields["maxAge"] = float64(report.MaxAge)
				if len(fields) > 0 {
					pt, err := client.NewPoint(
						*influxConfig.Measurement,
						map[string]string{
							"makerSymbol": makerSymbol,
							"takerSymbol": report.TakerSymbol,
							"type":        "spread-report",
						},
						fields,
						time.Now().UTC(),
					)
					if err != nil {
						logger.Debugf("SpreadReport NewPoint error %v", err)
					} else {
						select {
						case influxWriter.PushCh <- pt:
						default:
						}
					}
				}
			}
			saveTimer.Reset(*influxConfig.SaveInterval)
			break
		}
	}
}
