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

func updatePositions() {
	if swapAccount == nil || swapAccount.AvailableBalance == nil {
		return
	}
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
		position, okTakerPositions := swapPositions[swapSymbol]
		mir, okMir := swapMirs[swapSymbol]
		targetPosition, okTargetPosition := swapMirPositions[swapSymbol]
		if !okTakerPositions || !okMir || !okTargetPosition {
			if time.Now().Truncate(time.Second*15).Add(*swapConfig.LoopInterval).Sub(time.Now()) > 0 {
				logger.Debugf("%s POS %v MIR %v TARGET %v", swapSymbol, okTakerPositions, okMir, okTargetPosition)
			}
			continue
		}

		diff := targetPosition - position.EntryPrice*position.PositionAmt
		size := math.Round(diff/mir.LastPrice/swapStepSizes[swapSymbol]) * swapStepSizes[swapSymbol]
		if size*position.PositionAmt >= 0 && math.Abs(size*mir.LastPrice) < swapMinNotional[swapSymbol]*1.5 {
			if time.Now().Truncate(time.Second*15).Add(*swapConfig.LoopInterval).Sub(time.Now()) > 0 {
				logger.Debugf("%s VALUE %f < MIN NOTIONAL %f", swapSymbol, size*mir.LastPrice, swapMinNotional[swapSymbol]*1.5)
			}
			continue
		}
		if size == 0 {
			continue
		}
		reduceOnly := false
		if size*position.PositionAmt >= 0 {
			swapUSDTAvailable -= math.Abs(size) * mir.LastPrice
		} else if math.Abs(size) <= math.Abs(position.PositionAmt) {
			reduceOnly = true
		} else {
			swapUSDTAvailable -= math.Abs(position.PositionAmt-size) * mir.LastPrice
		}
		side := "BUY"
		if size < 0 {
			side = "SELL"
			size = -size
		}
		takerOrder := bnswap.NewOrderParams{
			Symbol:           swapSymbol,
			Side:             side,
			Type:             common.OrderTypeMarket,
			Quantity:         size,
			ReduceOnly:       reduceOnly,
			NewClientOrderId: fmt.Sprintf("%d", time.Now().Unix()*10000+int64(rand.Intn(10000))),
		}
		swapOrderSilentTimes[swapSymbol] = time.Now().Add(*swapConfig.OrderSilent)
		swapOrderRequestChs[swapSymbol] <- takerOrder
	}
}
