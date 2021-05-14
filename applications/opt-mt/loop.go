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
	for _, takerSymbol := range tSymbols {
		makerSymbol := tmSymbolsMap[takerSymbol]
		if time.Now().Sub(tPositionsUpdateTimes[takerSymbol]) > mtConfig.BalancePositionMaxAge {
			continue
		}
		if time.Now().Sub(mPositionsUpdateTimes[makerSymbol]) > mtConfig.BalancePositionMaxAge {
			continue
		}
		if tOrderSilentTimes[takerSymbol].Sub(time.Now()).Seconds() > 0 {
			continue
		}

		makerPosition, okPosition := mPositions[makerSymbol]
		takerPosition, okTakerBalance := tPositions[takerSymbol]
		spread, okSpread := mtSpreads[makerSymbol]
		if !okPosition || !okTakerBalance || !okSpread {
			continue
		}
		takerTakerDepth := spread.TakerDepth

		takerStepSize := tStepSizes[takerSymbol]
		takerMinNotional := tMinNotional[takerSymbol]

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
			ClientID:   tExchange.GenerateClientID(),
		}
		logger.Debugf("TAKER ORDER %v", takerOrder)

		hedgeMarkPrice, okHedgeMarkPrice := tHedgeMarkPrices[takerSymbol]
		if !mtConfig.HedgeInstantly && okHedgeMarkPrice {
			if takerOrder.Side == common.OrderSideBuy &&
				spread.TakerDir < 0 &&
				spread.TakerDepth.BestAskPrice < hedgeMarkPrice*(1.0-mtConfig.HedgeTrackOffset) {
				logger.Debugf(
					"%s taker change size %f dir %f mark price %f -> %f",
					takerSymbol, takerSizeDiff, spread.TakerDir,
					hedgeMarkPrice, spread.TakerDepth.BestAskPrice,
				)
				tHedgeMarkPrices[takerSymbol] = spread.TakerDepth.BestAskPrice
				tOrderSilentTimes[takerSymbol] = time.Now().Add(mtConfig.HedgeCheckInterval)
				mOrderSilentTimes[makerSymbol] = time.Now().Add(mtConfig.HedgeCheckInterval)
				tPositionsUpdateTimes[takerSymbol] = time.Now()
				mPositionsUpdateTimes[takerSymbol] = time.Now()
				continue
			} else if takerOrder.Side == common.OrderSideSell &&
				spread.TakerDir > 0 &&
				spread.TakerDepth.BestBidPrice > hedgeMarkPrice*(1.0+mtConfig.HedgeTrackOffset) {
				logger.Debugf(
					"%s taker change size %f dir %f mark price %f -> %f",
					takerSymbol, -takerSizeDiff, spread.TakerDir,
					hedgeMarkPrice, spread.TakerDepth.BestAskPrice,
				)
				tHedgeMarkPrices[takerSymbol] = spread.TakerDepth.BestBidPrice
				tOrderSilentTimes[takerSymbol] = time.Now().Add(mtConfig.HedgeCheckInterval)
				mOrderSilentTimes[makerSymbol] = time.Now().Add(mtConfig.HedgeCheckInterval)
				tPositionsUpdateTimes[takerSymbol] = time.Now()
				mPositionsUpdateTimes[takerSymbol] = time.Now()
				continue
			}
		}
		mOrderSilentTimes[makerSymbol] = time.Now().Add(mtConfig.OrderSilent)
		tOrderSilentTimes[takerSymbol] = time.Now().Add(mtConfig.OrderSilent)
		tPositionsUpdateTimes[takerSymbol] = time.Unix(0, 0)
		if !mtConfig.DryRun {
			tOrderRequestChMap[takerSymbol] <- common.OrderRequest{
				New: &takerOrder,
			}
		}
		if okHedgeMarkPrice {
			delete(tHedgeMarkPrices, takerSymbol)
		}
	}
	mtUnHedgeValue = unHedgedValue
}

