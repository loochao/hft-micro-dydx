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
	if xAccount != nil && yAccount != nil {
		entryStep := (xAccount.GetFree() + yAccount.GetFree()) * xyConfig.EnterFreePct
		if entryStep < xyConfig.EnterMinimalStep {
			entryStep = xyConfig.EnterMinimalStep
		}
		entryTarget = entryStep * xyConfig.EnterTargetFactor
	}

	totalUnHedgeValue := 0.0
	takerURPnl := 0.0
	makerURPnl := 0.0
	for _, makerSymbol := range xSymbols {
		takerSymbol := xySymbolsMap[makerSymbol]
		delta := xyDeltas[makerSymbol]
		fields := make(map[string]interface{})
		if makerPosition, ok := xPositions[makerSymbol]; ok {
			fields["makerSize"] = makerPosition.GetSize()
			if spread, ok := xySpreads[makerSymbol]; ok {
				makerValue := makerPosition.GetSize() * makerPosition.GetPrice()
				fields["makerValue"] = makerValue
				fields["shortTop"] = delta.ShortTop + xyConfig.EnterOffsetDelta*(math.Max(makerValue, 0)/entryTarget)
				fields["shortBot"] = delta.ShortBot + xyConfig.EnterOffsetDelta*(math.Max(makerValue, 0)/entryTarget)
				fields["longBot"] = delta.LongBot + xyConfig.EnterOffsetDelta*(math.Min(makerValue, 0)/entryTarget)
				fields["longTop"] = delta.LongTop + xyConfig.EnterOffsetDelta*(math.Min(makerValue, 0)/entryTarget)
				if makerPosition.GetPrice() != 0 {
					makerURPnl += makerPosition.GetSize() * (spread.MakerDepth.MidPrice - makerPosition.GetPrice())
				}
				if takerPosition, ok := yPositions[takerSymbol]; ok {
					unHedgedValue := (takerPosition.GetSize() + makerPosition.GetSize()) * spread.MakerDepth.MidPrice
					fields["unHedgedValue"] = unHedgedValue
					totalUnHedgeValue += math.Abs(unHedgedValue)
					if takerPosition.GetPrice() != 0 {
						takerURPnl += takerPosition.GetSize() * (spread.TakerDepth.MidPrice - takerPosition.GetPrice())
					}
				}
			}
		}
		if takerPosition, ok := yPositions[takerSymbol]; ok {
			fields["takerSize"] = takerPosition.GetSize()
			fields["takerValue"] = takerPosition.GetPrice() * takerPosition.GetSize()
		}
		if fr, ok := xFundingRates[makerSymbol]; ok {
			fields["makerFundingRate"] = fr.GetFundingRate()
		}
		if fr, ok := yFundingRates[takerSymbol]; ok {
			fields["takerFundingRate"] = fr.GetFundingRate()
		}
		if fr, ok := xyFundingRates[makerSymbol]; ok {
			fields["fundingRate"] = fr
		}
		if spread, ok := xySpreads[makerSymbol]; ok {

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
		if realisedSpread, ok := xyRealisedSpread[makerSymbol]; ok {
			fields["realisedSpread"] = realisedSpread
		}
		if xSystemStatus == common.SystemStatusReady {
			fields["makerSystemStatus"] = 1.0
		} else {
			fields["makerSystemStatus"] = -1.0
		}
		if ySystemStatus == common.SystemStatusReady {
			fields["takerSystemStatus"] = 1.0
		} else {
			fields["takerSystemStatus"] = -1.0
		}
		pt, err := client.NewPoint(
			xyConfig.InternalInflux.Measurement,
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
			err = xyInfluxWriter.PushPoint(pt)
			if err != nil {
				logger.Debugf("xyInfluxWriter.PushPoint error %v", err)
			}
		}
	}

	if yAccount != nil &&
		xAccount != nil {
		totalBalance := yAccount.GetBalance() + xAccount.GetBalance()
		netWorth := totalBalance / xyConfig.StartValue
		fields := make(map[string]interface{})
		fields["totalUnHedgeValue"] = totalUnHedgeValue
		fields["totalBalance"] = totalBalance
		fields["takerBalance"] = yAccount.GetBalance()
		fields["makerBalance"] = xAccount.GetBalance()
		fields["netWorth"] = netWorth
		fields["startValue"] = xyConfig.StartValue
		fields["netWorth"] = netWorth
		fields["takerAvailable"] = yAccount.GetFree()
		fields["takerURPnl"] = takerURPnl
		fields["makerAvailable"] = xAccount.GetFree()
		fields["makerURPnl"] = makerURPnl
		pt, err := client.NewPoint(
			xyConfig.InternalInflux.Measurement,
			map[string]string{
				"type": "balance",
			},
			fields,
			time.Now().UTC(),
		)
		if err != nil {
			logger.Debugf("client.NewPoint error %v", err)
		} else {
			err = xyInfluxWriter.PushPoint(pt)
			if err != nil {
				logger.Debugf("xyInfluxWriter.PushPoint error %v", err)
			}
		}
	}
}

func handleExternalInfluxSave() {

	if yAccount != nil && xAccount != nil {
		totalBalance := yAccount.GetBalance() + xAccount.GetBalance()
		netWorth := totalBalance / xyConfig.StartValue
		fields := make(map[string]interface{})
		fields["netWorth"] = netWorth
		for name, start := range xyConfig.StartValues {
			if start > 0 {
				fields["currentValue_"+strings.ToLower(name)] = netWorth * start
			}
		}
		if len(fields) > 0 {
			pt, err := client.NewPoint(
				xyConfig.ExternalInflux.Measurement,
				map[string]string{
					"name": *xyConfig.Name,
				},
				fields,
				time.Now().UTC(),
			)
			if err != nil {
				logger.Debugf("client.NewPoint error %v", err)
			} else {
				err = xyExternalInfluxWriter.PushPoint(pt)
				if err != nil {
					logger.Debugf("xyExternalInfluxWriter.PushPoint error %v", err)
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
