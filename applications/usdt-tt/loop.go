package main

import (
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"time"
)

func hedgeYSymbol(ySymbol, xSymbol string) float64 {
	yPosition, okYPosition := yPositions[ySymbol]
	targetSize, okTargetSize := yTargetPositionSizes[ySymbol]
	spread, okSpread := xySpreads[xSymbol]
	if !okYPosition || !okSpread || !okTargetSize {
		return 0
	}

	yDepth := spread.YDepth
	yStepSize := yStepSizes[ySymbol]
	yMinNotional := yMinNotionals[ySymbol]
	yMultiplier := yMultipliers[ySymbol]
	ySizeDiff := targetSize/yMultiplier - yPosition.GetSize()
	if math.Abs(ySizeDiff) < yStepSize {
		return 0
	}
	ySizeDiff = math.Round(ySizeDiff/yStepSize) * yStepSize

	if yExchange.IsSpot() {
		if math.Abs(ySizeDiff) < yStepSize {
			return 0
		} else if ySizeDiff < 0 && -ySizeDiff*yMultiplier*yDepth.MidPrice < yMinNotional {
			return 0
		} else if ySizeDiff > 0 && ySizeDiff*yMultiplier*yDepth.MidPrice < yMinNotional {
			return 0
		}
	} else {
		//期货以close仓位，没有minNotional限制
		if math.Abs(ySizeDiff) < yStepSize {
			return 0
		} else if ySizeDiff < 0 && yPosition.GetSize() <= 0 && -ySizeDiff*yMultiplier*yDepth.MidPrice < yMinNotional {
			return 0
		} else if ySizeDiff > 0 && yPosition.GetSize() >= 0 && ySizeDiff*yMultiplier*yDepth.MidPrice < yMinNotional {
			return 0
		}
	}

	reduceOnly := false
	if ySizeDiff*yPosition.GetSize() < 0 && math.Abs(ySizeDiff) <= math.Abs(yPosition.GetSize()) {
		reduceOnly = true
	}
	side := common.OrderSideBuy
	if ySizeDiff < 0 {
		side = common.OrderSideSell
		ySizeDiff = -ySizeDiff
	}
	yOrder := common.NewOrderParam{
		Symbol:     ySymbol,
		Side:       side,
		Type:       common.OrderTypeMarket,
		Size:       ySizeDiff,
		ReduceOnly: reduceOnly,
		ClientID:   yExchange.GenerateClientID(),
	}
	if !xyConfig.DryRun {
		select {
		case yOrderRequestChMap[ySymbol] <- common.OrderRequest{
			New: &yOrder,
		}:
			yOrderSilentTimes[ySymbol] = time.Now().Add(xyConfig.OrderSilent)
			yPositionsUpdateTimes[ySymbol] = time.Unix(0, 0)
		default:
			logger.Debugf("yOrderRequestChMap[ySymbol] <- common.OrderRequest %s failed, ch len %d", ySymbol, len(yOrderRequestChMap[ySymbol]))
		}
	} else {
		yOrderSilentTimes[ySymbol] = time.Now().Add(xyConfig.OrderSilent)
		yPositionsUpdateTimes[ySymbol] = time.Unix(0, 0)
	}
	return math.Abs(ySizeDiff * yMultiplier * yDepth.MidPrice)
}

func updateYPositions() {
	if xAccount == nil {
		if time.Now().Sub(time.Now().Truncate(xyConfig.LogInterval)) < xyConfig.LoopInterval {
			logger.Debugf("xACCOUNT not ready")
		}
		return
	}
	if yAccount == nil {
		if time.Now().Sub(time.Now().Truncate(xyConfig.LogInterval)) < xyConfig.LoopInterval {
			logger.Debugf("yACCOUNT not ready")
		}
		return
	}
	unHedgedValue := 0.0
	for _, ySymbol := range ySymbols {
		xSymbol := yxSymbolsMap[ySymbol]
		if _, ok := xyConfig.NotTradePairs[xSymbol]; ok {
			continue
		}
		if time.Now().Sub(yPositionsUpdateTimes[ySymbol]) > xyConfig.BalancePositionMaxAge {
			continue
		}
		if yOrderSilentTimes[ySymbol].Sub(time.Now()).Seconds() > 0 {
			continue
		}
		unHedgedValue += hedgeYSymbol(ySymbol, xSymbol)
	}
	xyUnHedgeValue = unHedgedValue
}

