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

func updateNewOrders() {
	if swapAccount == nil || swapAccount.AvailableBalance == nil {
		return
	}
	enterStep := *swapAccount.AvailableBalance * *swapConfig.EnterFreePct
	if enterStep < *swapConfig.EnterMinimalStep {
		enterStep = *swapConfig.EnterMinimalStep
	}
	enterTarget := *swapConfig.EnterTargetFactor * enterStep
	swapUSDTAvailable := *swapAccount.AvailableBalance

	for _, swapSymbol := range swapSymbols {
		if time.Now().Sub(swapPositionsUpdateTimes[swapSymbol]) > *swapConfig.PositionMaxAge {
			if time.Now().Truncate(time.Second*15).Add(*swapConfig.LoopInterval).Sub(time.Now()) > 0 {
				logger.Debugf("%s POSITION NOT UPDATE", swapSymbol)
			}
			continue
		}
		if swapOrderSilentTimes[swapSymbol].Sub(time.Now()).Seconds() > 0 {
			if time.Now().Truncate(time.Second*15).Add(*swapConfig.LoopInterval).Sub(time.Now()) > 0 {
				logger.Debugf("%s ORDER IN SILENT", swapSymbol)
			}
			continue
		}
		swapPosition, okTakerPositions := swapPositions[swapSymbol]
		swapDepth, okDepth := swapWalkedDepths[swapSymbol]
		mergedSignal, okSignal := swapMergedSignals[swapSymbol]
		if !okTakerPositions || !okDepth || !okSignal {
			if time.Now().Truncate(time.Second*15).Add(*swapConfig.LoopInterval).Sub(time.Now()) > 0 {
				logger.Debugf("%s POS %v DEPTH %v SIGNAL %v", swapSymbol, okTakerPositions, okDepth, okSignal)
			}
			continue
		}

		lastEnterPrice, okLastEnterPrice := swapLastEnterPrices[swapSymbol]

		//logger.Debugf("%v", mergedSignal)
		swapStepSize := swapStepSizes[swapSymbol]
		swapTickSize := swapTickSizes[swapSymbol]
		takerMinNotional := swapMinNotional[swapSymbol]

		swapSizeDiff := 0.0
		targetValue := 0.0
		swapOrderPrice := 0.0
		enterValue := 0.0
		openValue := 0.0

		//还在加多档期
		if mergedSignal.Value < -*swapConfig.EnterThreshold &&
			time.Now().Sub(swapEnterSilentTimes[swapSymbol]) > 0 {
			swapOrderPrice = math.Floor(swapDepth.TakerAsk/swapTickSize) * swapTickSize
			if swapPosition.PositionAmt > 0 &&
				okLastEnterPrice &&
				lastEnterPrice+swapDepth.TakerFarAsk-swapDepth.TakerFarBid > swapOrderPrice {
				if time.Now().Truncate(time.Second*15).Add(*swapConfig.LoopInterval).Sub(time.Now()) > 0 {
					logger.Debugf("%s LONG FAILED, TAKER ASK %f > LAST ENTER PRICE %f", swapSymbol, swapDepth.TakerAsk, lastEnterPrice)
				}
				//已有多仓，且上次加仓成本比现在高，不加仓
				continue
			}
			if swapPosition.PositionAmt >= 0 {
				targetValue = swapPosition.PositionAmt*swapPosition.EntryPrice + enterStep
				if targetValue > enterTarget {
					targetValue = enterTarget
				}
				swapSizeDiff = math.Floor((targetValue-swapPosition.PositionAmt*swapPosition.EntryPrice)/swapOrderPrice/swapStepSize) * swapStepSize
				openValue = swapSizeDiff * swapOrderPrice
			} else {
				if -swapPosition.PositionAmt*swapPosition.EntryPrice > enterTarget/2 {
					//超过一半目标仓位，减半仓
					swapSizeDiff = math.Floor(-swapPosition.PositionAmt/2/swapStepSize) * swapStepSize
				} else {
					//直接换仓
					swapSizeDiff = math.Floor((enterStep/swapOrderPrice-swapPosition.PositionAmt)/swapStepSize) * swapStepSize
					openValue = enterStep
				}
			}
			enterValue = swapSizeDiff * swapOrderPrice
			if enterValue < 0.8*enterStep {
				if time.Now().Sub(swapLogSilentTimes[swapSymbol]) > 0 {
					logger.Debugf(
						"%s FAILED LONG OPEN, ENTRY VALUE %f LESS THAN 0.8*ENTRY_STEP %f, SIZE %f",
						swapSymbol,
						enterValue,
						enterStep*0.8,
						swapSizeDiff,
					)
					swapLogSilentTimes[swapSymbol] = time.Now().Add(*swapConfig.LogInterval)
				}
				continue
			}
			if enterValue > swapUSDTAvailable*float64(*swapConfig.Leverage) {
				if time.Now().Sub(swapLogSilentTimes[swapSymbol]) > 0 {
					logger.Debugf(
						"%s FAILED LONG OPEN, ENTRY VALUE %f MORE THAN swapUSDTAvailable %f, SIZE %f",
						swapSymbol,
						enterValue,
						swapUSDTAvailable,
						swapSizeDiff,
					)
					swapLogSilentTimes[swapSymbol] = time.Now().Add(*swapConfig.LogInterval)
				}
				continue
			}
			if enterValue < takerMinNotional {
				if time.Now().Sub(swapLogSilentTimes[swapSymbol]) > 0 {
					logger.Debugf(
						"%s FAILED LONG OPEN, ORDER VALUE %f LESS THAN NOTIONAL %f, SIZE %f",
						swapSymbol,
						enterValue,
						takerMinNotional,
						swapSizeDiff,
					)
					swapLogSilentTimes[swapSymbol] = time.Now().Add(*swapConfig.LogInterval)
				}
				continue
			}
			logger.Debugf("%s OPEN LONG@%f %f -> %f", swapSymbol, swapOrderPrice, swapPosition.PositionAmt, swapPosition.PositionAmt+swapSizeDiff)
		} else if mergedSignal.Value > *swapConfig.EnterThreshold &&
			time.Now().Sub(swapEnterSilentTimes[swapSymbol]) > 0 {

			swapOrderPrice = math.Ceil(swapDepth.TakerBid/swapTickSize) * swapTickSize
			if swapPosition.PositionAmt < 0 &&
				okLastEnterPrice &&
				lastEnterPrice-(swapDepth.TakerFarAsk-swapDepth.TakerFarBid) < swapOrderPrice {
				//已有多仓，且上次加仓成本比现在高，不加仓
				if time.Now().Truncate(time.Second*15).Add(*swapConfig.LoopInterval).Sub(time.Now()) > 0 {
					logger.Debugf("%s SHORT FAILED, TAKER BID %f > LAST ENTER PRICE %f", swapSymbol, swapDepth.TakerAsk, lastEnterPrice)
				}
				continue
			}
			if swapPosition.PositionAmt <= 0 {
				targetValue = swapPosition.PositionAmt*swapPosition.EntryPrice - enterStep
				if targetValue < -enterTarget {
					targetValue = -enterTarget
				}
				swapSizeDiff = math.Floor((targetValue-swapPosition.PositionAmt*swapPosition.EntryPrice)/swapOrderPrice/swapStepSize) * swapStepSize
				openValue = swapSizeDiff * swapOrderPrice
			} else {
				if swapPosition.PositionAmt*swapPosition.EntryPrice > enterTarget/2 {
					//减半仓
					swapSizeDiff = math.Floor(-swapPosition.PositionAmt/2/swapStepSize) * swapStepSize
				} else {
					//直接换仓
					swapSizeDiff = math.Floor((-enterStep/swapOrderPrice-swapPosition.PositionAmt)/swapStepSize) * swapStepSize
					openValue = -enterStep
				}
			}
			enterValue = swapSizeDiff * swapOrderPrice
			if -enterValue < 0.8*enterStep {
				if time.Now().Sub(swapLogSilentTimes[swapSymbol]) > 0 {
					logger.Debugf(
						"%s FAILED SHORT OPEN, ENTRY VALUE %f LESS THAN 0.8*ENTRY_STEP %f, SIZE %f",
						swapSymbol,
						-enterValue,
						enterStep*0.8,
						swapSizeDiff,
					)
					swapLogSilentTimes[swapSymbol] = time.Now().Add(*swapConfig.LogInterval)
				}
				continue
			}
			if -enterValue > swapUSDTAvailable*float64(*swapConfig.Leverage) {
				if time.Now().Sub(swapLogSilentTimes[swapSymbol]) > 0 {
					logger.Debugf(
						"%s FAILED SHORT OPEN, ENTRY VALUE %f MORE THAN swapUSDTAvailable %f, SIZE %f",
						swapSymbol,
						-enterValue,
						swapUSDTAvailable,
						swapSizeDiff,
					)
					swapLogSilentTimes[swapSymbol] = time.Now().Add(*swapConfig.LogInterval)
				}
				continue
			}
			if -enterValue < takerMinNotional {
				if time.Now().Sub(swapLogSilentTimes[swapSymbol]) > 0 {
					logger.Debugf(
						"%s FAILED SHORT TOP OPEN, ORDER VALUE %f LESS THAN NOTIONAL %f, SIZE %f",
						swapSymbol,
						-enterValue,
						takerMinNotional,
						swapSizeDiff,
					)
					swapLogSilentTimes[swapSymbol] = time.Now().Add(*swapConfig.LogInterval)
				}
				continue
			}
			logger.Debugf("%s OPEN SHORT@%f %f -> %f", swapSymbol, swapOrderPrice, swapPosition.PositionAmt, swapPosition.PositionAmt+swapSizeDiff)
		}

		if math.Abs(swapSizeDiff) < swapStepSize {
			continue
		} else if swapSizeDiff < 0 && swapPosition.PositionAmt <= 0 && -swapSizeDiff*swapOrderPrice < takerMinNotional {
			continue
		} else if swapSizeDiff > 0 && swapPosition.PositionAmt >= 0 && swapSizeDiff*swapOrderPrice < takerMinNotional {
			continue
		}
		swapUSDTAvailable -= math.Abs(openValue) / float64(*swapConfig.Leverage)
		reduceOnly := false
		if swapSizeDiff*swapPosition.PositionAmt < 0 && math.Abs(swapSizeDiff) <= math.Abs(swapPosition.PositionAmt) {
			reduceOnly = true
		}
		side := "BUY"

		if swapSizeDiff > 0 && swapOrderPrice > swapDepth.BestBidPrice {
			swapOrderPrice = swapDepth.BestBidPrice
		}
		if swapSizeDiff < 0 {
			side = "SELL"
			swapSizeDiff = -swapSizeDiff
			if swapOrderPrice < swapDepth.BestAskPrice {
				swapOrderPrice = swapDepth.BestAskPrice
			}
		}
		takerOrder := bnswap.NewOrderParams{
			Symbol:           swapSymbol,
			Side:             side,
			Type:             common.OrderTypeMarket,
			Quantity:         swapSizeDiff,
			ReduceOnly:       reduceOnly,
			NewClientOrderId: fmt.Sprintf("%d", time.Now().Unix()*10000+int64(rand.Intn(10000))),
		}
		swapOrderSilentTimes[swapSymbol] = time.Now().Add(*swapConfig.OrderSilent)
		swapOrderRequestChs[swapSymbol] <- takerOrder
	}
}
