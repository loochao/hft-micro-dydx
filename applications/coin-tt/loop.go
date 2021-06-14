package main

import (
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"time"
)

func hedgeYSymbol(ySymbol, xSymbol string) float64 {
	yPosition, okYPosition := yPositions[ySymbol]
	targetContractValue, okTargetContractValue := yTargetContractValues[ySymbol]
	spread, okSpread := xySpreads[xSymbol]
	if !okYPosition || !okSpread || !okTargetContractValue {
		return 0
	}

	yDepth := spread.YDepth
	yStepSize := yStepSizes[ySymbol]
	yMinNotional := yMinNotionals[ySymbol]
	yMultiplier := yMultipliers[ySymbol]
	ySizeDiff := targetContractValue/yMultiplier - yPosition.GetSize()
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
	return math.Abs(ySizeDiff * yMultiplier)
}

func updateYPositions() {
	unHedgedValue := 0.0
	for _, ySymbol := range ySymbols {
		xSymbol := yxSymbolsMap[ySymbol]
		if _, ok := xBalances[xyConfig.XSymbolAssetMap[xSymbol]]; !ok {
			if time.Now().Sub(time.Now().Truncate(xyConfig.LogInterval)) < xyConfig.LoopInterval {
				logger.Debugf("xAccount not ready for %s", xSymbol)
			}
			continue
		}
		if _, ok := yBalances[xyConfig.YSymbolAssetMap[ySymbol]]; !ok {
			if time.Now().Sub(time.Now().Truncate(xyConfig.LogInterval)) < xyConfig.LoopInterval {
				logger.Debugf("yAccount not ready for %s", ySymbol)
			}
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
	xTargetContractValue, okXTargetContractValue := xTargetContractValues[xSymbol]
	spread, okSpread := xySpreads[xSymbol]
	if !okXPosition || !okSpread || !okXTargetContractValue {
		return
	}
	xDepth := spread.XDepth
	xStepSize := xStepSizes[xSymbol]
	xMinNotional := xMinNotionals[xSymbol]
	xMultiplier := xMultipliers[xSymbol]
	xSizeDiff := xTargetContractValue/xMultiplier - xPosition.GetSize()
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

	//logger.Debugf("updateXPositions %s size %f position %f -> %f", xSymbol, xSizeDiff, xPosition.GetSize(), xTargetContractValue)

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
	for _, xSymbol := range xSymbols {
		ySymbol := xySymbolsMap[xSymbol]
		if _, ok := xBalances[xyConfig.XSymbolAssetMap[xSymbol]]; !ok {
			if time.Now().Sub(time.Now().Truncate(xyConfig.LogInterval)) < xyConfig.LoopInterval {
				logger.Debugf("xAccount not ready for %s", xSymbol)
			}
			continue
		}
		if _, ok := yBalances[xyConfig.YSymbolAssetMap[ySymbol]]; !ok {
			if time.Now().Sub(time.Now().Truncate(xyConfig.LogInterval)) < xyConfig.LoopInterval {
				logger.Debugf("yBalance not ready for %s", ySymbol)
			}
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

	//第一步，默认以X为准，对冲Y
	for xSymbol, ySymbol := range xySymbolsMap {
		xBalance, ok := xBalances[xyConfig.XSymbolAssetMap[xSymbol]]
		if !ok {
			if time.Now().Sub(time.Now().Truncate(xyConfig.LogInterval)) < xyConfig.LoopInterval {
				logger.Debugf("xBalance not ready for %s", xSymbol)
			}
			continue
		}
		yBalance, ok := yBalances[xyConfig.YSymbolAssetMap[ySymbol]]
		if !ok {
			if time.Now().Sub(time.Now().Truncate(xyConfig.LogInterval)) < xyConfig.LoopInterval {
				logger.Debugf("yBalance not ready for %s", ySymbol)
			}
			continue
		}
		//在信号触发期间，以信号为准
		if time.Now().Sub(xyTargetPositionUpdateSilentTimes[xSymbol]) < 0 {
			continue
		}

		_, okXPosition := xPositions[xSymbol]
		yPosition, okYPosition := yPositions[ySymbol]
		yMultiplier := yMultipliers[ySymbol]
		spread, okSpread := xySpreads[xSymbol]

		//以y调整x
		if okXPosition && okYPosition && okSpread {
			spotValue := spread.XDepth.MidPrice*xBalance.GetBalance() + spread.YDepth.MidPrice*yBalance.GetBalance()
			yValue := yPosition.GetSize() * yMultiplier
			yTargetContractValues[ySymbol] = yValue
			xTargetContractValues[xSymbol] = -spotValue - yValue
			if time.Now().Sub(time.Now().Truncate(xyConfig.LogInterval)) < xyConfig.LoopInterval {
				logger.Debugf("target x %f y %f", -spotValue-yValue, yValue)
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

	//遍历合约 从最大的rank 开始，能保证FR强的先下单, 优先做空
	for _, rank := range xyDualEnds {
		xSymbol := xyRankSymbolMap[rank]
		ySymbol := xySymbolsMap[xSymbol]
		xBalance, ok := xBalances[xyConfig.XSymbolAssetMap[xSymbol]]
		if !ok {
			if time.Now().Sub(time.Now().Truncate(xyConfig.LogInterval)) < xyConfig.LoopInterval {
				logger.Debugf("xBalance not ready for %s", xSymbol)
			}
			continue
		}
		yBalance, ok := yBalances[xyConfig.YSymbolAssetMap[ySymbol]]
		if !ok {
			if time.Now().Sub(time.Now().Truncate(xyConfig.LogInterval)) < xyConfig.LoopInterval {
				logger.Debugf("yBalance not ready for %s", ySymbol)
			}
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
		_, okXPosition := xPositions[xSymbol]
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

		coinUSDAvailable := math.Min(xBalance.GetFree()*xDepth.MidPrice*xyConfig.XExchange.Leverage, yBalance.GetFree()*yDepth.MidPrice*xyConfig.YExchange.Leverage)

		yMultiplier := yMultipliers[ySymbol]

		xyUsdStepSize := xyUsdStepSizes[xSymbol]

		ySize := yPosition.GetSize()
		yValue := math.Abs(ySize*yMultiplier) * spread.YDepth.MidPrice
		spotValue := xBalance.GetBalance()*xDepth.MidPrice + yBalance.GetBalance()*yDepth.MidPrice

		maxYTargetValue := math.Round(spotValue * xyConfig.EnterTarget)

		offsetFactor := yValue / spotValue / xyConfig.EnterTarget
		offsetStep := math.Min(xyConfig.EnterStep/xyConfig.EnterTarget, offsetFactor)
		if time.Now().Sub(time.Now().Truncate(xyConfig.LogInterval)) < xyConfig.LoopInterval {
			logger.Debugf("%s offset factor %f offset step %f", xSymbol, offsetFactor, offsetStep)
		}

		shortTop := xyConfig.ShortEnterDelta + xyConfig.EnterOffsetDelta*offsetFactor
		shortBot := xyConfig.ShortExitDelta + xyConfig.ExitOffsetDelta*(offsetFactor-offsetStep)
		longBot := xyConfig.LongEnterDelta - xyConfig.EnterOffsetDelta*offsetFactor
		longTop := xyConfig.LongExitDelta - xyConfig.ExitOffsetDelta*(offsetFactor-offsetStep)
		entryStep := spotValue * xyConfig.EnterStep

		if spread.ShortLastLeave < shortBot &&
			spread.ShortMedianLeave < shortBot &&
			fundingRate < xyConfig.MinimalKeepFundingRate &&
			ySize < 0 {

			entryValue := math.Min(2*entryStep, yValue)
			entryValue = math.Round(entryValue/xyUsdStepSize) * xyUsdStepSize
			if yValue-entryValue < xyUsdStepSize {
				entryValue = yValue
			}
			yTargetContractValues[ySymbol] += entryValue
			xTargetContractValues[xSymbol] = -spotValue - yTargetContractValues[ySymbol]
			xyTargetPositionUpdateSilentTimes[xSymbol] = time.Now().Add(xyConfig.EnterSilent)
			xOrderSilentTimes[xSymbol] = time.Now()
			yOrderSilentTimes[ySymbol] = time.Now()
			delete(xLastFilledBuyPrices, xSymbol)
			delete(xLastFilledSellPrices, xSymbol)
			delete(yLastFilledBuyPrices, ySymbol)
			delete(yLastFilledSellPrices, ySymbol)
			logger.Debugf(
				"%s %s SHORT BOT REDUCE %f < %f, %f < %f, VALUE %f, TARGET X %f TARGET Y %f",
				xSymbol, ySymbol,
				spread.ShortLastLeave, shortBot,
				spread.ShortMedianLeave, shortBot,
				entryValue,
				xTargetContractValues[xSymbol],
				yTargetContractValues[ySymbol],
			)
			hedgeXSymbol(xSymbol)
			hedgeYSymbol(ySymbol, xSymbol)
		} else if spread.LongLastLeave > longTop &&
			spread.LongMedianLeave > longTop &&
			fundingRate > -xyConfig.MinimalKeepFundingRate &&
			ySize > 0 {

			entryValue := math.Min(2*entryStep, yValue)
			entryValue = math.Round(entryValue/xyUsdStepSize) * xyUsdStepSize
			if yValue-entryValue < xyUsdStepSize {
				entryValue = yValue
			}
			yTargetContractValues[ySymbol] -= entryValue
			xTargetContractValues[xSymbol] = -spotValue - yTargetContractValues[ySymbol]

			xyTargetPositionUpdateSilentTimes[xSymbol] = time.Now().Add(xyConfig.EnterSilent)
			xOrderSilentTimes[xSymbol] = time.Now()
			yOrderSilentTimes[ySymbol] = time.Now()
			delete(xLastFilledBuyPrices, xSymbol)
			delete(xLastFilledSellPrices, xSymbol)
			delete(yLastFilledBuyPrices, ySymbol)
			delete(yLastFilledSellPrices, ySymbol)
			logger.Debugf(
				"%s %s LONG TOP REDUCE %f > %f, %f > %f, VALUE %f, TARGET X %f, TARGET Y %f",
				xSymbol, ySymbol,
				spread.LongLastLeave, longTop,
				spread.LongMedianLeave, longTop,
				entryValue,
				xTargetContractValues[xSymbol],
				yTargetContractValues[ySymbol],
			)
			hedgeXSymbol(xSymbol)
			hedgeYSymbol(ySymbol, xSymbol)
		} else if !yExchange.IsSpot() &&
			spread.ShortLastEnter > shortTop &&
			spread.ShortMedianEnter > shortTop &&
			fundingRate > xyConfig.MinimalEnterFundingRate &&
			ySize <= 0 {

			targetYValue := yValue + math.Max(entryStep, xyUsdStepSize)
			if targetYValue > maxYTargetValue {
				targetYValue = maxYTargetValue
			}
			entryValue := targetYValue - yValue
			entryValue = math.Round(entryValue/xyUsdStepSize) * xyUsdStepSize
			if entryValue > coinUSDAvailable {
				if time.Now().Sub(time.Now().Truncate(xyConfig.LogInterval)) < xyConfig.LoopInterval {
					logger.Debugf(
						"%s %s FAILED SHORT TOP OPEN, ENTRY VALUE %f MORE THAN coinUSDAvailable %f, %f > %f, %f > %f",
						xSymbol,
						ySymbol,
						entryValue,
						coinUSDAvailable,
						spread.ShortLastEnter, shortTop,
						spread.ShortMedianEnter, shortTop,
					)
				}
				continue
			}

			yTargetContractValues[ySymbol] -= entryValue
			xTargetContractValues[xSymbol] = -spotValue - yTargetContractValues[ySymbol]

			xyTargetPositionUpdateSilentTimes[xSymbol] = time.Now().Add(xyConfig.EnterSilent)
			coinUSDAvailable -= entryValue
			xOrderSilentTimes[xSymbol] = time.Now()
			yOrderSilentTimes[ySymbol] = time.Now()
			delete(xLastFilledBuyPrices, xSymbol)
			delete(xLastFilledSellPrices, xSymbol)
			delete(yLastFilledBuyPrices, ySymbol)
			delete(yLastFilledSellPrices, ySymbol)
			logger.Debugf(
				"%s %s SHORT TOP OPEN %f > %f, %f > %f, VALUE %f, TARGET X %f, TARGET Y %f",
				xSymbol, ySymbol,
				spread.ShortLastEnter, shortTop,
				spread.ShortMedianEnter, shortTop,
				entryValue,
				xTargetContractValues[xSymbol],
				yTargetContractValues[ySymbol],
			)
			hedgeXSymbol(xSymbol)
			hedgeYSymbol(ySymbol, xSymbol)
		} else if !xExchange.IsSpot() &&
			spread.LongLastEnter < longBot &&
			spread.LongMedianEnter < longBot &&
			fundingRate < -xyConfig.MinimalEnterFundingRate &&
			ySize >= 0 {

			targetYValue := yValue + math.Max(entryStep, xyUsdStepSize)
			if targetYValue > maxYTargetValue {
				targetYValue = maxYTargetValue
			}
			entryValue := targetYValue - yValue
			entryValue = math.Round(entryValue/xyUsdStepSize) * xyUsdStepSize

			if entryValue > coinUSDAvailable {
				if time.Now().Sub(time.Now().Truncate(xyConfig.LogInterval)) < xyConfig.LoopInterval {
					logger.Debugf(
						"%s %s FAILED SHORT TOP OPEN, ENTRY VALUE %f MORE THAN coinUSDAvailable %f, %f < %f, %f < %f",
						xSymbol,
						ySymbol,
						entryValue,
						coinUSDAvailable,
						spread.LongLastEnter, longBot,
						spread.LongMedianEnter, longBot,
					)
				}
				continue
			}

			yTargetContractValues[ySymbol] -= entryValue
			xTargetContractValues[xSymbol] = -spotValue - yTargetContractValues[ySymbol]

			xyTargetPositionUpdateSilentTimes[xSymbol] = time.Now().Add(xyConfig.EnterSilent)
			coinUSDAvailable -= entryValue
			xOrderSilentTimes[xSymbol] = time.Now()
			yOrderSilentTimes[ySymbol] = time.Now()
			delete(xLastFilledBuyPrices, xSymbol)
			delete(xLastFilledSellPrices, xSymbol)
			delete(yLastFilledBuyPrices, ySymbol)
			delete(yLastFilledSellPrices, ySymbol)
			logger.Debugf(
				"%s %s LONG BOT OPEN %f < %f, %f < %f, VALUE %f, TARGET X %f, TARGET Y %f",
				xSymbol, ySymbol,
				spread.LongLastEnter, longBot,
				spread.LongMedianEnter, longBot,
				entryValue,
				xTargetContractValues[xSymbol],
				yTargetContractValues[ySymbol],
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
