package main

import (
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"math/rand"
	"time"
)

func updateTakerPositions() {
	unHedgedValue := 0.0
	for _, takerSymbol := range ySymbols {
		makerSymbol := yxSymbolsMap[takerSymbol]
		if time.Now().Sub(yPositionsUpdateTimes[takerSymbol]) > xyConfig.BalancePositionMaxAge {
			continue
		}
		if time.Now().Sub(xPositionsUpdateTimes[makerSymbol]) > xyConfig.BalancePositionMaxAge {
			continue
		}
		if yOrderSilentTimes[takerSymbol].Sub(time.Now()).Seconds() > 0 {
			continue
		}

		makerPosition, okPosition := xPositions[makerSymbol]
		takerPosition, okTakerBalance := yPositions[takerSymbol]
		spread, okSpread := xySpreads[makerSymbol]
		if !okPosition || !okTakerBalance || !okSpread {
			continue
		}
		takerTakerDepth := spread.TakerDepth

		takerStepSize := yStepSizes[takerSymbol]
		takerMinNotional := yMinNotional[takerSymbol]

		makerSize := makerPosition.GetSize()

		takerSizeDiff := -makerSize - takerPosition.GetSize()
		takerSizeDiff = math.Round(takerSizeDiff/takerStepSize) * takerStepSize
		if takerSizeDiff > 0 {
			unHedgedValue += math.Abs(takerSizeDiff * takerTakerDepth.TakerAsk)
		} else {
			unHedgedValue += math.Abs(takerSizeDiff * takerTakerDepth.TakerBid)
		}

		if math.Abs(takerSizeDiff) < takerStepSize {
			continue
		} else if takerSizeDiff < 0 && takerPosition.GetSize() <= 0 && -takerSizeDiff*takerTakerDepth.TakerBid < takerMinNotional {
			continue
		} else if takerSizeDiff > 0 && takerPosition.GetSize() >= 0 && takerSizeDiff*takerTakerDepth.TakerAsk < takerMinNotional {
			continue
		}

		logger.Debugf("updateTakerPositions %s SIZE DIFF %f POS %f -> %f", takerSymbol, takerSizeDiff, takerPosition.GetSize(), -makerSize)

		reduceOnly := false
		if takerSizeDiff*takerPosition.GetSize() < 0 && math.Abs(takerSizeDiff) <= math.Abs(takerPosition.GetSize()) {
			reduceOnly = true
		}
		side := common.OrderSideBuy
		if takerSizeDiff < 0 {
			side = common.OrderSideSell
			takerSizeDiff = -takerSizeDiff
		}
		takerOrder := common.NewOrderParam{
			Symbol:     takerSymbol,
			Side:       side,
			Type:       common.OrderTypeMarket,
			Size:       takerSizeDiff,
			ReduceOnly: reduceOnly,
			ClientID:   yExchange.GenerateClientID(),
		}
		logger.Debugf("TAKER ORDER %v", takerOrder)

		hedgeMarkPrice, okHedgeMarkPrice := yHedgeMarkPrices[takerSymbol]
		if !xyConfig.HedgeInstantly && okHedgeMarkPrice {
			if takerOrder.Side == common.OrderSideBuy &&
				spread.TakerDir < 0 &&
				spread.TakerDepth.BestAskPrice < hedgeMarkPrice*(1.0-xyConfig.HedgeTrackOffset) {
				logger.Debugf(
					"%s taker change size %f dir %f mark price %f -> %f",
					takerSymbol, takerSizeDiff, spread.TakerDir,
					hedgeMarkPrice, spread.TakerDepth.BestAskPrice,
				)
				yHedgeMarkPrices[takerSymbol] = spread.TakerDepth.BestAskPrice
				yOrderSilentTimes[takerSymbol] = time.Now().Add(xyConfig.HedgeCheckInterval)
				xOrderSilentTimes[makerSymbol] = time.Now().Add(xyConfig.HedgeCheckInterval)
				yPositionsUpdateTimes[takerSymbol] = time.Now()
				xPositionsUpdateTimes[takerSymbol] = time.Now()
				continue
			} else if takerOrder.Side == common.OrderSideSell &&
				spread.TakerDir > 0 &&
				spread.TakerDepth.BestBidPrice > hedgeMarkPrice*(1.0+xyConfig.HedgeTrackOffset) {
				logger.Debugf(
					"%s taker change size %f dir %f mark price %f -> %f",
					takerSymbol, -takerSizeDiff, spread.TakerDir,
					hedgeMarkPrice, spread.TakerDepth.BestAskPrice,
				)
				yHedgeMarkPrices[takerSymbol] = spread.TakerDepth.BestBidPrice
				yOrderSilentTimes[takerSymbol] = time.Now().Add(xyConfig.HedgeCheckInterval)
				xOrderSilentTimes[makerSymbol] = time.Now().Add(xyConfig.HedgeCheckInterval)
				yPositionsUpdateTimes[takerSymbol] = time.Now()
				xPositionsUpdateTimes[takerSymbol] = time.Now()
				continue
			}
		}
		xOrderSilentTimes[makerSymbol] = time.Now().Add(xyConfig.OrderSilent)
		yOrderSilentTimes[takerSymbol] = time.Now().Add(xyConfig.OrderSilent)
		yPositionsUpdateTimes[takerSymbol] = time.Unix(0, 0)
		if !xyConfig.DryRun {
			yOrderRequestChMap[takerSymbol] <- common.OrderRequest{
				New: &takerOrder,
			}
		}
		if okHedgeMarkPrice {
			delete(yHedgeMarkPrices, takerSymbol)
		}
	}
	xyUnHedgeValue = unHedgedValue
}

