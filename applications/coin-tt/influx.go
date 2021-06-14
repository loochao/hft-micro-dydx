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

	if xyConfig.InternalInflux.Address == "" {
		return
	}

	totalUnHedgeValue := 0.0
	totalXUSDValue := 0.0
	totalYUSDValue := 0.0
	yURPnl := 0.0
	xURPnl := 0.0
	totalURPnl := 0.0
	hasAllSymbols := true
	totalUSDValue := 0.0
	for _, xSymbol := range xSymbols {
		ySymbol := xySymbolsMap[xSymbol]
		xPosition, okXPosition := xPositions[xSymbol]
		yPosition, okYPosition := yPositions[ySymbol]
		xBalance, okXBalance := xBalances[xyConfig.XSymbolAssetMap[xSymbol]]
		yBalance, okYBalance := yBalances[xyConfig.YSymbolAssetMap[ySymbol]]
		spread, okSpread := xySpreads[xSymbol]
		yMultiplier := yMultipliers[ySymbol]
		xMultiplier := xMultipliers[xSymbol]
		fields := make(map[string]interface{})
		if okXPosition && okYPosition && okSpread && okXBalance && okYBalance {

			xSize := xPosition.GetSize()
			xValue := xSize * xMultiplier
			ySize := yPosition.GetSize()
			yValue := ySize * yMultiplier

			fields["yBalanceInUSD"] = yBalance.GetBalance() * spread.YDepth.MidPrice
			fields["xBalanceInUSD"] = xBalance.GetBalance() * spread.XDepth.MidPrice
			fields["yBalanceInCoin"] = yBalance.GetBalance()
			fields["xBalanceInCoin"] = xBalance.GetBalance()
			spotValue := xBalance.GetBalance()*spread.XDepth.MidPrice + yBalance.GetBalance()*spread.YDepth.MidPrice
			totalUSDValue += spotValue
			totalXUSDValue += xBalance.GetBalance()*spread.XDepth.MidPrice
			totalYUSDValue += yBalance.GetBalance()*spread.YDepth.MidPrice

			offsetFactor := math.Abs(yValue) / spotValue / xyConfig.EnterTarget
			offsetStep := math.Min(xyConfig.EnterStep/xyConfig.EnterTarget, offsetFactor)

			shortTop := xyConfig.ShortEnterDelta + xyConfig.EnterOffsetDelta*offsetFactor
			shortBot := xyConfig.ShortExitDelta + xyConfig.ExitOffsetDelta*(offsetFactor-offsetStep)
			longBot := xyConfig.LongEnterDelta - xyConfig.EnterOffsetDelta*offsetFactor
			longTop := xyConfig.LongExitDelta - xyConfig.ExitOffsetDelta*(offsetFactor-offsetStep)

			unHedgeValue := math.Abs(xValue + yValue)
			totalUnHedgeValue += unHedgeValue

			fields["xPosEventTime"] = xPosition.GetEventTime().UnixNano()
			fields["xPosParseTime"] = xPosition.GetParseTime().UnixNano()
			fields["yPosEventTime"] = yPosition.GetEventTime().UnixNano()
			fields["yPosParseTime"] = yPosition.GetParseTime().UnixNano()
			fields["unHedgeValue"] = unHedgeValue
			fields["xSize"] = xSize
			fields["xValue"] = xValue
			fields["ySize"] = ySize
			fields["yValue"] = yValue
			fields["xyValue"] = xValue + yValue

			fields["shortTop"] = shortTop
			fields["shortBot"] = shortBot
			fields["longBot"] = longBot
			fields["longTop"] = longTop

			if yPosition.GetPrice() != 0 {
				yURPnl += yValue * (1.0/yPosition.GetPrice() - 1.0/spread.YDepth.MidPrice) * spread.YDepth.MidPrice
			}
			if xPosition.GetPrice() != 0 {
				xURPnl += xValue * (1.0/xPosition.GetPrice() - 1.0/spread.XDepth.MidPrice) * spread.XDepth.MidPrice
			}
			totalURPnl += yURPnl + xURPnl

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
		} else {
			logger.Debugf("%s %s save failed, okXPosition %v okYPosition %v okSpread %v okXBalance %v okYBalance %v", xSymbol, ySymbol, okXPosition, okYPosition, okSpread, okXBalance, okYBalance)
			hasAllSymbols = false
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

	if hasAllSymbols {
		netWorth := totalUSDValue / xyConfig.StartValue
		fields := make(map[string]interface{})
		fields["totalUnHedgeValue"] = totalUnHedgeValue
		fields["totalUSDValue"] = totalUSDValue
		fields["xURPnl"] = xURPnl
		fields["yURPnl"] = yURPnl
		fields["xyTurnover"] = (xTimedPositionChange.Sum() + yTimedPositionChange.Sum()) / totalUSDValue
		fields["xTurnover"] = xTimedPositionChange.Sum() / totalXUSDValue
		fields["yTurnover"] = yTimedPositionChange.Sum() / totalYUSDValue
		fields["xyURPnl"] = totalURPnl
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
			if influxConfig.Address == "" {
				saveTimer.Reset(influxConfig.SaveInterval)
				break
			}
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
				fields["ageDiff"] = report.AgeDiff.Seconds()

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
