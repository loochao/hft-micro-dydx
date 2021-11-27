package main

import (
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/influx/client"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/mackerelio/go-osstat/memory"
	"math"
	"runtime"
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
	cpuUsage *CpuUsage,
) {
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
	xTradeVolume := 0.0
	yTradeVolume := 0.0
	xTotalBidValue := 0.0
	xTotalAskValue := 0.0
	yTotalBidValue := 0.0
	yTotalAskValue := 0.0

	totalSuccessCount := 0.0
	totalXSlippage := 0.0
	totalYSlippage := 0.0
	totalSuccessRatioCount := 0.0
	totalXSlippageCount := 0.0
	totalYSlippageCount := 0.0
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

			if strat.xSlippageTM.Len() > 0 {
				fields["xSlippage"] = strat.xSlippageTM.Mean
				totalXSlippage += strat.xSlippageTM.Mean * float64(strat.xSlippageTM.Len())
				totalXSlippageCount += float64(strat.xSlippageTM.Len())
			}

			if strat.ySlippageTM.Len() > 0 {
				fields["ySlippage"] = strat.ySlippageTM.Mean
				totalYSlippage += strat.ySlippageTM.Mean * float64(strat.ySlippageTM.Len())
				totalYSlippageCount += float64(strat.ySlippageTM.Len())
			}

			if strat.xySuccessRatioTM.Len() > 0 {
				fields["xySuccessRatio"] = strat.xySuccessRatioTM.Mean
				totalSuccessCount += strat.xySuccessRatioTM.Mean * float64(strat.xySuccessRatioTM.Len())
				totalSuccessRatioCount += float64(strat.xySuccessRatioTM.Len())
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

			xTradeVolume += strat.xTimedPositionChange.Sum()
			yTradeVolume += strat.yTimedPositionChange.Sum()

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

		if strat.stats.Ready.True() {
			fields["statsReady"] = 1.0
		} else {
			fields["statsReady"] = 0.0
		}

		//stats不管策略状态，都需要保存
		if strat.targetWeightUpdated.True() {
			fields["targetWeight"] = strat.targetWeight.Load()
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
		fields["statsXBidOffset"] = strat.stats.XBidOffset.Load()
		fields["statsXAskOffset"] = strat.stats.XAskOffset.Load()
		fields["statsYBidOffset"] = strat.stats.YBidOffset.Load()
		fields["statsYAskOffset"] = strat.stats.YAskOffset.Load()
		if strat.stats.XMiddlePrice.Load() > 0 {
			fields["statsXMiddlePrice"] = strat.stats.XMiddlePrice.Load()
		}
		if strat.stats.YMiddlePrice.Load() > 0 {
			fields["statsYMiddlePrice"] = strat.stats.YMiddlePrice.Load()
		}
		if strat.tickerCount > 0 && strat.tickerMatchCount > 0 {
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

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fields := make(map[string]interface{})
	fields["alloc"] = m.Alloc
	fields["totalAlloc"] = m.TotalAlloc
	fields["sys"] = m.Sys
	fields["numGC"] = m.NumGC
	fields["cpuSystem"] = cpuUsage.System
	fields["cpuUser"] = cpuUsage.User
	fields["cpuIdle"] = cpuUsage.Idle
	mm, err := memory.Get()
	if err != nil {
		logger.Debugf("get memory error %v", err)
	} else {
		fields["memoryTotal"] = mm.Total
		fields["memoryFree"] = mm.Free
		fields["memoryCached"] = mm.Cached
		fields["memoryUsed"] = mm.Used
	}

	pt, err := client.NewPoint(
		xyConfig.InternalInflux.Measurement,
		map[string]string{
			"type": "runtime",
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
		if totalXSlippageCount != 0 {
			fields["meanXSlippage"] = totalXSlippage / totalXSlippageCount
		}
		if totalYSlippageCount != 0 {
			fields["meanYSlippage"] = totalYSlippage / totalYSlippageCount
		}
		fields["xURPnl"] = xURPnl
		fields["yURPnl"] = yURPnl
		if totalBalance != 0 {
			fields["xyTurnover"] = (xTradeVolume + yTradeVolume) / totalBalance
		}
		if xBalance != 0 {
			fields["xTurnover"] = xTradeVolume / xBalance
		}
		if yBalance != 0 {
			fields["yTurnover"] = yTradeVolume / yBalance
		}
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