func updateMakerNewOrders() {

	if mAccount == nil {
		if time.Now().Sub(time.Now().Truncate(mtConfig.LogInterval)) < mtConfig.LoopInterval {
			logger.Debugf("mACCOUNT not ready")
		}
		return
	}
	if tAccount == nil {
		if time.Now().Sub(time.Now().Truncate(mtConfig.LogInterval)) < mtConfig.LoopInterval {
			logger.Debugf("tACCOUNT not ready")
		}
		return
	}

	if len(mtRankSymbolMap) == 0 {
		if time.Now().Sub(time.Now().Truncate(mtConfig.LogInterval)) < mtConfig.LoopInterval {
			logger.Debugf("mtRankSymbolMap not ready")
		}
		return
	}

	if mtUnHedgeValue > mtConfig.MaxUnHedgeValue {
		if time.Now().Sub(time.Now().Truncate(mtConfig.LogInterval)) < mtConfig.LoopInterval {
			logger.Debugf("taker unhedged value %f > %f", mtUnHedgeValue, mtConfig.MaxUnHedgeValue)
		}
		return
	}

	entryStep := (mAccount.GetFree() + tAccount.GetFree()) * mtConfig.EnterFreePct
	if entryStep < mtConfig.EnterMinimalStep {
		entryStep = mtConfig.EnterMinimalStep
	}
	entryTarget := entryStep * mtConfig.EnterTargetFactor

	//得是两个市场的最小可用资金, 以防有一边用完了钱, 开不了仓
	makerUSDTAvailable := math.Min(mAccount.GetFree()*mtConfig.MakerExchange.Leverage, tAccount.GetFree()*mtConfig.TakerExchange.Leverage)

	//遍历合约 从最大的rank 开始，能保证FR强的先下单, 优先做空
	for _, rank := range mtDualEnds {
		makerSymbol := mtRankSymbolMap[rank]
		takerSymbol := mtSymbolsMap[makerSymbol]

		spread, okSpread := mtSpreads[makerSymbol]

		if okSpread && time.Now().Sub(time.Now().Truncate(mtConfig.LogInterval)) < mtConfig.LoopInterval {
			logger.Debugf("%s maker dir %f  taker dir %f", makerSymbol, spread.MakerDir, spread.TakerDir)
		}
		//需要保证两边都有仓位更新，才调整现货仓位
		if time.Now().Sub(mPositionsUpdateTimes[makerSymbol]) > mtConfig.BalancePositionMaxAge {
			if time.Now().Sub(time.Now().Truncate(mtConfig.LogInterval)) < mtConfig.LoopInterval {
				logger.Debugf("maker position too old %s", makerSymbol)
			}
			continue
		}
		if time.Now().Sub(tPositionsUpdateTimes[takerSymbol]) > mtConfig.BalancePositionMaxAge {
			if time.Now().Sub(time.Now().Truncate(mtConfig.LogInterval)) < mtConfig.LoopInterval {
				logger.Debugf("taker position too old %s", makerSymbol)
			}
			continue
		}
		if time.Now().Sub(mOrderSilentTimes[makerSymbol]) < 0 {
			if time.Now().Sub(time.Now().Truncate(mtConfig.LogInterval)) < mtConfig.LoopInterval {
				logger.Debugf("taker order silent %s", makerSymbol)
			}
			continue
		}
		if time.Now().Sub(mEnterSilentTimes[makerSymbol]) < 0 {
			if time.Now().Sub(time.Now().Truncate(mtConfig.LogInterval)) < mtConfig.LoopInterval {
				logger.Debugf("maker enter silent %s", makerSymbol)
			}
			continue
		}
		if _, ok := mOpenOrders[makerSymbol]; ok {
			if time.Now().Sub(time.Now().Truncate(mtConfig.LogInterval)) < mtConfig.LoopInterval {
				logger.Debugf("has open order %s", makerSymbol)
			}
			continue
		}
		makerPosition, okMakerPosition := mPositions[makerSymbol]
		fundingRate, okFundingRate := mtFundingRates[makerSymbol]
		if !okSpread || !okMakerPosition || !okFundingRate {
			if time.Now().Sub(time.Now().Truncate(mtConfig.LogInterval)) < mtConfig.LoopInterval {
				logger.Debugf("spread %v maker position %v fundingRate %v %s", okSpread, okFundingRate, okMakerPosition, makerSymbol)
			}
			continue
		}
		if time.Now().Sub(time.Now().Truncate(mtConfig.LogInterval)) < mtConfig.LoopInterval {
			logger.Debugf("%s maker dir %f  taker dir %f", makerSymbol, spread.MakerDir, spread.TakerDir)
		}

		if time.Now().Sub(spread.Time) > mtConfig.SpreadTimeToLive {
			if time.Now().Sub(time.Now().Truncate(mtConfig.LogInterval)) < mtConfig.LoopInterval {
				logger.Debugf("spread too old %s %v", makerSymbol, spread.Time)
			}
			continue
		}
		makerDepth := spread.MakerDepth
		makerMultiplier := mStepSizes[makerSymbol]
		makerTickSize := mTickSizes[makerSymbol]
		takerMinNotional := tMinNotional[takerSymbol]
		makerTakerStepSize := mtStepSizes[makerSymbol]

		makerValue := makerPosition.GetSize() * makerPosition.GetPrice()
		offset := mOrderOffsets[makerSymbol]
		delta := mtDeltas[makerSymbol]
		shortTop := delta.ShortTop + mtConfig.OffsetDelta*(math.Max(makerValue, 0)/entryTarget)
		shortBot := delta.ShortBot + mtConfig.OffsetDelta*(math.Max(makerValue, 0)/entryTarget)
		longBot := delta.LongBot + mtConfig.OffsetDelta*(math.Min(makerValue, 0)/entryTarget)
		longTop := delta.LongTop + mtConfig.OffsetDelta*(math.Min(makerValue, 0)/entryTarget)

		if time.Now().Sub(time.Now().Truncate(mtConfig.LogInterval)) < mtConfig.LoopInterval {
			logger.Debugf("loop %s", makerSymbol)
		}

		if spread.ShortLastLeave < shortBot &&
			spread.ShortMedianLeave < shortBot &&
			fundingRate < mtConfig.MinimalKeepFundingRate &&
			makerPosition.GetSize() > 0 {
			makerSize := makerPosition.GetSize()
			price := math.Ceil(makerDepth.MakerAsk*(1.0+offset.Top)/makerTickSize) * makerTickSize
			entryValue := math.Max(4*entryStep, makerSize*price*0.5)
			if fundingRate > mtConfig.MinimalKeepFundingRate*0.5 {
				entryValue = math.Max(2*entryStep, makerSize*price*0.5)
			}
			size := entryValue / price
			size = math.Round(size/makerTakerStepSize) * makerTakerStepSize
			size = math.Round(size / makerMultiplier)
			entryValue = size * makerMultiplier * price
			if makerSize*price-entryValue < entryStep {
				size = makerPosition.GetSize()
			}
			if size > 0 {
				logger.Debugf(
					"SHORT BOT REDUCE %s %f < %f, %f < %f, SIZE %f PRICE %f",
					makerSymbol,
					spread.ShortLastLeave, shortBot,
					spread.ShortMedianLeave, shortBot,
					size, price,
				)

				order := common.NewOrderParam{
					Symbol:      makerSymbol,
					Side:        common.OrderSideSell,
					Type:        common.OrderTypeLimit,
					Price:       price,
					TimeInForce: common.OrderTimeInForceGTC,
					Size:        size,
					PostOnly:    true,
					ReduceOnly:  true,
					ClientID:    mExchange.GenerateClientID(),
				}
				mOpenOrders[makerSymbol] = order
				mOrderSilentTimes[makerSymbol] = time.Now()
				mOrderSilentTimes[makerSymbol] = time.Now().Add(mtConfig.OrderSilent)
				if !mtConfig.DryRun {
					mOrderRequestChMap[makerSymbol] <- common.OrderRequest{New: &order}
				}
				return
			}
		} else if spread.LongLastLeave > longTop &&
			spread.LongMedianLeave > longTop &&
			fundingRate > -mtConfig.MinimalKeepFundingRate &&
			makerPosition.GetSize() < 0 {

			makerSize := -makerPosition.GetSize()
			price := math.Floor(makerDepth.MakerBid*(1.0+offset.Bot)/makerTickSize) * makerTickSize
			entryValue := math.Max(4*entryStep, makerSize*price*0.5)
			if fundingRate < -mtConfig.MinimalKeepFundingRate/2 {
				entryValue = math.Max(2*entryStep, makerSize*price*0.5)
			}
			size := entryValue / price
			size = math.Round(size/makerTakerStepSize) * makerTakerStepSize
			size = math.Round(size / makerMultiplier)
			if makerSize*price-entryValue < entryStep {
				size = -makerPosition.GetSize()
			}
			if size > 0 {
				logger.Debugf(
					"LONG TOP REDUCE %s %f > %f, %f > %f, SIZE %f PRICE %f",
					makerSymbol,
					spread.LongLastLeave, longTop,
					spread.LongMedianLeave, longTop,
					size, price,
				)
				order := common.NewOrderParam{
					Symbol:      makerSymbol,
					Side:        common.OrderSideBuy,
					Type:        common.OrderTypeLimit,
					Price:       price,
					TimeInForce: common.OrderTimeInForceGTC,
					Size:        size,
					PostOnly:    true,
					ReduceOnly:  true,
					ClientID:    mExchange.GenerateClientID(),
				}
				mOpenOrders[makerSymbol] = order
				mOrderSilentTimes[makerSymbol] = time.Now()
				mOrderSilentTimes[makerSymbol] = time.Now().Add(mtConfig.OrderSilent)
				if !mtConfig.DryRun {
					mOrderRequestChMap[makerSymbol] <- common.OrderRequest{New: &order}
				}
				return
			}
		} else if spread.ShortLastEnter > shortTop &&
			spread.ShortMedianEnter > shortTop &&
			delta.ShortTop - delta.ShortBot > mtConfig.MinimalDelta &&
			fundingRate > mtConfig.MinimalEnterFundingRate &&
			makerPosition.GetSize() >= 0 {
			makerSize := makerPosition.GetSize()
			price := math.Floor(makerDepth.MakerBid*(1.0+offset.Bot)/makerTickSize) * makerTickSize
			targetValue := makerSize*price + entryStep
			if targetValue > entryTarget {
				targetValue = entryTarget
			}
			entryValue := targetValue - makerSize*price
			if entryValue > makerUSDTAvailable*0.8 {
				entryValue = makerUSDTAvailable * 0.8
			}
			size := entryValue / price
			size = math.Round(size/makerTakerStepSize) * makerTakerStepSize
			size = math.Round(size / makerMultiplier)

			entryValue = size * makerMultiplier * price

			////不及一个0.8*EntryStep, 不操作
			//if entryValue < entryStep*0.8 {
			//	if time.Now().Sub(mtLogSilentTimes[makerSymbol]) > 0 {
			//		logger.Debugf(
			//			"FAILED SHORT TOP OPEN, ENTRY VALUE %f LESS THAN 0.8*ENTRY_STEP %f, %s %f > %f, %f > %f, SIZE %f PRICE %f",
			//			entryValue,
			//			entryStep*0.8,
			//			makerSymbol,
			//			spread.ShortLastEnter, shortTop,
			//			spread.ShortMedianEnter, shortTop,
			//			size, price,
			//		)
			//		mtLogSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.LogInterval)
			//	}
			//	continue
			//}
			if entryValue > makerUSDTAvailable {
				if time.Now().Sub(mtLogSilentTimes[makerSymbol]) > 0 {
					logger.Debugf(
						"FAILED SHORT TOP OPEN, ENTRY VALUE %f MORE THAN WithdrawAvailable %f, %s %f > %f, %f > %f, SIZE %f PRICE %f",
						entryValue,
						makerUSDTAvailable,
						makerSymbol,
						spread.ShortLastEnter, shortTop,
						spread.ShortMedianEnter, shortTop,
						size, price,
					)
					mtLogSilentTimes[makerSymbol] = time.Now().Add(mtConfig.LogInterval)
				}
				continue
			}
			if entryValue < takerMinNotional {
				if time.Now().Sub(mtLogSilentTimes[makerSymbol]) > 0 {
					logger.Debugf(
						"FAILED SHORT TOP OPEN, ORDER VALUE %f LESS THAN NOTIONAL %f, %s %f > %f, %f > %f, SIZE %f PRICE %f",
						entryValue,
						takerMinNotional,
						makerSymbol,
						spread.ShortLastEnter, shortTop,
						spread.ShortMedianEnter, shortTop,
						size, price,
					)
					mtLogSilentTimes[makerSymbol] = time.Now().Add(mtConfig.LogInterval)
				}
				continue
			}
			mtLogSilentTimes[makerSymbol] = time.Now()
			logger.Debugf(
				"SHORT TOP OPEN %s %f > %f, %f > %f, SIZE %f PRICE %f",
				makerSymbol,
				spread.ShortLastEnter, shortTop,
				spread.ShortMedianEnter, shortTop,
				size, price,
			)
			makerUSDTAvailable -= entryValue
			order := common.NewOrderParam{
				Symbol:      makerSymbol,
				Side:        common.OrderSideBuy,
				Type:        common.OrderTypeLimit,
				Price:       price,
				TimeInForce: common.OrderTimeInForceGTC,
				Size:        size,
				PostOnly:    true,
				ReduceOnly:  false,
				ClientID:    mExchange.GenerateClientID(),
			}
			mOpenOrders[makerSymbol] = order
			mOrderSilentTimes[makerSymbol] = time.Now()
			mOrderSilentTimes[makerSymbol] = time.Now().Add(mtConfig.OrderSilent)
			if !mtConfig.DryRun {
				mOrderRequestChMap[makerSymbol] <- common.OrderRequest{New: &order}
			}
		} else if spread.LongLastEnter < longBot &&
			spread.LongMedianEnter < longBot &&
			delta.LongTop - delta.LongBot > mtConfig.MinimalDelta &&
			fundingRate < -mtConfig.MinimalEnterFundingRate &&
			makerPosition.GetSize() <= 0 {

			makerSize := -makerPosition.GetSize()
			price := math.Ceil(makerDepth.MakerAsk*(1.0+offset.Top)/makerTickSize) * makerTickSize
			targetValue := makerSize*price + entryStep
			if targetValue > entryTarget {
				targetValue = entryTarget
			}
			entryValue := targetValue - makerSize*price
			if entryValue > makerUSDTAvailable*0.8 {
				entryValue = makerUSDTAvailable * 0.8
			}
			size := entryValue / price
			size = math.Round(size/makerTakerStepSize) * makerTakerStepSize
			size = math.Round(size / makerMultiplier)

			entryValue = size * makerMultiplier * price

			//不及一个0.8*EntryStep, 不操作
			//if entryValue < entryStep*0.8 {
			//	if time.Now().Sub(mtLogSilentTimes[makerSymbol]) > 0 {
			//		logger.Debugf(
			//			"FAILED LONG BOT OPEN, ENTRY VALUE %f LESS THAN 0.8*ENTRY_STEP %f, %s %f < %f, %f < %f, SIZE %f",
			//			entryValue,
			//			entryStep*0.8,
			//			makerSymbol,
			//			spread.LongLastEnter, ongBot,
			//			spread.LongMedianEnter, quantile.LongBot,
			//			size,
			//		)
			//		mtLogSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.LogInterval)
			//	}
			//	continue
			//}
			if entryValue > makerUSDTAvailable {
				if time.Now().Sub(mtLogSilentTimes[makerSymbol]) > 0 {
					logger.Debugf(
						"FAILED LONG BOT OPEN, ENTRY VALUE %f MORE THAN WithdrawAvailable %f, %s %f < %f, %f < %f, SIZE %f PRICE %f",
						entryValue,
						makerUSDTAvailable,
						makerSymbol,
						spread.LongLastEnter, longBot,
						spread.LongMedianEnter, longBot,
						size, price,
					)
					mtLogSilentTimes[makerSymbol] = time.Now().Add(mtConfig.LogInterval)
				}
				continue
			}
			if entryValue < takerMinNotional {
				if time.Now().Sub(mtLogSilentTimes[makerSymbol]) > 0 {
					logger.Debugf(
						"FAILED LONG BOT OPEN, ORDER VALUE %f LESS THAN NOTIONAL %f, %s %f < %f, %f < %f, SIZE %f PRICE %f",
						entryValue,
						takerMinNotional,
						makerSymbol,
						spread.LongLastEnter, longBot,
						spread.LongMedianEnter, longBot,
						size, price,
					)
					mtLogSilentTimes[makerSymbol] = time.Now().Add(mtConfig.LogInterval)
				}
				continue
			}
			mtLogSilentTimes[makerSymbol] = time.Now()
			logger.Debugf(
				"LONG BOT OPEN %s %f < %f, %f < %f, SIZE %f PRICE %f",
				makerSymbol,
				spread.LongLastEnter, longBot,
				spread.LongMedianEnter, longBot,
				size, price,
			)
			makerUSDTAvailable -= entryValue
			order := common.NewOrderParam{
				Symbol:      makerSymbol,
				Side:        common.OrderSideSell,
				Type:        common.OrderTypeLimit,
				Price:       price,
				TimeInForce: common.OrderTimeInForceGTC,
				Size:        size,
				PostOnly:    true,
				ReduceOnly:  false,
				ClientID:    fmt.Sprintf("%d%04d", time.Now().Unix(), rand.Intn(10000)),
			}
			mOpenOrders[makerSymbol] = order
			mOrderSilentTimes[makerSymbol] = time.Now()
			mOrderSilentTimes[makerSymbol] = time.Now().Add(mtConfig.OrderSilent)
			if !mtConfig.DryRun {
				mOrderRequestChMap[makerSymbol] <- common.OrderRequest{New: &order}
			}
		}
	}
}

