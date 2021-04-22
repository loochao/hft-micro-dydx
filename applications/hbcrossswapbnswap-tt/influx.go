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

	if tAccount != nil &&
		tAccount.MarginBalance != nil &&
		mAccount != nil {
		totalBalance := *tAccount.MarginBalance + mAccount.MarginBalance
		netWorth := totalBalance / *mtConfig.StartValue
		fields := make(map[string]interface{})
		fields["totalBalance"] = totalBalance
		fields["takerBalance"] = *tAccount.MarginBalance
		fields["makerBalance"] = mAccount.MarginBalance
		fields["netWorth"] = netWorth
		fields["startValue"] = *mtConfig.StartValue
		fields["netWorth"] = netWorth
		if tAccount.AvailableBalance != nil {
			fields["takerAvailable"] = *tAccount.AvailableBalance
		}
		if tAccount.UnrealizedProfit != nil {
			fields["takerUnrealizedProfit"] = *tAccount.UnrealizedProfit
		}
		fields["makerAvailable"] = mAccount.WithdrawAvailable
		fields["makerUnrealizedProfit"] = mAccount.ProfitUnreal
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
		if buyPosition, ok := mBuyPositions[makerSymbol]; ok {
			if sellPosition, ok := mSellPositions[makerSymbol]; ok {
				fields["makerSize"] = (buyPosition.Volume - sellPosition.Volume) * mContractSizes[makerSymbol]
				if spread, ok := mtSpreads[makerSymbol]; ok {
					fields["makerValue"] = (buyPosition.Volume - sellPosition.Volume) * mContractSizes[makerSymbol] * spread.MakerDepth.MakerBid
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
			fields["makerFundingRate"] = fr.FundingRate
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
		if quantile, ok := mtQuantiles[makerSymbol]; ok {
			fields["quantileShortBot"] = quantile.ShortBot
			fields["quantileShortTop"] = quantile.ShortTop
			fields["quantileLongBot"] = quantile.LongBot
			fields["quantileLongTop"] = quantile.LongTop
			fields["quantileMid"] = quantile.Mid
			fields["quantileMaClose"] = quantile.MaClose
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
			logger.Debugf("new buyPosition point error %v", err)
		} else {
			go mtInfluxWriter.Push(pt)
		}
	}
}

func handleExternalInfluxSave() {
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
				logger.Debugf("Margin NewPoint error %v", err)
			} else {
				go mtExternalInfluxWriter.Push(pt)
			}
		}
	}
}

func watchReports(
	ctx context.Context,
	influxWriter *common.InfluxWriter,
	influxConfig InfluxConfig,
	depthReportCh chan common.DepthReport,
	spreadReportCh chan common.SpreadReport,
) {
	depthReports := make(map[string]common.DepthReport)
	spreadReports := make(map[string]common.SpreadReport)
	saveTimer := time.NewTimer(*influxConfig.SaveInterval)
	for {
		select {
		case <-ctx.Done():
			return
		case spreadReport := <-spreadReportCh:
			spreadReports[spreadReport.MakerSymbol] = spreadReport
			break
		case depthReport := <-depthReportCh:
			depthReports[depthReport.Exchange] = depthReport
			break
		case <-saveTimer.C:
			for exchange, report := range depthReports {
				fields := make(map[string]interface{})
				fields["avgLen"] = report.AvgLen
				fields["dropRatio"] = report.DropRatio
				fields["bias"] = report.Bias
				fields["decay"] = report.Decay
				fields["emaTimeDelta"] = report.EmaTimeDelta
				if len(fields) > 0 {
					pt, err := client.NewPoint(
						*influxConfig.Measurement,
						map[string]string{
							"exchange": exchange,
							"type":     "depth-report",
						},
						fields,
						time.Now().UTC(),
					)
					if err != nil {
						logger.Debugf("DepthReport NewPoint error %v", err)
					} else {
						select {
						case influxWriter.PushCh <- pt:
						default:
						}
					}
				}
			}
			for makerSymbol, report := range spreadReports {
				fields := make(map[string]interface{})
				fields["matchRatio"] = report.MatchRatio
				fields["maxAgeDiff"] = float64(report.MaxAgeDiff)
				fields["maxAge"] = float64(report.MaxAge)
				if len(fields) > 0 {
					pt, err := client.NewPoint(
						*influxConfig.Measurement,
						map[string]string{
							"makerSymbol": makerSymbol,
							"takerSymbol": report.TakerSymbol,
							"type":     "spread-report",
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
