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
	if xyInternalInfluxWriter == nil {
		return
	}
	if yCommissionAssetValue == nil || xCommissionAssetValue == nil {
		logger.Debugf("miss commission %v %v", yCommissionAssetValue == nil, xCommissionAssetValue == nil)
		return
	}
	totalUnHedgeValue := 0.0
	totalRiskExposure := 0.0
	totalXSymbolValue := 0.0
	totalYSymbolValue := 0.0
	yURPnl := 0.0
	xURPnl := 0.0
	totalURPnl := 0.0
	hasAllSymbols := true
	xTurnoverVolume := 0.0
	yTurnoverVolume := 0.0
	x30DayVolume := 0.0
	y30DayVolume := 0.0
	xTotalBidValue := 0.0
	xTotalAskValue := 0.0
	yTotalBidValue := 0.0
	yTotalAskValue := 0.0

	totalSuccessCount := 0.0
	totalSpreadSlippage := 0.0
	totalXSlippage := 0.0
	totalYSlippage := 0.0
	totalSuccessRatioCount := 0.0
	totalXSlippageWeight := 0.0
	totalYSlippageWeight := 0.0
	totalSpreadSlippageWeight := 0.0
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
			strat.xMidPrice > 0 &&
			strat.yMidPrice > 0 {

			if strat.XSlippageTM.Len() > 0 {
				fields["xSlippage"] = strat.XSlippageTM.Mean
				totalXSlippage += strat.XSlippageTM.Mean * strat.XSlippageTM.Weight
				totalXSlippageWeight += strat.XSlippageTM.Weight
			}

			fields["ySlippageFactor"] = strat.ySlippageFactor

			if strat.YSlippageTM.Len() > 0 {
				fields["ySlippage"] = strat.YSlippageTM.Mean
				totalYSlippage += strat.YSlippageTM.Mean * strat.YSlippageTM.Weight
				totalYSlippageWeight += strat.YSlippageTM.Weight
			}

			if strat.XYSpreadSlippageTM.Len() > 0 {
				fields["xySpreadSlippage"] = strat.XYSpreadSlippageTM.Mean
				totalSpreadSlippage += strat.XYSpreadSlippageTM.Mean * strat.XYSpreadSlippageTM.Weight
				totalSpreadSlippageWeight += strat.XYSpreadSlippageTM.Weight
			}

			if strat.XYSuccessRatioTM.Len() > 0 {
				fields["xySuccessRatio"] = strat.XYSuccessRatioTM.Mean
				totalSuccessCount += strat.XYSuccessRatioTM.Mean * float64(strat.XYSuccessRatioTM.Len())
				totalSuccessRatioCount += float64(strat.XYSuccessRatioTM.Len())
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
			ySize := strat.yPosition.GetSize() * strat.yMultiplier
			xValue := 0.0
			yValue := 0.0
			if strat.isXSpot || strat.xPosition.GetPrice() == 0 {
				xValue = xSize * strat.xMidPrice
			} else {
				xValue = xSize * strat.xPosition.GetPrice()
			}
			if strat.isYSpot || strat.yPosition.GetPrice() == 0 {
				yValue = ySize * strat.yMidPrice
			} else {
				yValue = ySize * strat.yPosition.GetPrice()
			}
			xAbsValue := math.Abs(xValue)
			yAbsValue := math.Abs(yValue)
			xyMidPrice := (strat.xMidPrice + strat.yMidPrice) * 0.5

			riskExposure := (xSize + ySize) * xyMidPrice
			totalRiskExposure += riskExposure
			unHedgeValue := math.Abs(riskExposure)
			totalUnHedgeValue += unHedgeValue
			totalXSymbolValue += xAbsValue
			totalYSymbolValue += yAbsValue

			xTurnoverVolume += strat.XTurnoverVolume.Sum
			yTurnoverVolume += strat.YTurnoverVolume.Sum
			x30DayVolume += strat.X30DayVolume.Sum
			y30DayVolume += strat.Y30DayVolume.Sum

			fields["unHedgeValue"] = unHedgeValue
			fields["riskExposure"] = riskExposure
			fields["xSize"] = xSize
			fields["xAbsValue"] = xAbsValue
			fields["xValue"] = xValue
			fields["ySize"] = ySize
			fields["yAbsValue"] = yAbsValue
			fields["yValue"] = yValue
			fields["xyValue"] = xValue + yValue
			totalURPnl += xValue + yValue

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
			if strat.yPosition.GetPrice() != 0 {
				yURPnl += yValue * (strat.yMidPrice - strat.yPosition.GetPrice()) / strat.yPosition.GetPrice()
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
				xSymbol, ySymbol, strat.xPosition != nil, strat.yPosition != nil, strat.xMidPrice, strat.yMidPrice,
			)
			hasAllSymbols = false
		}

		if strat.Stats.Ready {
			fields["statsReady"] = 1.0
		} else {
			fields["statsReady"] = 0.0
		}

		//stats不管策略状态，都需要保存
		fields["statsXTimeDeltaBot"] = strat.Stats.XEventTimeDeltaBot.Seconds()
		fields["statsXTimeDeltaMid"] = strat.Stats.XEventTimeDeltaMid.Seconds()
		fields["statsXTimeDeltaTop"] = strat.Stats.XEventTimeDeltaTop.Seconds()

		fields["statsYTimeDeltaBot"] = strat.Stats.YEventTimeDeltaBot.Seconds()
		fields["statsYTimeDeltaMid"] = strat.Stats.YEventTimeDeltaMid.Seconds()
		fields["statsYTimeDeltaTop"] = strat.Stats.YEventTimeDeltaTop.Seconds()

		fields["statsXYTimeDeltaBot"] = strat.Stats.XYEventTimeDeltaBot.Seconds()
		fields["statsXYTimeDeltaMid"] = strat.Stats.XYEventTimeDeltaMid.Seconds()
		fields["statsXYTimeDeltaTop"] = strat.Stats.XYEventTimeDeltaTop.Seconds()

		fields["statsXParseTimeDeltaMid"] = strat.Stats.XParseTimeDeltaMid.Seconds()
		fields["statsYParseTimeDeltaMid"] = strat.Stats.YParseTimeDeltaMid.Seconds()

		if strat.Stats.XMiddlePrice > 0 {
			fields["statsXMiddlePrice"] = strat.Stats.XMiddlePrice
		}
		if strat.Stats.YMiddlePrice > 0 {
			fields["statsYMiddlePrice"] = strat.Stats.YMiddlePrice
		}
		if strat.tickerCount > 0 && strat.tickerMatchCount > 0 {
			fields["tickerMatchRatio"] = float64(strat.tickerMatchCount) / float64(strat.tickerCount)
		}

		fields["statsSpreadEnterOffset"] = strat.Stats.SpreadEnterOffset
		fields["statsSpreadLeaveOffset"] = strat.Stats.SpreadLeaveOffset
		fields["statsSpreadLongEnterBot"] = strat.Stats.SpreadLongEnterBot
		fields["statsSpreadLongLeaveTop"] = strat.Stats.SpreadLongLeaveTop
		fields["statsSpreadShortEnterTop"] = strat.Stats.SpreadShortEnterTop
		fields["statsSpreadShortLeaveBot"] = strat.Stats.SpreadShortLeaveBot
		fields["statsSpreadMiddle"] = strat.Stats.SpreadMiddle

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
		fields["totalRiskExposure"] = totalRiskExposure
		fields["totalBalance"] = totalBalance
		fields["xCommissionAssetValue"] = *xCommissionAssetValue
		fields["yCommissionAssetValue"] = *yCommissionAssetValue
		fields["yBalance"] = yBalance
		fields["xBalance"] = xBalance
		fields["yAvailable"] = yAccount.GetFree()
		fields["xAvailable"] = xAccount.GetFree()
		fields["xTotalBidValue"] = xTotalBidValue
		fields["xTotalAskValue"] = xTotalAskValue
		fields["yTotalBidValue"] = yTotalBidValue
		fields["yTotalAskValue"] = yTotalAskValue
		if totalSuccessRatioCount != 0 {
			fields["totalSuccessCount"] = int(totalSuccessCount)
			fields["totalFailureCount"] = int(totalSuccessRatioCount - totalSuccessCount)
			fields["totalSuccessRatio"] = totalSuccessCount / totalSuccessRatioCount
		}
		if totalXSlippageWeight != 0 {
			fields["meanXSlippage"] = totalXSlippage / totalXSlippageWeight
		}
		if totalYSlippageWeight != 0 {
			fields["meanYSlippage"] = totalYSlippage / totalYSlippageWeight
		}
		if totalSpreadSlippageWeight != 0 {
			fields["meanSpreadSlippage"] = totalSpreadSlippage / totalSpreadSlippageWeight
		}
		fields["xURPnl"] = xURPnl
		fields["yURPnl"] = yURPnl
		fields["x30DayVolume"] = x30DayVolume
		fields["y30DayVolume"] = y30DayVolume
		if totalBalance != 0 {
			fields["xyTurnover"] = (xTurnoverVolume + yTurnoverVolume) / totalBalance
		}
		if xBalance != 0 {
			fields["xTurnover"] = xTurnoverVolume / xBalance
		}
		if yBalance != 0 {
			fields["yTurnover"] = yTurnoverVolume / yBalance
		}
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
			err = xyInternalInfluxWriter.PushPoint(pt)
			if err != nil {
				logger.Debugf("xyInfluxWriter.PushPoint error %v", err)
			}
		}
		if xyExternalInfluxWriter != nil &&
			time.Now().Sub(*lastExternalSaveTime) > xyConfig.ExternalInflux.SaveInterval {
			*lastExternalSaveTime = time.Now()
			fields = make(map[string]interface{})
			fields["netWorth"] = netWorth
			fields["x30DayVolume"] = x30DayVolume
			fields["y30DayVolume"] = y30DayVolume
			fields["yBalance"] = yBalance
			fields["xBalance"] = xBalance
			if xBalance != 0 {
				fields["xTurnover"] = xTurnoverVolume / xBalance
			}
			if yBalance != 0 {
				fields["yTurnover"] = yTurnoverVolume / yBalance
			}
			if totalBalance != 0 {
				fields["xyTurnover"] = (xTurnoverVolume + yTurnoverVolume) / totalBalance
			}
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
