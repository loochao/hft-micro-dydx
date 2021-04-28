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
	if time.Now().Sub(mtGlobalSilent) < 0 {
		return
	}

	if tAccount != nil &&
		tAccount.MarginBalance != nil {
		totalBalance := *tAccount.MarginBalance
		netWorth := totalBalance / *mtConfig.StartValue
		fields := make(map[string]interface{})
		fields["totalBalance"] = totalBalance
		fields["takerBalance"] = *tAccount.MarginBalance
		fields["netWorth"] = netWorth
		fields["startValue"] = *mtConfig.StartValue
		fields["netWorth"] = netWorth
		if tAccount.AvailableBalance != nil {
			fields["takerAvailable"] = *tAccount.AvailableBalance
		}
		if tAccount.UnrealizedProfit != nil {
			fields["takerUnrealizedProfit"] = *tAccount.UnrealizedProfit
		}
		pt, err := client.NewPoint(
			*mtConfig.InternalInflux.Measurement,
			map[string]string{
				"type": "balance",
			},
			fields,
			time.Now().UTC(),
		)
		if err != nil {
			logger.Debugf("Spot Balance NewPoint error %v", err)
		} else {
			go mtInfluxWriter.PushPoint(pt)
		}
	}

	for _, takerSymbol := range tSymbols {
		fields := make(map[string]interface{})
		if takerPosition, ok := tPositions[takerSymbol]; ok {
			fields["takerSize"] = takerPosition.PositionAmt
			if spread, ok := mtSpreads[takerSymbol]; ok {
				fields["takerValue"] = spread.TakerDepth.TakerBid * takerPosition.PositionAmt
			}
		}
		if spread, ok := mtSpreads[takerSymbol]; ok {

			fields["spreadShortLastEnter"] = spread.LastEnter
			fields["spreadShortLastLeave"] = spread.LastLeave
			fields["spreadShortMedianEnter"] = spread.MedianEnter
			fields["spreadShortMedianLeave"] = spread.MedianLeave


			fields["takerTakerBid"] = spread.TakerDepth.TakerBid
			fields["takerTakerAsk"] = spread.TakerDepth.TakerAsk

			fields["makerTakerBid"] = spread.MakerDepth.TakerBid
			fields["makerTakerAsk"] = spread.MakerDepth.TakerAsk

			fields["age"] = spread.Age.Seconds()
			fields["ageDiff"] = spread.AgeDiff.Seconds()
		}
		if realisedSpread, ok := mtRealisedSpread[takerSymbol]; ok {
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
				"type":        "symbol",
			},
			fields,
			time.Now().UTC(),
		)
		if err != nil {
			logger.Debugf("new buyPosition point error %v", err)
		} else {
			go mtInfluxWriter.PushPoint(pt)
		}
	}
}

func handleExternalInfluxSave() {

	if time.Now().Sub(mtGlobalSilent) < 0 {
		return
	}

	if tAccount != nil &&
		tAccount.MarginBalance != nil {
		totalBalance := *tAccount.MarginBalance
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
				logger.Debugf("Margin NewPoint error %v", err)
			} else {
				go mtExternalInfluxWriter.PushPoint(pt)
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
						logger.Debugf("SpreadReport NewPoint error %v", err)
					} else {
						select {
						case influxWriter.pushCh <- pt:
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
