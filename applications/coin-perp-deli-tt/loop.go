package main

import (
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"time"
)

func hedgeSymbol(symbol string) {
	position, okPosition := xyPositions[symbol]
	targetValue, okTargetSize := xyTargetValues[symbol]
	if !okPosition || !okTargetSize {
		return
	}
	stepSize := xyStepSizes[symbol]
	multiplier := xyMultipliers[symbol]
	sizeDiff := math.Round(targetValue/multiplier - position.GetSize())
	sizeDiff = math.Round(sizeDiff/stepSize) * stepSize
	if math.Abs(sizeDiff) < stepSize {
		return
	}
	logger.Debugf("updatePositions %s size %f position %f -> %f", symbol, sizeDiff, position.GetSize(), targetValue/multiplier)

	reduceOnly := false
	if sizeDiff*position.GetSize() < 0 && math.Abs(sizeDiff) <= math.Abs(position.GetSize()) {
		reduceOnly = true
	}
	side := common.OrderSideBuy
	if sizeDiff < 0 {
		side = common.OrderSideSell
		sizeDiff = -sizeDiff
	}
	order := common.NewOrderParam{
		Symbol:     symbol,
		Side:       side,
		Type:       common.OrderTypeMarket,
		Size:       sizeDiff,
		ReduceOnly: reduceOnly,
		ClientID:   xyExchange.GenerateClientID(),
	}
	logger.Debugf("%s order %v", symbol, order)
	if xyConfig.DryRun {
		xyOrderSilentTimes[symbol] = time.Now().Add(xyConfig.OrderSilent)
		xyPositionsUpdateTimes[symbol] = time.Unix(0, 0)
	} else {
		select {
		case xyOrderRequestChMap[symbol] <- common.OrderRequest{
			New: &order,
		}:
			xyOrderSilentTimes[symbol] = time.Now().Add(xyConfig.OrderSilent)
			xyPositionsUpdateTimes[symbol] = time.Unix(0, 0)
		default:
			logger.Debugf("yOrderRequestChMap[symbol] <- common.OrderRequest %s failed, ch len %d", symbol, len(xyOrderRequestChMap[symbol]))
		}
	}
}

func updateYPositions() {
	for _, ySymbol := range ySymbols {
		xSymbol := yxSymbolsMap[ySymbol]
		xAsset := xyConfig.SymbolAssetMap[xSymbol]
		_, okBalance := xyBalanceMap[xAsset]
		_, okSpread := xySpreads[xSymbol]
		if !okSpread {
			continue
		}
		if !okBalance || !okSpread {
			if time.Now().Sub(time.Now().Truncate(xyConfig.LogInterval)) < xyConfig.LoopInterval {
				logger.Debugf("okBalance %v || okSpread %v", okBalance, okSpread)
			}
			continue
		}
		if time.Now().Sub(xyPositionsUpdateTimes[ySymbol]) > xyConfig.BalancePositionMaxAge {
			continue
		}
		if xyOrderSilentTimes[ySymbol].Sub(time.Now()).Seconds() > 0 {
			continue
		}
		hedgeSymbol(ySymbol)
	}
}

func updateXPositions() {
	for _, xSymbol := range xSymbols {
		xAsset := xyConfig.SymbolAssetMap[xSymbol]
		_, okBalance := xyBalanceMap[xAsset]
		_, okSpread := xySpreads[xSymbol]
		if !okSpread {
			continue
		}
		if !okBalance || !okSpread {
			if time.Now().Sub(time.Now().Truncate(xyConfig.LogInterval)) < xyConfig.LoopInterval {
				logger.Debugf("okBalance %v || okSpread %v", okBalance, okSpread)
			}
			continue
		}
		if time.Now().Sub(xyPositionsUpdateTimes[xSymbol]) > xyConfig.BalancePositionMaxAge {
			continue
		}
		if xyOrderSilentTimes[xSymbol].Sub(time.Now()).Seconds() > 0 {
			continue
		}
		hedgeSymbol(xSymbol)
	}
}

