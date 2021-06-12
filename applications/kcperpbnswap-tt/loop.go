package main

import (
	"fmt"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/kucoin-usdtfuture"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"math/rand"
	"time"
)

func updateTakerPositions() {
	unHedgedValue := 0.0
	for _, takerSymbol := range tSymbols {
		makerSymbol := tmSymbolsMap[takerSymbol]
		if time.Now().Sub(mtTriggerSilentTimes[makerSymbol]) < 0 {
			continue
		}
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
		takerSizeDiff = math.Round(takerSizeDiff/takerStepSize) * takerStepSize
		if takerSizeDiff > 0 {
			unHedgedValue += math.Abs(takerSizeDiff * takerTakerDepth.TakerAsk)
		} else {
			unHedgedValue += math.Abs(takerSizeDiff * takerTakerDepth.TakerBid)
		}

		if math.Abs(takerSizeDiff) < takerStepSize {
			continue
		} else if takerSizeDiff < 0 && takerPosition.PositionAmt <= 0 && -takerSizeDiff*takerTakerDepth.TakerBid*(1.0-*mtConfig.EnterSlippage) < takerMinNotional {
			continue
		} else if takerSizeDiff > 0 && takerPosition.PositionAmt >= 0 && takerSizeDiff*takerTakerDepth.TakerAsk*(1.0+*mtConfig.EnterSlippage) < takerMinNotional {
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

func updateTriggerOrders() {

	if mAccount == nil {
		return
	}

	if tAccount == nil || tAccount.AvailableBalance == nil {
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
	takerUSDTAvailable := *tAccount.AvailableBalance

	for _, makerSymbol := range mSymbols {
		takerSymbol := mtSymbolsMap[makerSymbol]
		if time.Now().Sub(mtTriggerSilentTimes[makerSymbol]) < 0 {
			continue
		}
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
		makerPosition, okMakerPosition := mPositions[makerSymbol]
		takerPosition, okTakerPosition := tPositions[takerSymbol]
		if !okSpread || !okQuantile || !okMakerPosition || !okTakerPosition {
			continue
		}

		if time.Now().Sub(spread.Time) > *mtConfig.SpreadTimeToLive {
			continue
		}
		makerDepth := spread.MakerDepth
		makerMultiplier := mMultipliers[makerSymbol]
		takerMinNotional := tMinNotional[takerSymbol]
		makerTakerStepSize := mtStepSizes[makerSymbol]

		if spread.ShortLastLeave < quantile.ShortBot &&
			spread.ShortMedianLeave < quantile.ShortBot &&
			makerPosition.CurrentQty > 0 &&
			takerPosition.PositionAmt < 0 {

			makerSize := makerPosition.CurrentQty * makerMultiplier
			makerRefPrice := makerDepth.TakerBid
			entryValue := math.Max(4*entryStep, makerSize*makerRefPrice*0.5)
			makerOrderSize := entryValue / makerRefPrice
			makerOrderSize = math.Round(makerOrderSize/makerTakerStepSize) * makerTakerStepSize
			makerOrderSize = math.Round(makerOrderSize / makerMultiplier)
			entryValue = makerOrderSize * makerMultiplier * makerRefPrice
			if makerSize*makerRefPrice-entryValue < entryStep {
				makerOrderSize = makerPosition.CurrentQty
			}
			takerOrderSize := math.Round(makerOrderSize*makerMultiplier/makerTakerStepSize) * makerTakerStepSize
			if takerOrderSize > -takerPosition.PositionAmt {
				takerOrderSize = -takerPosition.PositionAmt
			}
			if makerOrderSize > 0 && takerOrderSize > 0 {

				logger.Debugf(
					"SHORT BOT REDUCE %s %f < %f, %f < %f, SIZE %f",
					makerSymbol,
					spread.ShortLastLeave, quantile.ShortBot,
					spread.ShortMedianLeave, quantile.ShortBot,
					makerOrderSize,
				)
				clientOID := fmt.Sprintf("CS%d%04d", time.Now().Unix(), rand.Intn(10000))
				makerOrder := kucoin_usdtfuture.NewOrderParam{
					Symbol:     makerSymbol,
					Side:       kucoin_usdtfuture.OrderSideSell,
					Type:       kucoin_usdtfuture.OrderTypeMarket,
					Size:       int64(makerOrderSize),
					ReduceOnly: true,
					ClientOid:  clientOID,
					Leverage:   *mtConfig.Leverage,
				}
				takerOrder := bnswap.NewOrderParams{
					Symbol:           takerSymbol,
					Side:             common.OrderSideBuy,
					Type:             common.OrderTypeMarket,
					Quantity:         takerOrderSize,
					ReduceOnly:       true,
					NewClientOrderId: clientOID,
				}

				mtTriggerSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.TriggerInterval)

				mOrderSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.OrderSilent)
				mHttpPositionUpdateSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.HttpSilent)
				mOrderRequestChs[makerSymbol] <- makerOrder

				tOrderSilentTimes[takerSymbol] = time.Now().Add(*mtConfig.OrderSilent)
				tHttpPositionUpdateSilentTimes[takerSymbol] = time.Now().Add(*mtConfig.HttpSilent)
				tOrderRequestChs[takerSymbol] <- takerOrder
			}
		} else if spread.LongLastLeave > quantile.LongTop &&
			spread.LongMedianLeave > quantile.LongTop &&
			makerPosition.CurrentQty < 0 &&
			takerPosition.PositionAmt > 0 {

			makerSize := -makerPosition.CurrentQty * makerMultiplier
			makerRefPrice := makerDepth.TakerAsk
			entryValue := math.Max(4*entryStep, makerSize*makerRefPrice*0.5)
			makerOrderSize := entryValue / makerRefPrice
			makerOrderSize = math.Round(makerOrderSize/makerTakerStepSize) * makerTakerStepSize
			makerOrderSize = math.Round(makerOrderSize / makerMultiplier)
			if makerSize*makerRefPrice-entryValue < entryStep {
				makerOrderSize = -makerPosition.CurrentQty
			}
			takerOrderSize := math.Round(makerOrderSize*makerMultiplier/makerTakerStepSize) * makerTakerStepSize
			if takerOrderSize > takerPosition.PositionAmt {
				takerOrderSize = takerPosition.PositionAmt
			}
			if makerOrderSize > 0 {
				logger.Debugf(
					"LONG TOP REDUCE %s %f > %f, %f > %f, VOLUME %f",
					makerSymbol,
					spread.LongLastLeave, quantile.LongTop,
					spread.LongMedianLeave, quantile.LongTop,
					makerOrderSize,
				)
				clientOID := fmt.Sprintf("CS%d%04d", time.Now().Unix(), rand.Intn(10000))
				makerOrder := kucoin_usdtfuture.NewOrderParam{
					Symbol:     makerSymbol,
					Side:       kucoin_usdtfuture.OrderSideBuy,
					Type:       kucoin_usdtfuture.OrderTypeMarket,
					Size:       int64(makerOrderSize),
					ReduceOnly: true,
					ClientOid:  clientOID,
					Leverage:   *mtConfig.Leverage,
				}
				takerOrder := bnswap.NewOrderParams{
					Symbol:           takerSymbol,
					Side:             common.OrderSideSell,
					Type:             common.OrderTypeMarket,
					Quantity:         takerOrderSize,
					ReduceOnly:       true,
					NewClientOrderId: clientOID,
				}
				mtTriggerSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.TriggerInterval)

				mOrderSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.OrderSilent)
				mHttpPositionUpdateSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.HttpSilent)
				mOrderRequestChs[makerSymbol] <- makerOrder

				tOrderSilentTimes[takerSymbol] = time.Now().Add(*mtConfig.OrderSilent)
				tHttpPositionUpdateSilentTimes[takerSymbol] = time.Now().Add(*mtConfig.HttpSilent)
				tOrderRequestChs[takerSymbol] <- takerOrder
			}
		} else if spread.ShortLastEnter > quantile.ShortTop &&
			spread.ShortMedianEnter > quantile.ShortTop &&
			makerPosition.CurrentQty >= 0 &&
			takerPosition.PositionAmt <= 0 {

			makerSize := makerPosition.CurrentQty * makerMultiplier
			makerRefPrice := makerDepth.TakerAsk
			targetValue := makerSize*makerRefPrice + entryStep
			if targetValue > entryTarget {
				targetValue = entryTarget
			}
			entryValue := targetValue - makerSize*makerRefPrice
			if entryValue > makerUSDTAvailable*0.8 {
				entryValue = makerUSDTAvailable * 0.8
			}
			makerOrderSize := entryValue / makerRefPrice
			makerOrderSize = math.Round(makerOrderSize/makerTakerStepSize) * makerTakerStepSize
			makerOrderSize = math.Round(makerOrderSize / makerMultiplier)
			entryValue = makerOrderSize * makerMultiplier * makerRefPrice
			takerOrderSize := math.Round(makerOrderSize*makerMultiplier/makerTakerStepSize) * makerTakerStepSize

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
						makerOrderSize,
					)
					mtLogSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.LogInterval)
				}
				continue
			}
			if entryValue > makerUSDTAvailable {
				if time.Now().Sub(mtLogSilentTimes[makerSymbol]) > 0 {
					logger.Debugf(
						"FAILED SHORT TOP OPEN, ENTRY VALUE %f MORE THAN makerUSDTAvailable %f, %s %f > %f, %f > %f, SIZE %f",
						entryValue,
						makerUSDTAvailable,
						makerSymbol,
						spread.ShortLastEnter, quantile.ShortTop,
						spread.ShortMedianEnter, quantile.ShortTop,
						makerOrderSize,
					)
					mtLogSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.LogInterval)
				}
				continue
			}
			if entryValue > takerUSDTAvailable {
				if time.Now().Sub(mtLogSilentTimes[makerSymbol]) > 0 {
					logger.Debugf(
						"FAILED SHORT TOP OPEN, ENTRY VALUE %f MORE THAN takerUSDTAvailable %f, %s %f > %f, %f > %f, SIZE %f",
						entryValue,
						takerUSDTAvailable,
						makerSymbol,
						spread.ShortLastEnter, quantile.ShortTop,
						spread.ShortMedianEnter, quantile.ShortTop,
						makerOrderSize,
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
						makerOrderSize,
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
				makerOrderSize,
			)
			makerUSDTAvailable -= entryValue
			takerUSDTAvailable -= entryValue
			clientOID := fmt.Sprintf("OS%d%04d", time.Now().Unix(), rand.Intn(10000))
			makerOrder := kucoin_usdtfuture.NewOrderParam{
				Symbol:     makerSymbol,
				Side:       kucoin_usdtfuture.OrderSideBuy,
				Type:       kucoin_usdtfuture.OrderTypeMarket,
				Size:       int64(makerOrderSize),
				ReduceOnly: false,
				ClientOid:  clientOID,
				Leverage:   *mtConfig.Leverage,
			}
			takerOrder := bnswap.NewOrderParams{
				Symbol:           takerSymbol,
				Side:             common.OrderSideSell,
				Type:             common.OrderTypeMarket,
				Quantity:         takerOrderSize,
				ReduceOnly:       false,
				NewClientOrderId: clientOID,
			}
			mtTriggerSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.TriggerInterval)

			mOrderSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.OrderSilent)
			mHttpPositionUpdateSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.HttpSilent)
			mOrderRequestChs[makerSymbol] <- makerOrder

			tOrderSilentTimes[takerSymbol] = time.Now().Add(*mtConfig.OrderSilent)
			tHttpPositionUpdateSilentTimes[takerSymbol] = time.Now().Add(*mtConfig.HttpSilent)
			tOrderRequestChs[takerSymbol] <- takerOrder
		} else if spread.LongLastEnter < quantile.LongBot &&
			spread.LongMedianEnter < quantile.LongBot &&
			makerPosition.CurrentQty <= 0 &&
			takerPosition.PositionAmt >= 0 {

			makerSize := -makerPosition.CurrentQty * makerMultiplier
			makerRefPrice := makerDepth.TakerBid
			targetValue := makerSize*makerRefPrice + entryStep
			if targetValue > entryTarget {
				targetValue = entryTarget
			}
			entryValue := targetValue - makerSize*makerRefPrice
			if entryValue > makerUSDTAvailable*0.8 {
				entryValue = makerUSDTAvailable * 0.8
			}
			makerOrderSize := entryValue / makerRefPrice
			makerOrderSize = math.Round(makerOrderSize/makerTakerStepSize) * makerTakerStepSize
			makerOrderSize = math.Round(makerOrderSize / makerMultiplier)
			entryValue = makerOrderSize * makerMultiplier * makerRefPrice
			takerOrderSize := math.Round(makerOrderSize*makerMultiplier/makerTakerStepSize) * makerTakerStepSize

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
						makerOrderSize,
					)
					mtLogSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.LogInterval)
				}
				continue
			}
			if entryValue > makerUSDTAvailable {
				if time.Now().Sub(mtLogSilentTimes[makerSymbol]) > 0 {
					logger.Debugf(
						"FAILED SHORT TOP OPEN, ENTRY VALUE %f MORE THAN makerUSDTAvailable %f, %s %f > %f, %f > %f, SIZE %f",
						entryValue,
						makerUSDTAvailable,
						makerSymbol,
						spread.ShortLastEnter, quantile.ShortTop,
						spread.ShortMedianEnter, quantile.ShortTop,
						makerOrderSize,
					)
					mtLogSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.LogInterval)
				}
				continue
			}
			if entryValue > takerUSDTAvailable {
				if time.Now().Sub(mtLogSilentTimes[makerSymbol]) > 0 {
					logger.Debugf(
						"FAILED SHORT TOP OPEN, ENTRY VALUE %f MORE THAN takerUSDTAvailable %f, %s %f > %f, %f > %f, SIZE %f",
						entryValue,
						takerUSDTAvailable,
						makerSymbol,
						spread.ShortLastEnter, quantile.ShortTop,
						spread.ShortMedianEnter, quantile.ShortTop,
						makerOrderSize,
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
						makerOrderSize,
					)
					mtLogSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.LogInterval)
				}
				continue
			}
			mtLogSilentTimes[makerSymbol] = time.Now()
			logger.Debugf(
				"LONG TOP OPEN %s %f < %f, %f < %f, SIZE %f",
				makerSymbol,
				spread.LongLastEnter, quantile.LongBot,
				spread.LongMedianEnter, quantile.LongBot,
				makerOrderSize,
			)
			makerUSDTAvailable -= entryValue
			takerUSDTAvailable -= entryValue
			clientOID := fmt.Sprintf("OL%d%04d", time.Now().Unix(), rand.Intn(10000))
			makerOrder := kucoin_usdtfuture.NewOrderParam{
				Symbol:     makerSymbol,
				Side:       kucoin_usdtfuture.OrderSideSell,
				Type:       kucoin_usdtfuture.OrderTypeMarket,
				Size:       int64(makerOrderSize),
				ReduceOnly: false,
				ClientOid:  clientOID,
				Leverage:   *mtConfig.Leverage,
			}
			takerOrder := bnswap.NewOrderParams{
				Symbol:           takerSymbol,
				Side:             common.OrderSideBuy,
				Type:             common.OrderTypeMarket,
				Quantity:         takerOrderSize,
				ReduceOnly:       false,
				NewClientOrderId: clientOID,
			}
			mtTriggerSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.TriggerInterval)

			mOrderSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.OrderSilent)
			mHttpPositionUpdateSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.HttpSilent)
			mOrderRequestChs[makerSymbol] <- makerOrder

			tOrderSilentTimes[takerSymbol] = time.Now().Add(*mtConfig.OrderSilent)
			tHttpPositionUpdateSilentTimes[takerSymbol] = time.Now().Add(*mtConfig.HttpSilent)
			tOrderRequestChs[takerSymbol] <- takerOrder
		}
	}
}

