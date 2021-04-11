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

	if bnRankSymbolMap == nil {
		return
	}

	entryValue := *bnswapUSDTAsset.AvailableBalance * *bnConfig.EnterFreePct
	if entryValue < *bnConfig.EnterMinimalStep {
		entryValue = *bnConfig.EnterMinimalStep
	}

	half := len(bnRankSymbolMap) / 2
	for i, symbol := range bnRankSymbolMap {
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
		size := 0.0
		price := 0.0
		if i <= half {
			price = markPrice.MarkPrice * (1 - *bnConfig.EnterSlippage)
			price = math.Floor(price/swapTickSize) * swapTickSize
			size = math.Round(entryValue/price/swapStepSize)*swapStepSize - position.PositionAmt
		} else {
			price = markPrice.MarkPrice * (1 + *bnConfig.EnterSlippage)
			price = math.Ceil(price/swapTickSize) * swapTickSize
			size = -math.Round(entryValue/price/swapStepSize)*swapStepSize - position.PositionAmt
		}
		if math.Abs(price*size) < swapMinNotional {
			continue
		}
		id, _ := common.GenerateShortId()
		clOrdID := fmt.Sprintf(
			"%sLONG%d",
			id, int(10000*markPrice.FundingRate),
		)
		if i <= half {
			clOrdID = fmt.Sprintf(
				"%sSHORT%d",
				id, int(10000*markPrice.FundingRate),
			)
		}
		side := common.OrderSideBuy
		if size < 0 {
			size = -size
			side = common.OrderSideSell
		}
		clOrdID = strings.ReplaceAll(clOrdID, ".", "_")
		order := bnswap.NewOrderParams{
			Symbol:           symbol,
			Price:            price,
			ReduceOnly:       false,
			Side:             side,
			Quantity:         size,
			TimeInForce:      common.OrderTimeInForceFOK,
			Type:             common.OrderTypeLimit,
			NewClientOrderId: clOrdID,
		}
		bnswapOrderSilentTimes[symbol] = time.Now().Add(*bnConfig.OrderSilent)
		bnswapPositionsUpdateTimes[symbol] = time.Unix(0, 0)
		bnswapOrderNewChs[symbol] <- order
	}
}