func hedgeXSymbol(xSymbol string) {
	xPosition, okXPosition := xPositions[xSymbol]
	xTargetSize, okXTargetSize := xTargetPositionSizes[xSymbol]
	spread, okSpread := xySpreads[xSymbol]
	if !okXPosition || !okSpread || !okXTargetSize {
		return
	}
	xDepth := spread.XDepth
	xStepSize := xStepSizes[xSymbol]
	xMinNotional := xMinNotionals[xSymbol]
	xMultiplier := xMultipliers[xSymbol]
	xSizeDiff := xTargetSize/xMultiplier - xPosition.GetSize()
	if math.Abs(xSizeDiff) < xStepSize {
		return
	}
	xSizeDiff = math.Round(xSizeDiff/xStepSize) * xStepSize
	if xExchange.IsSpot() {
		if math.Abs(xSizeDiff) < xStepSize {
			return
		} else if xSizeDiff < 0 && -xSizeDiff*xMultiplier*xDepth.MidPrice < xMinNotional {
			return
		} else if xSizeDiff > 0 && xSizeDiff*xMultiplier*xDepth.MidPrice < xMinNotional {
			return
		}
	} else {
		if math.Abs(xSizeDiff) < xStepSize {
			return
		} else if xSizeDiff < 0 && xPosition.GetSize() <= 0 && -xSizeDiff*xMultiplier*xDepth.MidPrice < xMinNotional {
			return
		} else if xSizeDiff > 0 && xPosition.GetSize() >= 0 && xSizeDiff*xMultiplier*xDepth.MidPrice < xMinNotional {
			return
		}
	}

	//logger.Debugf("updateXPositions %s size %f position %f -> %f", xSymbol, xSizeDiff, xPosition.GetSize(), xTargetSize)

	reduceOnly := false
	if xSizeDiff*xPosition.GetSize() < 0 && math.Abs(xSizeDiff) <= math.Abs(xPosition.GetSize()) {
		reduceOnly = true
	}
	side := common.OrderSideBuy
	if xSizeDiff < 0 {
		side = common.OrderSideSell
		xSizeDiff = -xSizeDiff
	}
	yOrder := common.NewOrderParam{
		Symbol:     xSymbol,
		Side:       side,
		Type:       common.OrderTypeMarket,
		Size:       xSizeDiff,
		ReduceOnly: reduceOnly,
		ClientID:   xExchange.GenerateClientID(),
	}
	if !xyConfig.DryRun {
		select {
		case xOrderRequestChMap[xSymbol] <- common.OrderRequest{
			New: &yOrder,
		}:
			xOrderSilentTimes[xSymbol] = time.Now().Add(xyConfig.OrderSilent)
			xPositionsUpdateTimes[xSymbol] = time.Unix(0, 0)
		default:
			logger.Debugf("xOrderRequestChMap[xSymbol] <- common.OrderRequest %s failed, ch len %d", xSymbol, len(xOrderRequestChMap[xSymbol]))
		}
	} else {
		xOrderSilentTimes[xSymbol] = time.Now().Add(xyConfig.OrderSilent)
		xPositionsUpdateTimes[xSymbol] = time.Unix(0, 0)
	}
}

func updateXPositions() {
	if xAccount == nil {
		if time.Now().Sub(time.Now().Truncate(xyConfig.LogInterval)) < xyConfig.LoopInterval {
			logger.Debugf("xACCOUNT not ready")
		}
		return
	}
	if yAccount == nil {
		if time.Now().Sub(time.Now().Truncate(xyConfig.LogInterval)) < xyConfig.LoopInterval {
			logger.Debugf("yACCOUNT not ready")
		}
		return
	}
	for _, xSymbol := range xSymbols {
		if _, ok := xyConfig.NotTradePairs[xSymbol]; ok {
			continue
		}
		if time.Now().Sub(xPositionsUpdateTimes[xSymbol]) > xyConfig.BalancePositionMaxAge {
			continue
		}
		if xOrderSilentTimes[xSymbol].Sub(time.Now()).Seconds() > 0 {
			continue
		}
		hedgeXSymbol(xSymbol)
	}
}

