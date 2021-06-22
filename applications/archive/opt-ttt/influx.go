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
	yURPnl := 0.0
	xURPnl := 0.0
	for _, xSymbol := range xSymbols {
		ySymbol := xySymbolsMap[xSymbol]
		fields := make(map[string]interface{})
		if xPosition, ok := xPositions[xSymbol]; ok {
			fields["makerSize"] = xPosition.GetSize()
			if spread, ok := xySpreads[xSymbol]; ok {
				xValue := xPosition.GetSize() * xPosition.GetPrice()
				fields["xValue"] = xValue

				if yPosition, ok := yPositions[ySymbol]; ok {
					if entryTarget != 0 {
						xValue := math.Abs(xPosition.GetSize()) * xPosition.GetPrice()
						yValue := math.Abs(yPosition.GetSize()) * yPosition.GetPrice()
						offsetFactor := (xValue + yValue) * 0.5 / entryTarget
						shortTop := xyConfig.ShortEnterDelta + xyConfig.EnterOffsetDelta*offsetFactor
						shortBot := xyConfig.ShortExitDelta
						longBot := xyConfig.LongEnterDelta - xyConfig.EnterOffsetDelta*offsetFactor
						longTop := xyConfig.LongExitDelta
						fields["shortTop"] = shortTop
						fields["shortBot"] = shortBot
						fields["longBot"] = longBot
						fields["longTop"] = longTop
					}
					unHedgedValue := (yPosition.GetSize() + xPosition.GetSize()) * spread.XDepth.MidPrice
					fields["unHedgedValue"] = unHedgedValue
					totalUnHedgeValue += math.Abs(unHedgedValue)
					if yPosition.GetPrice() != 0 {
						yURPnl += yPosition.GetSize() * (spread.YDepth.MidPrice - yPosition.GetPrice())
					}
				}

				if xPosition.GetPrice() != 0 {
					xURPnl += xPosition.GetSize() * (spread.XDepth.MidPrice - xPosition.GetPrice())
				}
			}
		}
		if yPosition, ok := yPositions[ySymbol]; ok {
			fields["ySize"] = yPosition.GetSize()
			fields["yValue"] = yPosition.GetPrice() * yPosition.GetSize()
		}
		if fr, ok := xFundingRates[xSymbol]; ok {
			fields["xFundingRate"] = fr.GetFundingRate()
		}
		if fr, ok := yFundingRates[ySymbol]; ok {
			fields["yFundingRate"] = fr.GetFundingRate()
		}
		if fr, ok := xyFundingRates[xSymbol]; ok {
			fields["fundingRate"] = fr
		}
		if spread, ok := xySpreads[xSymbol]; ok {

			fields["spreadShortLastEnter"] = spread.ShortLastEnter
			fields["spreadShortLastLeave"] = spread.ShortLastLeave
			fields["spreadShortMedianEnter"] = spread.ShortMedianEnter
			fields["spreadShortMedianLeave"] = spread.ShortMedianLeave

			fields["spreadLongLastEnter"] = spread.LongLastEnter
			fields["spreadLongLastLeave"] = spread.LongLastLeave
			fields["spreadLongMedianEnter"] = spread.LongMedianEnter
			fields["spreadLongMedianLeave"] = spread.LongMedianLeave

			fields["yMakerBid"] = spread.YDepth.MakerBid
			fields["yMakerAsk"] = spread.YDepth.MakerAsk
			fields["yTakerBid"] = spread.YDepth.TakerBid
			fields["yTakerAsk"] = spread.YDepth.TakerAsk
			fields["yBestBidPrice"] = spread.YDepth.BestBidPrice
			fields["yBestAskPrice"] = spread.YDepth.BestAskPrice

			fields["xMakerBid"] = spread.XDepth.MakerBid
			fields["xMakerAsk"] = spread.XDepth.MakerAsk
			fields["xTakerBid"] = spread.XDepth.TakerBid
			fields["xTakerAsk"] = spread.XDepth.TakerAsk
			fields["xBestBidPrice"] = spread.XDepth.BestBidPrice
			fields["xBestAskPrice"] = spread.XDepth.BestAskPrice

			fields["yDir"] = spread.YDir
			fields["xDir"] = spread.XDir
			fields["dir"] =  spread.XDir*xyConfig.XYDirRatio + spread.YDir*(1.0-xyConfig.XYDirRatio)

			fields["age"] = spread.Age.Seconds()
			fields["ageDiff"] = spread.AgeDiff.Seconds()
		}
		if realisedSpread, ok := xyRealisedSpread[xSymbol]; ok {
			fields["realisedSpread"] = realisedSpread
		}
		if xSystemStatus == common.SystemStatusReady {
			fields["xSystemStatus"] = 1.0
		} else {
			fields["xSystemStatus"] = -1.0
		}
		if ySystemStatus == common.SystemStatusReady {
			fields["ySystemStatus"] = 1.0
		} else {
			fields["ySystemStatus"] = -1.0
		}
		pt, err := client.NewPoint(
			xyConfig.InternalInflux.Measurement,
			map[string]string{
				"ySymbol": ySymbol,
				"xSymbol": xSymbol,
				"type":    "symbol",
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
		fields["yBalance"] = yAccount.GetBalance()
		fields["xBalance"] = xAccount.GetBalance()
		fields["netWorth"] = netWorth
		fields["startValue"] = xyConfig.StartValue
		fields["netWorth"] = netWorth
		fields["yAvailable"] = yAccount.GetFree()
		fields["yURPnl"] = yURPnl
		fields["xAvailable"] = xAccount.GetFree()
		fields["xURPnl"] = xURPnl
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
	spreadReportCh chan SpreadReport,
) {
	spreadReports := make(map[string]SpreadReport)
	saveTimer := time.NewTimer(influxConfig.SaveInterval)
	for {
		select {
		case <-ctx.Done():
			return
		case spreadReport := <-spreadReportCh:
			//logger.Debugf("%s", spreadReport.ToString())
			spreadReports[spreadReport.XSymbol] = spreadReport
			break
		case <-saveTimer.C:
			for _, report := range spreadReports {
				fields := make(map[string]interface{})
				fields["matchRatio"] = report.MatchRatio
				fields["adjustedAgeDiff"] = float64(report.AdjustedAgeDiff / time.Millisecond)
				fields["xTimeDeltaEma"] = report.XTimeDeltaEma
				fields["yTimeDeltaEma"] = report.YTimeDeltaEma
				fields["xTimeDelta"] = report.XTimeDelta
				fields["yTimeDelta"] = report.YTimeDelta
				fields["xDepthFilterRatio"] = report.XDepthFilterRatio
				fields["yDepthFilterRatio"] = report.XDepthFilterRatio
				fields["xExpireRatio"] = report.XExpireRatio
				fields["yExpireRatio"] = report.YExpireRatio
				if len(fields) > 0 {
					pt, err := client.NewPoint(
						influxConfig.Measurement,
						map[string]string{
							"xSymbol": report.XSymbol,
							"ySymbol": report.YSymbol,
							"type":    "spread-report",
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
