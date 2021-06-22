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
	takerUSDTAvailable := *tAccount.AvailableBalance

	for _, takerSymbol := range tSymbols {
		if time.Now().Sub(tPositionsUpdateTimes[takerSymbol]) > *mtConfig.PositionMaxAge {
			continue
		}
		if tOrderSilentTimes[takerSymbol].Sub(time.Now()).Seconds() > 0 {
			continue
		}
		if _, ok := tOpenOrders[takerSymbol]; ok {
			continue
		}
		takerPosition, okTakerPositions := tPositions[takerSymbol]
		takerTakerDepth, okSpread := mtDepths[takerSymbol]
		takerSignal, okSignal := mtSignals[takerSymbol]
		if !okTakerPositions || !okSpread || !okSignal {
			continue
		}
		takerStepSize := tStepSizes[takerSymbol]
		takerTickSize := tTickSizes[takerSymbol]
		takerMinNotional := tMinNotional[takerSymbol]

		takerSizeDiff := 0.0
		takerPrice := 0.0
		entryValue := 0.0

		//还在加多档期
		if takerSignal.Direction > 0 &&
			takerPosition.PositionAmt <= 0 {

			takerPrice = math.Floor(takerTakerDepth.MidPrice/takerTickSize) * takerTickSize
			takerSizeDiff = -takerPosition.PositionAmt + math.Floor(entryStep/takerPrice/takerStepSize)*takerStepSize
			entryValue = takerSizeDiff * takerPrice
			takerUSDTAvailable += -takerPosition.PositionAmt * takerPrice //补偿, 这一部分不占仓位
			takerSizeDiff = math.Floor(entryValue/takerPrice/takerStepSize) * takerStepSize
			entryValue = takerPrice * takerSizeDiff
			if entryValue < 0.8*entryStep {
				if time.Now().Sub(mtLogSilentTimes[takerSymbol]) > 0 {
					logger.Debugf(
						"%s FAILED LONG OPEN, ENTRY VALUE %f LESS THAN 0.8*ENTRY_STEP %f, SIZE %f",
						takerSymbol,
						entryValue,
						entryStep*0.8,
						takerSizeDiff,
					)
					mtLogSilentTimes[takerSymbol] = time.Now().Add(*mtConfig.LogInterval)
				}
				continue
			}
			if entryValue > takerUSDTAvailable {
				if time.Now().Sub(mtLogSilentTimes[takerSymbol]) > 0 {
					logger.Debugf(
						"%s FAILED LONG OPEN, ENTRY VALUE %f MORE THAN takerUSDTAvailable %f, SIZE %f",
						takerSymbol,
						entryValue,
						takerUSDTAvailable,
						takerSizeDiff,
					)
					mtLogSilentTimes[takerSymbol] = time.Now().Add(*mtConfig.LogInterval)
				}
				continue
			}
			if entryValue < takerMinNotional {
				if time.Now().Sub(mtLogSilentTimes[takerSymbol]) > 0 {
					logger.Debugf(
						"%s FAILED LONG OPEN, ORDER VALUE %f LESS THAN NOTIONAL %f, SIZE %f",
						takerSymbol,
						entryValue,
						takerMinNotional,
						takerSizeDiff,
					)
					mtLogSilentTimes[takerSymbol] = time.Now().Add(*mtConfig.LogInterval)
				}
				continue
			}
			if takerPosition.PositionAmt >= 0 {
				takerUSDTAvailable -= entryValue
			}
		} else if takerSignal.Direction < 0 &&
			takerPosition.PositionAmt >= 0 {

			takerPrice = math.Ceil(takerTakerDepth.MidPrice/takerTickSize) * takerTickSize

			takerSizeDiff = -takerPosition.PositionAmt - math.Floor(entryStep/takerPrice/takerStepSize)*takerStepSize
			entryValue = takerSizeDiff * takerPrice
			takerUSDTAvailable += takerPosition.PositionAmt * takerPrice //补偿, 这一部分不占仓位
			takerSizeDiff = math.Ceil(entryValue/takerPrice/takerStepSize) * takerStepSize
			entryValue = takerPrice * takerSizeDiff

			if -entryValue < 0.8*entryStep {
				if time.Now().Sub(mtLogSilentTimes[takerSymbol]) > 0 {
					logger.Debugf(
						"%s FAILED SHORT OPEN, ENTRY VALUE %f LESS THAN 0.8*ENTRY_STEP %f, SIZE %f",
						takerSymbol,
						-entryValue,
						entryStep*0.8,
						takerSizeDiff,
					)
					mtLogSilentTimes[takerSymbol] = time.Now().Add(*mtConfig.LogInterval)
				}
				continue
			}
			if -entryValue > takerUSDTAvailable {
				if time.Now().Sub(mtLogSilentTimes[takerSymbol]) > 0 {
					logger.Debugf(
						"%s FAILED SHORT OPEN, ENTRY VALUE %f MORE THAN takerUSDTAvailable %f, SIZE %f",
						takerSymbol,
						-entryValue,
						takerUSDTAvailable,
						takerSizeDiff,
					)
					mtLogSilentTimes[takerSymbol] = time.Now().Add(*mtConfig.LogInterval)
				}
				continue
			}
			if -entryValue < takerMinNotional {
				if time.Now().Sub(mtLogSilentTimes[takerSymbol]) > 0 {
					logger.Debugf(
						"%s FAILED SHORT TOP OPEN, ORDER VALUE %f LESS THAN NOTIONAL %f, SIZE %f",
						takerSymbol,
						-entryValue,
						takerMinNotional,
						takerSizeDiff,
					)
					mtLogSilentTimes[takerSymbol] = time.Now().Add(*mtConfig.LogInterval)
				}
				continue
			}
			if takerPosition.PositionAmt >= 0 {
				takerUSDTAvailable -= -entryValue
			}
		} else if takerPosition.PositionAmt > 0 {
			//logger.Debugf("CLOSE LONG %s", takerSymbol)
			if tCloseTimeouts[takerSymbol].Sub(time.Now()) > 0 {
				takerPrice = (1.0 + float64(tCloseTimeouts[takerSymbol].Sub(time.Now()))/float64(*mtConfig.CloseTimeout)**mtConfig.CloseProfitPct) * takerPosition.EntryPrice
				takerPrice = math.Ceil(takerPrice/takerTickSize) * takerTickSize
			}
			takerSizeDiff = -takerPosition.PositionAmt
		} else if takerPosition.PositionAmt < 0 {
			//logger.Debugf("CLOSE SHORT %s", takerSymbol)
			if tCloseTimeouts[takerSymbol].Sub(time.Now()) > 0 {
				takerPrice = (1.0 - float64(tCloseTimeouts[takerSymbol].Sub(time.Now()))/float64(*mtConfig.CloseTimeout)**mtConfig.CloseProfitPct) * takerPosition.EntryPrice
				takerPrice = math.Floor(takerPrice/takerTickSize) * takerTickSize
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

		if takerSizeDiff > 0 && takerPrice > takerTakerDepth.BestBidPrice {
			takerPrice = takerTakerDepth.BestBidPrice
		}
		if takerSizeDiff < 0 {
			side = "SELL"
			takerSizeDiff = -takerSizeDiff
			if takerPrice < takerTakerDepth.BestAskPrice {
				takerPrice = takerTakerDepth.BestAskPrice
			}
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
			logger.Debugf("%s CLOSE TIMEOUT %v", takerSymbol, time.Now().Sub(tCloseTimeouts[takerSymbol]))
			takerOrder.Type = common.OrderTypeMarket
			takerOrder.Price = 0
			takerOrder.TimeInForce = ""
		}
		tOpenOrders[takerSymbol] = TakerOpenOrder{NewOrderParams: &takerOrder, Symbol: takerSymbol}
		tOrderSilentTimes[takerSymbol] = time.Now().Add(*mtConfig.OrderSilent)
		tOrderRequestChs[takerSymbol] <- TakerOrderRequest{New: &takerOrder}
	}
}