func updateTargetPositionSizes() {

	if xAccount == nil {
		if time.Now().Sub(time.Now().Truncate(xyConfig.LogInterval)) < xyConfig.LoopInterval {
			logger.Debugf("xACCOUNT not ready")
		}
		return
	}
	if yAccount == nil {
		if time.Now().Sub(time.Now().Truncate(xyConfig.LogInterval)) < xyConfig.LoopInterval {
			logger.Debugf("yACCOUNT not ready")
		}
		return
	}

	//第一步，默认以X为准，对冲Y
	for xSymbol, ySymbol := range xySymbolsMap {
		if _, ok := xyConfig.NotTradePairs[xSymbol]; ok {
			continue
		}
		//在信号触发期间，以信号为准
		if time.Now().Sub(xyTargetPositionUpdateSilentTimes[xSymbol]) < 0 {
			continue
		}

		xPosition, okXPosition := xPositions[xSymbol]
		yPosition, okYPosition := yPositions[ySymbol]
		xySpotStepSize := xySpotStepSizes[xSymbol]
		xMultiplier := xMultipliers[xSymbol]
		yMultiplier := yMultipliers[ySymbol]

		//其他时间以仓位小的为准
		if okXPosition && okYPosition {
			if math.Abs(xPosition.GetSize()*xMultiplier)-math.Abs(yPosition.GetSize()*yMultiplier) >= xySpotStepSize {
				yTargetPositionSizes[ySymbol] = yPosition.GetSize() * yMultiplier
				xTargetPositionSizes[xSymbol] = -yPosition.GetSize() * yMultiplier
			} else if math.Abs(xPosition.GetSize()*xMultiplier)-math.Abs(yPosition.GetSize()*yMultiplier) <= -xySpotStepSize {
				xTargetPositionSizes[xSymbol] = xPosition.GetSize() * xMultiplier
				yTargetPositionSizes[ySymbol] = -xPosition.GetSize() * xMultiplier
			} else {
				xTargetPositionSizes[xSymbol] = xPosition.GetSize() * xMultiplier
				yTargetPositionSizes[ySymbol] = yPosition.GetSize() * yMultiplier
			}
		}
	}

	if len(xyRankSymbolMap) == 0 {
		if time.Now().Sub(time.Now().Truncate(xyConfig.LogInterval)) < xyConfig.LoopInterval {
			logger.Debugf("xyRankSymbolMap not ready")
		}
		return
	}

	if xyUnHedgeValue > xyConfig.MaxUnHedgeValue {
		if time.Now().Sub(time.Now().Truncate(xyConfig.LogInterval)) < xyConfig.LoopInterval {
			logger.Debugf("xyUnHedgeValue %f > xyConfig.MaxUnHedgeValue %f", xyUnHedgeValue, xyConfig.MaxUnHedgeValue)
		}
		return
	}

	entryStep := (xAccount.GetFree() + yAccount.GetFree()) * xyConfig.EnterFreePct
	if entryStep < xyConfig.EnterMinimalStep {
		entryStep = xyConfig.EnterMinimalStep
	}
	entryTarget := entryStep * xyConfig.EnterTargetFactor

	//得是两个市场的最小可用资金, 以防有一边用完了钱, 开不了仓
	xyUSDTAvailable := math.Min(xAccount.GetFree()*xyConfig.XExchange.Leverage, yAccount.GetFree()*xyConfig.YExchange.Leverage)

	//遍历合约 从最大的rank 开始，能保证FR强的先下单, 优先做空
	for _, rank := range xyDualEnds {
		xSymbol := xyRankSymbolMap[rank]
		ySymbol := xySymbolsMap[xSymbol]
		if _, ok := xyConfig.NotTradePairs[xSymbol]; ok {
			continue
		}

		//需要保证两边都有仓位更新，才调整现货仓位
		if time.Now().Sub(xPositionsUpdateTimes[xSymbol]) > xyConfig.BalancePositionMaxAge {
			if time.Now().Sub(time.Now().Truncate(xyConfig.LogInterval)) < xyConfig.LoopInterval {
				logger.Debugf("%s x position too old", xSymbol)
			}
			continue
		}
		if time.Now().Sub(yPositionsUpdateTimes[ySymbol]) > xyConfig.BalancePositionMaxAge {
			if time.Now().Sub(time.Now().Truncate(xyConfig.LogInterval)) < xyConfig.LoopInterval {
				logger.Debugf("%s y position too old", ySymbol)
			}
			continue
		}
		if time.Now().Sub(xyTargetPositionUpdateSilentTimes[xSymbol]) < 0 {
			//if time.Now().Sub(time.Now().Truncate(xyConfig.LogInterval)) < xyConfig.LoopInterval {
			//	logger.Debugf("%s %s in target update silent", xSymbol, ySymbol)
			//}
			continue
		}

		spread, okSpread := xySpreads[xSymbol]
		xPosition, okXPosition := xPositions[xSymbol]
		yPosition, okYPosition := yPositions[ySymbol]
		fundingRate, okFundingRate := xyFundingRates[xSymbol]
		if !okSpread || !okXPosition || !okYPosition || !okFundingRate {
			if time.Now().Sub(time.Now().Truncate(xyConfig.LogInterval)) < xyConfig.LoopInterval {
				logger.Debugf("%s %s spread %v x position %v y position %v fundingRate %v", xSymbol, ySymbol, okSpread, okXPosition, okYPosition, okFundingRate)
			}
			continue
		}

		if time.Now().Sub(spread.Time) > xyConfig.SpreadTimeToLive {
			//if time.Now().Sub(time.Now().Truncate(xyConfig.LogInterval)) < xyConfig.LoopInterval {
			//	logger.Debugf("%s %s spread too old %v", xSymbol, ySymbol, spread.Time)
			//}
			continue
		}
		xDepth := spread.XDepth
		yDepth := spread.YDepth
		xStepSize := xStepSizes[xSymbol]
		xMinNotional := xMinNotionals[xSymbol]
		yMinNotional := yMinNotionals[ySymbol]
		xMultiplier := xMultipliers[xSymbol]
		yMultiplier := yMultipliers[ySymbol]

		xySpotStepSize := xySpotStepSizes[xSymbol]

		xSize := xPosition.GetSize() * xMultiplier
		ySize := yPosition.GetSize() * yMultiplier
		xValue := math.Abs(xSize) * spread.XDepth.MidPrice
		yValue := math.Abs(ySize) * spread.YDepth.MidPrice
		offsetFactor := (xValue + yValue) * 0.5 / entryTarget
		shortTop := xyConfig.ShortEnterDelta + xyConfig.EnterOffsetDelta*offsetFactor
		shortBot := xyConfig.ShortExitDelta + xyConfig.ExitOffsetDelta*offsetFactor
		longBot := xyConfig.LongEnterDelta - xyConfig.EnterOffsetDelta*offsetFactor
		longTop := xyConfig.LongExitDelta - xyConfig.ExitOffsetDelta*offsetFactor

		midPrice := (xDepth.MidPrice + yDepth.MidPrice) * 0.5

		if spread.ShortLastLeave < shortBot &&
			spread.ShortMedianLeave < shortBot &&
			fundingRate < xyConfig.MinimalKeepFundingRate &&
			xSize >= xStepSize {

			entryValue := math.Min(4*entryStep, math.Min(xValue, yValue))
			if fundingRate > xyConfig.MinimalKeepFundingRate*0.5 {
				entryValue = math.Min(2*entryStep, math.Min(xValue, yValue))
			}
			size := entryValue / midPrice
			size = math.Round(size/xySpotStepSize) * xySpotStepSize
			entryValue = size * midPrice

			if xValue-entryValue < xySpotStepSize || yValue-entryValue < xySpotStepSize {
				//两种情况都把x全平，间接y全平
				size = xSize
			}
			//谁小以谁为准
			if xValue <= yValue {
				xTargetPositionSizes[xSymbol] -= size
				if xTargetPositionSizes[xSymbol] < 0 {
					xTargetPositionSizes[xSymbol] = 0
				}
				yTargetPositionSizes[ySymbol] = -xTargetPositionSizes[xSymbol]
			} else {
				yTargetPositionSizes[ySymbol] += size
				if yTargetPositionSizes[ySymbol] > 0 {
					yTargetPositionSizes[ySymbol] = 0
				}
				xTargetPositionSizes[xSymbol] = -yTargetPositionSizes[ySymbol]
			}
			xyTargetPositionUpdateSilentTimes[xSymbol] = time.Now().Add(xyConfig.EnterSilent)
			xOrderSilentTimes[xSymbol] = time.Now()
			yOrderSilentTimes[ySymbol] = time.Now()
			delete(xLastFilledBuyPrices, xSymbol)
			delete(xLastFilledSellPrices, xSymbol)
			delete(yLastFilledBuyPrices, ySymbol)
			delete(yLastFilledSellPrices, ySymbol)
			logger.Debugf(
				"%s %s SHORT BOT REDUCE %f < %f, %f < %f, SIZE %f, TARGET X %f TARGET Y %f",
				xSymbol, ySymbol,
				spread.ShortLastLeave, shortBot,
				spread.ShortMedianLeave, shortBot,
				size,
				xTargetPositionSizes[xSymbol],
				yTargetPositionSizes[ySymbol],
			)
			hedgeXSymbol(xSymbol)
			hedgeYSymbol(ySymbol, xSymbol)
		} else if spread.LongLastLeave > longTop &&
			spread.LongMedianLeave > longTop &&
			fundingRate > -xyConfig.MinimalKeepFundingRate &&
			xSize <= -xStepSize {

			entryValue := math.Min(4*entryStep, math.Min(xValue, yValue))
			if fundingRate < -xyConfig.MinimalKeepFundingRate*0.5 {
				entryValue = math.Min(2*entryStep, math.Min(xValue, yValue))
			}
			size := entryValue / midPrice
			size = math.Round(size/xySpotStepSize) * xySpotStepSize
			entryValue = size * midPrice
			if xValue-entryValue < xySpotStepSize || yValue-entryValue < xySpotStepSize {
				size = -xSize
			}
			//谁小以谁为准
			if xValue <= yValue {
				xTargetPositionSizes[xSymbol] += size
				if xTargetPositionSizes[xSymbol] > 0 {
					xTargetPositionSizes[xSymbol] = 0
				}
				yTargetPositionSizes[ySymbol] = -xTargetPositionSizes[xSymbol]
			} else {
				yTargetPositionSizes[ySymbol] -= size
				if yTargetPositionSizes[ySymbol] < 0 {
					yTargetPositionSizes[ySymbol] = 0
				}
				xTargetPositionSizes[xSymbol] = -yTargetPositionSizes[ySymbol]
			}
			xyTargetPositionUpdateSilentTimes[xSymbol] = time.Now().Add(xyConfig.EnterSilent)

			xOrderSilentTimes[xSymbol] = time.Now()
			yOrderSilentTimes[ySymbol] = time.Now()
			delete(xLastFilledBuyPrices, xSymbol)
			delete(xLastFilledSellPrices, xSymbol)
			delete(yLastFilledBuyPrices, ySymbol)
			delete(yLastFilledSellPrices, ySymbol)
			logger.Debugf(
				"%s %s LONG TOP REDUCE %f > %f, %f > %f, SIZE %f, TARGET X %f, TARGET Y %f",
				xSymbol, ySymbol,
				spread.LongLastLeave, longTop,
				spread.LongMedianLeave, longTop,
				size,
				xTargetPositionSizes[xSymbol],
				yTargetPositionSizes[ySymbol],
			)
			hedgeXSymbol(xSymbol)
			hedgeYSymbol(ySymbol, xSymbol)
		} else if !yExchange.IsSpot() &&
			spread.ShortLastEnter > shortTop &&
			spread.ShortMedianEnter > shortTop &&
			fundingRate > xyConfig.MinimalEnterFundingRate &&
			xSize >= 0 {

			targetValue := math.Max(xValue, yValue) + entryStep
			if targetValue > entryTarget {
				targetValue = entryTarget
			}
			entryValue := targetValue - math.Max(xValue, yValue)
			size := entryValue / midPrice
			size = math.Round(size/xySpotStepSize) * xySpotStepSize
			entryValue = size * midPrice

			if entryValue > xyUSDTAvailable {
				if time.Now().Sub(time.Now().Truncate(xyConfig.LogInterval)) < xyConfig.LoopInterval {
					logger.Debugf(
						"%s %s FAILED SHORT TOP OPEN, ENTRY VALUE %f MORE THAN xyUSDTAvailable %f, %f > %f, %f > %f, SIZE %f",
						xSymbol,
						ySymbol,
						entryValue,
						xyUSDTAvailable,
						spread.ShortLastEnter, shortTop,
						spread.ShortMedianEnter, shortTop,
						size,
					)
				}
				continue
			}
			if entryValue < yMinNotional || entryValue < xMinNotional || entryValue == 0 {
				if time.Now().Sub(xyLogSilentTimes[xSymbol]) > 0 {
					logger.Debugf(
						"%s %s FAILED SHORT TOP OPEN, ORDER VALUE %f TOO SMALL, %f > %f, %f > %f, SIZE %f",
						xSymbol, ySymbol,
						entryValue,
						spread.ShortLastEnter, shortTop,
						spread.ShortMedianEnter, shortTop,
						size,
					)
					xyLogSilentTimes[xSymbol] = time.Now().Add(xyConfig.LogInterval)
				}
				continue
			}
			//谁大以谁为准
			if xValue >= yValue {
				xTargetPositionSizes[xSymbol] += size
				yTargetPositionSizes[ySymbol] = -xTargetPositionSizes[xSymbol]
			} else {
				yTargetPositionSizes[ySymbol] -= size
				xTargetPositionSizes[xSymbol] = -yTargetPositionSizes[ySymbol]
			}
			xyTargetPositionUpdateSilentTimes[xSymbol] = time.Now().Add(xyConfig.EnterSilent)
			xyUSDTAvailable -= entryValue
			xOrderSilentTimes[xSymbol] = time.Now()
			yOrderSilentTimes[ySymbol] = time.Now()
			delete(xLastFilledBuyPrices, xSymbol)
			delete(xLastFilledSellPrices, xSymbol)
			delete(yLastFilledBuyPrices, ySymbol)
			delete(yLastFilledSellPrices, ySymbol)
			logger.Debugf(
				"%s %s SHORT TOP OPEN %f > %f, %f > %f, SIZE %f, TARGET X %f, TARGET Y %f",
				xSymbol, ySymbol,
				spread.ShortLastEnter, shortTop,
				spread.ShortMedianEnter, shortTop,
				size,
				xTargetPositionSizes[xSymbol],
				yTargetPositionSizes[ySymbol],
			)
			hedgeXSymbol(xSymbol)
			hedgeYSymbol(ySymbol, xSymbol)
		} else if !xExchange.IsSpot() &&
			spread.LongLastEnter < longBot &&
			spread.LongMedianEnter < longBot &&
			fundingRate < -xyConfig.MinimalEnterFundingRate &&
			xSize <= 0 {

			targetValue := math.Max(xValue, yValue) + entryStep
			if targetValue > entryTarget {
				targetValue = entryTarget
			}
			entryValue := targetValue - math.Max(xValue, yValue)
			size := entryValue / midPrice
			size = math.Round(size/xySpotStepSize) * xySpotStepSize
			entryValue = size * midPrice
			if entryValue > xyUSDTAvailable {
				if time.Now().Sub(time.Now().Truncate(xyConfig.LogInterval)) < xyConfig.LoopInterval {
					logger.Debugf(
						"%s %s FAILED SHORT TOP OPEN, ENTRY VALUE %f MORE THAN xyUSDTAvailable %f, %f < %f, %f < %f, SIZE %f",
						xSymbol,
						ySymbol,
						entryValue,
						xyUSDTAvailable,
						spread.LongLastEnter, longBot,
						spread.LongMedianEnter, longBot,
						size,
					)
				}
				continue
			}
			if entryValue < yMinNotional || entryValue < xMinNotional || entryValue == 0 {
				if time.Now().Sub(xyLogSilentTimes[xSymbol]) > 0 {
					logger.Debugf(
						"%s %s FAILED SHORT TOP OPEN, ORDER VALUE %f TOO SMALL, %f < %f, %f < %f, SIZE %f",
						xSymbol, ySymbol,
						entryValue,
						spread.LongLastEnter, longBot,
						spread.LongMedianEnter, longBot,
						size,
					)
					xyLogSilentTimes[xSymbol] = time.Now().Add(xyConfig.LogInterval)
				}
				continue
			}
			//谁大以谁为准
			if xValue >= yValue {
				xTargetPositionSizes[xSymbol] -= size
				yTargetPositionSizes[ySymbol] = -xTargetPositionSizes[xSymbol]
			} else {
				yTargetPositionSizes[ySymbol] += size
				xTargetPositionSizes[xSymbol] = -yTargetPositionSizes[ySymbol]
			}
			xyTargetPositionUpdateSilentTimes[xSymbol] = time.Now().Add(xyConfig.EnterSilent)
			xyUSDTAvailable -= entryValue
			xOrderSilentTimes[xSymbol] = time.Now()
			yOrderSilentTimes[ySymbol] = time.Now()
			delete(xLastFilledBuyPrices, xSymbol)
			delete(xLastFilledSellPrices, xSymbol)
			delete(yLastFilledBuyPrices, ySymbol)
			delete(yLastFilledSellPrices, ySymbol)
			logger.Debugf(
				"%s %s LONG BOT OPEN %f < %f, %f < %f, SIZE %f, TARGET X %f, TARGET Y %f",
				xSymbol, ySymbol,
				spread.LongLastEnter, longBot,
				spread.LongMedianEnter, longBot,
				size,
				xTargetPositionSizes[xSymbol],
				yTargetPositionSizes[ySymbol],
			)
			hedgeXSymbol(xSymbol)
			hedgeYSymbol(ySymbol, xSymbol)

		}
	}
}

