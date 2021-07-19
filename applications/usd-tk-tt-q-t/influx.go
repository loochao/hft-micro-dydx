package main

import (
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/influx/client"
	"github.com/geometrybase/hft-micro/logger"
	"strings"
	"time"
)

func handleSave(
	xAccount common.Balance,
	xExchange, yExchange common.UsdExchange,
	strategiesMap map[string]*XYStrategy,
	xSymbols []string,
	xSystemStatus, ySystemStatus common.SystemStatus,
	xyConfig *Config,
	xCommissionAssetValue *float64,
	xyInternalInfluxWriter, xyExternalInfluxWriter *common.InfluxWriter,
	lastExternalSaveTime *time.Time,
) {
	if xCommissionAssetValue == nil {
		return
	}
	totalXSymbolValue := 0.0
	xURPnl := 0.0
	hasAllSymbols := true
	xTradeVolume := 0.0
	for _, xSymbol := range xSymbols {
		st, ok := strategiesMap[xSymbol]
		if !ok {
			hasAllSymbols = false
			continue
		}
		ySymbol := st.ySymbol
		fields := make(map[string]interface{})
		if st.xPosition != nil &&
			st.spread != nil &&
			st.midPrice != 0 {

			totalXSymbolValue += st.xAbsValue

			xTradeVolume += st.xTimedPositionChange.Sum()

			//fields["xPosEventTime"] = st.xPosition.GetEventTime().UnixNano()
			//fields["xPosParseTime"] = st.xPosition.GetParseTime().UnixNano()
			//fields["yPosEventTime"] = st.yPosition.GetEventTime().UnixNano()
			//fields["yPosParseTime"] = st.yPosition.GetParseTime().UnixNano()
			fields["xSize"] = st.xSize
			fields["xAbsValue"] = st.xAbsValue
			fields["xValue"] = st.xValue
			fields["shortTop"] = st.shortTop
			fields["shortBot"] = st.shortBot
			fields["longBot"] = st.longBot
			fields["longTop"] = st.longTop
			fields["enterTarget"] = st.enterTarget
			fields["enterStep"] = st.enterStep
			fields["enterValue"] = st.enterValue
			fields["offsetFactor"] = st.offsetFactor
			if st.quantileMiddle != nil {
				fields["quantileMiddle"] = *st.quantileMiddle
			}

			if st.xPosition.GetPrice() != 0 {
				xURPnl += st.xValue * (st.xMidPrice - st.xPosition.GetPrice()) / st.xPosition.GetPrice()
				fields["xURPnlBySymbol"] = st.xValue * (st.xMidPrice - st.xPosition.GetPrice()) / st.xPosition.GetPrice()
			}

			fields["spreadTimeDelta"] = st.spread.ParseTime.Sub(st.spread.EventTime).Seconds()

			fields["spreadShortLastEnter"] = st.spread.ShortLastEnter
			fields["spreadShortLastLeave"] = st.spread.ShortLastLeave
			fields["spreadShortMedianEnter"] = st.spread.ShortMedianEnter
			fields["spreadShortMedianLeave"] = st.spread.ShortMedianLeave

			fields["spreadLongLastEnter"] = st.spread.LongLastEnter
			fields["spreadLongLastLeave"] = st.spread.LongLastLeave
			fields["spreadLongMedianEnter"] = st.spread.LongMedianEnter
			fields["spreadLongMedianLeave"] = st.spread.LongMedianLeave

			fields["xBidPrice"] = st.xTicker.GetBidPrice()
			fields["xAskPrice"] = st.xTicker.GetAskPrice()
			fields["xMidPrice"] = st.xMidPrice

			fields["yBidPrice"] = st.yTicker.GetBidPrice()
			fields["yAskPrice"] = st.yTicker.GetAskPrice()
			fields["yMidPrice"] = st.yMidPrice

			if st.spreadReport != nil {
				fields["matchRatio"] = st.spreadReport.MatchRatio
				fields["xTimeDeltaEma"] = st.spreadReport.XTimeDeltaEma
				fields["yTimeDeltaEma"] = st.spreadReport.YTimeDeltaEma
				fields["xTimeDelta"] = st.spreadReport.XTimeDelta
				fields["yTimeDelta"] = st.spreadReport.YTimeDelta
				fields["xTickerFilterRatio"] = st.spreadReport.XTickerFilterRatio
				fields["yTickerFilterRatio"] = st.spreadReport.XTickerFilterRatio
				fields["xExpireRatio"] = st.spreadReport.XExpireRatio
				fields["yExpireRatio"] = st.spreadReport.YExpireRatio
			}

		} else {
			logger.Debugf(
				"%s %s save failed, okXPosition %v okSpread %v midPrice %v",
				xSymbol, ySymbol, st.xPosition != nil, st.spread != nil, st.midPrice,
			)
			hasAllSymbols = false
		}
		if st.xFundingRate != nil {
			fields["xFundingRate"] = st.xFundingRate.GetFundingRate()
		}
		if st.yFundingRate != nil {
			fields["yFundingRate"] = st.yFundingRate.GetFundingRate()
		}
		if st.xyFundingRate != nil {
			fields["xyFundingRate"] = *st.xyFundingRate
		}
		if st.realisedSpread != nil {
			fields["realisedSpread"] = *st.realisedSpread
		}
		if st.adjustedRealisedSpread != nil {
			fields["adjustedRealisedSpread"] = *st.adjustedRealisedSpread
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
			err = xyInternalInfluxWriter.PushPoint(pt)
			if err != nil {
				logger.Debugf("xyInfluxWriter.PushPoint error %v", err)
			}
		}
	}

	if xAccount != nil &&
		hasAllSymbols {
		xBalance := xAccount.GetBalance()
		if xExchange.IsSpot() {
			xBalance += totalXSymbolValue
		}
		totalBalance := xBalance  + *xCommissionAssetValue
		netWorth := totalBalance / xyConfig.StartValue
		fields := make(map[string]interface{})
		fields["totalBalance"] = totalBalance
		fields["xCommissionAssetValue"] = *xCommissionAssetValue
		fields["xBalance"] = xBalance
		fields["xAvailable"] = xAccount.GetFree()
		fields["xURPnl"] = xURPnl
		if xBalance != 0 {
			fields["xTurnover"] = xTradeVolume / xBalance
		}
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
			err = xyInternalInfluxWriter.PushPoint(pt)
			if err != nil {
				logger.Debugf("xyInfluxWriter.PushPoint error %v", err)
			}
		}
		if time.Now().Sub(*lastExternalSaveTime) > xyConfig.ExternalInflux.SaveInterval {
			*lastExternalSaveTime = time.Now()
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
}
