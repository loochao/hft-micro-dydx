package main

import (
	"fmt"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"math/rand"
	"time"
)

func updateTakerNewOrders() {
	if tAccount == nil || tAccount.AvailableBalance == nil {
		return
	}
	entryStep := *tAccount.AvailableBalance * *mtConfig.EnterFreePct
	if entryStep < *mtConfig.EnterMinimalStep {
		entryStep = *mtConfig.EnterMinimalStep
	}
	entryTarget := entryStep * *mtConfig.EnterTargetFactor
	takerUSDTAvailable := *tAccount.AvailableBalance

	for _, takerSymbol := range tSymbols {
		makerSymbol := tmSymbolsMap[takerSymbol]
		if time.Now().Sub(tPositionsUpdateTimes[takerSymbol]) > *mtConfig.BalancePositionMaxAge {
			continue
		}
		if tOrderSilentTimes[takerSymbol].Sub(time.Now()).Seconds() > 0 {
			continue
		}
		if _, ok := tOpenOrders[takerSymbol]; ok {
			continue
		}
		takerPosition, okTakerPositions := tPositions[takerSymbol]
		spread, okSpread := mtSpreads[makerSymbol]
		if !okTakerPositions || !okSpread {
			continue
		}
		takerTakerDepth := spread.TakerDepth
		takerStepSize := tStepSizes[takerSymbol]
		takerTickSize := tTickSizes[takerSymbol]
		takerMinNotional := tMinNotional[takerSymbol]

		takerSizeDiff := 0.0
		takerPrice := 0.0

		//还在加多档期
		if mtTriggeredDirection[takerSymbol] > 0 &&
			mtEnterTimeouts[takerSymbol].Sub(time.Now()) > 0 &&
			mtEnterSilentTimes[takerSymbol].Sub(time.Now()) < 0 {
			entryValue := takerPosition.PositionAmt*takerPosition.EntryPrice + entryStep
			if entryValue > entryTarget {
				entryValue = entryTarget
			}
			entryValue -= takerPosition.PositionAmt * takerPosition.EntryPrice
			takerPrice = math.Floor(takerTakerDepth.MidPrice/takerTickSize) * takerTickSize
			takerSizeDiff = math.Floor(entryValue/takerStepSize) * takerStepSize
			entryValue = takerPrice * takerSizeDiff
			if entryValue < 0.8*entryStep {
				if time.Now().Sub(mtLogSilentTimes[makerSymbol]) > 0 {
					logger.Debugf(
						"%s FAILED LONG OPEN, ENTRY VALUE %f LESS THAN 0.8*ENTRY_STEP %f, SIZE %f",
						takerSymbol,
						entryValue,
						entryStep*0.8,
						takerSizeDiff,
					)
					mtLogSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.LogInterval)
				}
				continue
			}
			if entryValue > takerUSDTAvailable {
				if time.Now().Sub(mtLogSilentTimes[makerSymbol]) > 0 {
					logger.Debugf(
						"%s FAILED LONG OPEN, ENTRY VALUE %f MORE THAN takerUSDTAvailable %f, SIZE %f",
						takerSymbol,
						entryValue,
						takerUSDTAvailable,
						takerSizeDiff,
					)
					mtLogSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.LogInterval)
				}
				continue
			}
			if entryValue < takerMinNotional {
				if time.Now().Sub(mtLogSilentTimes[makerSymbol]) > 0 {
					logger.Debugf(
						"%s FAILED LONG OPEN, ORDER VALUE %f LESS THAN NOTIONAL %f, SIZE %f",
						takerSymbol,
						entryValue,
						takerMinNotional,
						takerSizeDiff,
					)
					mtLogSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.LogInterval)
				}
				continue
			}
		} else if mtTriggeredDirection[takerSymbol] < 0 &&
			mtEnterTimeouts[takerSymbol].Sub(time.Now()) > 0 &&
			mtEnterSilentTimes[takerSymbol].Sub(time.Now()) < 0 {
			entryValue := takerPosition.PositionAmt*takerPosition.EntryPrice - entryStep
			if entryValue < -entryTarget {
				entryValue = -entryTarget
			}
			entryValue -= takerPosition.PositionAmt * takerPosition.EntryPrice
			takerPrice = math.Ceil(takerTakerDepth.MidPrice/takerTickSize) * takerTickSize
			takerSizeDiff = math.Ceil(entryValue/takerStepSize) * takerStepSize
			entryValue = takerPrice * takerSizeDiff
			if -entryValue < 0.8*entryStep {
				if time.Now().Sub(mtLogSilentTimes[makerSymbol]) > 0 {
					logger.Debugf(
						"%s FAILED SHORT OPEN, ENTRY VALUE %f LESS THAN 0.8*ENTRY_STEP %f, SIZE %f",
						takerSymbol,
						-entryValue,
						entryStep*0.8,
						takerSizeDiff,
					)
					mtLogSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.LogInterval)
				}
				continue
			}
			if -entryValue > takerUSDTAvailable {
				if time.Now().Sub(mtLogSilentTimes[makerSymbol]) > 0 {
					logger.Debugf(
						"%s FAILED SHORT TOP OPEN, ENTRY VALUE %f MORE THAN takerUSDTAvailable %f, SIZE %f",
						takerSymbol,
						-entryValue,
						takerUSDTAvailable,
						takerSizeDiff,
					)
					mtLogSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.LogInterval)
				}
				continue
			}
			if -entryValue < takerMinNotional {
				if time.Now().Sub(mtLogSilentTimes[makerSymbol]) > 0 {
					logger.Debugf(
						"%s FAILED SHORT TOP OPEN, ORDER VALUE %f LESS THAN NOTIONAL %f, SIZE %f",
						takerSymbol,
						-entryValue,
						takerMinNotional,
						takerSizeDiff,
					)
					mtLogSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.LogInterval)
				}
				continue
			}
		}else if takerPosition.PositionAmt > 0 {
			if mtCloseTimeouts[takerSymbol].Sub(time.Now()) > 0 {
				takerPrice =  (1.0 + float64(mtCloseTimeouts[takerSymbol].Sub(time.Now()))/float64(*mtConfig.CloseTimeout))*takerTakerDepth.MidPrice
				takerPrice = math.Ceil(takerPrice/takerTickSize)*takerTickSize
			}
			takerSizeDiff = -takerPosition.PositionAmt
		}else if takerPosition.PositionAmt < 0 {
			if mtCloseTimeouts[takerSymbol].Sub(time.Now()) > 0 {
				takerPrice =  (1.0 - float64(mtCloseTimeouts[takerSymbol].Sub(time.Now()))/float64(*mtConfig.CloseTimeout))*takerTakerDepth.MidPrice
				takerPrice = math.Floor(takerPrice/takerTickSize)*takerTickSize
			}
			takerSizeDiff = -takerPosition.PositionAmt
		}

		if math.Abs(takerSizeDiff) < takerStepSize {
			continue
		} else if takerSizeDiff < 0 && takerPosition.PositionAmt <= 0 && -takerSizeDiff*takerPrice < takerMinNotional {
			continue
		} else if takerSizeDiff > 0 && takerPosition.PositionAmt >= 0 && takerSizeDiff*takerPrice < takerMinNotional {
			continue
		}
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
			Type:             common.OrderTypeLimit,
			Price:            takerPrice,
			TimeInForce:      common.OrderTimeInForceGTX,
			Quantity:         takerSizeDiff,
			ReduceOnly:       reduceOnly,
			NewClientOrderId: fmt.Sprintf("%d", time.Now().Unix()*10000+int64(rand.Intn(10000))),
		}
		if takerPrice == 0 {
			logger.Debugf("%s HEDGE TIMEOUT", takerSymbol)
			takerOrder.Type = common.OrderTypeMarket
			takerOrder.Price = 0
			takerOrder.TimeInForce = ""
		}
		tOpenOrders[takerSymbol] = TakerOpenOrder{NewOrderParams: &takerOrder, Symbol: takerSymbol}
		tOrderSilentTimes[takerSymbol] = time.Now().Add(*mtConfig.OrderSilent)
		tPositionsUpdateTimes[takerSymbol] = time.Unix(0, 0)
		tOrderRequestChs[takerSymbol] <- TakerOrderRequest{New: &takerOrder}
	}
}