func handleUpdateFundingRates() {
	if len(xFundingRates) != len(yFundingRates) {
		//if time.Now().Sub(time.Now().Truncate(xyConfig.LogInterval)) < xyConfig.LogInterval {
		//	logger.Debugf("len(xFundingRates) %d != len(yFundingRates) %d", len(xFundingRates), len(yFundingRates))
		//}
		return
	}
	frs := make([]float64, len(xSymbols))
	for i, xSymbol := range xSymbols {
		ySymbol := xySymbolsMap[xSymbol]
		if xFr, ok := xFundingRates[xSymbol]; ok {
			if yFr, ok := yFundingRates[ySymbol]; ok {
				frs[i] = yFr.GetFundingRate() - xFr.GetFundingRate()
				xyFundingRates[xSymbol] = frs[i]
			} else {
				logger.Debugf("MISS PREMIUM INDEX FOR TAKER %s", xSymbol)
				return
			}
		} else {
			logger.Debugf("MISS FUNDING RATE FOR MAKER %s", xSymbol)
			return
		}
	}
	var err error
	if len(xyRankSymbolMap) == 0 {
		for i, fr := range frs {
			logger.Debugf(
				"MERGED FR %s %f %s %f -> %f",
				xSymbols[i], xFundingRates[xSymbols[i]].GetFundingRate(),
				xySymbolsMap[xSymbols[i]], yFundingRates[xySymbolsMap[xSymbols[i]]].GetFundingRate(),
				fr,
			)
		}
		xyRankSymbolMap, err = common.RankSymbols(xSymbols, frs)
		if err != nil {
			logger.Debugf("RankSymbols error %v", err)
		}
		logger.Debugf("SYMBOLS FR RANK %v", xyRankSymbolMap)
	} else {
		xyRankSymbolMap, err = common.RankSymbols(xSymbols, frs)
		if err != nil {
			logger.Debugf("RankSymbols error %v", err)
		}
	}
}
