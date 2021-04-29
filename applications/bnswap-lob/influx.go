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

	if swapAccount != nil &&
		swapAccount.MarginBalance != nil {
		totalBalance := *swapAccount.MarginBalance
		netWorth := totalBalance / *mtConfig.StartValue
		fields := make(map[string]interface{})
		fields["totalBalance"] = totalBalance
		fields["takerBalance"] = *swapAccount.MarginBalance
		fields["netWorth"] = netWorth
		fields["startValue"] = *mtConfig.StartValue
		fields["netWorth"] = netWorth
		if swapAccount.AvailableBalance != nil {
			fields["takerAvailable"] = *swapAccount.AvailableBalance
		}
		if swapAccount.UnrealizedProfit != nil {
			fields["takerUnrealizedProfit"] = *swapAccount.UnrealizedProfit
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
			logger.Debugf("client.NewPoint() error %v", err)
		} else {
			err = swapInternalInfluxWriter.PushPoint(pt)
			if err != nil {
				logger.Debugf("swapInternalInfluxWriter.PushPoint(pt) error %v", err)
			}
		}
	}

	for _, symbol := range swapSymbols {
		fields := make(map[string]interface{})
		if takerPosition, ok := swapPositions[symbol]; ok {
			fields["takerSize"] = takerPosition.PositionAmt
			if walkedDepth, ok := swapWalkedDepths[symbol]; ok {
				fields["takerValue"] = walkedDepth.MidPrice * takerPosition.PositionAmt
			}
		}
		if walkedDepth, ok := swapWalkedDepths[symbol]; ok {
			fields["swapMidPrice"] = walkedDepth.MidPrice
			fields["swapMircoPrice"] = walkedDepth.MircoPrice
			fields["swapAskPrice"] = walkedDepth.AskPrice
			fields["swapBidPrice"] = walkedDepth.BidPrice
			fields["swapAskSize"] = walkedDepth.AskSize
			fields["swapBidSize"] = walkedDepth.BidSize
			fields["swapAskBidRatio"] = walkedDepth.AskBidRatio
			fields["swapBidAskRatio"] = walkedDepth.BidAskRatio
			fields["swapEmaBidAskRatio"] = walkedDepth.EmaBidAskRatio
			fields["swapEmaAskBidRatio"] = walkedDepth.EmaAskBidRatio
		}
		if walkedDepth, ok := spotWalkedDepths[symbol]; ok {
			fields["spotMidPrice"] = walkedDepth.MidPrice
			fields["spotMircoPrice"] = walkedDepth.MircoPrice
			fields["spotAskPrice"] = walkedDepth.AskPrice
			fields["spotBidPrice"] = walkedDepth.BidPrice
			fields["spotAskSize"] = walkedDepth.AskSize
			fields["spotBidSize"] = walkedDepth.BidSize
			fields["spotAskBidRatio"] = walkedDepth.AskBidRatio
			fields["spotBidAskRatio"] = walkedDepth.BidAskRatio
			fields["spotEmaBidAskRatio"] = walkedDepth.EmaBidAskRatio
			fields["spotEmaAskBidRatio"] = walkedDepth.EmaAskBidRatio
		}
		if realisedSpread, ok := swapRealisedSpread[symbol]; ok {
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
				"symbol": symbol,
				"type":   "symbol",
			},
			fields,
			time.Now().UTC(),
		)
		if err != nil {
			logger.Debugf("client.NewPoint() error %v", err)
		} else {
			err = swapInternalInfluxWriter.PushPoint(pt)
			if err != nil {
				logger.Debugf("swapInternalInfluxWriter.PushPoint(pt) error %v", err)
			}
		}
	}
}

func handleExternalInfluxSave() {

	if time.Now().Sub(mtGlobalSilent) < 0 {
		return
	}

	if swapAccount != nil &&
		swapAccount.MarginBalance != nil {
		totalBalance := *swapAccount.MarginBalance
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
				logger.Debugf("client.NewPoint() error %v", err)
			} else {
				err = swapExternalInfluxWriter.PushPoint(pt)
				if err != nil {
					logger.Debugf(" swapExternalInfluxWriter.PushPoint(pt) error %v", err)
				}
			}
		}
	}
}

func swapReportsSaveLoop(
	ctx context.Context,
	influxWriter *common.InfluxWriter,
	influxConfig InfluxConfig,
	depthReportCh chan DepthReport,
) {
	depthReports := make(map[string]DepthReport)
	saveTimer := time.NewTimer(*influxConfig.SaveInterval)
	for {
		select {
		case <-ctx.Done():
			return
		case spreadReport := <-depthReportCh:
			depthReports[spreadReport.Symbol] = spreadReport
			break
		case <-saveTimer.C:
			for _, report := range depthReports {
				fields := make(map[string]interface{})
				fields["filterRatio"] = report.FilterRatio
				fields["timeDeltaEma"] = report.TimeDeltaEma
				fields["timeDelta"] = report.TimeDelta
				fields["msgAvgLen"] = report.MsgAvgLen
				if len(fields) > 0 {
					pt, err := client.NewPoint(
						*influxConfig.Measurement,
						map[string]string{
							"symbol": report.Symbol,
							"type":   "swap-report",
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

func spotReportsSaveLoop(
	ctx context.Context,
	influxWriter *common.InfluxWriter,
	influxConfig InfluxConfig,
	depthReportCh chan DepthReport,
) {
	depthReports := make(map[string]DepthReport)
	saveTimer := time.NewTimer(*influxConfig.SaveInterval)
	for {
		select {
		case <-ctx.Done():
			return
		case spreadReport := <-depthReportCh:
			depthReports[spreadReport.Symbol] = spreadReport
			break
		case <-saveTimer.C:
			for _, report := range depthReports {
				fields := make(map[string]interface{})
				fields["filterRatio"] = report.FilterRatio
				fields["timeDeltaEma"] = report.TimeDeltaEma
				fields["timeDelta"] = report.TimeDelta
				fields["msgAvgLen"] = report.MsgAvgLen
				if len(fields) > 0 {
					pt, err := client.NewPoint(
						*influxConfig.Measurement,
						map[string]string{
							"symbol": report.Symbol,
							"type":   "spot-report",
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
