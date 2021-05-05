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

	if !bnswapAssetUpdatedForInflux || !bnspotBalanceUpdatedForInflux ||
		time.Now().Sub(bnSaveSilentTime).Seconds() < 0 {
		return
	}
	bnswapAssetUpdatedForInflux = false
	bnspotBalanceUpdatedForInflux = false

	var totalSpotBalance, totalSwapUSDTBalance, totalSwapBnBBalance *float64

	if bnspotUSDTBalance != nil {
		spotBalance := bnspotUSDTBalance.Free + bnspotUSDTBalance.Locked
		getAllBalances := true
		for _, symbol := range bnSymbols {
			balance, okBalance := bnspotBalances[symbol]
			premiumIndex, okPremiumIndex := bnswapPremiumIndexes[symbol]
			if okBalance && okPremiumIndex {
				spotBalance += premiumIndex.IndexPrice * (balance.Free + balance.Locked)
			} else {
				logger.Debugf("%s miss balance %v or premium index %v", symbol, okBalance, okPremiumIndex)
				getAllBalances = false
				break
			}
		}
		if getAllBalances {
			totalSpotBalance = &spotBalance
			fields := make(map[string]interface{})
			fields["spotBalance"] = *totalSpotBalance
			fields["spotUsdtFreeBalance"] = bnspotUSDTBalance.Free
			fields["spotUsdtLockedBalance"] = bnspotUSDTBalance.Locked
			pt, err := client.NewPoint(
				*bnConfig.InternalInflux.Measurement,
				map[string]string{
					"type": "spotBalance",
				},
				fields,
				time.Now().UTC(),
			)
			if err != nil {
				logger.Debugf("client.NewPoint() error %v", err)
			} else {
				err = bnInternalInfluxWriter.PushPoint(pt)
				if err != nil {
					logger.Debugf("bnExternalInfluxWriter.PushPoint(pt) error %v", err)
				}
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
		if bnswapAvgFundingRate != nil {
			fields["avgFundingRate"] = *bnswapAvgFundingRate
		}
		if bnswapBNBAsset != nil && bnswapBNBAsset.MarginBalance != nil {
			if markPrice, ok := bnswapPremiumIndexes[bnBNBSymbol]; ok {
				balance := *bnswapBNBAsset.MarginBalance * markPrice.IndexPrice
				fields["swapBNBMarginBalance"] = *bnswapBNBAsset.MarginBalance
				fields["swapBNBBalance"] = balance
				totalSwapBnBBalance = &balance
			}
		}
		pt, err := client.NewPoint(
			*bnConfig.InternalInflux.Measurement,
			map[string]string{
				"type": "swapBalance",
			},
			fields,
			time.Now().UTC(),
		)
		if err != nil {
			logger.Debugf("client.NewPoint() error %v", err)
		} else {
			err = bnInternalInfluxWriter.PushPoint(pt)
			if err != nil {
				logger.Debugf("bnExternalInfluxWriter.PushPoint(pt) error %v", err)
			}
		}
		totalSwapUSDTBalance = bnswapUSDTAsset.MarginBalance
	}

	totalUnHedgeValue := 0.0
	entryTarget := 0.0
	if bnswapUSDTAsset != nil && bnspotUSDTBalance != nil {
		entryStep := (*bnswapUSDTAsset.AvailableBalance + bnspotUSDTBalance.Free) * *bnConfig.EnterFreePct
		if entryStep < *bnConfig.EnterMinimalStep {
			entryStep = *bnConfig.EnterMinimalStep
		}
		entryTarget = entryStep * *bnConfig.EnterTargetFactor
	}
	for _, symbol := range bnSymbols {
		fields := make(map[string]interface{})
		if position, ok := bnswapPositions[symbol]; ok {
			fields["swapBalance"] = position.PositionAmt
			if premiumIndex, ok := bnswapPremiumIndexes[symbol]; ok {
				fields["swapValue"] = premiumIndex.IndexPrice * position.PositionAmt
			}
		}
		if spotBalance, ok := bnspotBalances[symbol]; ok {
			fields["spotBalance"] = spotBalance.Free + spotBalance.Locked
			if premiumIndex, ok := bnswapPremiumIndexes[symbol]; ok {
				fields["spotValue"] = premiumIndex.IndexPrice * (spotBalance.Free + spotBalance.Locked)
				if entryTarget != 0 {
					fields["enterDelta"] = *bnConfig.EnterDelta + *bnConfig.OffsetDelta*(premiumIndex.IndexPrice * (spotBalance.Free + spotBalance.Locked)/entryTarget)
					fields["exitDelta"] = *bnConfig.ExitDelta + *bnConfig.OffsetDelta*(premiumIndex.IndexPrice * (spotBalance.Free + spotBalance.Locked)/entryTarget)
				}

				if position, ok := bnswapPositions[symbol]; ok {
					if symbol == bnBNBSymbol {
						fields["unHedgeValue"] = (position.PositionAmt + spotBalance.Free + spotBalance.Locked + *bnswapBNBAsset.MarginBalance) * premiumIndex.IndexPrice
						totalUnHedgeValue += (position.PositionAmt + spotBalance.Free + spotBalance.Locked + *bnswapBNBAsset.MarginBalance) * premiumIndex.IndexPrice
					} else {
						fields["unHedgeValue"] = (position.PositionAmt + spotBalance.Free + spotBalance.Locked) * premiumIndex.IndexPrice
						totalUnHedgeValue += (position.PositionAmt + spotBalance.Free + spotBalance.Locked) * premiumIndex.IndexPrice
					}
				}
			}
		}
		if markPrice, ok := bnswapPremiumIndexes[symbol]; ok {
			fields["swapNextFundingRate"] = markPrice.FundingRate
		}
		if spread, ok := bnSpreads[symbol]; ok {
			fields["shortLastEnter"] = spread.ShortLastEnter
			fields["shortLastLeave"] = spread.ShortLastLeave
			fields["shortMedianEnter"] = spread.ShortMedianEnter
			fields["shortMedianLeave"] = spread.ShortMedianLeave

			fields["spotTakerBid"] = spread.MakerDepth.TakerBid
			fields["spotMakerBid"] = spread.MakerDepth.MakerBid
			fields["spotTakerAsk"] = spread.MakerDepth.TakerAsk
			fields["spotMakerAsk"] = spread.MakerDepth.MakerAsk
			fields["spotMidPrice"] = spread.MakerDepth.MidPrice
			fields["spotTakerFarAsk"] = spread.MakerDepth.TakerFarAsk
			fields["spotTakerFarBid"] = spread.MakerDepth.TakerFarBid
			if order, ok := bnspotOpenOrders[symbol]; ok {
				fields["spotOpenOrderPrice"] = order.Price
			}
			fields["swapTakerBid"] = spread.TakerDepth.TakerBid
			fields["swapMakerBid"] = spread.TakerDepth.MakerBid
			fields["swapTakerAsk"] = spread.TakerDepth.TakerAsk
			fields["swapMakerAsk"] = spread.TakerDepth.MakerAsk
			fields["swapMidPrice"] = spread.TakerDepth.MidPrice

			fields["age"] = spread.Age.Seconds()
			fields["ageDiff"] = spread.AgeDiff.Seconds()
		}
		if realisedSpread, ok := bnRealisedSpread[symbol]; ok {
			fields["realisedSpread"] = realisedSpread
		}
		//if quantile, ok := bnQuantiles[symbol]; ok {
		//	fields["quantileBot"] = quantile.Bot
		//	fields["quantileTop"] = quantile.Top
		//	fields["quantileOriginalTop"] = quantile.OriginalTop
		//	fields["quantileOriginalBot"] = quantile.OriginalBot
		//	fields["quantileMid"] = quantile.Mid
		//	fields["quantileMaClose"] = quantile.MaClose
		//	fields["quantileMeanFr"] = quantile.MeanFr
		//}
		pt, err := client.NewPoint(
			*bnConfig.InternalInflux.Measurement,
			map[string]string{
				"symbol": symbol,
				"type":   "singleBalance",
			},
			fields,
			time.Now().UTC(),
		)
		if err != nil {
			logger.Debugf("client.NewPoint() error %v", err)
		} else {
			err = bnInternalInfluxWriter.PushPoint(pt)
			if err != nil {
				logger.Debugf("bnExternalInfluxWriter.PushPoint(pt) error %v", err)
			}
		}
	}

	if totalSpotBalance != nil && totalSwapUSDTBalance != nil && totalSwapBnBBalance != nil {
		netWorth := (*totalSpotBalance + *totalSwapUSDTBalance + *totalSwapBnBBalance) / *bnConfig.StartValue
		fields := make(map[string]interface{})
		fields["totalBalance"] = *totalSpotBalance + *totalSwapUSDTBalance + *totalSwapBnBBalance
		fields["totalUnHedgeValue"] = totalUnHedgeValue
		fields["swapBalance"] = *totalSwapUSDTBalance + *totalSwapBnBBalance
		fields["spotBalance"] = *totalSpotBalance
		fields["netWorth"] = (*totalSpotBalance + *totalSwapUSDTBalance + *totalSwapBnBBalance) / *bnConfig.StartValue
		fields["startValue"] = *bnConfig.StartValue
		fields["netWorth"] = netWorth
		if bnGlobalSilent.Sub(time.Now()) > 0 {
			fields["globalSilent"] = 1.0
		} else {
			fields["globalSilent"] = 0.0
		}
		for name, start := range bnConfig.StartValues {
			if start > 0 {
				fields["refStartValue_"+strings.ToLower(name)] = start
				fields["currentValue_"+strings.ToLower(name)] = netWorth * start
			}
		}
		pt, err := client.NewPoint(
			*bnConfig.InternalInflux.Measurement,
			map[string]string{
				"type": "totalBalance",
			},
			fields,
			time.Now().UTC(),
		)
		if err != nil {
			logger.Debugf("client.NewPoint() error %v", err)
		} else {
			err = bnInternalInfluxWriter.PushPoint(pt)
			if err != nil {
				logger.Debugf("bnExternalInfluxWriter.PushPoint(pt) error %v", err)
			}
		}
	}
}

func handleExternalInfluxSave() {
	if !bnswapAssetUpdatedForExternalInflux ||
		!bnspotBalanceUpdatedForExternalInflux ||
		time.Now().Sub(bnSaveSilentTime).Seconds() < 0 {
		return
	}
	bnswapAssetUpdatedForExternalInflux = false
	bnspotBalanceUpdatedForExternalInflux = false

	var totalSpotBalance, totalSwapUSDTBalance, totalSwapBnBBalance *float64

	if bnspotUSDTBalance != nil {
		spotBalance := bnspotUSDTBalance.Free + bnspotUSDTBalance.Locked
		getAllBalances := true
		for _, symbol := range bnSymbols {
			balance, okBalance := bnspotBalances[symbol]
			markPrice, okMP := bnswapPremiumIndexes[symbol]
			if okBalance && okMP {
				spotBalance += markPrice.IndexPrice * (balance.Free + balance.Locked)
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
		if spread, ok := bnSpreads[bnBNBSymbol]; ok {
			balance := *bnswapBNBAsset.MarginBalance * (spread.MakerDepth.MakerBid + spread.MakerDepth.MakerAsk) * 0.5
			totalSwapBnBBalance = &balance
		}
	}

	if bnswapUSDTAsset != nil && bnswapUSDTAsset.MarginBalance != nil {
		totalSwapUSDTBalance = bnswapUSDTAsset.MarginBalance
	}

	if totalSpotBalance != nil && totalSwapUSDTBalance != nil && totalSwapBnBBalance != nil {
		fields := make(map[string]interface{})
		netWorth := (*totalSpotBalance + *totalSwapUSDTBalance + *totalSwapBnBBalance) / *bnConfig.StartValue
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
				logger.Debugf("client.NewPoint() error %v", err)
			} else {
				err = bnExternalInfluxWriter.PushPoint(pt)
				if err != nil {
					logger.Debugf("bnExternalInfluxWriter.PushPoint(pt) error %v", err)
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
			for symbol, report := range spreadReports {
				fields := make(map[string]interface{})
				fields["matchRatio"] = report.MatchRatio
				fields["maxAgeDiff"] = float64(report.MaxAgeDiff)
				fields["spotTimeDeltaEma"] = report.MakerTimeDeltaEma
				fields["swapTimeDeltaEma"] = report.TakerTimeDeltaEma
				fields["spotTimeDelta"] = report.MakerTimeDelta
				fields["swapTimeDelta"] = report.TakerTimeDelta
				fields["spotDepthFilterRatio"] = report.MakerDepthFilterRatio
				fields["swapDepthFilterRatio"] = report.TakerDepthFilterRatio
				fields["spotMsgAvgLen"] = report.MakerMsgAvgLen
				fields["swapMsgAvgLen"] = report.TakerMsgAvgLen
				if len(fields) > 0 {
					pt, err := client.NewPoint(
						*influxConfig.Measurement,
						map[string]string{
							"symbol": symbol,
							"type":   "spread-report",
						},
						fields,
						time.Now().UTC(),
					)
					if err != nil {
						logger.Debugf("client.NewPoint() error %v", err)
					} else {
						err = influxWriter.PushPoint(pt)
						if err != nil {
							logger.Debugf("influxWriter.PushPoint(pt) error %v", err)
						}
					}
				}
			}
			saveTimer.Reset(*influxConfig.SaveInterval)
			break
		}
	}
}