func handleUpdateFundingRates() {
	if len(mFundingRates) != len(tFundingRates) {
		if time.Now().Sub(time.Now().Truncate(mtConfig.LogInterval)) < mtConfig.LogInterval {
			logger.Debugf("len(mFundingRates) %d != len(tFundingRates) %d", len(mFundingRates), len(tFundingRates))
		}
		return
	}
	frs := make([]float64, len(mSymbols))
	for i, makerSymbol := range mSymbols {
		takerSymbol := mtSymbolsMap[makerSymbol]
		if makerFr, ok := mFundingRates[makerSymbol]; ok {
			if takerFr, ok := tFundingRates[takerSymbol]; ok {
				frs[i] = takerFr.GetFundingRate() - makerFr.GetFundingRate()
				mtFundingRates[makerSymbol] = frs[i]
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
	if len(mtRankSymbolMap) == 0 {
		for i, fr := range frs {
			logger.Debugf(
				"MERGED FR %s %f %s %f -> %f",
				mSymbols[i], mFundingRates[mSymbols[i]].GetFundingRate(),
				mtSymbolsMap[mSymbols[i]], tFundingRates[mtSymbolsMap[mSymbols[i]]].GetFundingRate(),
				fr,
			)
		}
		mtRankSymbolMap, err = common.RankSymbols(mSymbols, frs)
		if err != nil {
			logger.Debugf("RankSymbols error %v", err)
		}
		logger.Debugf("SYMBOLS FR RANK %v", mtRankSymbolMap)
	} else {
		mtRankSymbolMap, err = common.RankSymbols(mSymbols, frs)
		if err != nil {
			logger.Debugf("RankSymbols error %v", err)
		}
	}
}
