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
			//有可能因为交易系统BUG Spread算不出来，这种时间Balance多半为0,
			balance, okBalance := kcspotBalances[spotSymbol]
			if balance.Available+balance.Holds > 0 {
				spread, okSpread := kcSpreads[spotSymbol]
				if okBalance && okSpread {
					spotBalance += spread.MakerDepth.TakerBid * (balance.Available + balance.Holds)
				} else {
					getAllBalances = false
					logger.Debugf("miss balance or spread %s", spotSymbol)
					break
				}
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
				go kcInternalInfluxWriter.PushPoint(pt)
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
			logger.Debugf("client.NewPoint error %v", err)
		} else {
			err = kcInternalInfluxWriter.PushPoint(pt)
			if err != nil {
				logger.Debugf("kcInternalInfluxWriter.PushPoint error %v", err)
			}
		}
		tp := kcperpUSDTAccount.MarginBalance + kcperpUSDTAccount.UnrealisedPNL
		totalPerpUSDTBalance = &tp
	}

	totalUnHedgedValue := 0.0
	for _, perpSymbol := range kcperpSymbols {
		spotSymbol := kcpsSymbolsMap[perpSymbol]
		fields := make(map[string]interface{})
		if position, ok := kcperpPositions[perpSymbol]; ok {
			fields["perpCurrentQty"] = position.CurrentQty
			if spread, ok := kcSpreads[spotSymbol]; ok {
				fields["perpValue"] = position.CurrentQty * kcperpMultipliers[perpSymbol] * spread.TakerDepth.TakerBid
				if spotBalance, ok := kcspotBalances[spotSymbol]; ok {
					unHedgedValue := (position.CurrentQty*kcperpMultipliers[perpSymbol] + (spotBalance.Available + spotBalance.Holds)) * spread.MakerDepth.MidPrice
					totalUnHedgedValue += unHedgedValue
					fields["unHedgedValue"] = unHedgedValue
				}
			}
		}
		if spotBalance, ok := kcspotBalances[spotSymbol]; ok {
			fields["spotBalance"] = spotBalance.Available + spotBalance.Holds
			if spread, ok := kcSpreads[spotSymbol]; ok {
				fields["spotValue"] = spread.MakerDepth.MidPrice * (spotBalance.Available + spotBalance.Holds)
			}
		}
		if fr, ok := kcperpFundingRates[perpSymbol]; ok {
			fields["perpNextFundingRate"] = fr.Value
			fields["perpPredictedFundingRate"] = fr.PredictedValue
		}
		if spread, ok := kcSpreads[spotSymbol]; ok {
			fields["shortLastEnter"] = spread.LastEnter
			fields["shortLastLeave"] = spread.LastLeave
			fields["shortMedianEnter"] = spread.MedianEnter
			fields["shortMedianLeave"] = spread.MedianLeave

			fields["spotTakerBid"] = spread.MakerDepth.TakerBid
			fields["spotTakerAsk"] = spread.MakerDepth.TakerAsk
			fields["spotTakerFarAsk"] = spread.MakerDepth.TakerFarAsk
			fields["spotTakerFarBid"] = spread.MakerDepth.TakerFarBid
			if order, ok := kcspotOpenOrders[spotSymbol]; ok {
				fields["spotOpenOrderPrice"] = order.Price
			}
			fields["perpTakerBid"] = spread.TakerDepth.TakerBid
			fields["perpTakerAsk"] = spread.TakerDepth.TakerAsk

			fields["age"] = spread.Age.Seconds()
			fields["ageDiff"] = spread.AgeDiff.Seconds()
		}
		if realisedSpread, ok := kcRealisedSpread[spotSymbol]; ok {
			fields["realisedSpread"] = realisedSpread
		}
		if quantile, ok := kcQuantiles[spotSymbol]; ok {
			fields["quantileBot"] = quantile.Bot
			fields["quantileTop"] = quantile.Top
			fields["quantileOriginalBot"] = quantile.OriginalBot
			fields["quantileOriginalTop"] = quantile.OriginalTop
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
			logger.Debugf("client.NewPoint error %v", err)
		} else {
			err = kcInternalInfluxWriter.PushPoint(pt)
			if err != nil {
				logger.Debugf("kcInternalInfluxWriter.PushPoint error %v", err)
			}
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
		fields["totalUnHedgedValue"] = totalUnHedgedValue
		fields["netWorth"] = netWorth
		if time.Now().Sub(kcGlobalSilent) > 0 {
			fields["globalSilent"] = 0
		} else {
			fields["globalSilent"] = 1
		}
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
			logger.Debugf("client.NewPoint error %v", err)
		} else {
			err = kcInternalInfluxWriter.PushPoint(pt)
			if err != nil {
				logger.Debugf("kcInternalInfluxWriter.PushPoint error %v", err)
			}
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
			if balance.Available+balance.Holds > 0 {
				spread, okSpread := kcSpreads[spotSymbol]
				if okBalance && okSpread {
					spotBalance += spread.MakerDepth.TakerBid * (balance.Available + balance.Holds)
				} else {
					getAllBalances = false
					break
				}
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
				logger.Debugf("client.NewPoint error %v", err)
			} else {
				err = kcExternalInfluxWriter.PushPoint(pt)
				if err != nil {
					logger.Debugf("kcExternalInfluxWriter.PushPoint error %v", err)
				}
			}
		}
	}
}

func reportsSaveLoop(
	ctx context.Context,
	influxWriter *common.InfluxWriter,
	influxConfig InfluxConfig,
	spreadReportCh chan common.SpreadReport,
) {
	spreadReports := make(map[string]common.SpreadReport)
	saveTimer := time.NewTimer(*influxConfig.SaveInterval)
	for {
		select {
		case <-ctx.Done():
			return
		case spreadReport := <-spreadReportCh:
			//logger.Debugf("%s", spreadReport.ToString())
			spreadReports[spreadReport.MakerSymbol] = spreadReport
			break
		case <-saveTimer.C:
			for makerSymbol, report := range spreadReports {
				fields := make(map[string]interface{})
				fields["matchRatio"] = report.MatchRatio
				fields["maxAgeDiff"] = float64(report.AdjustedAgeDiff)
				fields["spotTimeDeltaEma"] = report.MakerTimeDeltaEma
				fields["perpTimeDeltaEma"] = report.TakerTimeDeltaEma
				fields["spotTimeDelta"] = report.MakerTimeDelta
				fields["perpTimeDelta"] = report.TakerTimeDelta
				fields["spotDepthFilterRatio"] = report.MakerDepthFilterRatio
				fields["perpDepthFilterRatio"] = report.MakerDepthFilterRatio
				fields["spotMsgAvgLen"] = report.MakerMsgAvgLen
				fields["perpMsgAvgLen"] = report.TakerMsgAvgLen
				if len(fields) > 0 {
					pt, err := client.NewPoint(
						*influxConfig.Measurement,
						map[string]string{
							"spotSymbol": makerSymbol,
							"perpSymbol": report.TakerSymbol,
							"type":       "spread-report",
						},
						fields,
						time.Now().UTC(),
					)
					if err != nil {
						logger.Debugf("client.NewPoint error %v", err)
					} else {
						err = influxWriter.PushPoint(pt)
						if err != nil {
							logger.Debugf("influxWriter.PushPoint error %v", err)
						}
					}
				}
			}
			saveTimer.Reset(*influxConfig.SaveInterval)
			break
		}
	}
}
