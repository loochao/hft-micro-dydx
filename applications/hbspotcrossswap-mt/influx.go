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

	if !hbcrossswapAssetUpdatedForInflux || !hbspotBalanceUpdatedForInflux {
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
				spotBalance += spread.MakerDepth.TakerBid * balance.Balance
			} else {
				logger.Debugf("%s MISS BALANCE %v OR MAKER SPREAD %v", spotSymbol, okBalance, okSpread)
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
				logger.Debugf("client.NewPoint error %v", err)
			} else {
				err = hbInternalInfluxWriter.PushPoint(pt)
				if err != nil {
					logger.Debugf("influxWriter.PushPoint error %v", err)
				}
			}
		}
	}

	if hbcrossswapAccount != nil {
		fields := make(map[string]interface{})
		fields["swapMarginBalance"] = hbcrossswapAccount.MarginBalance
		fields["swapWithdrawAvailable"] = hbcrossswapAccount.WithdrawAvailable
		fields["swapProfitUnreal"] = hbcrossswapAccount.ProfitUnreal
		fields["swapMarginPosition"] = hbcrossswapAccount.MarginPosition
		pt, err := client.NewPoint(
			*hbConfig.InternalInflux.Measurement,
			map[string]string{
				"type": "swapBalance",
			},
			fields,
			time.Now().UTC(),
		)
		if err != nil {
			logger.Debugf("client.NewPoint error %v", err)
		} else {
			err = hbInternalInfluxWriter.PushPoint(pt)
			if err != nil {
				logger.Debugf("influxWriter.PushPoint error %v", err)
			}
		}
		tp := hbcrossswapAccount.MarginBalance
		totalPerpUSDTBalance = &tp
	}

	for _, swapSymbol := range hbcrossswapSymbols {
		spotSymbol := hbSwapSpotSymbolsMap[swapSymbol]
		fields := make(map[string]interface{})
		if position, ok := hbcrossswapPositions[swapSymbol]; ok {
			fields["swapSize"] = -position.Volume * hbcrossswapContractSizes[swapSymbol]
			if spread, ok := hbSpreads[spotSymbol]; ok {
				fields["swapValue"] = -position.Volume * hbcrossswapContractSizes[swapSymbol] * spread.TakerDepth.TakerAsk
			}
		}
		if spotBalance, ok := hbspotBalances[spotSymbol]; ok {
			fields["spotBalance"] = spotBalance.Balance
			if spread, ok := hbSpreads[spotSymbol]; ok {
				fields["spotValue"] = spread.MakerDepth.TakerBid * spotBalance.Balance
			}
		}
		if fr, ok := hbcrossswapFundingRates[swapSymbol]; ok {
			fields["swapNextFundingRate"] = fr.FundingRate
			fields["swapEstimatedRate"] = fr.EstimatedRate
		}
		if spread, ok := hbSpreads[spotSymbol]; ok {
			fields["lastEnterSpread"] = spread.ShortMedianEnter
			fields["lastExitSpread"] = spread.ShortLastLeave
			fields["medianEnterSpread"] = spread.ShortMedianEnter
			fields["medianExitSpread"] = spread.ShortMedianLeave

			fields["spotTakerBidVWAP"] = spread.MakerDepth.TakerBid
			fields["spotMakerBidVWAP"] = spread.MakerDepth.MakerBid
			fields["spotTakerAskVWAP"] = spread.MakerDepth.TakerAsk
			fields["spotMakerAskVWAP"] = spread.MakerDepth.MakerAsk
			fields["spotTakerAskFarPrice"] = spread.MakerDepth.TakerFarAsk
			fields["spotTakerBidFarPrice"] = spread.MakerDepth.TakerFarBid
			if order, ok := hbspotOpenOrders[spotSymbol]; ok {
				fields["spotOpenOrderPrice"] = order.Price
			}

			fields["swapTakerBidVWAP"] = spread.TakerDepth.TakerBid
			fields["swapMakerBidVWAP"] = spread.TakerDepth.MakerBid
			fields["swapTakerAskVWAP"] = spread.TakerDepth.TakerAsk
			fields["swapMakerAskVWAP"] = spread.TakerDepth.MakerAsk

			fields["age"] = spread.Age.Seconds()
			fields["ageDiff"] = spread.AgeDiff.Seconds()
		}
		if realisedSpread, ok := hbRealisedSpread[spotSymbol]; ok {
			fields["realisedSpread"] = realisedSpread
		}
		if quantile, ok := hbQuantiles[spotSymbol]; ok {
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
			logger.Debugf("client.NewPoint error %v", err)
		} else {
			err = hbInternalInfluxWriter.PushPoint(pt)
			if err != nil {
				logger.Debugf("influxWriter.PushPoint error %v", err)
			}
		}
	}

	if totalSpotBalance != nil && totalPerpUSDTBalance != nil {
		netWorth := (*totalSpotBalance + *totalPerpUSDTBalance) / *hbConfig.StartValue
		fields := make(map[string]interface{})
		fields["totalBalance"] = *totalSpotBalance + *totalPerpUSDTBalance
		fields["swapBalance"] = *totalPerpUSDTBalance
		fields["spotBalance"] = *totalSpotBalance
		fields["unHedgeValue"] = hbUnHedgeValue
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
			logger.Debugf("client.NewPoint error %v", err)
		} else {
			err = hbInternalInfluxWriter.PushPoint(pt)
			if err != nil {
				logger.Debugf("influxWriter.PushPoint error %v", err)
			}
		}
	}
}

func handleExternalInfluxSave() {
	if !hbcrossswapAssetUpdatedForExternalInflux ||
		!hbspotBalanceUpdatedForExternalInflux {
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
				spotBalance += spread.MakerDepth.TakerBid * balance.Balance
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
		tp := hbcrossswapAccount.MarginBalance
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
				logger.Debugf("client.NewPoint error %v", err)
			} else {
				err = hbExternalInfluxWriter.PushPoint(pt)
				if err != nil {
					logger.Debugf("influxWriter.PushPoint error %v", err)
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
				fields["maxAgeDiff"] = float64(report.MaxAgeDiff)
				fields["spotTimeDeltaEma"] = report.MakerTimeDeltaEma
				fields["swapTimeDeltaEma"] = report.TakerTimeDeltaEma
				fields["spotTimeDelta"] = report.MakerTimeDelta
				fields["swapTimeDelta"] = report.TakerTimeDelta
				fields["spotDepthFilterRatio"] = report.MakerDepthFilterRatio
				fields["swapDepthFilterRatio"] = report.MakerDepthFilterRatio
				fields["spotMsgAvgLen"] = report.MakerMsgAvgLen
				fields["swapMsgAvgLen"] = report.TakerMsgAvgLen
				if len(fields) > 0 {
					pt, err := client.NewPoint(
						*influxConfig.Measurement,
						map[string]string{
							"spotSymbol": makerSymbol,
							"swapSymbol": report.TakerSymbol,
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
