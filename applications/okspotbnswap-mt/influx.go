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
		tAccount.MarginBalance != nil &&
		mAccount != nil {

		mTotalBalance := mAccount.Balance
		getAll := true
		for makerSymbol, b := range mBalances {
			if b.Balance != 0 {
				if spread, ok := mtSpreads[makerSymbol]; ok {
					mTotalBalance += b.Balance * spread.MakerDepth.TakerBid
				} else {
					logger.Debugf("MISS SPREAD FOR %s", makerSymbol)
					getAll = false
					break
				}
			}
		}

		fields := make(map[string]interface{})
		if getAll {
			totalBalance := *tAccount.MarginBalance + mTotalBalance
			netWorth := totalBalance / *mtConfig.StartValue
			fields["totalBalance"] = totalBalance
			fields["makerBalance"] = mTotalBalance
			fields["netWorth"] = netWorth
		}

		fields["takerBalance"] = *tAccount.MarginBalance
		fields["startValue"] = *mtConfig.StartValue
		fields["makerAvailable"] = mAccount.Available
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
			go mtInfluxWriter.Push(pt)
		}
	}

	for _, makerSymbol := range mSymbols {
		takerSymbol := mtSymbolsMap[makerSymbol]
		fields := make(map[string]interface{})
		if makerBalance, ok := mBalances[makerSymbol]; ok {
			fields["makerSize"] = makerBalance.Balance
			if spread, ok := mtSpreads[makerSymbol]; ok {
				fields["makerValue"] = makerBalance.Balance * spread.MakerDepth.MidPrice
			}
		}
		if takerPosition, ok := tPositions[takerSymbol]; ok {
			fields["takerSize"] = takerPosition.PositionAmt
			if spread, ok := mtSpreads[makerSymbol]; ok {
				fields["takerValue"] = spread.TakerDepth.TakerBid * takerPosition.PositionAmt
			}
		}
		if fr, ok := mtFundingRates[makerSymbol]; ok {
			fields["fundingRate"] = fr
		}
		if spread, ok := mtSpreads[makerSymbol]; ok {

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
		if realisedSpread, ok := mtRealisedSpread[makerSymbol]; ok {
			fields["realisedSpread"] = realisedSpread
		}
		if quantile, ok := mtQuantiles[makerSymbol]; ok {
			fields["quantileShortBot"] = quantile.ShortBot
			fields["quantileShortTop"] = quantile.ShortTop
			fields["quantileLongBot"] = quantile.LongBot
			fields["quantileLongTop"] = quantile.LongTop
			fields["quantileMid"] = quantile.Mid
			fields["quantileMaClose"] = quantile.MaClose
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
			logger.Debugf("new makerBalance point error %v", err)
		} else {
			go mtInfluxWriter.Push(pt)
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

		mTotalBalance := mAccount.Balance
		getAll := true
		for makerSymbol, b := range mBalances {
			if b.Balance != 0 {
				if spread, ok := mtSpreads[makerSymbol]; ok {
					mTotalBalance += b.Balance * spread.MakerDepth.TakerBid
				} else {
					getAll = false
					break
				}
			}
		}

		if getAll {
			totalBalance := *tAccount.MarginBalance + mTotalBalance
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
					go mtExternalInfluxWriter.Push(pt)
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
						logger.Debugf("SpreadReport NewPoint error %v", err)
					} else {
						select {
						case influxWriter.PushCh <- pt:
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
