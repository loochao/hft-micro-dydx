package main

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/influx/client"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"strings"
	"time"
)

func handleSave() {
	if time.Now().Sub(mtGlobalSilent) < 0 {
		return
	}

	entryTarget := 0.0
	if mAccount != nil || tAccount != nil || tAccount.AvailableBalance != nil {
		entryStep := (mAccount.AvailableBalance + *tAccount.AvailableBalance) * *mtConfig.EnterFreePct
		if entryStep < *mtConfig.EnterMinimalStep {
			entryStep = *mtConfig.EnterMinimalStep
		}
		entryTarget = entryStep * *mtConfig.EnterTargetFactor
	}


	totalUnHedgeValue := 0.0
	for _, makerSymbol := range mSymbols {
		takerSymbol := mtSymbolsMap[makerSymbol]
		fields := make(map[string]interface{})
		if makerPosition, ok := mPositions[makerSymbol]; ok {
			fields["makerSize"] = makerPosition.CurrentQty * mMultipliers[makerSymbol]
			if spread, ok := mtSpreads[makerSymbol]; ok {
				fields["makerValue"] = makerPosition.CurrentQty * mMultipliers[makerSymbol] * spread.MakerDepth.MakerBid
				makerValue := makerPosition.AvgEntryPrice * makerPosition.CurrentQty * mMultipliers[makerSymbol]
				//offset := mOrderOffsets[makerSymbol]
				fields["shortTop"] = *mtConfig.EnterDelta + *mtConfig.OffsetDelta*(math.Max(makerValue, 0)/entryTarget)
				fields["shortBot"] = *mtConfig.ExitDelta + *mtConfig.OffsetDelta*(math.Max(makerValue, 0)/entryTarget)
				fields["longBot"] = -*mtConfig.EnterDelta + *mtConfig.OffsetDelta*(math.Min(makerValue, 0)/entryTarget)
				fields["longTop"] = -*mtConfig.ExitDelta + *mtConfig.OffsetDelta*(math.Min(makerValue, 0)/entryTarget)
				if takerPosition, ok := tPositions[takerSymbol]; ok {
					unHedgedValue := (takerPosition.PositionAmt +  makerPosition.CurrentQty * mMultipliers[makerSymbol])* spread.MakerDepth.MakerAsk
					fields["unHedgedValue"] = unHedgedValue
					totalUnHedgeValue += math.Abs(unHedgedValue)
				}
			}
		}
		if takerPosition, ok := tPositions[takerSymbol]; ok {
			fields["takerSize"] = takerPosition.PositionAmt
			if spread, ok := mtSpreads[makerSymbol]; ok {
				fields["takerValue"] = spread.TakerDepth.TakerBid * takerPosition.PositionAmt
			}
		}
		if fr, ok := mFundingRates[makerSymbol]; ok {
			fields["makerFundingRate"] = fr.Value
		}
		if pi, ok := tPremiumIndexes[takerSymbol]; ok {
			fields["takerFundingRate"] = pi.FundingRate
		}
		if fr, ok := mtFundingRates[makerSymbol]; ok {
			fields["fundingRate"] = fr
		}
		if spread, ok := mtSpreads[makerSymbol]; ok {

			fields["spreadShortLastEnter"] = spread.ShortLastEnter
			fields["spreadShortLastLeave"] = spread.ShortLastLeave
			fields["spreadShortMedianEnter"] = spread.ShortMedianEnter
			fields["spreadShortMedianLeave"] = spread.ShortMedianLeave

			fields["spreadLongLastEnter"] = spread.LongLastEnter
			fields["spreadLongLastLeave"] = spread.LongLastLeave
			fields["spreadLongMedianEnter"] = spread.LongMedianEnter
			fields["spreadLongMedianLeave"] = spread.LongMedianLeave

			fields["takerMakerBid"] = spread.TakerDepth.MakerBid
			fields["takerMakerAsk"] = spread.TakerDepth.MakerAsk
			fields["takerTakerBid"] = spread.TakerDepth.TakerBid
			fields["takerTakerAsk"] = spread.TakerDepth.TakerAsk

			fields["makerMakerBid"] = spread.MakerDepth.MakerBid
			fields["makerMakerAsk"] = spread.MakerDepth.MakerAsk
			fields["makerTakerBid"] = spread.MakerDepth.TakerBid
			fields["makerTakerAsk"] = spread.MakerDepth.TakerAsk

			fields["age"] = spread.Age.Seconds()
			fields["ageDiff"] = spread.AgeDiff.Seconds()
		}
		if realisedSpread, ok := mtRealisedSpread[makerSymbol]; ok {
			fields["realisedSpread"] = realisedSpread
		}
		if time.Now().Sub(mtGlobalSilent) > 0 {
			fields["globalSilent"] = 0
		} else {
			fields["globalSilent"] = 1
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
			logger.Debugf("client.NewPoint error %v", err)
		} else {
			err = mtInfluxWriter.PushPoint(pt)
			if err != nil {
				logger.Debugf("mtInfluxWriter.PushPoint error %v", err)
			}
		}
	}

	if tAccount != nil &&
		tAccount.MarginBalance != nil &&
		mAccount != nil {
		totalBalance := *tAccount.MarginBalance + mAccount.MarginBalance + mAccount.UnrealisedPNL
		netWorth := totalBalance / *mtConfig.StartValue
		fields := make(map[string]interface{})
		fields["totalUnHedgeValue"] = totalUnHedgeValue
		fields["totalBalance"] = totalBalance
		fields["takerBalance"] = *tAccount.MarginBalance
		fields["makerBalance"] = mAccount.MarginBalance + mAccount.UnrealisedPNL
		fields["netWorth"] = netWorth
		fields["startValue"] = *mtConfig.StartValue
		fields["netWorth"] = netWorth
		if tAccount.AvailableBalance != nil {
			fields["takerAvailable"] = *tAccount.AvailableBalance
		}
		if tAccount.UnrealizedProfit != nil {
			fields["takerUnrealizedProfit"] = *tAccount.UnrealizedProfit
		}
		fields["makerAvailable"] = mAccount.AvailableBalance
		fields["makerUnrealizedProfit"] = mAccount.UnrealisedPNL
		pt, err := client.NewPoint(
			*mtConfig.InternalInflux.Measurement,
			map[string]string{
				"type": "balance",
			},
			fields,
			time.Now().UTC(),
		)
		if err != nil {
			logger.Debugf("client.NewPoint error %v", err)
		} else {
			err = mtInfluxWriter.PushPoint(pt)
			if err != nil {
				logger.Debugf("mtInfluxWriter.PushPoint error %v", err)
			}
		}
	}
}

