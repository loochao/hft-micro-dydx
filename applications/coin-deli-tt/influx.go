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

	totalUnHedgeValue := 0.0
	totalValue := 0.0
	hasAllSymbols := true
	for _, xSymbol := range xSymbols {
		ySymbol := xySymbolsMap[xSymbol]
		xPosition, okXPosition := xyPositions[xSymbol]
		yPosition, okYPosition := xyPositions[ySymbol]
		balance, okBalance := xyBalanceMap[xyConfig.SymbolAssetMap[xSymbol]]
		spread, okSpread := xySpreads[xSymbol]

		fields := make(map[string]interface{})
		if okXPosition && okYPosition && okSpread && okBalance {

			xContractSize := xyMultipliers[xSymbol]
			yContractSize := xyMultipliers[ySymbol]
			xDepth := spread.XDepth
			//yDepth := spread.YDepth
			xSize := xPosition.GetSize()
			xValue := xSize * xContractSize
			ySize := yPosition.GetSize()
			yValue := ySize * yContractSize
			balanceInUSD := balance.GetBalance() * xDepth.MidPrice
			totalValue += balanceInUSD

			fields["balanceInCoin"] = balance.GetBalance()
			fields["balanceInUSD"] = balanceInUSD
			fields["xPosEventTime"] = xPosition.GetEventTime().UnixNano()
			fields["xPosParseTime"] = xPosition.GetParseTime().UnixNano()
			fields["yPosEventTime"] = yPosition.GetEventTime().UnixNano()
			fields["yPosParseTime"] = yPosition.GetParseTime().UnixNano()
			fields["xSize"] = xSize
			fields["ySize"] = ySize
			fields["xValue"] = xValue
			fields["yValue"] = yValue
			fields["xAdjValue"] = xValue + balanceInUSD
			fields["xyUnHedgeValue"] = xValue + yValue + balanceInUSD

			expireDate := xyConfig.ExpireDates[ySymbol]
			expireRatio := float64(time.Now().Sub(expireDate)) / float64(xyConfig.DeliDuration)

			offsetFactor := math.Abs(yValue) / balanceInUSD / xyConfig.EnterTarget
			offsetStep := xyConfig.EnterStep / xyConfig.EnterTarget
			shortTop := xyConfig.ShortEnterDelta + xyConfig.EnterOffsetDelta*offsetFactor*expireRatio
			shortBot := xyConfig.ShortExitDelta + xyConfig.ExitOffsetDelta*(offsetFactor-offsetStep)*expireRatio
			longBot := xyConfig.LongEnterDelta - xyConfig.EnterOffsetDelta*offsetFactor*expireRatio
			longTop := xyConfig.LongExitDelta - xyConfig.ExitOffsetDelta*(offsetFactor-offsetStep)*expireRatio

			fields["expireRatio"] = expireRatio
			fields["shortTop"] = shortTop
			fields["shortBot"] = shortBot
			fields["longBot"] = longBot
			fields["longTop"] = longTop

			fields["spreadTime"] = spread.Time.UnixNano()
			fields["spreadShortLastEnter"] = spread.ShortLastEnter
			fields["spreadShortLastLeave"] = spread.ShortLastLeave
			fields["spreadShortMedianEnter"] = spread.ShortMedianEnter
			fields["spreadShortMedianLeave"] = spread.ShortMedianLeave

			fields["spreadLongLastEnter"] = spread.LongLastEnter
			fields["spreadLongLastLeave"] = spread.LongLastLeave
			fields["spreadLongMedianEnter"] = spread.LongMedianEnter
			fields["spreadLongMedianLeave"] = spread.LongMedianLeave

			fields["yBidPrice"] = spread.YDepth.BidPrice
			fields["yAskPrice"] = spread.YDepth.AskPrice
			fields["yMidPrice"] = spread.YDepth.MidPrice
			fields["xBidPrice"] = spread.XDepth.BidPrice
			fields["xAskPrice"] = spread.XDepth.AskPrice
			fields["xMidPrice"] = spread.XDepth.MidPrice

			fields["age"] = spread.Age.Seconds()
			fields["ageDiff"] = spread.AgeDiff.Seconds()
		} else {
			logger.Debugf("%s %s save failed, okXPosition %v okYPosition %v okSpread %v okBalance %v", xSymbol, ySymbol, okXPosition, okYPosition, okSpread, okBalance)
			hasAllSymbols = false
		}
		if fr, ok := xFundingRates[xSymbol]; ok {
			fields["xFundingRate"] = fr.GetFundingRate()
		}
		if realisedSpread, ok := xyRealisedSpread[xSymbol]; ok {
			fields["realisedSpread"] = realisedSpread
		}
		if xySystemStatus == common.SystemStatusReady {
			fields["xySystemStatus"] = 1.0
		} else {
			fields["xySystemStatus"] = -1.0
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

	if hasAllSymbols {
		totalBalance := totalValue
		netWorth := totalBalance / xyConfig.StartValue
		fields := make(map[string]interface{})
		fields["totalUnHedgeValue"] = totalUnHedgeValue
		fields["totalBalance"] = totalBalance

		fields["xyTurnover"] = xyTimedPositionChange.Sum() / totalBalance
		fields["netWorth"] = netWorth
		fields["startValue"] = xyConfig.StartValue
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

		fields = make(map[string]interface{})
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
