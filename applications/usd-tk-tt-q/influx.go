package main

import (
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/influx/client"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"strings"
	"time"
)

func handleSave(
	xAccount, yAccount common.Balance,
	xExchange, yExchange common.UsdExchange,
	strategiesMap map[string]*XYStrategy,
	xSymbols []string,
	xSystemStatus, ySystemStatus common.SystemStatus,
	xyConfig *Config,
	xCommissionAssetValue, yCommissionAssetValue *float64,
	xyInternalInfluxWriter, xyExternalInfluxWriter *common.InfluxWriter,
	lastExternalSaveTime *time.Time,
) {
	if yCommissionAssetValue == nil || xCommissionAssetValue == nil {
		logger.Debugf("miss commission %v %v", yCommissionAssetValue == nil, xCommissionAssetValue == nil)
		return
	}
	totalUnHedgeValue := 0.0
	totalXSymbolValue := 0.0
	totalYSymbolValue := 0.0
	yURPnl := 0.0
	xURPnl := 0.0
	totalURPnl := 0.0
	hasAllSymbols := true
	xTradeVolume := 0.0
	yTradeVolume := 0.0
	for _, xSymbol := range xSymbols {
		st, ok := strategiesMap[xSymbol]
		if !ok {
			hasAllSymbols = false
			continue
		}
		ySymbol := st.ySymbol
		fields := make(map[string]interface{})
		if st.xPosition != nil &&
			st.yPosition != nil &&
			st.spread != nil &&
			st.midPrice != 0 {

			unHedgeValue := math.Abs(st.xSize+st.ySize) * st.midPrice
			totalUnHedgeValue += unHedgeValue
			totalXSymbolValue += st.xAbsValue
			totalYSymbolValue += st.yAbsValue

			xTradeVolume += st.xTimedPositionChange.Sum()
			yTradeVolume += st.yTimedPositionChange.Sum()

			fields["unHedgeValue"] = unHedgeValue
			fields["xSize"] = st.xSize
			fields["xAbsValue"] = st.xAbsValue
			fields["xValue"] = st.xValue
			fields["ySize"] = st.ySize
			fields["yAbsValue"] = st.yAbsValue
			fields["yValue"] = st.yValue
			fields["xyValue"] = st.xValue + st.yValue
			totalURPnl += st.xValue + st.yValue
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
				fields["quantileEnterTop"] = st.quantileEnterTop
				fields["quantileEnterBot"] = st.quantileEnterBot
				fields["quantileExitTop"] = st.quantileExitTop
				fields["quantileExitBot"] = st.quantileExitBot
				fields["enterOffsetDelta"] = *st.enterOffset
				fields["exitOffsetDelta"] = *st.exitOffset
			}
			if st.fundingRateFactor != nil {
				fields["fundingRateFactor"] = *st.fundingRateFactor
			}

			if st.xPosition.GetPrice() != 0 {
				xURPnl += st.xValue * (st.xMidPrice - st.xPosition.GetPrice()) / st.xPosition.GetPrice()
			}
			if st.yPosition.GetPrice() != 0 {
				yURPnl += st.yValue * (st.yMidPrice - st.yPosition.GetPrice()) / st.yPosition.GetPrice()
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

		} else {
			logger.Debugf(
				"%s %s save failed, okXPosition %v okYPosition %v okSpread %v midPrice %v",
				xSymbol, ySymbol, st.xPosition != nil, st.yPosition != nil, st.spread != nil, st.midPrice,
			)
			hasAllSymbols = false
		}

		//只要有report， 存一份
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

	if yAccount != nil &&
		xAccount != nil &&
		hasAllSymbols {
		xBalance := xAccount.GetBalance()
		yBalance := yAccount.GetBalance()
		if xExchange.IsSpot() {
			xBalance += totalXSymbolValue
		}
		if yExchange.IsSpot() {
			yBalance += totalYSymbolValue
		}
		totalBalance := xBalance + yBalance + *xCommissionAssetValue + *yCommissionAssetValue
		netWorth := totalBalance / xyConfig.StartValue
		fields := make(map[string]interface{})
		fields["totalUnHedgeValue"] = totalUnHedgeValue
		fields["totalBalance"] = totalBalance
		fields["xCommissionAssetValue"] = *xCommissionAssetValue
		fields["yCommissionAssetValue"] = *yCommissionAssetValue
		fields["yBalance"] = yBalance
		fields["xBalance"] = xBalance
		fields["yAvailable"] = yAccount.GetFree()
		fields["xAvailable"] = xAccount.GetFree()
		fields["xURPnl"] = xURPnl
		fields["yURPnl"] = yURPnl
		fields["xyTurnover"] = (xTradeVolume + yTradeVolume) / totalBalance
		fields["xTurnover"] = xTradeVolume / xBalance
		fields["yTurnover"] = yTradeVolume / yBalance
		fields["xyURPnl"] = totalURPnl
		fields["netWorth"] = netWorth
		fields["startValue"] = xyConfig.StartValue
		fields["xPriceFactor"] = xExchange.GetPriceFactor()
		fields["yPriceFactor"] = yExchange.GetPriceFactor()
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
