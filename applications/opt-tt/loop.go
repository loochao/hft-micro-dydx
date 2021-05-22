package main

import (
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"time"
)

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
		if time.Now().Sub(yPositionsUpdateTimes[ySymbol]) > xyConfig.BalancePositionMaxAge {
			continue
		}
		if yOrderSilentTimes[ySymbol].Sub(time.Now()).Seconds() > 0 {
			continue
		}

		yPosition, okYPosition := yPositions[ySymbol]
		_, okXPosition := xPositions[xSymbol]
		targetSize, okTargetSize := yTargetPositionSizes[ySymbol]
		spread, okSpread := xySpreads[xSymbol]
		if !okYPosition || !okSpread || !okTargetSize || !okXPosition {
			continue
		}
		yDepth := spread.YDepth

		yStepSize := yStepSizes[ySymbol]
		yMinNotional := yMinNotionals[ySymbol]
		ySizeDiff := targetSize - yPosition.GetSize()
		ySizeDiff = math.Round(ySizeDiff/yStepSize) * yStepSize
		unHedgedValue += math.Abs(ySizeDiff * yDepth.MidPrice)

		if math.Abs(ySizeDiff) < yStepSize {
			continue
		} else if ySizeDiff < 0 && yPosition.GetSize() <= 0 && -ySizeDiff*yDepth.BestBidPrice < yMinNotional {
			continue
		} else if ySizeDiff > 0 && yPosition.GetSize() >= 0 && ySizeDiff*yDepth.BestAskPrice < yMinNotional {
			continue
		}

		hedgeMarkPrice, okHedgeMarkPrice := yHedgeMarkPrices[ySymbol]
		if xyEnterTradeOrders[xSymbol] == EnterTradeOrderXY && okHedgeMarkPrice {
			if ySizeDiff < 0 && yDepth.BestBidPrice > hedgeMarkPrice*(1.0+xyConfig.EnterOffsetDelta) {
				logger.Debugf("%s updateYPositions size %f push mark price %f -> %f", ySymbol, ySizeDiff, hedgeMarkPrice, yDepth.BestBidPrice)
				yHedgeMarkPrices[ySymbol] = yDepth.BestBidPrice
				yOrderSilentTimes[ySymbol] = time.Now().Add(xyConfig.HedgeCheckInterval)
				yPositionsUpdateTimes[ySymbol] = time.Now()
				continue
			} else if ySizeDiff > 0 && yDepth.BestAskPrice < hedgeMarkPrice*(1.0-xyConfig.EnterOffsetDelta) {
				logger.Debugf("%s updateYPositions size %f push mark price %f -> %f", ySymbol, ySizeDiff, hedgeMarkPrice, yDepth.BestAskPrice)
				yHedgeMarkPrices[ySymbol] = yDepth.BestAskPrice
				yOrderSilentTimes[ySymbol] = time.Now().Add(xyConfig.HedgeCheckInterval)
				yPositionsUpdateTimes[ySymbol] = time.Now()
				continue
			}
		}

		logger.Debugf("updateYPositions %s size %f position %f -> %f", ySymbol, ySizeDiff, yPosition.GetSize(), targetSize)

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
		logger.Debugf("y order %v", yOrder)
		if !xyConfig.DryRun {
			select {
			case yOrderRequestChMap[ySymbol] <- common.OrderRequest{
				New: &yOrder,
			}:
				yOrderSilentTimes[ySymbol] = time.Now().Add(xyConfig.OrderSilent)
				yPositionsUpdateTimes[ySymbol] = time.Unix(0, 0)
				if okHedgeMarkPrice {
					delete(yHedgeMarkPrices, ySymbol)
				}
			default:
				logger.Debugf("yOrderRequestChMap[ySymbol] <- common.OrderRequest failed, ch len %d", len(yOrderRequestChMap[ySymbol]))
			}
		}
	}
	xyUnHedgeValue = unHedgedValue
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
		ySymbol := xySymbolsMap[xSymbol]
		if time.Now().Sub(xPositionsUpdateTimes[xSymbol]) > xyConfig.BalancePositionMaxAge {
			continue
		}
		if xOrderSilentTimes[xSymbol].Sub(time.Now()).Seconds() > 0 {
			continue
		}
		xPosition, okXPosition := xPositions[xSymbol]
		_, okYPosition := yPositions[ySymbol]
		xTargetSize, okXTargetSize := xTargetPositionSizes[xSymbol]
		spread, okSpread := xySpreads[xSymbol]
		if !okXPosition || !okSpread || !okXTargetSize || !okYPosition {
			continue
		}
		xDepth := spread.XDepth

		xStepSize := xStepSizes[xSymbol]
		xMinNotional := xMinNotionals[xSymbol]
		xSizeDiff := xTargetSize - xPosition.GetSize()
		xSizeDiff = math.Round(xSizeDiff/xStepSize) * xStepSize

		if math.Abs(xSizeDiff) < xStepSize {
			continue
		} else if xSizeDiff < 0 && xPosition.GetSize() <= 0 && -xSizeDiff*xDepth.BestBidPrice < xMinNotional {
			continue
		} else if xSizeDiff > 0 && xPosition.GetSize() >= 0 && xSizeDiff*xDepth.BestAskPrice < xMinNotional {
			continue
		}

		xHedgeMarkPrice, okXHedgeMarkPrice := xHedgeMarkPrices[xSymbol]
		if xyEnterTradeOrders[xSymbol] == EnterTradeOrderYX && okXHedgeMarkPrice {
			if xSizeDiff < 0 && xDepth.BestBidPrice > xHedgeMarkPrice*(1.0+xyConfig.EnterOffsetDelta) {
				logger.Debugf("%s updateXPositions size %f push mark price %f -> %f", xSymbol, xSizeDiff, xHedgeMarkPrice, xDepth.BestBidPrice)
				xHedgeMarkPrices[xSymbol] = xDepth.BestBidPrice
				xOrderSilentTimes[xSymbol] = time.Now().Add(xyConfig.HedgeCheckInterval)
				xPositionsUpdateTimes[xSymbol] = time.Now()
				continue
			} else if xSizeDiff > 0 && xDepth.BestAskPrice < xHedgeMarkPrice*(1.0-xyConfig.EnterOffsetDelta) {
				logger.Debugf("%s updateXPositions size %f push mark price %f -> %f", xSymbol, xSizeDiff, xHedgeMarkPrice, xDepth.BestAskPrice)
				xHedgeMarkPrices[xSymbol] = xDepth.BestAskPrice
				xOrderSilentTimes[xSymbol] = time.Now().Add(xyConfig.HedgeCheckInterval)
				xPositionsUpdateTimes[xSymbol] = time.Now()
				continue
			}
		}

		logger.Debugf("updateXPositions %s size %f position %f -> %f", xSymbol, xSizeDiff, xPosition.GetSize(), xTargetSize)

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
		logger.Debugf("x order %v", yOrder)
		if !xyConfig.DryRun {
			select {
			case xOrderRequestChMap[xSymbol] <- common.OrderRequest{
				New: &yOrder,
			}:
				xOrderSilentTimes[xSymbol] = time.Now().Add(xyConfig.OrderSilent)
				xPositionsUpdateTimes[xSymbol] = time.Unix(0, 0)
				if okXHedgeMarkPrice {
					delete(xHedgeMarkPrices, xSymbol)
				}
			default:
				logger.Debugf("xOrderRequestChMap[xSymbol] <- common.OrderRequest %d failed, ch len %d", len(xOrderRequestChMap[xSymbol]))
			}
		}
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
		//在信号触发期间，以信号为准
		if time.Now().Sub(xyTargetPositionUpdateSilentTimes[xSymbol]) < 0 {
			continue
		}
		//其他时间以X为准
		if xPosition, okXPosition := xPositions[xSymbol]; okXPosition {
			xTargetPositionSizes[xSymbol] = xPosition.GetSize()
			yTargetPositionSizes[ySymbol] = -xPosition.GetSize()
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

		spread, okSpread := xySpreads[xSymbol]
		//需要保证两边都有仓位更新，才调整现货仓位
		if time.Now().Sub(xPositionsUpdateTimes[xSymbol]) > xyConfig.BalancePositionMaxAge {
			//if time.Now().Sub(time.Now().Truncate(xyConfig.LogInterval)) < xyConfig.LoopInterval {
			//	logger.Debugf("%s x position too old", xSymbol)
			//}
			continue
		}
		if time.Now().Sub(yPositionsUpdateTimes[ySymbol]) > xyConfig.BalancePositionMaxAge {
			//if time.Now().Sub(time.Now().Truncate(xyConfig.LogInterval)) < xyConfig.LoopInterval {
			//	logger.Debugf("%s y position too old", xSymbol)
			//}
			continue
		}
		if time.Now().Sub(xyTargetPositionUpdateSilentTimes[xSymbol]) < 0 {
			//if time.Now().Sub(time.Now().Truncate(xyConfig.LogInterval)) < xyConfig.LoopInterval {
			//	logger.Debugf("%s %s in target update silent", xSymbol, ySymbol)
			//}
			continue
		}
		xPosition, okXPosition := xPositions[xSymbol]
		yPosition, okYPosition := yPositions[ySymbol]
		fundingRate, okFundingRate := xyFundingRates[xSymbol]
		if !okSpread || !okXPosition || !okYPosition || !okFundingRate {
			//if time.Now().Sub(time.Now().Truncate(xyConfig.LogInterval)) < xyConfig.LoopInterval {
			//	logger.Debugf("%s %s spread %v x position %v y position %v fundingRate %v", xSymbol, ySymbol, okSpread, okXPosition, okYPosition, okFundingRate)
			//}
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

		xyStepSize := xyStepSizes[xSymbol]
		xValue := math.Abs(xPosition.GetSize()) * xPosition.GetPrice()
		yValue := math.Abs(yPosition.GetSize()) * yPosition.GetPrice()
		offsetFactor := (xValue + yValue) * 0.5 / entryTarget
		shortTop := xyConfig.ShortEnterDelta + xyConfig.EnterOffsetDelta*offsetFactor
		shortBot := xyConfig.ShortExitDelta
		longBot := xyConfig.LongEnterDelta - xyConfig.EnterOffsetDelta*offsetFactor
		longTop := xyConfig.LongExitDelta

		xSize := xPosition.GetSize()
		midPrice := (xDepth.MidPrice + yDepth.MidPrice) * 0.5

		xyMergedDirs[xSymbol] = spread.XDir*xyConfig.XYDirRatio + spread.YDir*(1.0-xyConfig.XYDirRatio)

		if spread.ShortLastLeave < shortBot &&
			spread.ShortMedianLeave < shortBot &&
			fundingRate < xyConfig.MinimalKeepFundingRate &&
			xSize >= xStepSize {

			entryValue := math.Min(4*entryStep, math.Min(xValue, yValue))
			if fundingRate > xyConfig.MinimalKeepFundingRate*0.5 {
				entryValue = math.Min(2*entryStep, math.Min(xValue, yValue))
			}
			size := entryValue / midPrice
			size = math.Round(size/xyStepSize) * xyStepSize
			entryValue = size * midPrice
			if xValue-entryValue < entryStep {
				size = xPosition.GetSize()
			}
			if yValue-entryValue < entryStep {
				size = xPosition.GetSize()
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

			if xyMergedDirs[xSymbol] < 0 {
				xyEnterTradeOrders[xSymbol] = EnterTradeOrderXY
				xOrderSilentTimes[xSymbol] = time.Now()
				xHedgeMarkPrices[xSymbol] = xDepth.BestBidPrice
				yOrderSilentTimes[ySymbol] = time.Now().Add(xyConfig.HedgeCheckInterval)
				yHedgeMarkPrices[ySymbol] = yDepth.BestAskPrice
			} else if xyMergedDirs[xSymbol] > 0 {
				xyEnterTradeOrders[xSymbol] = EnterTradeOrderYX
				xOrderSilentTimes[xSymbol] = time.Now().Add(xyConfig.HedgeCheckInterval)
				xHedgeMarkPrices[xSymbol] = xDepth.BestBidPrice
				yOrderSilentTimes[ySymbol] = time.Now()
				yHedgeMarkPrices[ySymbol] = yDepth.BestAskPrice
			} else {
				xyEnterTradeOrders[xSymbol] = EnterTradeOrderUnknown
				xOrderSilentTimes[xSymbol] = time.Now()
				xHedgeMarkPrices[xSymbol] = xDepth.BestBidPrice
				yOrderSilentTimes[ySymbol] = time.Now()
				yHedgeMarkPrices[ySymbol] = yDepth.BestAskPrice
			}
			logger.Debugf(
				"%s %s SHORT BOT REDUCE %f < %f, %f < %f, SIZE %f, ENTER ORDER %s, MARK PRICE X %f, MARK PRICE Y %f",
				xSymbol, ySymbol,
				spread.ShortLastLeave, shortTop,
				spread.ShortMedianLeave, shortTop,
				size,
				xyEnterTradeOrders[xSymbol],
				xHedgeMarkPrices[xSymbol], yHedgeMarkPrices[ySymbol],
			)

		} else if spread.LongLastLeave > longTop &&
			spread.LongMedianLeave > longTop &&
			fundingRate > -xyConfig.MinimalKeepFundingRate &&
			xSize <= -xStepSize {

			entryValue := math.Min(4*entryStep, math.Min(xValue, yValue))
			if fundingRate < -xyConfig.MinimalKeepFundingRate*0.5 {
				entryValue = math.Min(2*entryStep, math.Min(xValue, yValue))
			}
			size := entryValue / midPrice
			size = math.Round(size/xyStepSize) * xyStepSize
			entryValue = size * midPrice
			if xValue-entryValue < entryStep {
				size = -xPosition.GetSize()
			}
			if yValue-entryValue < entryStep {
				size = -xPosition.GetSize()
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
			xyMergedDirs[xSymbol] = spread.XDir*xyConfig.XYDirRatio + spread.YDir*(1.0-xyConfig.XYDirRatio)

			if xyMergedDirs[xSymbol] < 0 {
				xyEnterTradeOrders[xSymbol] = EnterTradeOrderYX
				xOrderSilentTimes[xSymbol] = time.Now().Add(xyConfig.HedgeCheckInterval)
				xHedgeMarkPrices[xSymbol] = xDepth.BestAskPrice
				yOrderSilentTimes[ySymbol] = time.Now()
				yHedgeMarkPrices[ySymbol] = yDepth.BestBidPrice
			} else if xyMergedDirs[xSymbol] > 0 {
				xyEnterTradeOrders[xSymbol] = EnterTradeOrderXY
				xOrderSilentTimes[xSymbol] = time.Now()
				xHedgeMarkPrices[xSymbol] = xDepth.BestAskPrice
				yOrderSilentTimes[ySymbol] = time.Now().Add(xyConfig.HedgeCheckInterval)
				yHedgeMarkPrices[ySymbol] = yDepth.BestBidPrice
			} else {
				xyEnterTradeOrders[xSymbol] = EnterTradeOrderUnknown
				xOrderSilentTimes[xSymbol] = time.Now()
				xHedgeMarkPrices[xSymbol] = xDepth.BestAskPrice
				yOrderSilentTimes[ySymbol] = time.Now()
				yHedgeMarkPrices[ySymbol] = yDepth.BestBidPrice
			}

			logger.Debugf(
				"%s %s LONG TOP REDUCE %f > %f, %f > %f, SIZE %f, ENTER ORDER %s MARK PRICE X %f MARK PRICE Y %f",
				xSymbol, ySymbol,
				spread.LongLastLeave, longTop,
				spread.LongMedianLeave, longTop,
				size,
				xyEnterTradeOrders[xSymbol],
				xHedgeMarkPrices[xSymbol], yHedgeMarkPrices[ySymbol],
			)
		} else if spread.ShortLastEnter > shortTop &&
			spread.ShortMedianEnter > shortTop &&
			fundingRate > xyConfig.MinimalEnterFundingRate &&
			xSize >= 0 {

			targetValue := math.Max(xValue, yValue) + entryStep
			if targetValue > entryTarget {
				targetValue = entryTarget
			}
			entryValue := targetValue - math.Max(xValue, yValue)
			size := entryValue / midPrice
			size = math.Round(size/xyStepSize) * xyStepSize
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
			if xyMergedDirs[xSymbol] < 0 {
				xyEnterTradeOrders[xSymbol] = EnterTradeOrderYX
				xOrderSilentTimes[xSymbol] = time.Now().Add(xyConfig.HedgeCheckInterval)
				xHedgeMarkPrices[xSymbol] = xDepth.BestAskPrice
				yOrderSilentTimes[ySymbol] = time.Now()
				yHedgeMarkPrices[ySymbol] = yDepth.BestBidPrice
			} else if xyMergedDirs[xSymbol] > 0 {
				xyEnterTradeOrders[xSymbol] = EnterTradeOrderXY
				xOrderSilentTimes[xSymbol] = time.Now()
				xHedgeMarkPrices[xSymbol] = xDepth.BestAskPrice
				yOrderSilentTimes[ySymbol] = time.Now().Add(xyConfig.HedgeCheckInterval)
				yHedgeMarkPrices[ySymbol] = yDepth.BestBidPrice
			} else {
				xyEnterTradeOrders[xSymbol] = EnterTradeOrderUnknown
				xOrderSilentTimes[xSymbol] = time.Now()
				xHedgeMarkPrices[xSymbol] = xDepth.BestAskPrice
				yOrderSilentTimes[ySymbol] = time.Now()
				yHedgeMarkPrices[ySymbol] = yDepth.BestBidPrice
			}
			logger.Debugf(
				"%s %s SHORT TOP OPEN %f > %f, %f > %f, SIZE %f, ENTER ORDER %s, MARK PRICE X %f MARK PRICE Y %f",
				xSymbol, ySymbol,
				spread.ShortLastEnter, shortTop,
				spread.ShortMedianEnter, shortTop,
				size,
				xyEnterTradeOrders[xSymbol],
				xHedgeMarkPrices[xSymbol], yHedgeMarkPrices[ySymbol],
			)
		} else if spread.LongLastEnter < longBot &&
			spread.LongMedianEnter < longBot &&
			fundingRate < -xyConfig.MinimalEnterFundingRate &&
			xSize <= 0 {

			targetValue := math.Max(xValue, yValue) + entryStep
			if targetValue > entryTarget {
				targetValue = entryTarget
			}
			entryValue := targetValue - math.Max(xValue, yValue)
			size := entryValue / midPrice
			size = math.Round(size/xyStepSize) * xyStepSize
			entryValue = size * midPrice
			if entryValue > xyUSDTAvailable {
				if time.Now().Sub(time.Now().Truncate(xyConfig.LogInterval)) < xyConfig.LoopInterval {
					logger.Debugf(
						"%s %s FAILED SHORT TOP OPEN, ENTRY VALUE %f MORE THAN xyUSDTAvailable %f, %f < %f, %f < %f, SIZE %f",
						xSymbol,
						ySymbol,
						entryValue,
						xyUSDTAvailable,
						spread.LongLastEnter, longTop,
						spread.LongMedianEnter, longTop,
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
						spread.LongLastEnter, longTop,
						spread.LongMedianEnter, longTop,
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
			if xyMergedDirs[xSymbol] < 0 {
				xyEnterTradeOrders[xSymbol] = EnterTradeOrderXY
				xOrderSilentTimes[xSymbol] = time.Now()
				xHedgeMarkPrices[xSymbol] = xDepth.BestBidPrice
				yOrderSilentTimes[ySymbol] = time.Now().Add(xyConfig.HedgeCheckInterval)
				yHedgeMarkPrices[ySymbol] = yDepth.BestAskPrice
			} else if xyMergedDirs[xSymbol] > 0 {
				xyEnterTradeOrders[xSymbol] = EnterTradeOrderYX
				xOrderSilentTimes[xSymbol] = time.Now().Add(xyConfig.HedgeCheckInterval)
				xHedgeMarkPrices[xSymbol] = xDepth.BestBidPrice
				yOrderSilentTimes[ySymbol] = time.Now()
				yHedgeMarkPrices[ySymbol] = yDepth.BestAskPrice
			} else {
				xyEnterTradeOrders[xSymbol] = EnterTradeOrderUnknown
				xOrderSilentTimes[xSymbol] = time.Now()
				xHedgeMarkPrices[xSymbol] = xDepth.BestBidPrice
				yOrderSilentTimes[ySymbol] = time.Now()
				yHedgeMarkPrices[ySymbol] = yDepth.BestAskPrice
			}
			logger.Debugf(
				"%s %s LONG BOT OPEN %f < %f, %f < %f, SIZE %f, ENTER ORDER %s, MARK PRICE X %f, MARK PRICE Y %f",
				xSymbol, ySymbol,
				spread.LongLastEnter, longBot,
				spread.LongMedianEnter, longBot,
				size,
				xyEnterTradeOrders[xSymbol],
				xHedgeMarkPrices[xSymbol], yHedgeMarkPrices[ySymbol],
			)
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
	for i, makerSymbol := range xSymbols {
		takerSymbol := xySymbolsMap[makerSymbol]
		if makerFr, ok := xFundingRates[makerSymbol]; ok {
			if takerFr, ok := yFundingRates[takerSymbol]; ok {
				frs[i] = takerFr.GetFundingRate() - makerFr.GetFundingRate()
				xyFundingRates[makerSymbol] = frs[i]
			} else {
				logger.Debugf("MISS PREMIUM INDEX FOR TAKER %s", makerSymbol)
				return
			}
		} else {
			logger.Debugf("MISS FUNDING RATE FOR MAKER %s", makerSymbol)
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