func updateTargetPositionSizes() {

	if xAccount == nil {
		if time.Now().Sub(time.Now().Truncate(xyConfig.LogInterval)) < xyConfig.LoopInterval {
			logger.Debugf("mACCOUNT not ready")
		}
		return
	}
	if yAccount == nil {
		if time.Now().Sub(time.Now().Truncate(xyConfig.LogInterval)) < xyConfig.LoopInterval {
			logger.Debugf("tACCOUNT not ready")
		}
		return
	}

	//第一步，默认以X为准，对冲Y
	for xSymbol, ySymbol := range xySymbolsMap {
		//在信号触发期间，以信号为准
		if time.Now().Sub(xyTargetPositionUpdateSilentTimes[xSymbol]) < 0 {
			continue
		}
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
			logger.Debugf("taker unhedged value %f > %f", xyUnHedgeValue, xyConfig.MaxUnHedgeValue)
		}
		return
	}

	entryStep := (xAccount.GetFree() + yAccount.GetFree()) * xyConfig.EnterFreePct
	if entryStep < xyConfig.EnterMinimalStep {
		entryStep = xyConfig.EnterMinimalStep
	}
	entryTarget := entryStep * xyConfig.EnterTargetFactor

	//得是两个市场的最小可用资金, 以防有一边用完了钱, 开不了仓
	makerUSDTAvailable := math.Min(xAccount.GetFree()*xyConfig.XExchange.Leverage, yAccount.GetFree()*xyConfig.YExchange.Leverage)

	//遍历合约 从最大的rank 开始，能保证FR强的先下单, 优先做空
	for _, rank := range xyDualEnds {
		xSymbol := xyRankSymbolMap[rank]
		ySymbol := xySymbolsMap[xSymbol]

		spread, okSpread := xySpreads[xSymbol]
		//需要保证两边都有仓位更新，才调整现货仓位
		if time.Now().Sub(xPositionsUpdateTimes[xSymbol]) > xyConfig.BalancePositionMaxAge {
			//if time.Now().Sub(time.Now().Truncate(xyConfig.LogInterval)) < xyConfig.LoopInterval {
			//	logger.Debugf("maker position too old %s", xSymbol)
			//}
			continue
		}
		if time.Now().Sub(yPositionsUpdateTimes[ySymbol]) > xyConfig.BalancePositionMaxAge {
			//if time.Now().Sub(time.Now().Truncate(xyConfig.LogInterval)) < xyConfig.LoopInterval {
			//	logger.Debugf("taker position too old %s", xSymbol)
			//}
			continue
		}
		if time.Now().Sub(xyTargetPositionUpdateSilentTimes[xSymbol]) < 0 {
			//if time.Now().Sub(time.Now().Truncate(xyConfig.LogInterval)) < xyConfig.LoopInterval {
			//	logger.Debugf("taker order silent %s", xSymbol)
			//}
			continue
		}
		xPosition, okXPosition := xPositions[xSymbol]
		yPosition, okYPosition := yPositions[ySymbol]
		fundingRate, okFundingRate := xyFundingRates[xSymbol]
		if !okSpread || !okXPosition || !okYPosition || !okFundingRate {
			//if time.Now().Sub(time.Now().Truncate(xyConfig.LogInterval)) < xyConfig.LoopInterval {
			//	logger.Debugf("spread %v x position %v fundingRate %v %s", okSpread, okFundingRate, okXPosition, xSymbol)
			//}
			continue
		}

		if time.Now().Sub(spread.Time) > xyConfig.SpreadTimeToLive {
			//if time.Now().Sub(time.Now().Truncate(xyConfig.LogInterval)) < xyConfig.LoopInterval {
			//	logger.Debugf("spread too old %s %v", xSymbol, spread.Time)
			//}
			continue
		}
		xDepth := spread.XDepth
		yDepth := spread.YDepth
		xStepSize := xStepSizes[xSymbol]
		yStepSize := yStepSizes[ySymbol]

		xyStepSize := xyStepSizes[xSymbol]
		xValue := xPosition.GetSize() * xPosition.GetPrice()
		yValue := yPosition.GetSize() * yPosition.GetPrice()
		offsetFactor := (math.Abs(xValue) + math.Abs(yValue)) / entryTarget
		shortTop := xyConfig.ShortEnterDelta + xyConfig.EnterOffsetDelta*offsetFactor
		shortBot := xyConfig.ShortExitDelta
		longBot := xyConfig.LongEnterDelta - xyConfig.EnterOffsetDelta*offsetFactor
		longTop := xyConfig.LongExitDelta

		if spread.ShortLastLeave < shortBot &&
			spread.ShortMedianLeave < shortBot &&
			fundingRate < xyConfig.MinimalKeepFundingRate &&
			xPosition.GetSize() >= xStepSize {

			makerSize := xPosition.GetSize()
			price := math.Ceil(xDepth.MakerAsk*(1.0+offset.Top)/makerTickSize) * makerTickSize
			entryValue := math.Max(4*entryStep, makerSize*price*0.5)
			if fundingRate > xyConfig.MinimalKeepFundingRate*0.5 {
				entryValue = math.Max(2*entryStep, makerSize*price*0.5)
			}
			size := entryValue / price
			size = math.Round(size/xyStepSize) * xyStepSize
			entryValue = size * price
			if makerSize*price-entryValue < entryStep {
				size = xPosition.GetSize()
			}
			if size > 0 {
				logger.Debugf(
					"SHORT BOT REDUCE %s %f < %f, %f < %f, SIZE %f PRICE %f",
					xSymbol,
					spread.ShortLastLeave, shortBot,
					spread.ShortMedianLeave, shortBot,
					size, price,
				)

				order := common.NewOrderParam{
					Symbol:      xSymbol,
					Side:        common.OrderSideSell,
					Type:        common.OrderTypeLimit,
					Price:       price,
					TimeInForce: common.OrderTimeInForceGTC,
					Size:        size,
					PostOnly:    true,
					ReduceOnly:  true,
					ClientID:    xExchange.GenerateClientID(),
				}
				xOpenOrders[xSymbol] = order
				xOrderSilentTimes[xSymbol] = time.Now()
				xOrderSilentTimes[xSymbol] = time.Now().Add(xyConfig.OrderSilent)
				if !xyConfig.DryRun {
					xOrderRequestChMap[xSymbol] <- common.OrderRequest{New: &order}
				}
				return
			}
		} else if spread.LongLastLeave > longTop &&
			spread.LongMedianLeave > longTop &&
			fundingRate > -xyConfig.MinimalKeepFundingRate &&
			xPosition.GetSize() < 0 {

			makerSize := -xPosition.GetSize()
			price := math.Floor(xDepth.MakerBid*(1.0+offset.Bot)/makerTickSize) * makerTickSize
			entryValue := math.Max(4*entryStep, makerSize*price*0.5)
			if fundingRate < -xyConfig.MinimalKeepFundingRate/2 {
				entryValue = math.Max(2*entryStep, makerSize*price*0.5)
			}
			size := entryValue / price
			size = math.Round(size/xyStepSize) * xyStepSize
			if makerSize*price-entryValue < entryStep {
				size = -xPosition.GetSize()
			}
			if size > 0 {
				logger.Debugf(
					"LONG TOP REDUCE %s %f > %f, %f > %f, SIZE %f PRICE %f",
					xSymbol,
					spread.LongLastLeave, longTop,
					spread.LongMedianLeave, longTop,
					size, price,
				)
				order := common.NewOrderParam{
					Symbol:      xSymbol,
					Side:        common.OrderSideBuy,
					Type:        common.OrderTypeLimit,
					Price:       price,
					TimeInForce: common.OrderTimeInForceGTC,
					Size:        size,
					PostOnly:    true,
					ReduceOnly:  true,
					ClientID:    xExchange.GenerateClientID(),
				}
				xOpenOrders[xSymbol] = order
				xOrderSilentTimes[xSymbol] = time.Now()
				xOrderSilentTimes[xSymbol] = time.Now().Add(xyConfig.OrderSilent)
				if !xyConfig.DryRun {
					xOrderRequestChMap[xSymbol] <- common.OrderRequest{New: &order}
				}
				return
			}
		} else if spread.ShortLastEnter > shortTop &&
			spread.ShortMedianEnter > shortTop &&
			delta.ShortTop-delta.ShortBot > xyConfig.BasicLongEnterDelta &&
			fundingRate > xyConfig.MinimalEnterFundingRate &&
			xPosition.GetSize() >= 0 {
			makerSize := xPosition.GetSize()
			price := math.Floor(xDepth.MakerBid*(1.0+offset.Bot)/makerTickSize) * makerTickSize
			targetValue := makerSize*price + entryStep
			if targetValue > entryTarget {
				targetValue = entryTarget
			}
			entryValue := targetValue - makerSize*price
			if entryValue > makerUSDTAvailable*0.8 {
				entryValue = makerUSDTAvailable * 0.8
			}
			size := entryValue / price
			size = math.Round(size/xyStepSize) * xyStepSize

			entryValue = size * price

			////不及一个0.8*EntryStep, 不操作
			//if entryValue < entryStep*0.8 {
			//	if time.Now().Sub(xyLogSilentTimes[xSymbol]) > 0 {
			//		logger.Debugf(
			//			"FAILED SHORT TOP OPEN, ENTRY VALUE %f LESS THAN 0.8*ENTRY_STEP %f, %s %f > %f, %f > %f, SIZE %f PRICE %f",
			//			entryValue,
			//			entryStep*0.8,
			//			xSymbol,
			//			spread.ShortLastEnter, shortTop,
			//			spread.ShortMedianEnter, shortTop,
			//			size, price,
			//		)
			//		xyLogSilentTimes[xSymbol] = time.Now().Add(*xyConfig.LogInterval)
			//	}
			//	continue
			//}
			if entryValue > makerUSDTAvailable {
				if time.Now().Sub(xyLogSilentTimes[xSymbol]) > 0 {
					logger.Debugf(
						"FAILED SHORT TOP OPEN, ENTRY VALUE %f MORE THAN WithdrawAvailable %f, %s %f > %f, %f > %f, SIZE %f PRICE %f",
						entryValue,
						makerUSDTAvailable,
						xSymbol,
						spread.ShortLastEnter, shortTop,
						spread.ShortMedianEnter, shortTop,
						size, price,
					)
					xyLogSilentTimes[xSymbol] = time.Now().Add(xyConfig.LogInterval)
				}
				continue
			}
			if entryValue < takerMinNotional {
				if time.Now().Sub(xyLogSilentTimes[xSymbol]) > 0 {
					logger.Debugf(
						"FAILED SHORT TOP OPEN, ORDER VALUE %f LESS THAN NOTIONAL %f, %s %f > %f, %f > %f, SIZE %f PRICE %f",
						entryValue,
						takerMinNotional,
						xSymbol,
						spread.ShortLastEnter, shortTop,
						spread.ShortMedianEnter, shortTop,
						size, price,
					)
					xyLogSilentTimes[xSymbol] = time.Now().Add(xyConfig.LogInterval)
				}
				continue
			}
			xyLogSilentTimes[xSymbol] = time.Now()
			logger.Debugf(
				"SHORT TOP OPEN %s %f > %f, %f > %f, SIZE %f PRICE %f",
				xSymbol,
				spread.ShortLastEnter, shortTop,
				spread.ShortMedianEnter, shortTop,
				size, price,
			)
			makerUSDTAvailable -= entryValue
			order := common.NewOrderParam{
				Symbol:      xSymbol,
				Side:        common.OrderSideBuy,
				Type:        common.OrderTypeLimit,
				Price:       price,
				TimeInForce: common.OrderTimeInForceGTC,
				Size:        size,
				PostOnly:    true,
				ReduceOnly:  false,
				ClientID:    xExchange.GenerateClientID(),
			}
			xOpenOrders[xSymbol] = order
			xOrderSilentTimes[xSymbol] = time.Now()
			xOrderSilentTimes[xSymbol] = time.Now().Add(xyConfig.OrderSilent)
			if !xyConfig.DryRun {
				xOrderRequestChMap[xSymbol] <- common.OrderRequest{New: &order}
			}
		} else if spread.LongLastEnter < longBot &&
			spread.LongMedianEnter < longBot &&
			delta.LongTop-delta.LongBot > xyConfig.BasicLongEnterDelta &&
			fundingRate < -xyConfig.MinimalEnterFundingRate &&
			xPosition.GetSize() <= 0 {

			makerSize := -xPosition.GetSize()
			price := math.Ceil(xDepth.MakerAsk*(1.0+offset.Top)/makerTickSize) * makerTickSize
			targetValue := makerSize*price + entryStep
			if targetValue > entryTarget {
				targetValue = entryTarget
			}
			entryValue := targetValue - makerSize*price
			if entryValue > makerUSDTAvailable*0.8 {
				entryValue = makerUSDTAvailable * 0.8
			}
			size := entryValue / price
			size = math.Round(size/xyStepSize) * xyStepSize

			entryValue = size * price

			//不及一个0.8*EntryStep, 不操作
			//if entryValue < entryStep*0.8 {
			//	if time.Now().Sub(xyLogSilentTimes[xSymbol]) > 0 {
			//		logger.Debugf(
			//			"FAILED LONG BOT OPEN, ENTRY VALUE %f LESS THAN 0.8*ENTRY_STEP %f, %s %f < %f, %f < %f, SIZE %f",
			//			entryValue,
			//			entryStep*0.8,
			//			xSymbol,
			//			spread.LongLastEnter, ongBot,
			//			spread.LongMedianEnter, quantile.LongBot,
			//			size,
			//		)
			//		xyLogSilentTimes[xSymbol] = time.Now().Add(*xyConfig.LogInterval)
			//	}
			//	continue
			//}
			if entryValue > makerUSDTAvailable {
				if time.Now().Sub(xyLogSilentTimes[xSymbol]) > 0 {
					logger.Debugf(
						"FAILED LONG BOT OPEN, ENTRY VALUE %f MORE THAN WithdrawAvailable %f, %s %f < %f, %f < %f, SIZE %f PRICE %f",
						entryValue,
						makerUSDTAvailable,
						xSymbol,
						spread.LongLastEnter, longBot,
						spread.LongMedianEnter, longBot,
						size, price,
					)
					xyLogSilentTimes[xSymbol] = time.Now().Add(xyConfig.LogInterval)
				}
				continue
			}
			if entryValue < takerMinNotional {
				if time.Now().Sub(xyLogSilentTimes[xSymbol]) > 0 {
					logger.Debugf(
						"FAILED LONG BOT OPEN, ORDER VALUE %f LESS THAN NOTIONAL %f, %s %f < %f, %f < %f, SIZE %f PRICE %f",
						entryValue,
						takerMinNotional,
						xSymbol,
						spread.LongLastEnter, longBot,
						spread.LongMedianEnter, longBot,
						size, price,
					)
					xyLogSilentTimes[xSymbol] = time.Now().Add(xyConfig.LogInterval)
				}
				continue
			}
			xyLogSilentTimes[xSymbol] = time.Now()
			logger.Debugf(
				"LONG BOT OPEN %s %f < %f, %f < %f, SIZE %f PRICE %f",
				xSymbol,
				spread.LongLastEnter, longBot,
				spread.LongMedianEnter, longBot,
				size, price,
			)
			makerUSDTAvailable -= entryValue
			order := common.NewOrderParam{
				Symbol:      xSymbol,
				Side:        common.OrderSideSell,
				Type:        common.OrderTypeLimit,
				Price:       price,
				TimeInForce: common.OrderTimeInForceGTC,
				Size:        size,
				PostOnly:    true,
				ReduceOnly:  false,
				ClientID:    fmt.Sprintf("%d%04d", time.Now().Unix(), rand.Intn(10000)),
			}
			xOpenOrders[xSymbol] = order
			xOrderSilentTimes[xSymbol] = time.Now()
			xOrderSilentTimes[xSymbol] = time.Now().Add(xyConfig.OrderSilent)
			if !xyConfig.DryRun {
				xOrderRequestChMap[xSymbol] <- common.OrderRequest{New: &order}
			}
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
