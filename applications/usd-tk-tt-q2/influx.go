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
	stratMap map[string]*XYStrategy,
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
		strat, ok := stratMap[xSymbol]
		if !ok {
			hasAllSymbols = false
			continue
		}
		ySymbol := strat.ySymbol
		fields := make(map[string]interface{})
		if strat.xPosition != nil &&
			strat.yPosition != nil &&
			strat.spreadReady &&
			strat.xyMidPrice != 0 {

			unHedgeValue := math.Abs(strat.xSize+strat.ySize) * strat.xyMidPrice
			totalUnHedgeValue += unHedgeValue
			totalXSymbolValue += strat.xAbsValue
			totalYSymbolValue += strat.yAbsValue

			xTradeVolume += strat.xTimedPositionChange.Sum()
			yTradeVolume += strat.yTimedPositionChange.Sum()

			fields["unHedgeValue"] = unHedgeValue
			fields["xSize"] = strat.xSize
			fields["xAbsValue"] = strat.xAbsValue
			fields["xValue"] = strat.xValue
			fields["ySize"] = strat.ySize
			fields["yAbsValue"] = strat.yAbsValue
			fields["yValue"] = strat.yValue
			fields["xyValue"] = strat.xValue + strat.yValue
			totalURPnl += strat.xValue + strat.yValue
			fields["thresholdShortTop"] = strat.thresholdShortTop
			fields["thresholdShortBot"] = strat.thresholdShortBot
			fields["thresholdLongBot"] = strat.thresholdLongBot
			fields["thresholdLongTop"] = strat.thresholdLongTop
			fields["enterTarget"] = strat.enterTarget
			fields["enterStep"] = strat.enterStep
			fields["enterValue"] = strat.enterValue
			fields["offsetFactor"] = strat.offsetFactor
			if strat.tdSpreadMiddle != 0 {
				fields["thresholdLongTop"] = strat.thresholdLongTop
				fields["thresholdShortTop"] = strat.thresholdShortTop
				fields["thresholdShortBot"] = strat.thresholdShortBot
				fields["thresholdLongBot"] = strat.thresholdLongBot
				fields["tdSpreadMiddle"] = strat.tdSpreadMiddle
				fields["tdSpreadEnterOffset"] = strat.tdSpreadEnterOffset
				fields["tdSpreadExitOffset"] = strat.tdSpreadExitOffset
			}
			if strat.xFundingRateFactor != nil {
				fields["xFundingRateFactor"] = *strat.xFundingRateFactor
			}
			if strat.yFundingRateFactor != nil {
				fields["yFundingRateFactor"] = *strat.yFundingRateFactor
			}

			if strat.xPosition.GetPrice() != 0 {
				xURPnl += strat.xValue * (strat.xMidPrice - strat.xPosition.GetPrice()) / strat.xPosition.GetPrice()
			}
			if strat.yPosition.GetPrice() != 0 {
				yURPnl += strat.yValue * (strat.yMidPrice - strat.yPosition.GetPrice()) / strat.yPosition.GetPrice()
			}

			fields["spreadTimeDelta"] = strat.spreadEventTime.Sub(strat.spreadTickerTime).Seconds()
			fields["spreadLastLong"] = strat.spreadLastLong
			fields["spreadLastShort"] = strat.spreadLastShort
			fields["spreadMedianLong"] = strat.spreadMedianLong
			fields["spreadMedianShort"] = strat.spreadMedianShort

			fields["xBidPrice"] = strat.xTicker.GetBidPrice()
			fields["xAskPrice"] = strat.xTicker.GetAskPrice()
			fields["xMidPrice"] = strat.xMidPrice

			fields["yBidPrice"] = strat.yTicker.GetBidPrice()
			fields["yAskPrice"] = strat.yTicker.GetAskPrice()
			fields["yMidPrice"] = strat.yMidPrice

		} else {
			logger.Debugf(
				"%s %s save failed, okXPosition %v okYPosition %v okSpread %v xyMidPrice %v",
				xSymbol, ySymbol, strat.xPosition != nil, strat.yPosition != nil, strat.spreadReady, strat.xyMidPrice,
			)
			hasAllSymbols = false
		}


		//stats不管策略状态，都需要保存

		if strat.stats.Ready.True() {
			fields["statsReady"] = 1.0
			if strat.targetWeightUpdated.True() {
				fields["targetWeight"] = strat.targetWeight.Load()
			}
		}else{
			fields["statsReady"] = 0.0
		}

		fields["statsXTimeDeltaBot"] = strat.stats.XTimeDeltaBot.Load().Seconds()
		fields["statsXTimeDeltaMid"] = strat.stats.XTimeDeltaMid.Load().Seconds()
		fields["statsXTimeDeltaTop"] = strat.stats.XTimeDeltaTop.Load().Seconds()

		fields["statsYTimeDeltaBot"] = strat.stats.YTimeDeltaBot.Load().Seconds()
		fields["statsYTimeDeltaMid"] = strat.stats.YTimeDeltaMid.Load().Seconds()
		fields["statsYTimeDeltaTop"] = strat.stats.YTimeDeltaTop.Load().Seconds()

		fields["statsXYTimeDeltaBot"] = strat.stats.XYTimeDeltaBot.Load().Seconds()
		fields["statsXYTimeDeltaMid"] = strat.stats.XYTimeDeltaMid.Load().Seconds()
		fields["statsXYTimeDeltaTop"] = strat.stats.XYTimeDeltaTop.Load().Seconds()

		fields["statsXBidSize"] = strat.stats.XBidSize.Load()
		fields["statsXAskSize"] = strat.stats.XAskSize.Load()
		fields["statsYBidSize"] = strat.stats.YBidSize.Load()
		fields["statsYAskSize"] = strat.stats.YAskSize.Load()
		if strat.stats.XMiddlePrice.Load() > 0 {
			fields["statsXMiddlePrice"] = strat.stats.XMiddlePrice.Load()
		}
		if strat.stats.YMiddlePrice.Load() > 0 {
			fields["statsYMiddlePrice"] = strat.stats.YMiddlePrice.Load()
		}
		if strat.tickerCount > 0 && strat.tickerMatchCount > 0{
			fields["tickerMatchRatio"] = float64(strat.tickerMatchCount) / float64(strat.tickerCount)
		}

		fields["statsSpreadEnterOffset"] = strat.stats.SpreadEnterOffset.Load()
		fields["statsSpreadLeaveOffset"] = strat.stats.SpreadLeaveOffset.Load()
		fields["statsSpreadLongEnterBot"] = strat.stats.SpreadLongEnterBot.Load()
		fields["statsSpreadLongLeaveTop"] = strat.stats.SpreadLongLeaveTop.Load()
		fields["statsSpreadShortEnterTop"] = strat.stats.SpreadShortEnterTop.Load()
		fields["statsSpreadShortLeaveBot"] = strat.stats.SpreadShortLeaveBot.Load()
		fields["statsSpreadMiddle"] = strat.stats.SpreadMiddle.Load()

		if strat.xFundingRate != nil {
			fields["xFundingRate"] = strat.xFundingRate.GetFundingRate()
		}
		if strat.xFundingRateFactor != nil {
			fields["xFundingRateFactor"] = *strat.xFundingRateFactor
		}
		if strat.yFundingRate != nil {
			fields["yFundingRate"] = strat.yFundingRate.GetFundingRate()
		}
		if strat.yFundingRateFactor != nil {
			fields["yFundingRateFactor"] = *strat.yFundingRateFactor
		}
		if strat.xAdjustedFundingRate != nil {
			fields["xAdjustedFundingRate"] = *strat.xAdjustedFundingRate
		}
		if strat.yAdjustedFundingRate != nil {
			fields["yAdjustedFundingRate"] = *strat.yAdjustedFundingRate
		}
		if strat.xyFundingRate != nil {
			fields["xyFundingRate"] = *strat.xyFundingRate
		}
		if strat.realisedSpread != nil {
			fields["realisedSpread"] = *strat.realisedSpread
		}
		if strat.adjustedRealisedSpread != nil {
			fields["adjustedRealisedSpread"] = *strat.adjustedRealisedSpread
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
