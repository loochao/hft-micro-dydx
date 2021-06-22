package main

import (
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
)

func handleMirPosition(mir common.MIR) {
	swapMirs[mir.Symbol] = mir
	position, ok := swapPositions[mir.Symbol]
	if !ok {
		return
	}
	if _, ok := swapMirPositions[mir.Symbol]; !ok {
		swapMirPositions[mir.Symbol] = position.PositionAmt * position.EntryPrice
	}
	if _, ok := swapLastMirPrices[mir.Symbol]; !ok {
		swapLastMirPrices[mir.Symbol] = position.EntryPrice
	}
	if mir.Value > 0 {
		if swapMirPositions[mir.Symbol] >= 0 {
			swapMirPositions[mir.Symbol] = -*swapConfig.EnterStep
			swapLastMirPrices[mir.Symbol] = mir.LastPrice
		} else if mir.LastPrice < swapLastMirPrices[mir.Symbol] && swapMirPositions[mir.Symbol] > -*swapConfig.EnterTarget {
			swapMirPositions[mir.Symbol] -= *swapConfig.EnterStep
			swapLastMirPrices[mir.Symbol] = mir.LastPrice
		}
	} else {
		if swapMirPositions[mir.Symbol] <= 0 {
			swapMirPositions[mir.Symbol] = *swapConfig.EnterStep
			swapLastMirPrices[mir.Symbol] = mir.LastPrice
		} else if mir.LastPrice > swapLastMirPrices[mir.Symbol] && swapMirPositions[mir.Symbol] < *swapConfig.EnterTarget {
			swapMirPositions[mir.Symbol] += *swapConfig.EnterStep
			swapLastMirPrices[mir.Symbol] = mir.LastPrice
		}
	}
	logger.Debugf("MIR UPDATE %s %f %f", mir.Symbol, mir.Value, swapMirPositions[mir.Symbol])
}
