package main

import (
	"fmt"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/hbcrossswap"
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

		makerBuyPosition, okMakerBuyPosition := mBuyPositions[makerSymbol]
		makerSellPosition, okMakerSellPosition := mSellPositions[makerSymbol]
		takerPosition, okTakerBalance := tPositions[takerSymbol]
		spread, okSpread := mtSpreads[makerSymbol]
		if !okMakerBuyPosition || !okMakerSellPosition || !okTakerBalance || !okSpread {
			continue
		}
		takerOrderBook := spread.TakerOrderBook

		makerContractSize := mContractSizes[makerSymbol]

		takerStepSize := tStepSizes[takerSymbol]
		takerTickSize := tTickSizes[takerSymbol]
		takerMinNotional := tMinNotional[takerSymbol]

		makerSize := (makerBuyPosition.Volume - makerSellPosition.Volume) * makerContractSize

		takerSizeDiff := -makerSize - takerPosition.PositionAmt
		if takerSizeDiff > 0 {
			unHedgedValue += math.Abs(takerSizeDiff * takerOrderBook.AskPrice)
		} else {
			unHedgedValue += math.Abs(takerSizeDiff * takerOrderBook.BidPrice)
		}
		takerSizeDiff = math.Round(takerSizeDiff/takerStepSize) * takerStepSize

		//只做空SWAP，所以开空是加仓，开多是减仓，减仓大小受当前空仓大小限制, 加仓受MinNotional限制
		if takerSizeDiff <= 0 && -takerSizeDiff*takerOrderBook.BidPrice*(1.0-*mtConfig.EnterSlippage) < takerMinNotional {
			continue
		} else if takerSizeDiff >= 0 && takerSizeDiff*takerOrderBook.AskPrice*(1.0+*mtConfig.EnterSlippage) < takerMinNotional {
			continue
		}
		logger.Debugf("updateTakerPositions %s SIZE DIFF %f POS %f -> %f", takerSymbol, takerSizeDiff, takerPosition.PositionAmt, -makerSize)

		reduceOnly := false
		if takerSizeDiff*takerPosition.PositionAmt < 0 && math.Abs(takerSizeDiff) <= math.Abs(takerPosition.PositionAmt) {
			reduceOnly = true
		}
		price := math.Round(takerOrderBook.AskPrice*(1.0+*mtConfig.EnterSlippage)/takerTickSize) * takerTickSize
		side := "BUY"
		if takerSizeDiff < 0 {
			side = "SELL"
			takerSizeDiff = -takerSizeDiff
			price = math.Round(takerOrderBook.BidPrice*(1.0-*mtConfig.EnterSlippage)/takerTickSize) * takerTickSize
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

func updateMakerPositions() {

	if mAccount == nil {
		return
	}

	if tAccount == nil || tAccount.AvailableBalance == nil {
		return
	}

	if len(mtRankSymbolMap) == 0 {
		return
	}

	if mtUnHedgeValue > *mtConfig.MaxUnHedgeValue {
		if time.Now().Sub(mtUnHedgeLogSilentTimes) > 0 {
			logger.Debugf("TAKER UN HEDGE VALUE %f > %f", mtUnHedgeValue, *mtConfig.MaxUnHedgeValue)
			mtUnHedgeLogSilentTimes = time.Now().Add(*mtConfig.LogInterval)
		}
		return
	}

	entryStep := (mAccount.WithdrawAvailable + *tAccount.AvailableBalance) * *mtConfig.EnterFreePct
	if entryStep < *mtConfig.EnterMinimalStep {
		entryStep = *mtConfig.EnterMinimalStep
	}
	entryTarget := entryStep * *mtConfig.EnterTargetFactor

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
		quantile, okQuantile := mtQuantiles[makerSymbol]
		spread, okSpread := mtSpreads[makerSymbol]
		makerBuyPosition, okMakerBuyPosition := mBuyPositions[makerSymbol]
		makerSellPosition, okMakerSellPosition := mSellPositions[makerSymbol]
		fundingRate, okFundingRate := mtFundingRates[makerSymbol]
		if !okSpread || !okQuantile || !okMakerBuyPosition || !okMakerSellPosition || !okFundingRate {
			continue
		}
		if time.Now().Sub(spread.LastUpdateTime) > *mtConfig.SpreadTimeToLive {
			continue
		}
		makerOrderBook := spread.MakerOrderBook
		makerContractSize := mContractSizes[makerSymbol]
		makerTickSize := mTickSizes[makerSymbol]
		takerMinNotional := tMinNotional[takerSymbol]
		makerTakerStepSize := mtStepSizes[makerSymbol]
		makerStepSize := math.Ceil(makerContractSize/makerTickSize) * makerTickSize
		makerStepSize = math.Ceil(makerStepSize / makerTickSize)
		if time.Now().Sub(mtLogSilentTimes[makerSymbol]) > 0 {
			mtLogSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.LogInterval)
			logger.Debugf("LOOP %s", makerSymbol)
		}

		if spread.ShortLastExit < quantile.ShortBot &&
			spread.ShortMedianExit < quantile.ShortBot &&
			fundingRate < *mtConfig.MinimalKeepFundingRate &&
			makerBuyPosition.Volume > 0 {

			makerSize := makerBuyPosition.Volume * makerContractSize
			price := makerOrderBook.BidVWAP * (1.0 - *mtConfig.EnterSlippage)
			price = math.Ceil(price/makerTickSize) * makerTickSize
			entryValue := math.Max(4*entryStep, makerSize*price*0.5)
			if fundingRate > *mtConfig.MinimalKeepFundingRate*0.5 {
				entryValue = math.Max(2*entryStep, makerSize*price*0.5)
			}
			volume := entryValue / price
			volume = math.Round(volume/makerTakerStepSize) * makerTakerStepSize
			volume = math.Round(volume / makerContractSize)
			entryValue = volume * makerContractSize * price
			if makerSize*price-entryValue < entryStep {
				volume = makerBuyPosition.Volume
			}
			if volume > 0 {
				logger.Debugf(
					"SHORT BOT REDUCE %s %f < %f, %f < %f, SIZE %f",
					makerSymbol,
					spread.ShortLastExit, quantile.ShortBot,
					spread.ShortMedianExit, quantile.ShortBot,
					volume,
				)
				order := hbcrossswap.NewOrderParam{
					Symbol:        makerSymbol,
					ClientOrderID: time.Now().Unix()*10000 + int64(rand.Intn(10000)),
					//Price:          common.Float64(price),
					Volume:         int64(volume),
					Direction:      hbcrossswap.OrderDirectionSell,
					Offset:         hbcrossswap.OrderOffsetClose,
					LeverRate:      *mtConfig.Leverage,
					OrderPriceType: hbcrossswap.OrderPriceTypeFOKOptimal20FOK,
				}
				mOrderSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.OrderSilent)
				mOrderRequestChs[makerSymbol] <- order
				return
			}
		} else if spread.LongLastExit > quantile.LongTop &&
			spread.LongMedianExit > quantile.LongTop &&
			fundingRate > -*mtConfig.MinimalKeepFundingRate &&
			makerSellPosition.Volume > 0 {

			makerSize := makerSellPosition.Volume * makerContractSize
			price := makerOrderBook.AskVWAP * (1.0 + *mtConfig.EnterSlippage)
			price = math.Ceil(price/makerTickSize) * makerTickSize
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
					spread.LongLastExit, quantile.LongTop,
					spread.LongMedianExit, quantile.LongTop,
					volume,
				)
				order := hbcrossswap.NewOrderParam{
					Symbol:        makerSymbol,
					ClientOrderID: time.Now().Unix()*10000 + int64(rand.Intn(10000)),
					//Price:          common.Float64(price),
					Volume:         int64(volume),
					Direction:      hbcrossswap.OrderDirectionBuy,
					Offset:         hbcrossswap.OrderOffsetClose,
					LeverRate:      *mtConfig.Leverage,
					OrderPriceType: hbcrossswap.OrderPriceTypeFOKOptimal20FOK,
				}
				mOrderSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.OrderSilent)
				mHttpPositionUpdateSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.HttpSilent)
				mOrderRequestChs[makerSymbol] <- order
				return
			}
		} else if spread.ShortLastEnter > quantile.ShortTop &&
			spread.ShortMedianEnter > quantile.ShortTop &&
			//fundingRate > *mtConfig.MinimalEnterFundingRate &&
			makerSellPosition.Volume == 0 {
			makerSize := makerBuyPosition.Volume * makerContractSize
			price := makerOrderBook.AskVWAP * (1.0 + *mtConfig.EnterSlippage)
			price = math.Floor(price/makerTickSize) * makerTickSize
			targetValue := makerSize*price + entryStep
			if targetValue > entryTarget {
				targetValue = entryTarget
			}
			entryValue := targetValue - makerSize*price
			if entryValue > mAccount.WithdrawAvailable*0.8 {
				entryValue = mAccount.WithdrawAvailable * 0.8
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
			if entryValue > mAccount.WithdrawAvailable {
				if time.Now().Sub(mtLogSilentTimes[makerSymbol]) > 0 {
					logger.Debugf(
						"FAILED SHORT TOP OPEN, ENTRY VALUE %f MORE THAN WithdrawAvailable %f, %s %f > %f, %f > %f, SIZE %f",
						entryValue,
						mAccount.WithdrawAvailable,
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
			order := hbcrossswap.NewOrderParam{
				Symbol:        makerSymbol,
				ClientOrderID: time.Now().Unix()*10000 + int64(rand.Intn(10000)),
				//Price:          common.Float64(price),
				Volume:         int64(volume),
				Direction:      hbcrossswap.OrderDirectionBuy,
				Offset:         hbcrossswap.OrderOffsetOpen,
				LeverRate:      *mtConfig.Leverage,
				OrderPriceType: hbcrossswap.OrderPriceTypeFOKOptimal20FOK,
			}
			mOrderSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.OrderSilent)
			mHttpPositionUpdateSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.HttpSilent)
			mOrderRequestChs[makerSymbol] <- order
			return
		} else if spread.LongLastEnter < quantile.LongBot &&
			spread.LongMedianEnter < quantile.LongBot &&
			//fundingRate < -*mtConfig.MinimalEnterFundingRate &&
			makerBuyPosition.Volume == 0 {
			makerSize := makerSellPosition.Volume * makerContractSize
			price := makerOrderBook.BidVWAP * (1.0 - *mtConfig.EnterSlippage)
			price = math.Floor(price/makerTickSize) * makerTickSize
			targetValue := makerSize*price + entryStep
			if targetValue > entryTarget {
				targetValue = entryTarget
			}
			entryValue := targetValue - makerSize*price

			if entryValue > mAccount.WithdrawAvailable*0.8 {
				entryValue = mAccount.WithdrawAvailable * 0.8
			}

			volume := entryValue / price
			volume = math.Round(volume/makerTakerStepSize) * makerTakerStepSize
			volume = math.Round(volume / makerContractSize)

			entryValue = volume * makerContractSize * price

			//不及一个0.8*EntryStep, 不操作
			if entryValue < entryStep*0.8 {
				if time.Now().Sub(mtLogSilentTimes[makerSymbol]) > 0 {
					logger.Debugf(
						"FAILED LONG BOT OPEN, ENTRY VALUE %f LESS THAN 0.8*ENTRY_STEP %f, %s %f > %f, %f > %f, SIZE %f",
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
			if entryValue > mAccount.WithdrawAvailable {
				if time.Now().Sub(mtLogSilentTimes[makerSymbol]) > 0 {
					logger.Debugf(
						"FAILED LONG BOT OPEN, ENTRY VALUE %f MORE THAN WithdrawAvailable %f, %s %f > %f, %f > %f, SIZE %f",
						entryValue,
						mAccount.WithdrawAvailable,
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
						"FAILED LONG BOT OPEN, ORDER VALUE %f LESS THAN NOTIONAL %f, %s %f > %f, %f > %f, SIZE %f",
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
				"LONG BOT OPEN %s %f < %f, %f < %f, SIZE %f",
				makerSymbol,
				spread.LongLastEnter, quantile.LongBot,
				spread.LongMedianEnter, quantile.LongBot,
				volume,
			)
			order := hbcrossswap.NewOrderParam{
				Symbol:        makerSymbol,
				ClientOrderID: time.Now().Unix()*10000 + int64(rand.Intn(10000)),
				//Price:          common.Float64(price),
				Volume:         int64(volume),
				Direction:      hbcrossswap.OrderDirectionSell,
				Offset:         hbcrossswap.OrderOffsetOpen,
				LeverRate:      *mtConfig.Leverage,
				OrderPriceType: hbcrossswap.OrderPriceTypeFOKOptimal20FOK,
			}
			mHttpPositionUpdateSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.HttpSilent)
			mOrderSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.OrderSilent)
			mOrderRequestChs[makerSymbol] <- order
			return
		}
	}

}
func handleRestartSilent() {
	for _, makerSymbol := range mSymbols {
		mSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.RestartSilent)
	}
}

func handleUpdateFundingRates() {
	if mFundingRates == nil {
		return
	}
	if tPremiumIndexes == nil {
		return
	}
	//logger.Debugf("%v %v", mFundingRates, tPremiumIndexes)
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
