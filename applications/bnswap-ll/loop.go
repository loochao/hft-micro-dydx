package main

import (
	"fmt"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/common"
	"math"
	"strings"
	"time"
)

func updateSwapPosition() {

	if bnswapUSDTAsset == nil || bnswapUSDTAsset.AvailableBalance == nil {
		return
	}

	if bnSignal == nil || bnSignal.Direction == 0 {
		return
	}

	entryValue := *bnswapUSDTAsset.AvailableBalance * *bnConfig.EnterFreePct
	if entryValue < *bnConfig.EnterMinimalStep {
		entryValue = *bnConfig.EnterMinimalStep
	}

	if bnSignal.Direction < 0 {
		entryValue = -entryValue
	}

	for _, symbol := range bnSymbols {
		if symbol == bnBNBSymbol {
			continue
		}
		if time.Now().Sub(bnswapPositionsUpdateTimes[symbol]) > *bnConfig.PositionMaxAge {
			continue
		}
		if time.Now().Sub(bnswapOrderSilentTimes[symbol]) < 0 {
			continue
		}
		markPrice, okMarkPrice := bnswapMarkPrices[symbol]
		position, okPosition := bnswapPositions[symbol]
		if !okMarkPrice || !okPosition {
			continue
		}
		swapStepSize := bnswapStepSizes[symbol]
		swapTickSize := bnswapTickSizes[symbol]
		swapMinNotional := bnswapMinNotional[symbol]

		if entryValue > 0 {
			price := markPrice.MarkPrice * (1 + *bnConfig.EnterSlippage)
			price = math.Ceil(price/swapTickSize) * swapTickSize
			size := math.Round(entryValue/swapStepSize) * swapStepSize
			id, _ := common.GenerateShortId()
			clOrdID := fmt.Sprintf(
				"%sOPEN",
				id,
			)
			clOrdID = strings.ReplaceAll(clOrdID, ".", "_")
			order := bnswap.NewOrderParams{
				Symbol:           symbol,
				Price:            price,
				ReduceOnly:       false,
				Side:             common.OrderSideBuy,
				Quantity:         size,
				TimeInForce:      common.OrderTimeInForceFOK,
				Type:             common.OrderTypeLimit,
				NewClientOrderId: clOrdID,
			}
			if position.PositionAmt < 0 {
				order.Quantity -= position.PositionAmt
			}
			if price*size < swapMinNotional {
				if position.PositionAmt < 0 {
					order.ReduceOnly = true
					order.Quantity = -position.PositionAmt
				}else{
					continue
				}
			}
			if position.PositionAmt <= 0 {
				bnswapOrderSilentTimes[symbol] = time.Now().Add(*bnConfig.OrderSilent)
				bnswapPositionsUpdateTimes[symbol] = time.Unix(0, 0)
				bnswapOrderNewChs[symbol] <- order
			}
		} else {
			price := markPrice.MarkPrice * (1.0 - *bnConfig.EnterSlippage)
			price = math.Floor(price/swapTickSize) * swapTickSize
			size := math.Round(-entryValue/swapStepSize) * swapStepSize
			id, _ := common.GenerateShortId()
			clOrdID := fmt.Sprintf(
				"%sOPEN",
				id,
			)
			clOrdID = strings.ReplaceAll(clOrdID, ".", "_")
			order := bnswap.NewOrderParams{
				Symbol:           symbol,
				ReduceOnly:       false,
				Price:            price,
				Side:             common.OrderSideSell,
				Quantity:         size,
				TimeInForce:      common.OrderTimeInForceFOK,
				Type:             common.OrderTypeLimit,
				NewClientOrderId: clOrdID,
			}
			if position.PositionAmt > 0 {
				order.Quantity += position.PositionAmt
			}
			if price*size < swapMinNotional {
				if position.PositionAmt > 0 {
					order.ReduceOnly = true
					order.Quantity = position.PositionAmt
				}else{
					continue
				}
			}
			if position.PositionAmt >= 0 {
				bnswapOrderSilentTimes[symbol] = time.Now().Add(*bnConfig.OrderSilent)
				bnswapPositionsUpdateTimes[symbol] = time.Unix(0, 0)
				bnswapOrderNewChs[symbol] <- order
			}
		}
	}
}
