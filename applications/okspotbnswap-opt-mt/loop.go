package main

import (
	"fmt"
	"github.com/geometrybase/hft-micro/bnspot"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/geometrybase/hft-micro/okspot"
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
		if time.Now().Sub(mBalancesUpdateTimes[makerSymbol]) > *mtConfig.BalancePositionMaxAge {
			continue
		}

		if tOrderSilentTimes[takerSymbol].Sub(time.Now()).Seconds() > 0 {
			continue
		}

		makerBalance, okMakerBalance := mBalances[makerSymbol]
		takerPosition, okTakerBalance := tPositions[takerSymbol]
		spread, okSpread := mtSpreads[makerSymbol]
		if !okMakerBalance || !okTakerBalance || !okSpread {
			continue
		}
		takerTakerDepth := spread.TakerDepth

		takerStepSize := tStepSizes[takerSymbol]
		takerMinNotional := tMinNotional[takerSymbol]

		takerSizeDiff := -makerBalance.Balance - takerPosition.PositionAmt
		if takerSizeDiff > 0 {
			unHedgedValue += math.Abs(takerSizeDiff * takerTakerDepth.TakerAsk)
		} else {
			unHedgedValue += math.Abs(takerSizeDiff * takerTakerDepth.TakerBid)
		}
		takerSizeDiff = math.Round(takerSizeDiff/takerStepSize) * takerStepSize
		if math.Abs(takerSizeDiff) < takerStepSize {
			continue
		} else if takerSizeDiff < 0 && takerPosition.PositionAmt <= 0 && -takerSizeDiff*takerTakerDepth.TakerBid < takerMinNotional {
			continue
		} else if takerSizeDiff > 0 && takerPosition.PositionAmt >= 0 && takerSizeDiff*takerTakerDepth.TakerAsk < takerMinNotional {
			continue
		}

		logger.Debugf("updateTakerPositions %s SIZE DIFF %f POS %f -> %f", takerSymbol, takerSizeDiff, takerPosition.PositionAmt, -makerBalance.Balance)

		reduceOnly := false
		if takerSizeDiff*takerPosition.PositionAmt < 0 && math.Abs(takerSizeDiff) <= math.Abs(takerPosition.PositionAmt) {
			reduceOnly = true
		}
		side := "BUY"
		if takerSizeDiff < 0 {
			side = "SELL"
			takerSizeDiff = -takerSizeDiff
		}
		takerOrder := bnswap.NewOrderParams{
			Symbol:           takerSymbol,
			Side:             side,
			Type:             bnspot.OrderTypeMarket,
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
	//if time.Now().Sub(time.Now().Truncate(time.Second*15)) < *mtConfig.LoopInterval {
	//	logger.Debugf("updateMakerNewOrders")
	//}

	if mAccount == nil {
		//logger.Debugf("mACCOUNT NOT READY")
		return
	}

	if tAccount == nil || tAccount.AvailableBalance == nil {
		//if time.Now().Sub(time.Now().Truncate(time.Second*15)) < *mtConfig.LoopInterval {
		//	logger.Debugf("tACCOUNT NOT READY")
		//}
		return
	}

	if len(mtRankSymbolMap) == 0 {
		//if time.Now().Sub(time.Now().Truncate(time.Second*15)) < *mtConfig.LoopInterval {
		//	logger.Debugf("RankSymbolMAP NOT READY")
		//}
		return
	}

	if mtUnHedgeValue > *mtConfig.MaxUnHedgeValue {
		if time.Now().Sub(mtUnHedgeLogSilentTimes) > 0 {
			logger.Debugf("TAKER UN HEDGE VALUE %f > %f", mtUnHedgeValue, *mtConfig.MaxUnHedgeValue)
			mtUnHedgeLogSilentTimes = time.Now().Add(*mtConfig.LogInterval)
		}
		return
	}

	entryStep := (mAccount.Available + *tAccount.AvailableBalance) * *mtConfig.EnterFreePct
	if entryStep < *mtConfig.EnterMinimalStep {
		entryStep = *mtConfig.EnterMinimalStep
	}
	entryTarget := entryStep * *mtConfig.EnterTargetFactor

	makerUSDTAvailable := math.Min(mAccount.Available, *tAccount.AvailableBalance*float64(*mtConfig.Leverage))

	//遍历合约 从最大的rank 开始，能保证FR强的先下单, 优先做空
	for _, rank := range mtDualEnds {
		makerSymbol := mtRankSymbolMap[rank]
		takerSymbol := mtSymbolsMap[makerSymbol]
		//需要保证两边都有仓位更新，才调整现货仓位
		if time.Now().Sub(mBalancesUpdateTimes[makerSymbol]) > *mtConfig.BalancePositionMaxAge {
			//if time.Now().Sub(time.Now().Truncate(time.Second*15)) < *mtConfig.LoopInterval {
			//	logger.Debugf("%s time.Now().Sub(mBalancesUpdateTimes[makerSymbol]) %v", makerSymbol, time.Now().Sub(mBalancesUpdateTimes[makerSymbol]))
			//}
			continue
		}
		if time.Now().Sub(tPositionsUpdateTimes[takerSymbol]) > *mtConfig.BalancePositionMaxAge {
			//if time.Now().Sub(time.Now().Truncate(time.Second*15)) < *mtConfig.LoopInterval {
			//	logger.Debugf("%s time.Now().Sub(tPositionsUpdateTimes[takerSymbol]) %v", makerSymbol, time.Now().Sub(tPositionsUpdateTimes[takerSymbol]))
			//}
			continue
		}
		if time.Now().Sub(mOrderSilentTimes[makerSymbol]) < 0 {
			//if time.Now().Sub(time.Now().Truncate(time.Second*15)) < *mtConfig.LoopInterval {
			//	logger.Debugf("%s time.Now().Sub(mOrderSilentTimes[makerSymbol]) < 0", makerSymbol)
			//}
			continue
		}
		if time.Now().Sub(mSilentTimes[makerSymbol]) < 0 {
			//if time.Now().Sub(time.Now().Truncate(time.Second*15)) < *mtConfig.LoopInterval {
			//	logger.Debugf("%s time.Now().Sub(mSilentTimes[makerSymbol]) < 0", makerSymbol)
			//}
			continue
		}
		if _, ok := mOpenOrders[makerSymbol]; ok {
			//if time.Now().Sub(time.Now().Truncate(time.Second*15)) < *mtConfig.LoopInterval {
			//	logger.Debugf("%s mOpenOrders[makerSymbol]) < 0", makerSymbol)
			//}
			continue
		}
		spread, okSpread := mtSpreads[makerSymbol]
		makerBalance, okMakerBalance := mBalances[makerSymbol]
		fundingRate, okFundingRate := mtFundingRates[makerSymbol]
		if !okSpread || !okMakerBalance || !okFundingRate {
			continue
		}

		if time.Now().Sub(spread.Time) > *mtConfig.SpreadTimeToLive {
			continue
		}
		makerDepth := spread.MakerDepth
		makerTickSize := mTickSizes[makerSymbol]
		takerMinNotional := tMinNotional[takerSymbol]
		makerTakerStepSize := mtStepSizes[makerSymbol]
		makerStepSize := mStepSizes[makerSymbol]
		makerMinSize := mMinSizes[makerSymbol]

		currentSpotSize := makerBalance.Balance
		currentSpotValue := currentSpotSize * spread.MakerDepth.MidPrice
		makerOffset := mOrderOffsets[makerSymbol]
		enterDelta := *mtConfig.EnterDelta + *mtConfig.OffsetDelta*(currentSpotValue/entryTarget)
		exitDelta := *mtConfig.ExitDelta + *mtConfig.OffsetDelta*(currentSpotValue/entryTarget)

		if spread.LastLeave < exitDelta &&
			spread.MedianLeave < exitDelta &&
			fundingRate < *mtConfig.MinimalKeepFundingRate &&
			makerBalance.Balance > makerMinSize {
			makerSize := makerBalance.Balance
			price := math.Ceil(makerDepth.MidPrice*(1.0+makerOffset.Top)/makerTickSize) * makerTickSize
			entryValue := math.Max(4*entryStep, makerSize*price*0.5)
			if fundingRate > *mtConfig.MinimalKeepFundingRate*0.5 {
				entryValue = math.Max(2*entryStep, makerSize*price*0.5)
			}
			size := entryValue / price
			size = math.Round(size/makerTakerStepSize) * makerTakerStepSize
			entryValue = size * price
			if makerSize*price-entryValue < entryStep {
				size = math.Floor(makerBalance.Balance/makerStepSize) * makerStepSize
			}
			if size > makerMinSize {
				logger.Debugf(
					"SHORT BOT REDUCE %s %f < %f, %f < %f, SIZE %f",
					makerSymbol,
					spread.LastLeave, exitDelta,
					spread.MedianLeave, exitDelta,
					size,
				)
				order := okspot.NewOrderParam{
					Symbol:    makerSymbol,
					ClientOID: fmt.Sprintf("M%d", time.Now().Unix()*10000+int64(rand.Intn(10000))),
					Side:      okspot.OrderSideSell,
					Type:      okspot.OrderLimit,
					OrderType: okspot.OrderTypePostOnly,
					Price:     &price,
					Size:      &size,
				}
				mOpenOrders[makerSymbol] = MakerOpenOrder{
					Symbol:        makerSymbol,
					NewOrderParam: &order,
				}
				mOrderSilentTimes[makerSymbol] = time.Now()
				mOrderSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.OrderSilent)
				mHttpPositionUpdateSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.HttpSilent)
				mOrderRequestChs[makerSymbol] <- MakerOrderRequest{New: &order}
				return
			}
		} else if spread.LastEnter > enterDelta &&
			spread.MedianEnter > enterDelta &&
			fundingRate > *mtConfig.MinimalEnterFundingRate {
			makerSize := makerBalance.Balance
			price := math.Floor(makerDepth.MidPrice*(1.0+makerOffset.Bot)/makerTickSize) * makerTickSize
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

			entryValue = size * price

			//不及一个0.8*EntryStep, 不操作
			if entryValue < entryStep*0.8 {
				if time.Now().Sub(mtLogSilentTimes[makerSymbol]) > 0 {
					logger.Debugf(
						"FAILED SHORT TOP OPEN, ENTRY VALUE %f LESS THAN 0.8*ENTRY_STEP %f, %s %f > %f, %f > %f, SIZE %f",
						entryValue,
						entryStep*0.8,
						makerSymbol,
						spread.LastEnter, enterDelta,
						spread.MedianEnter, enterDelta,
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
						spread.LastEnter, enterDelta,
						spread.MedianEnter, enterDelta,
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
						spread.LastEnter, enterDelta,
						spread.MedianEnter, enterDelta,
						size,
					)
					mtLogSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.LogInterval)
				}
				continue
			}
			if size < makerMinSize {
				if time.Now().Sub(mtLogSilentTimes[makerSymbol]) > 0 {
					logger.Debugf(
						"FAILED SHORT TOP OPEN, ORDER SIZE %f LESS THAN MIN SIZE %f, %s %f > %f, %f > %f, SIZE %f",
						size,
						makerMinSize,
						makerSymbol,
						spread.LastEnter, enterDelta,
						spread.MedianEnter, enterDelta,
						size,
					)
					mtLogSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.LogInterval)
				}
				continue
			}
			mtLogSilentTimes[makerSymbol] = time.Now()
			logger.Debugf(
				"SHORT TOP OPEN %s %f > %f, %f > %f, SIZE %f, PRICE %f",
				makerSymbol,
				spread.LastEnter, enterDelta,
				spread.MedianEnter, enterDelta,
				size, price,
			)
			makerUSDTAvailable -= entryValue
			order := okspot.NewOrderParam{
				Symbol:    makerSymbol,
				ClientOID: fmt.Sprintf("M%d", time.Now().Unix()*10000+int64(rand.Intn(10000))),
				Side:      okspot.OrderSideBuy,
				Type:      okspot.OrderLimit,
				OrderType: okspot.OrderTypePostOnly,
				Price:     &price,
				Size:      &size,
			}
			mOpenOrders[makerSymbol] = MakerOpenOrder{
				Symbol:        makerSymbol,
				NewOrderParam: &order,
			}
			mOrderSilentTimes[makerSymbol] = time.Now()
			mOrderSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.OrderSilent)
			mHttpPositionUpdateSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.HttpSilent)
			mOrderRequestChs[makerSymbol] <- MakerOrderRequest{New: &order}
		}
	}
}

func handleUpdateFundingRates() {
	if tPremiumIndexes == nil {
		return
	}
	frs := make([]float64, len(mSymbols))
	for i, makerSymbol := range mSymbols {
		takerSymbol := mtSymbolsMap[makerSymbol]
		if pi, ok := tPremiumIndexes[takerSymbol]; ok {
			frs[i] = pi.FundingRate
			mtFundingRates[makerSymbol] = frs[i]
		} else {
			logger.Debugf("MISS PREMIUM INDEX FOR TAKER %s", makerSymbol)
			return
		}
	}
	var err error
	if len(mtRankSymbolMap) == 0 {
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
