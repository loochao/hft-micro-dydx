package main

import (
	"fmt"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/kcperp"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"math/rand"
	"time"
)

func updateTakerPositions() {
	unHedgedValue := 0.0
	for _, takerSymbol := range tSymbols {
		makerSymbol := tmSymbolsMap[takerSymbol]
		if time.Now().Sub(tPositionsUpdateTimes[takerSymbol]) > *mtConfig.BalancePositionMaxAge {
			continue
		}
		if time.Now().Sub(mPositionsUpdateTimes[makerSymbol]) > *mtConfig.BalancePositionMaxAge {
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

		makerMultiplier := mMultipliers[makerSymbol]

		takerStepSize := tStepSizes[takerSymbol]
		takerTickSize := tTickSizes[takerSymbol]
		takerMinNotional := tMinNotional[takerSymbol]

		makerSize := makerPosition.CurrentQty * makerMultiplier

		takerSizeDiff := -makerSize - takerPosition.PositionAmt
		if takerSizeDiff > 0 {
			unHedgedValue += math.Abs(takerSizeDiff * takerTakerDepth.TakerAsk)
		} else {
			unHedgedValue += math.Abs(takerSizeDiff * takerTakerDepth.TakerBid)
		}
		takerSizeDiff = math.Round(takerSizeDiff/takerStepSize) * takerStepSize

		if takerSizeDiff <= 0 && -takerSizeDiff*takerTakerDepth.TakerBid*(1.0-*mtConfig.EnterSlippage) < takerMinNotional {
			continue
		} else if takerSizeDiff >= 0 && takerSizeDiff*takerTakerDepth.TakerAsk*(1.0+*mtConfig.EnterSlippage) < takerMinNotional {
			continue
		}

		logger.Debugf("updateTakerPositions %s SIZE DIFF %f POS %f -> %f", takerSymbol, takerSizeDiff, takerPosition.PositionAmt, -makerSize)

		reduceOnly := false
		if takerSizeDiff*takerPosition.PositionAmt < 0 && math.Abs(takerSizeDiff) <= math.Abs(takerPosition.PositionAmt) {
			reduceOnly = true
		}
		price := math.Round(takerTakerDepth.TakerAsk*(1.0+*mtConfig.EnterSlippage)/takerTickSize) * takerTickSize
		side := "BUY"
		if takerSizeDiff < 0 {
			side = "SELL"
			takerSizeDiff = -takerSizeDiff
			price = math.Round(takerTakerDepth.TakerBid*(1.0-*mtConfig.EnterSlippage)/takerTickSize) * takerTickSize
		}
		takerOrder := bnswap.NewOrderParams{
			Symbol:           takerSymbol,
			Side:             side,
			Type:             "LIMIT",
			Price:            price,
			TimeInForce:      "FOK",
			Quantity:         takerSizeDiff,
			ReduceOnly:       reduceOnly,
			NewClientOrderId: fmt.Sprintf("%d", time.Now().Unix()*10000+int64(rand.Intn(10000))),
		}
		logger.Debugf("TAKER ORDER %v", takerOrder.ToString())
		mOrderSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.OrderSilent)

		tOrderSilentTimes[takerSymbol] = time.Now().Add(*mtConfig.OrderSilent)
		tPositionsUpdateTimes[takerSymbol] = time.Unix(0, 0)
		tHttpPositionUpdateSilentTimes[takerSymbol] = time.Now().Add(*mtConfig.HttpSilent)
		tOrderRequestChs[takerSymbol] <- takerOrder
	}
	mtUnHedgeValue = unHedgedValue
}

