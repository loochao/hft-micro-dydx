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
	if swapAccount == nil || swapAccount.AvailableBalance == nil {
		return
	}
	entryStep := *swapAccount.AvailableBalance * *mtConfig.EnterFreePct
	if entryStep < *mtConfig.EnterMinimalStep {
		entryStep = *mtConfig.EnterMinimalStep
	}
	takerUSDTAvailable := *swapAccount.AvailableBalance

	for _, swapSymbol := range swapSymbols {
		if time.Now().Sub(swapPositionsUpdateTimes[swapSymbol]) > *mtConfig.PositionMaxAge {
			continue
		}
		if tOrderSilentTimes[swapSymbol].Sub(time.Now()).Seconds() > 0 {
			continue
		}
		if _, ok := swapOpenOrders[swapSymbol]; ok {
			continue
		}
		swapPosition, okTakerPositions := swapPositions[swapSymbol]
		swapTakerDepth, okSpread := swapWalkedDepths[swapSymbol]
		if !okTakerPositions || !okSpread {
			continue
		}
		takerStepSize := swapStepSizes[swapSymbol]
		swapTickSize := swapTickSizes[swapSymbol]
		takerMinNotional := swapMinNotional[swapSymbol]

		swapSizeDiff := 0.0
		swapOrderPrice := 0.0
		entryValue := 0.0

		//还在加多档期
		if swapTakerDepth.EmaBidAskRatio > 4 &&
			swapPosition.PositionAmt <= 0 {
			swapOrderPrice = math.Floor(swapTakerDepth.MidPrice/swapTickSize) * swapTickSize
			swapSizeDiff = -swapPosition.PositionAmt + math.Floor(entryStep/swapOrderPrice/takerStepSize)*takerStepSize
			entryValue = swapSizeDiff * swapOrderPrice
			takerUSDTAvailable += -swapPosition.PositionAmt * swapOrderPrice //补偿, 这一部分不占仓位
			swapSizeDiff = math.Floor(entryValue/swapOrderPrice/takerStepSize) * takerStepSize
			entryValue = swapOrderPrice * swapSizeDiff
			if entryValue < 0.8*entryStep {
				if time.Now().Sub(mtLogSilentTimes[swapSymbol]) > 0 {
					logger.Debugf(
						"%s FAILED LONG OPEN, ENTRY VALUE %f LESS THAN 0.8*ENTRY_STEP %f, SIZE %f",
						swapSymbol,
						entryValue,
						entryStep*0.8,
						swapSizeDiff,
					)
					mtLogSilentTimes[swapSymbol] = time.Now().Add(*mtConfig.LogInterval)
				}
				continue
			}
			if entryValue > takerUSDTAvailable {
				if time.Now().Sub(mtLogSilentTimes[swapSymbol]) > 0 {
					logger.Debugf(
						"%s FAILED LONG OPEN, ENTRY VALUE %f MORE THAN takerUSDTAvailable %f, SIZE %f",
						swapSymbol,
						entryValue,
						takerUSDTAvailable,
						swapSizeDiff,
					)
					mtLogSilentTimes[swapSymbol] = time.Now().Add(*mtConfig.LogInterval)
				}
				continue
			}
			if entryValue < takerMinNotional {
				if time.Now().Sub(mtLogSilentTimes[swapSymbol]) > 0 {
					logger.Debugf(
						"%s FAILED LONG OPEN, ORDER VALUE %f LESS THAN NOTIONAL %f, SIZE %f",
						swapSymbol,
						entryValue,
						takerMinNotional,
						swapSizeDiff,
					)
					mtLogSilentTimes[swapSymbol] = time.Now().Add(*mtConfig.LogInterval)
				}
				continue
			}
			if swapPosition.PositionAmt <= 0 {
				takerUSDTAvailable -= entryValue
			}
			logger.Debugf("%s OPEN LONG@%f %f %f", swapSymbol, swapOrderPrice,swapPosition.PositionAmt, swapTakerDepth.EmaBidAskRatio)
		} else if swapTakerDepth.EmaAskBidRatio > 4 &&
			swapPosition.PositionAmt >= 0 {

			swapOrderPrice = math.Ceil(swapTakerDepth.MidPrice/swapTickSize) * swapTickSize

			swapSizeDiff = -swapPosition.PositionAmt - math.Floor(entryStep/swapOrderPrice/takerStepSize)*takerStepSize
			entryValue = swapSizeDiff * swapOrderPrice
			takerUSDTAvailable += swapPosition.PositionAmt * swapOrderPrice //补偿, 这一部分不占仓位
			swapSizeDiff = math.Ceil(entryValue/swapOrderPrice/takerStepSize) * takerStepSize
			entryValue = swapOrderPrice * swapSizeDiff

			if -entryValue < 0.8*entryStep {
				if time.Now().Sub(mtLogSilentTimes[swapSymbol]) > 0 {
					logger.Debugf(
						"%s FAILED SHORT OPEN, ENTRY VALUE %f LESS THAN 0.8*ENTRY_STEP %f, SIZE %f",
						swapSymbol,
						-entryValue,
						entryStep*0.8,
						swapSizeDiff,
					)
					mtLogSilentTimes[swapSymbol] = time.Now().Add(*mtConfig.LogInterval)
				}
				continue
			}
			if -entryValue > takerUSDTAvailable {
				if time.Now().Sub(mtLogSilentTimes[swapSymbol]) > 0 {
					logger.Debugf(
						"%s FAILED SHORT OPEN, ENTRY VALUE %f MORE THAN takerUSDTAvailable %f, SIZE %f",
						swapSymbol,
						-entryValue,
						takerUSDTAvailable,
						swapSizeDiff,
					)
					mtLogSilentTimes[swapSymbol] = time.Now().Add(*mtConfig.LogInterval)
				}
				continue
			}
			if -entryValue < takerMinNotional {
				if time.Now().Sub(mtLogSilentTimes[swapSymbol]) > 0 {
					logger.Debugf(
						"%s FAILED SHORT TOP OPEN, ORDER VALUE %f LESS THAN NOTIONAL %f, SIZE %f",
						swapSymbol,
						-entryValue,
						takerMinNotional,
						swapSizeDiff,
					)
					mtLogSilentTimes[swapSymbol] = time.Now().Add(*mtConfig.LogInterval)
				}
				continue
			}
			if swapPosition.PositionAmt >= 0 {
				takerUSDTAvailable -= -entryValue
			}
			logger.Debugf("%s OPEN SHORT@%f %f %f", swapSymbol, swapOrderPrice,swapPosition.PositionAmt, swapTakerDepth.EmaAskBidRatio)
		} else if swapPosition.PositionAmt > 0 &&
			swapTakerDepth.EmaBidAskRatio < 1.0 {
			swapOrderPrice = math.Ceil(swapTakerDepth.MidPrice/swapTickSize) * swapTickSize
			swapSizeDiff = -swapPosition.PositionAmt
			logger.Debugf("%s CLOSE LONG@%f %f %f", swapSymbol, swapOrderPrice,swapPosition.PositionAmt, swapTakerDepth.EmaBidAskRatio)
		} else if swapPosition.PositionAmt < 0 &&
			swapTakerDepth.EmaAskBidRatio < 1.0 {
			swapOrderPrice = math.Floor(swapTakerDepth.MidPrice/swapTickSize) * swapTickSize
			swapSizeDiff = -swapPosition.PositionAmt
			logger.Debugf("%s CLOSE SHORT@%f %f %f", swapSymbol, swapOrderPrice,swapPosition.PositionAmt, swapTakerDepth.EmaAskBidRatio)
		}

		if math.Abs(swapSizeDiff) < takerStepSize {
			continue
		} else if swapSizeDiff < 0 && swapPosition.PositionAmt <= 0 && -swapSizeDiff*swapOrderPrice < takerMinNotional {
			continue
		} else if swapSizeDiff > 0 && swapPosition.PositionAmt >= 0 && swapSizeDiff*swapOrderPrice < takerMinNotional {
			continue
		}
		reduceOnly := false
		if swapSizeDiff*swapPosition.PositionAmt < 0 && math.Abs(swapSizeDiff) <= math.Abs(swapPosition.PositionAmt) {
			reduceOnly = true
		}
		side := "BUY"

		if swapSizeDiff > 0 && swapOrderPrice > swapTakerDepth.BidPrice {
			swapOrderPrice = swapTakerDepth.BidPrice
		}
		if swapSizeDiff < 0 {
			side = "SELL"
			swapSizeDiff = -swapSizeDiff
			if swapOrderPrice < swapTakerDepth.AskPrice{
				swapOrderPrice = swapTakerDepth.AskPrice
			}
		}
		takerOrder := bnswap.NewOrderParams{
			Symbol:           swapSymbol,
			Side:             side,
			Type:             common.OrderTypeLimit,
			Price:            swapOrderPrice,
			TimeInForce:      common.OrderTimeInForceGTX,
			Quantity:         swapSizeDiff,
			ReduceOnly:       reduceOnly,
			NewClientOrderId: fmt.Sprintf("%d", time.Now().Unix()*10000+int64(rand.Intn(10000))),
		}
		swapOpenOrders[swapSymbol] = TakerOpenOrder{NewOrderParams: &takerOrder, Symbol: swapSymbol}
		tOrderSilentTimes[swapSymbol] = time.Now().Add(*mtConfig.OrderSilent)
		swapOrderRequestChs[swapSymbol] <- TakerOrderRequest{New: &takerOrder}
	}
}
