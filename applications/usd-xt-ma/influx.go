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
	xAccount common.Balance,
	xExchange, yExchange common.UsdExchange,
	stratMap map[string]*XYStrategy,
	xSymbols []string,
	xSystemStatus, ySystemStatus common.SystemStatus,
	xyConfig *Config,
	xCommissionAssetValue *float64,
	xyInternalInfluxWriter, xyExternalInfluxWriter *common.InfluxWriter,
	lastExternalSaveTime *time.Time,
) {
	if xCommissionAssetValue == nil {
		logger.Debugf("miss x commission %v", xCommissionAssetValue == nil)
		return
	}
	totalUnHedgeValue := 0.0
	totalRiskExposure := 0.0
	totalXSymbolValue := 0.0
	xURPnl := 0.0
	hasAllSymbols := true
	xTradeVolume := 0.0
	xTotalBidValue := 0.0
	xTotalAskValue := 0.0
	yTotalBidValue := 0.0
	yTotalAskValue := 0.0

	totalXSlippage := 0.0
	totalXSlippageCount := 0.0
	for _, xSymbol := range xSymbols {
		strat, ok := stratMap[xSymbol]
		if !ok {
			hasAllSymbols = false
			continue
		}
		ySymbol := strat.ySymbol
		fields := make(map[string]interface{})
		if strat.xPosition != nil &&
			strat.xMidPrice > 0 &&
			strat.yMidPrice > 0 {

			if strat.xSlippageTM.Len() > 0 {
				fields["xSlippage"] = strat.xSlippageTM.Mean
				totalXSlippage += strat.xSlippageTM.Mean * float64(strat.xSlippageTM.Len())
				totalXSlippageCount += float64(strat.xSlippageTM.Len())
			}

			fields["xBidPrice"] = strat.xTicker.GetBidPrice()
			fields["xAskPrice"] = strat.xTicker.GetAskPrice()
			fields["xMidPrice"] = strat.xMidPrice
			fields["xBidSize"] = strat.xTicker.GetBidSize()
			fields["xAskSize"] = strat.xTicker.GetAskSize()

			fields["yBidPrice"] = strat.yTicker.GetBidPrice()
			fields["yAskPrice"] = strat.yTicker.GetAskPrice()
			fields["yMidPrice"] = strat.yMidPrice
			fields["yBidSize"] = strat.yTicker.GetBidSize()
			fields["yAskSize"] = strat.yTicker.GetAskSize()

			xTotalBidValue += strat.xTicker.GetBidSize() * strat.xMultiplier * strat.xTicker.GetBidPrice()
			xTotalAskValue += strat.xTicker.GetAskSize() * strat.xMultiplier * strat.xTicker.GetAskPrice()
			yTotalBidValue += strat.yTicker.GetBidSize() * strat.yMultiplier * strat.yTicker.GetBidPrice()
			yTotalAskValue += strat.yTicker.GetAskSize() * strat.yMultiplier * strat.yTicker.GetAskPrice()

			xSize := strat.xPosition.GetSize() * strat.xMultiplier
			xValue := 0.0
			if strat.xPosition.GetPrice() == 0 {
				xValue = xSize * strat.xMidPrice
			} else {
				xValue = xSize * strat.xPosition.GetPrice()
			}
			xAbsValue := math.Abs(xValue)
			xyMidPrice := (strat.xMidPrice + strat.yMidPrice) * 0.5

			riskExposure := xSize * xyMidPrice
			totalRiskExposure += riskExposure
			unHedgeValue := math.Abs(riskExposure)
			totalUnHedgeValue += unHedgeValue
			totalXSymbolValue += xAbsValue

			xTradeVolume += strat.xTimedPositionChange.Sum()

			fields["unHedgeValue"] = unHedgeValue
			fields["riskExposure"] = riskExposure
			fields["xSize"] = xSize
			fields["xAbsValue"] = xAbsValue
			fields["xValue"] = xValue

			//如果已经计算了tdSpreadMiddle, 这些数据都该有了
			if strat.tdSpreadMiddle != 0 {
				fields["enterTarget"] = strat.enterTarget
				fields["enterStep"] = strat.enterStep
				fields["enterValue"] = strat.enterValue
				fields["offsetFactor"] = strat.offsetFactor
				fields["thresholdLongTop"] = strat.thresholdLongTop
				fields["thresholdShortTop"] = strat.thresholdShortTop
				fields["thresholdShortBot"] = strat.thresholdShortBot
				fields["thresholdLongBot"] = strat.thresholdLongBot
				fields["tdSpreadMiddle"] = strat.tdSpreadMiddle
				fields["tdSpreadEnterOffset"] = strat.tdSpreadEnterOffset
				fields["tdSpreadExitOffset"] = strat.tdSpreadExitOffset
			}
			if strat.xPosition.GetPrice() != 0 {
				xURPnl += xValue * (strat.xMidPrice - strat.xPosition.GetPrice()) / strat.xPosition.GetPrice()
			}
			fields["xTickerTimeDeltaF"] = strat.xTickerTimeDelta.Seconds()
			fields["yTickerTimeDeltaF"] = strat.yTickerTimeDelta.Seconds()
			fields["xyTickerTimeDeltaF"] = strat.xyTickerTimeDelta.Seconds()
			if strat.spreadReady {
				fields["spreadTimeDelta"] = strat.spreadEventTime.Sub(strat.spreadTickerTime).Seconds()
				fields["spreadLastLong"] = strat.spreadLastLong
				fields["spreadLastShort"] = strat.spreadLastShort
				fields["spreadMedianLong"] = strat.spreadMedianLong
				fields["spreadMedianShort"] = strat.spreadMedianShort
			}

			if strat.spreadReady {
				fields["spreadReady"] = 1.0
			} else {
				fields["spreadReady"] = 0.0
			}

		} else {
			logger.Debugf(
				"%s %s save failed, okXPosition %v okYPosition %v xMidPrice %v yMidPrice %v",
				xSymbol, ySymbol, strat.xPosition != nil, strat.xMidPrice, strat.yMidPrice,
			)
			hasAllSymbols = false
		}

		if strat.stats.Ready {
			fields["statsReady"] = 1.0
		} else {
			fields["statsReady"] = 0.0
		}

		//stats不管策略状态，都需要保存
		fields["statsXTimeDeltaBot"] = strat.stats.XEventTimeDeltaBot.Seconds()
		fields["statsXTimeDeltaMid"] = strat.stats.XEventTimeDeltaMid.Seconds()
		fields["statsXTimeDeltaTop"] = strat.stats.XEventTimeDeltaTop.Seconds()

		fields["statsYTimeDeltaBot"] = strat.stats.YEventTimeDeltaBot.Seconds()
		fields["statsYTimeDeltaMid"] = strat.stats.YEventTimeDeltaMid.Seconds()
		fields["statsYTimeDeltaTop"] = strat.stats.YEventTimeDeltaTop.Seconds()

		fields["statsXYTimeDeltaBot"] = strat.stats.XYEventTimeDeltaBot.Seconds()
		fields["statsXYTimeDeltaMid"] = strat.stats.XYEventTimeDeltaMid.Seconds()
		fields["statsXYTimeDeltaTop"] = strat.stats.XYEventTimeDeltaTop.Seconds()

		fields["statsXParseTimeDeltaMid"] = strat.stats.XParseTimeDeltaMid.Seconds()
		fields["statsYParseTimeDeltaMid"] = strat.stats.YParseTimeDeltaMid.Seconds()

		if strat.stats.XMiddlePrice > 0 {
			fields["statsXMiddlePrice"] = strat.stats.XMiddlePrice
		}
		if strat.stats.YMiddlePrice > 0 {
			fields["statsYMiddlePrice"] = strat.stats.YMiddlePrice
		}
		if strat.tickerCount > 0 && strat.tickerMatchCount > 0 {
			fields["tickerMatchRatio"] = float64(strat.tickerMatchCount) / float64(strat.tickerCount)
		}

		fields["statsSpreadEnterOffset"] = strat.stats.SpreadEnterOffset
		fields["statsSpreadLeaveOffset"] = strat.stats.SpreadLeaveOffset
		fields["statsSpreadLongEnterBot"] = strat.stats.SpreadLongEnterBot
		fields["statsSpreadLongLeaveTop"] = strat.stats.SpreadLongLeaveTop
		fields["statsSpreadShortEnterTop"] = strat.stats.SpreadShortEnterTop
		fields["statsSpreadShortLeaveBot"] = strat.stats.SpreadShortLeaveBot
		fields["statsSpreadMiddle"] = strat.stats.SpreadMiddle

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

		if strat.tickerCount > 0 {
			fields["tickerCount"] = strat.tickerCount
			fields["tickerMatchCount"] = strat.tickerMatchCount
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

	////每小时录一次
	//if time.Now().Truncate(time.Minute).Sub(time.Now().Truncate(time.Hour).Add(time.Minute*10)) == 0 {
	//	var m runtime.MemStats
	//	runtime.ReadMemStats(&m)
	//	fields := make(map[string]interface{})
	//	fields["alloc"] = int64(m.Alloc/1000/1000)
	//	fields["totalAlloc"] = int64(m.TotalAlloc/1000/1000)
	//	fields["sys"] = int64(m.Sys/1000/1000)
	//	fields["numGC"] = int64(m.NumGC)
	//	mm, err := memory.Get()
	//	if err != nil {
	//		logger.Debugf("get memory error %v", err)
	//	} else {
	//		fields["memoryTotal"] = int64(mm.Total/1000/1000)
	//		fields["memoryFree"] = int64(mm.Free/1000/1000)
	//		fields["memoryCached"] = int64(mm.Cached/1000/1000)
	//		fields["memoryUsed"] = int64(mm.Used/1000/1000)
	//	}
	//	pt, err := client.NewPoint(
	//		xyConfig.InternalInflux.Measurement,
	//		map[string]string{
	//			"type": "runtime",
	//		},
	//		fields,
	//		time.Now().UTC(),
	//	)
	//	if err != nil {
	//		logger.Debugf("client.NewPoint error %v", err)
	//	} else {
	//		err = xyInternalInfluxWriter.PushPoint(pt)
	//		if err != nil {
	//			logger.Debugf("xyInfluxWriter.PushPoint error %v", err)
	//		}
	//	}
	//}

	if xAccount != nil &&
		hasAllSymbols {
		xBalance := xAccount.GetBalance()
		if xExchange.IsSpot() {
			xBalance += totalXSymbolValue
		}
		netWorth := xBalance / xyConfig.StartValue
		fields := make(map[string]interface{})
		fields["totalUnHedgeValue"] = totalUnHedgeValue
		fields["totalRiskExposure"] = totalRiskExposure
		fields["xCommissionAssetValue"] = *xCommissionAssetValue
		fields["xBalance"] = xBalance
		fields["xAvailable"] = xAccount.GetFree()
		fields["xTotalBidValue"] = xTotalBidValue
		fields["xTotalAskValue"] = xTotalAskValue
		fields["yTotalBidValue"] = yTotalBidValue
		fields["yTotalAskValue"] = yTotalAskValue
		if totalXSlippageCount != 0 {
			fields["meanXSlippage"] = totalXSlippage / totalXSlippageCount
		}
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