func handleExternalInfluxSave() {

	if time.Now().Sub(mtGlobalSilent) < 0 {
		return
	}

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
				logger.Debugf("client.NewPoint error %v", err)
			} else {
				err = mtExternalInfluxWriter.PushPoint(pt)
				if err != nil {
					logger.Debugf("mtExternalInfluxWriter.PushPoint error %v", err)
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
			for _, report := range spreadReports {
				fields := make(map[string]interface{})
				fields["matchRatio"] = report.MatchRatio
				fields["maxAgeDiff"] = float64(report.MaxAgeDiff)
				fields["makerTimeDeltaEma"] = report.MakerTimeDeltaEma
				fields["takerTimeDeltaEma"] = report.TakerTimeDeltaEma
				fields["makerTimeDelta"] = report.MakerTimeDelta
				fields["takerTimeDelta"] = report.TakerTimeDelta
				fields["makerDepthFilterRatio"] = report.MakerDepthFilterRatio
				fields["takerDepthFilterRatio"] = report.MakerDepthFilterRatio
				fields["makerMsgAvgLen"] = report.MakerMsgAvgLen
				fields["takerMsgAvgLen"] = report.TakerMsgAvgLen
				if len(fields) > 0 {
					pt, err := client.NewPoint(
						*influxConfig.Measurement,
						map[string]string{
							"makerSymbol": report.MakerSymbol,
							"takerSymbol": report.TakerSymbol,
							"type":        "spread-report",
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