func updateTargetPositionSizes() {

	for xSymbol, ySymbol := range xySymbolsMap {
		//在信号触发期间，以信号为准
		if time.Now().Sub(xyTargetPositionUpdateSilentTimes[xSymbol]) < 0 {
			continue
		}
		spread, okSpread := xySpreads[xSymbol]
		if !okSpread {
			continue
		}
		xAsset := xyConfig.SymbolAssetMap[xSymbol]
		xyBalance, okBalance := xyBalanceMap[xAsset]
		yPosition, okYPosition := xyPositions[ySymbol]
		if okYPosition && okBalance {
			//以y调整x
			spotValue := spread.XDepth.MidPrice * xyBalance.GetBalance()
			yValue := yPosition.GetSize() * xyMultipliers[ySymbol]
			xyTargetValues[ySymbol] = yValue
			xyTargetValues[xSymbol] = -spotValue - yValue
			//logger.Debugf("target x %f y %f", -spotValue-yValue, yValue)
		}
	}

	for _, xSymbol := range xSymbols {

		ySymbol := xySymbolsMap[xSymbol]
		xAsset := xyConfig.SymbolAssetMap[xSymbol]
		xyBalance, okBalance := xyBalanceMap[xAsset]
		if !okBalance {
			if time.Now().Sub(time.Now().Truncate(xyConfig.LogInterval)) < xyConfig.LoopInterval {
				logger.Debugf("okBalance %v ", okBalance)
			}
			continue
		}

		spread, okSpread := xySpreads[xSymbol]
		//需要保证两边都有仓位更新，才调整现货仓位
		if time.Now().Sub(xyPositionsUpdateTimes[xSymbol]) > xyConfig.BalancePositionMaxAge {
			if time.Now().Sub(time.Now().Truncate(xyConfig.LogInterval)) < xyConfig.LoopInterval {
				logger.Debugf("%s x position too old", xSymbol)
			}
			continue
		}
		if time.Now().Sub(xyPositionsUpdateTimes[ySymbol]) > xyConfig.BalancePositionMaxAge {
			if time.Now().Sub(time.Now().Truncate(xyConfig.LogInterval)) < xyConfig.LoopInterval {
				logger.Debugf("%s y position too old", ySymbol)
			}
			continue
		}
		if time.Now().Sub(xyTargetPositionUpdateSilentTimes[xSymbol]) < 0 {
			if time.Now().Sub(time.Now().Truncate(xyConfig.LogInterval)) < xyConfig.LoopInterval {
				logger.Debugf("%s %s in target update silent", xSymbol, ySymbol)
			}
			continue
		}
		yPosition, okYPosition := xyPositions[ySymbol]
		if !okSpread || !okYPosition {
			if time.Now().Sub(time.Now().Truncate(xyConfig.LogInterval)) < xyConfig.LoopInterval {
				logger.Debugf("%s %s spread %v y position %v", xSymbol, ySymbol, okSpread, okYPosition)
			}
			continue
		}

		if time.Now().Sub(spread.Time) > xyConfig.SpreadTimeToLive {
			if time.Now().Sub(time.Now().Truncate(xyConfig.LogInterval)) < xyConfig.LoopInterval {
				logger.Debugf("%s %s spread too old %v", xSymbol, ySymbol, spread.Time)
			}
			continue
		}
		xDepth := spread.XDepth

		xStepSize := xyStepSizes[xSymbol]
		yStepSize := xyStepSizes[ySymbol]
		xMultiplier := xyMultipliers[xSymbol]
		yMultiplier := xyMultipliers[ySymbol]
		xyStepSize := common.MergedStepSize(xStepSize, yStepSize)

		if xyBalance.GetFree()*xDepth.MidPrice < xyStepSize {
			if time.Now().Sub(time.Now().Truncate(xyConfig.LogInterval)) < xyConfig.LoopInterval {
				logger.Debugf("%s %s free balance %s too small", xSymbol, ySymbol, xyBalance.GetFree()*xDepth.MidPrice)
			}
			continue
		}

		ySize := yPosition.GetSize()
		yValue := math.Abs(ySize) * yMultiplier
		spotValue := xyBalance.GetBalance() * xDepth.MidPrice
		maxYTargetValue := math.Round(spotValue * xyConfig.EnterTarget)

		offsetFactor := yValue / spotValue / xyConfig.EnterTarget
		offsetStep := math.Min(xyConfig.EnterStep/xyConfig.EnterTarget, offsetFactor)
		if time.Now().Sub(time.Now().Truncate(xyConfig.LogInterval)) < xyConfig.LoopInterval {
			logger.Debugf("%s offset factor %f step %f", xSymbol, offsetFactor, offsetStep)
		}

		shortTop := xyConfig.ShortEnterDelta + xyConfig.EnterOffsetDelta*offsetFactor
		shortBot := xyConfig.ShortExitDelta + xyConfig.ExitOffsetDelta*(offsetFactor - offsetStep)
		longBot := xyConfig.LongEnterDelta - xyConfig.EnterOffsetDelta*offsetFactor
		longTop := xyConfig.LongExitDelta - xyConfig.ExitOffsetDelta*(offsetFactor - offsetStep)

		if spread.ShortLastLeave < shortBot &&
			spread.ShortMedianLeave < shortBot &&
			ySize < 0 {
			//如果y还有仓位，还可以平仓
			entryValue := math.Round(math.Min(spotValue*xyConfig.EnterStep, yValue))
			xyTargetValues[ySymbol] += entryValue
			xyTargetValues[xSymbol] = -spotValue - xyTargetValues[ySymbol]
			xyTargetPositionUpdateSilentTimes[xSymbol] = time.Now().Add(xyConfig.EnterSilent)
			xyOrderSilentTimes[xSymbol] = time.Now()
			xyOrderSilentTimes[ySymbol] = time.Now()
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
				xyTargetValues[xSymbol],
				xyTargetValues[ySymbol],
			)
			hedgeSymbol(xSymbol)
			hedgeSymbol(ySymbol)
		} else if spread.LongLastLeave > longTop &&
			spread.LongMedianLeave > longTop &&
			ySize > 0 {

			entryValue := math.Round(math.Min(spotValue*xyConfig.EnterStep, yValue))
			xyTargetValues[ySymbol] -= entryValue
			xyTargetValues[xSymbol] = -spotValue - xyTargetValues[ySymbol]
			xyTargetPositionUpdateSilentTimes[xSymbol] = time.Now().Add(xyConfig.EnterSilent)
			xyOrderSilentTimes[xSymbol] = time.Now()
			xyOrderSilentTimes[ySymbol] = time.Now()
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
				xyTargetValues[xSymbol],
				xyTargetValues[ySymbol],
			)
			hedgeSymbol(xSymbol)
			hedgeSymbol(ySymbol)
		} else if spread.ShortLastEnter > shortTop &&
			spread.ShortMedianEnter > shortTop &&
			ySize <= 0 {
			targetYValue := yValue + math.Max(math.Round(spotValue*xyConfig.EnterStep), common.MergedStepSize(xMultiplier, yMultiplier))
			if targetYValue > maxYTargetValue {
				targetYValue = maxYTargetValue
			}
			entryValue := targetYValue - yValue
			xyTargetValues[ySymbol] -= entryValue
			xyTargetValues[xSymbol] = -spotValue - xyTargetValues[ySymbol]
			xyTargetPositionUpdateSilentTimes[xSymbol] = time.Now().Add(xyConfig.EnterSilent)
			xyOrderSilentTimes[xSymbol] = time.Now()
			xyOrderSilentTimes[ySymbol] = time.Now()
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
				xyTargetValues[xSymbol],
				xyTargetValues[ySymbol],
			)
			hedgeSymbol(xSymbol)
			hedgeSymbol(ySymbol)
		} else if spread.LongLastEnter < longBot &&
			spread.LongMedianEnter < longBot &&
			ySize >= 0 {

			targetYValue := yValue + math.Max(math.Round(spotValue*xyConfig.EnterStep), common.MergedStepSize(xMultiplier, yMultiplier))
			if targetYValue > maxYTargetValue {
				targetYValue = maxYTargetValue
			}
			entryValue := targetYValue - yValue
			xyTargetValues[ySymbol] += entryValue
			xyTargetValues[xSymbol] = -spotValue - xyTargetValues[ySymbol]
			xyTargetPositionUpdateSilentTimes[xSymbol] = time.Now().Add(xyConfig.EnterSilent)
			xyOrderSilentTimes[xSymbol] = time.Now()
			xyOrderSilentTimes[ySymbol] = time.Now()
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
				xyTargetValues[xSymbol],
				xyTargetValues[ySymbol],
			)
			hedgeSymbol(xSymbol)
			hedgeSymbol(ySymbol)
		}
	}
}
