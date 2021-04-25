package main

import (
	"context"
	"fmt"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/hbcrossswap"
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
		if !okMakerBalance  || !okTakerBalance || !okSpread {
			continue
		}
		takerTakerDepth := spread.TakerDepth

		takerStepSize := tStepSizes[takerSymbol]
		takerTickSize := tTickSizes[takerSymbol]
		takerMinNotional := tMinNotional[takerSymbol]

		takerSizeDiff := -makerBalance.Balance - takerPosition.PositionAmt
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

		logger.Debugf("updateTakerPositions %s SIZE DIFF %f POS %f -> %f", takerSymbol, takerSizeDiff, takerPosition.PositionAmt, -makerBalance.Balance)

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

	entryStep := (mAccount.Available + *tAccount.AvailableBalance) * *mtConfig.EnterFreePct
	if entryStep < *mtConfig.EnterMinimalStep {
		entryStep = *mtConfig.EnterMinimalStep
	}
	entryTarget := entryStep * *mtConfig.EnterTargetFactor

	usdtAvailable := mAccount.Available

	//遍历合约 从最大的rank 开始，能保证FR强的先下单, 优先做空
	for _, rank := range mtDualEnds {
		makerSymbol := mtRankSymbolMap[rank]
		takerSymbol := mtSymbolsMap[makerSymbol]
		//需要保证两边都有仓位更新，才调整现货仓位
		if time.Now().Sub(mBalancesUpdateTimes[makerSymbol]) > *mtConfig.BalancePositionMaxAge {
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
		makerBalance, okMakerBalance := mBalances[makerSymbol]
		fundingRate, okFundingRate := tPremiumIndexes[takerSymbol]
		if !okSpread || !okQuantile || !okMakerBalance || !okFundingRate {
			continue
		}

		if time.Now().Sub(spread.Time) > *mtConfig.SpreadTimeToLive {
			continue
		}
		makerDepth := spread.MakerDepth
		makerContractSize := mStepSizes[makerSymbol]
		makerTickSize := mTickSizes[makerSymbol]
		takerMinNotional := tMinNotional[takerSymbol]
		makerTakerStepSize := mtStepSizes[makerSymbol]
		makerStepSize := math.Ceil(makerContractSize/makerTickSize) * makerTickSize
		makerStepSize = math.Ceil(makerStepSize / makerTickSize)

		//if time.Now().Sub(mtLogSilentTimes[makerSymbol]) > 0 {
		//	mtLogSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.LogInterval)
		//	logger.Debugf("LOOP %s", makerSymbol)
		//}
		makerUSDTAvailable := mAccount.WithdrawAvailable

		if spread.ShortLastLeave < quantile.ShortBot &&
			spread.ShortMedianLeave < quantile.ShortBot &&
			fundingRate < *mtConfig.MinimalKeepFundingRate &&
			makerBalance.Volume > 0 {
			makerSize := makerBalance.Volume * makerContractSize
			price := math.Ceil(makerDepth.MakerAsk/makerTickSize) * makerTickSize
			entryValue := math.Max(4*entryStep, makerSize*price*0.5)
			if fundingRate > *mtConfig.MinimalKeepFundingRate*0.5 {
				entryValue = math.Max(2*entryStep, makerSize*price*0.5)
			}
			volume := entryValue / price
			volume = math.Round(volume/makerTakerStepSize) * makerTakerStepSize
			volume = math.Round(volume / makerContractSize)
			entryValue = volume * makerContractSize * price
			if makerSize*price-entryValue < entryStep {
				volume = makerBalance.Volume
			}
			if volume > 0 {
				logger.Debugf(
					"SHORT BOT REDUCE %s %f < %f, %f < %f, SIZE %f",
					makerSymbol,
					spread.ShortLastLeave, quantile.ShortBot,
					spread.ShortMedianLeave, quantile.ShortBot,
					volume,
				)
				order := hbcrossswap.NewOrderParam{
					Symbol:         makerSymbol,
					ClientOrderID:  time.Now().Unix()*10000 + int64(rand.Intn(10000)),
					Price:          common.Float64(price),
					Volume:         int64(volume),
					Direction:      hbcrossswap.OrderDirectionSell,
					Offset:         hbcrossswap.OrderOffsetClose,
					LeverRate:      *mtConfig.Leverage,
					OrderPriceType: hbcrossswap.OrderPriceTypePostOnly,
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
			makerSellPosition.Volume > 0 {

			makerSize := makerSellPosition.Volume * makerContractSize
			price := math.Floor(makerDepth.MakerBid/makerTickSize) * makerTickSize
			entryValue := math.Max(4*entryStep, makerSize*price*0.5)
			if fundingRate < -*mtConfig.MinimalKeepFundingRate/2 {
				entryValue = math.Max(2*entryStep, makerSize*price*0.5)
			}
			volume := entryValue / price
			volume = math.Round(volume/makerTakerStepSize) * makerTakerStepSize
			volume = math.Round(volume / makerContractSize)
			if makerSize*price-entryValue < entryStep {
				volume = makerSellPosition.Volume
			}
			if volume > 0 {
				logger.Debugf(
					"LONG TOP REDUCE %s %f > %f, %f > %f, VOLUME %f",
					makerSymbol,
					spread.LongLastLeave, quantile.LongTop,
					spread.LongMedianLeave, quantile.LongTop,
					volume,
				)
				order := hbcrossswap.NewOrderParam{
					Symbol:         makerSymbol,
					ClientOrderID:  time.Now().Unix()*10000 + int64(rand.Intn(10000)),
					Price:          common.Float64(price),
					Volume:         int64(volume),
					Direction:      hbcrossswap.OrderDirectionBuy,
					Offset:         hbcrossswap.OrderOffsetClose,
					LeverRate:      *mtConfig.Leverage,
					OrderPriceType: hbcrossswap.OrderPriceTypePostOnly,
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
			makerSellPosition.Volume == 0 {
			makerSize := makerBalance.Volume * makerContractSize
			price := math.Floor(makerDepth.MakerBid/makerTickSize) * makerTickSize
			targetValue := makerSize*price + entryStep
			if targetValue > entryTarget {
				targetValue = entryTarget
			}
			entryValue := targetValue - makerSize*price
			if entryValue > makerUSDTAvailable*0.8 {
				entryValue = makerUSDTAvailable * 0.8
			}

			volume := entryValue / price
			volume = math.Round(volume/makerTakerStepSize) * makerTakerStepSize
			volume = math.Round(volume / makerContractSize)

			entryValue = volume * makerContractSize * price

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
						volume,
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
						volume,
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
						volume,
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
				volume,
			)
			makerUSDTAvailable -= entryValue
			order := hbcrossswap.NewOrderParam{
				Symbol:         makerSymbol,
				ClientOrderID:  time.Now().Unix()*10000 + int64(rand.Intn(10000)),
				Price:          common.Float64(price),
				Volume:         int64(volume),
				Direction:      hbcrossswap.OrderDirectionBuy,
				Offset:         hbcrossswap.OrderOffsetOpen,
				LeverRate:      *mtConfig.Leverage,
				OrderPriceType: hbcrossswap.OrderPriceTypePostOnly,
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
			makerBalance.Volume == 0 {
			makerSize := makerSellPosition.Volume * makerContractSize
			price := math.Ceil(makerDepth.MakerAsk/makerTickSize) * makerTickSize
			targetValue := makerSize*price + entryStep
			if targetValue > entryTarget {
				targetValue = entryTarget
			}
			entryValue := targetValue - makerSize*price

			if entryValue > makerUSDTAvailable*0.8 {
				entryValue = makerUSDTAvailable * 0.8
			}

			volume := entryValue / price
			volume = math.Round(volume/makerTakerStepSize) * makerTakerStepSize
			volume = math.Round(volume / makerContractSize)

			entryValue = volume * makerContractSize * price

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
						volume,
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
						volume,
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
						volume,
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
				volume,
			)
			makerUSDTAvailable -= entryValue
			order := hbcrossswap.NewOrderParam{
				Symbol:         makerSymbol,
				ClientOrderID:  time.Now().Unix()*10000 + int64(rand.Intn(10000)),
				Price:          common.Float64(price),
				Volume:         int64(volume),
				Direction:      hbcrossswap.OrderDirectionSell,
				Offset:         hbcrossswap.OrderOffsetOpen,
				LeverRate:      *mtConfig.Leverage,
				OrderPriceType: hbcrossswap.OrderPriceTypePostOnly,
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
	frs := make([]float64, len(mSymbols))
	for i, makerSymbol := range mSymbols {
		takerSymbol := mtSymbolsMap[makerSymbol]
		if fr, ok := mFundingRates[makerSymbol]; ok {
			if pi, ok := tPremiumIndexes[takerSymbol]; ok {
				frs[i] = pi.FundingRate - fr.FundingRate
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
				mSymbols[i], mFundingRates[mSymbols[i]].FundingRate,
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