func updateMakerNewOrders() {

	if mAccount == nil {
		//logger.Debugf("mACCOUNT NOT READY")
		return
	}

	if tAccount == nil || tAccount.AvailableBalance == nil {
		//logger.Debugf("tACCOUNT NOT READY")
		return
	}

	if len(mtRankSymbolMap) == 0 {
		//logger.Debugf("RankSymbolMAP NOT READY")
		return
	}

	if mtUnHedgeValue > *mtConfig.MaxUnHedgeValue {
		if time.Now().Sub(mtUnHedgeLogSilentTimes) > 0 {
			logger.Debugf("TAKER UN HEDGE VALUE %f > %f", mtUnHedgeValue, *mtConfig.MaxUnHedgeValue)
			mtUnHedgeLogSilentTimes = time.Now().Add(*mtConfig.LogInterval)
		}
		return
	}

	entryStep := (mAccount.AvailableBalance + *tAccount.AvailableBalance) * *mtConfig.EnterFreePct
	if entryStep < *mtConfig.EnterMinimalStep {
		entryStep = *mtConfig.EnterMinimalStep
	}
	entryTarget := entryStep * *mtConfig.EnterTargetFactor

	makerUSDTAvailable := mAccount.AvailableBalance

	//遍历合约 从最大的rank 开始，能保证FR强的先下单, 优先做空
	for _, rank := range mtDualEnds {
		makerSymbol := mtRankSymbolMap[rank]
		takerSymbol := mtSymbolsMap[makerSymbol]
		//需要保证两边都有仓位更新，才调整现货仓位
		if time.Now().Sub(mPositionsUpdateTimes[makerSymbol]) > *mtConfig.BalancePositionMaxAge {
			continue
		}
		if time.Now().Sub(tPositionsUpdateTimes[takerSymbol]) > *mtConfig.BalancePositionMaxAge {
			continue
		}
		if time.Now().Sub(mOrderSilentTimes[makerSymbol]) < 0 {
			continue
		}
		if time.Now().Sub(mSilentTimes[makerSymbol]) < 0 {
			continue
		}
		if _, ok := mOpenOrders[makerSymbol]; ok {
			continue
		}
		quantile, okQuantile := mtQuantiles[makerSymbol]
		spread, okSpread := mtSpreads[makerSymbol]
		makerPosition, okMakerPosition := mPositions[makerSymbol]
		fundingRate, okFundingRate := mtFundingRates[makerSymbol]
		if !okSpread || !okQuantile || !okMakerPosition || !okFundingRate {
			continue
		}

		if time.Now().Sub(spread.Time) > *mtConfig.SpreadTimeToLive {
			continue
		}
		makerDepth := spread.MakerDepth
		makerMultiplier := mMultipliers[makerSymbol]
		makerTickSize := mTickSizes[makerSymbol]
		takerMinNotional := tMinNotional[takerSymbol]
		makerTakerStepSize := mtStepSizes[makerSymbol]

		//if time.Now().Sub(mtLogSilentTimes[makerSymbol]) > 0 {
		//	mtLogSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.LogInterval)
		//	logger.Debugf("LOOP %s", makerSymbol)
		//}

		if spread.ShortLastLeave < quantile.ShortBot &&
			spread.ShortMedianLeave < quantile.ShortBot &&
			fundingRate < *mtConfig.MinimalKeepFundingRate &&
			makerPosition.CurrentQty > 0 {
			makerSize := makerPosition.CurrentQty * makerMultiplier
			price := math.Ceil(makerDepth.MakerAsk/makerTickSize) * makerTickSize
			entryValue := math.Max(4*entryStep, makerSize*price*0.5)
			if fundingRate > *mtConfig.MinimalKeepFundingRate*0.5 {
				entryValue = math.Max(2*entryStep, makerSize*price*0.5)
			}
			size := entryValue / price
			size = math.Round(size/makerTakerStepSize) * makerTakerStepSize
			size = math.Round(size / makerMultiplier)
			entryValue = size * makerMultiplier * price
			if makerSize*price-entryValue < entryStep {
				size = makerPosition.CurrentQty
			}
			if size > 0 {
				logger.Debugf(
					"SHORT BOT REDUCE %s %f < %f, %f < %f, SIZE %f",
					makerSymbol,
					spread.ShortLastLeave, quantile.ShortBot,
					spread.ShortMedianLeave, quantile.ShortBot,
					size,
				)

				order := kcperp.NewOrderParam{
					Symbol:      makerSymbol,
					Side:        kcperp.OrderSideSell,
					Type:        kcperp.OrderTypeLimit,
					Price:       common.Float64(price),
					TimeInForce: kcperp.OrderTimeInForceGTC,
					Size:        int64(size),
					PostOnly:    true,
					ReduceOnly:  true,
					ClientOid:   fmt.Sprintf("%d%04d", time.Now().Unix(), rand.Intn(10000)),
					Leverage:    *mtConfig.Leverage,
				}
				mOpenOrders[makerSymbol] = MakerOpenOrder{
					Symbol:        makerSymbol,
					NewOrderParam: &order,
				}
				mOrderSilentTimes[makerSymbol] = time.Now()
				mOrderCancelCounts[makerSymbol] = 0
				mOrderSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.OrderSilent)
				mHttpPositionUpdateSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.HttpSilent)
				mOrderRequestChs[makerSymbol] <- MakerOrderRequest{New: &order}
				return
			}
		} else if spread.LongLastLeave > quantile.LongTop &&
			spread.LongMedianLeave > quantile.LongTop &&
			fundingRate > -*mtConfig.MinimalKeepFundingRate &&
			makerPosition.CurrentQty < 0 {

			makerSize := -makerPosition.CurrentQty * makerMultiplier
			price := math.Floor(makerDepth.MakerBid/makerTickSize) * makerTickSize
			entryValue := math.Max(4*entryStep, makerSize*price*0.5)
			if fundingRate < -*mtConfig.MinimalKeepFundingRate/2 {
				entryValue = math.Max(2*entryStep, makerSize*price*0.5)
			}
			size := entryValue / price
			size = math.Round(size/makerTakerStepSize) * makerTakerStepSize
			size = math.Round(size / makerMultiplier)
			if makerSize*price-entryValue < entryStep {
				size = -makerPosition.CurrentQty
			}
			if size > 0 {
				logger.Debugf(
					"LONG TOP REDUCE %s %f > %f, %f > %f, VOLUME %f",
					makerSymbol,
					spread.LongLastLeave, quantile.LongTop,
					spread.LongMedianLeave, quantile.LongTop,
					size,
				)
				order := kcperp.NewOrderParam{
					Symbol:      makerSymbol,
					Side:        kcperp.OrderSideBuy,
					Type:        kcperp.OrderTypeLimit,
					Price:       common.Float64(price),
					TimeInForce: kcperp.OrderTimeInForceGTC,
					Size:        int64(size),
					PostOnly:    true,
					ReduceOnly:  true,
					ClientOid:   fmt.Sprintf("%d%04d", time.Now().Unix(), rand.Intn(10000)),
					Leverage:    *mtConfig.Leverage,
				}
				mOpenOrders[makerSymbol] = MakerOpenOrder{
					Symbol:        makerSymbol,
					NewOrderParam: &order,
				}
				mOrderSilentTimes[makerSymbol] = time.Now()
				mOrderCancelCounts[makerSymbol] = 0
				mOrderSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.OrderSilent)
				mHttpPositionUpdateSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.HttpSilent)
				mOrderRequestChs[makerSymbol] <- MakerOrderRequest{New: &order}
				return
			}
		} else if spread.ShortLastEnter > quantile.ShortTop &&
			spread.ShortMedianEnter > quantile.ShortTop &&
			fundingRate > *mtConfig.MinimalEnterFundingRate &&
			makerPosition.CurrentQty >= 0 {
			makerSize := makerPosition.CurrentQty * makerMultiplier
			price := math.Floor(makerDepth.MakerBid/makerTickSize) * makerTickSize
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
			if entryValue < entryStep*0.8 {
				if time.Now().Sub(mtLogSilentTimes[makerSymbol]) > 0 {
					logger.Debugf(
						"FAILED SHORT TOP OPEN, ENTRY VALUE %f LESS THAN 0.8*ENTRY_STEP %f, %s %f > %f, %f > %f, SIZE %f",
						entryValue,
						entryStep*0.8,
						makerSymbol,
						spread.ShortLastEnter, quantile.ShortTop,
						spread.ShortMedianEnter, quantile.ShortTop,
						size,
					)
					mtLogSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.LogInterval)
				}
				continue
			}
			if entryValue > makerUSDTAvailable {
				if time.Now().Sub(mtLogSilentTimes[makerSymbol]) > 0 {
					logger.Debugf(
						"FAILED SHORT TOP OPEN, ENTRY VALUE %f MORE THAN WithdrawAvailable %f, %s %f > %f, %f > %f, SIZE %f",
						entryValue,
						makerUSDTAvailable,
						makerSymbol,
						spread.ShortLastEnter, quantile.ShortTop,
						spread.ShortMedianEnter, quantile.ShortTop,
						size,
					)
					mtLogSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.LogInterval)
				}
				continue
			}
			if entryValue < takerMinNotional {
				if time.Now().Sub(mtLogSilentTimes[makerSymbol]) > 0 {
					logger.Debugf(
						"FAILED SHORT TOP OPEN, ORDER VALUE %f LESS THAN NOTIONAL %f, %s %f > %f, %f > %f, SIZE %f",
						entryValue,
						takerMinNotional,
						makerSymbol,
						spread.ShortLastEnter, quantile.ShortTop,
						spread.ShortMedianEnter, quantile.ShortTop,
						size,
					)
					mtLogSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.LogInterval)
				}
				continue
			}
			mtLogSilentTimes[makerSymbol] = time.Now()
			logger.Debugf(
				"SHORT TOP OPEN %s %f > %f, %f > %f, SIZE %f",
				makerSymbol,
				spread.ShortLastEnter, quantile.ShortTop,
				spread.ShortMedianEnter, quantile.ShortTop,
				size,
			)
			makerUSDTAvailable -= entryValue
			order := kcperp.NewOrderParam{
				Symbol:      makerSymbol,
				Side:        kcperp.OrderSideBuy,
				Type:        kcperp.OrderTypeLimit,
				Price:       common.Float64(price),
				TimeInForce: kcperp.OrderTimeInForceGTC,
				Size:        int64(size),
				PostOnly:    true,
				ReduceOnly:  false,
				ClientOid:   fmt.Sprintf("%d%04d", time.Now().Unix(), rand.Intn(10000)),
				Leverage:    *mtConfig.Leverage,
			}
			mOpenOrders[makerSymbol] = MakerOpenOrder{
				Symbol:        makerSymbol,
				NewOrderParam: &order,
			}
			mOrderSilentTimes[makerSymbol] = time.Now()
			mOrderCancelCounts[makerSymbol] = 0
			mOrderSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.OrderSilent)
			mHttpPositionUpdateSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.HttpSilent)
			mOrderRequestChs[makerSymbol] <- MakerOrderRequest{New: &order}
		} else if spread.LongLastEnter < quantile.LongBot &&
			spread.LongMedianEnter < quantile.LongBot &&
			fundingRate < -*mtConfig.MinimalEnterFundingRate &&
			makerPosition.CurrentQty <= 0 {

			makerSize := -makerPosition.CurrentQty * makerMultiplier
			price := math.Ceil(makerDepth.MakerAsk/makerTickSize) * makerTickSize
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
			if entryValue < entryStep*0.8 {
				if time.Now().Sub(mtLogSilentTimes[makerSymbol]) > 0 {
					logger.Debugf(
						"FAILED LONG BOT OPEN, ENTRY VALUE %f LESS THAN 0.8*ENTRY_STEP %f, %s %f < %f, %f < %f, SIZE %f",
						entryValue,
						entryStep*0.8,
						makerSymbol,
						spread.LongLastEnter, quantile.LongBot,
						spread.LongMedianEnter, quantile.LongBot,
						size,
					)
					mtLogSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.LogInterval)
				}
				continue
			}
			if entryValue > makerUSDTAvailable {
				if time.Now().Sub(mtLogSilentTimes[makerSymbol]) > 0 {
					logger.Debugf(
						"FAILED LONG BOT OPEN, ENTRY VALUE %f MORE THAN WithdrawAvailable %f, %s %f < %f, %f < %f, SIZE %f",
						entryValue,
						makerUSDTAvailable,
						makerSymbol,
						spread.LongLastEnter, quantile.LongBot,
						spread.LongMedianEnter, quantile.LongBot,
						size,
					)
					mtLogSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.LogInterval)
				}
				continue
			}
			if entryValue < takerMinNotional {
				if time.Now().Sub(mtLogSilentTimes[makerSymbol]) > 0 {
					logger.Debugf(
						"FAILED LONG BOT OPEN, ORDER VALUE %f LESS THAN NOTIONAL %f, %s %f < %f, %f < %f, SIZE %f",
						entryValue,
						takerMinNotional,
						makerSymbol,
						spread.LongLastEnter, quantile.LongBot,
						spread.LongMedianEnter, quantile.LongBot,
						size,
					)
					mtLogSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.LogInterval)
				}
				continue
			}
			mtLogSilentTimes[makerSymbol] = time.Now()
			logger.Debugf(
				"LONG BOT OPEN %s %f < %f, %f < %f, SIZE %f",
				makerSymbol,
				spread.LongLastEnter, quantile.LongBot,
				spread.LongMedianEnter, quantile.LongBot,
				size,
			)
			makerUSDTAvailable -= entryValue
			order := kcperp.NewOrderParam{
				Symbol:      makerSymbol,
				Side:        kcperp.OrderSideSell,
				Type:        kcperp.OrderTypeLimit,
				Price:       common.Float64(price),
				TimeInForce: kcperp.OrderTimeInForceGTC,
				Size:        int64(size),
				PostOnly:    true,
				ReduceOnly:  false,
				ClientOid:   fmt.Sprintf("%d%04d", time.Now().Unix(), rand.Intn(10000)),
				Leverage:    *mtConfig.Leverage,
			}
			mOpenOrders[makerSymbol] = MakerOpenOrder{
				Symbol:        makerSymbol,
				NewOrderParam: &order,
			}
			mOrderSilentTimes[makerSymbol] = time.Now()
			mOrderCancelCounts[makerSymbol] = 0
			mOrderSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.OrderSilent)
			mHttpPositionUpdateSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.HttpSilent)
			mOrderRequestChs[makerSymbol] <- MakerOrderRequest{New: &order}
		}
	}
}

func handleUpdateFundingRates() {
	if mFundingRates == nil {
		return
	}
	if tPremiumIndexes == nil {
		return
	}
	if len(mFundingRates) != len(tPremiumIndexes) {
		//logger.Debugf("FR M %d T %d", len(mFundingRates), len(tPremiumIndexes))
		return
	}
	frs := make([]float64, len(mSymbols))
	for i, makerSymbol := range mSymbols {
		takerSymbol := mtSymbolsMap[makerSymbol]
		if fr, ok := mFundingRates[makerSymbol]; ok {
			if pi, ok := tPremiumIndexes[takerSymbol]; ok {
				frs[i] = pi.FundingRate - fr.Value
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
				"%s %f %s %f -> %f",
				mSymbols[i], mFundingRates[mSymbols[i]].Value,
				mtSymbolsMap[mSymbols[i]], tPremiumIndexes[mtSymbolsMap[mSymbols[i]]].FundingRate,
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
