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

func handleInternalSave() {

	entryTarget := 0.0
	if mAccount != nil && tAccount != nil {
		entryStep := (mAccount.GetFree() + tAccount.GetFree()) * mtConfig.EnterFreePct
		if entryStep < mtConfig.EnterMinimalStep {
			entryStep = mtConfig.EnterMinimalStep
		}
		entryTarget = entryStep * mtConfig.EnterTargetFactor
	}

	totalUnHedgeValue := 0.0
	takerURPnl := 0.0
	makerURPnl := 0.0
	for _, makerSymbol := range mSymbols {
		takerSymbol := mtSymbolsMap[makerSymbol]
		delta := mtDeltas[makerSymbol]
		fields := make(map[string]interface{})
		if makerPosition, ok := mPositions[makerSymbol]; ok {
			fields["makerSize"] = makerPosition.GetSize()
			if spread, ok := mtSpreads[makerSymbol]; ok {
				makerValue := makerPosition.GetSize() * makerPosition.GetPrice()
				fields["makerValue"] = makerValue
				fields["shortTop"] = delta.ShortTop + mtConfig.OffsetDelta*(math.Max(makerValue, 0)/entryTarget)
				fields["shortBot"] = delta.ShortBot + mtConfig.OffsetDelta*(math.Max(makerValue, 0)/entryTarget)
				fields["longBot"] = delta.LongBot + mtConfig.OffsetDelta*(math.Min(makerValue, 0)/entryTarget)
				fields["longTop"] = delta.LongTop + mtConfig.OffsetDelta*(math.Min(makerValue, 0)/entryTarget)
				if makerPosition.GetPrice() != 0 {
					makerURPnl += makerPosition.GetSize() * (spread.MakerDepth.MidPrice - makerPosition.GetPrice())
				}
				if takerPosition, ok := tPositions[takerSymbol]; ok {
					unHedgedValue := (takerPosition.GetSize() + makerPosition.GetSize()) * spread.MakerDepth.MidPrice
					fields["unHedgedValue"] = unHedgedValue
					totalUnHedgeValue += math.Abs(unHedgedValue)
					if takerPosition.GetPrice() != 0 {
						takerURPnl += takerPosition.GetSize() * (spread.TakerDepth.MidPrice - takerPosition.GetPrice())
					}
				}
			}
		}
		if takerPosition, ok := tPositions[takerSymbol]; ok {
			fields["takerSize"] = takerPosition.GetSize()
			fields["takerValue"] = takerPosition.GetPrice() * takerPosition.GetSize()
		}
		if fr, ok := mFundingRates[makerSymbol]; ok {
			fields["makerFundingRate"] = fr.GetFundingRate()
		}
		if fr, ok := tFundingRates[takerSymbol]; ok {
			fields["takerFundingRate"] = fr.GetFundingRate()
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
			fields["takerBestBidPrice"] = spread.TakerDepth.BestBidPrice
			fields["takerBestAskPrice"] = spread.TakerDepth.BestAskPrice

			fields["makerMakerBid"] = spread.MakerDepth.MakerBid
			fields["makerMakerAsk"] = spread.MakerDepth.MakerAsk
			fields["makerTakerBid"] = spread.MakerDepth.TakerBid
			fields["makerTakerAsk"] = spread.MakerDepth.TakerAsk

			fields["takerDir"] = spread.TakerDir
			fields["makerDir"] = spread.MakerDir

			fields["age"] = spread.Age.Seconds()
			fields["ageDiff"] = spread.AgeDiff.Seconds()
		}
		if realisedSpread, ok := mtRealisedSpread[makerSymbol]; ok {
			fields["realisedSpread"] = realisedSpread
		}
		if mSystemStatus == common.SystemStatusReady {
			fields["makerSystemStatus"] = 1.0
		} else {
			fields["makerSystemStatus"] = -1.0
		}
		if tSystemStatus == common.SystemStatusReady {
			fields["takerSystemStatus"] = 1.0
		} else {
			fields["takerSystemStatus"] = -1.0
		}
		pt, err := client.NewPoint(
			mtConfig.InternalInflux.Measurement,
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
		mAccount != nil {
		totalBalance := tAccount.GetBalance() + mAccount.GetBalance()
		netWorth := totalBalance / mtConfig.StartValue
		fields := make(map[string]interface{})
		fields["totalUnHedgeValue"] = totalUnHedgeValue
		fields["totalBalance"] = totalBalance
		fields["takerBalance"] = tAccount.GetBalance()
		fields["makerBalance"] = mAccount.GetBalance()
		fields["netWorth"] = netWorth
		fields["startValue"] = mtConfig.StartValue
		fields["netWorth"] = netWorth
		fields["takerAvailable"] = tAccount.GetFree()
		fields["takerURPnl"] = takerURPnl
		fields["makerAvailable"] = mAccount.GetFree()
		fields["makerURPnl"] = makerURPnl
		pt, err := client.NewPoint(
			mtConfig.InternalInflux.Measurement,
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

	if tAccount != nil && mAccount != nil {
		totalBalance := tAccount.GetBalance() + mAccount.GetBalance()
		netWorth := totalBalance / mtConfig.StartValue
		fields := make(map[string]interface{})
		fields["netWorth"] = netWorth
		for name, start := range mtConfig.StartValues {
			if start > 0 {
				fields["currentValue_"+strings.ToLower(name)] = netWorth * start
			}
		}
		if len(fields) > 0 {
			pt, err := client.NewPoint(
				mtConfig.ExternalInflux.Measurement,
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
	influxConfig common.InfluxSettings,
	spreadReportCh chan common.SpreadReport,
) {
	spreadReports := make(map[string]common.SpreadReport)
	saveTimer := time.NewTimer(influxConfig.SaveInterval)
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
				fields["adjustedAgeDiff"] = float64(report.AdjustedAgeDiff / time.Millisecond)
				fields["makerTimeDeltaEma"] = report.MakerTimeDeltaEma
				fields["takerTimeDeltaEma"] = report.TakerTimeDeltaEma
				fields["makerTimeDelta"] = report.MakerTimeDelta
				fields["takerTimeDelta"] = report.TakerTimeDelta
				fields["makerDepthFilterRatio"] = report.MakerDepthFilterRatio
				fields["takerDepthFilterRatio"] = report.MakerDepthFilterRatio
				fields["makerExpireRatio"] = report.MakerExpireRatio
				fields["takerExpireRatio"] = report.TakerExpireRatio
				if len(fields) > 0 {
					pt, err := client.NewPoint(
						influxConfig.Measurement,
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
			saveTimer.Reset(influxConfig.SaveInterval)
			break
		}
	}
}
