package main

import (
	"fmt"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"strings"
	"time"
)

func updateSwapNewOrders() {

	if bnswapUSDTAsset == nil || bnswapUSDTAsset.AvailableBalance == nil {
		return
	}

	entryValue := *bnswapUSDTAsset.AvailableBalance * *bnConfig.EnterFreePct
	if entryValue < *bnConfig.EnterMinimalStep {
		entryValue = *bnConfig.EnterMinimalStep
	}

	//logger.Debugf("updateSwapNewOrders %f", entryValue)

	for _, symbol := range bnSymbols {
		if symbol == bnBNBSymbol {
			continue
		}
		if time.Now().Sub(bnswapPositionsUpdateTimes[symbol]) > *bnConfig.PositionMaxAge {
			continue
		}
		if _, ok := bnswapOpenOrders[symbol]; ok {
			//如果还有订单不操作
			continue
		}
		if time.Now().Sub(bnswapOrderSilentTimes[symbol]) < 0 {
			continue
		}
		spread, okSpread := bnSpreads[symbol]
		markPrice, okMarkPrice := bnswapMarkPrices[symbol]
		position, okPosition := bnswapPositions[symbol]
		if !okSpread || !okMarkPrice || !okPosition {
			continue
		}
		if time.Now().Sub(spread.EventTime) > *bnConfig.SpreadTimeToLive {
			continue
		}
		swapStepSize := bnswapStepSizes[symbol]
		swapTickSize := bnswapTickSizes[symbol]
		swapMinNotional := bnswapMinNotional[symbol]

		if position.PositionAmt != 0 {
			id, _ := common.GenerateShortId()
			clOrdID := fmt.Sprintf(
				"%sCLOSE",
				id,
			)
			clOrdID = strings.ReplaceAll(clOrdID, ".", "_")
			order := bnswap.NewOrderParams{
				ReduceOnly:       true,
				Symbol:           symbol,
				Quantity:         position.PositionAmt,
				TimeInForce:      common.OrderTimeInForceGTC,
				Type:             common.OrderTypeLimit,
				NewClientOrderId: clOrdID,
			}
			if order.Quantity < 0 {
				order.Quantity = -order.Quantity
				order.Side = common.OrderSideBuy
				order.Price = math.Floor(spread.OrderBook.CloseBidVWAP/swapTickSize) * swapTickSize
			} else {
				order.Side = common.OrderSideSell
				order.Price = math.Ceil(spread.OrderBook.CloseAskVWAP/swapTickSize) * swapTickSize
			}
			bnswapOrderSilentTimes[symbol] = time.Now().Add(*bnConfig.OrderSilent)
			bnswapOrderCancelCounts[symbol] = 0
			bnswapOpenOrders[symbol] = order
			bnswapLastOrderTimes[symbol] = time.Now()
			bnswapOrderRequestChs[symbol] <- SwapOrderRequest{New: &order}
		} else if markPrice.FundingRate > *bnConfig.MinimalLongFundingRate &&
			spread.MedianLong > *bnConfig.EnterMinimalSpread &&
			spread.MedianLong < *bnConfig.EnterMaximalSpread &&
			spread.LastLong > *bnConfig.EnterMinimalSpread &&
			spread.LastLong < *bnConfig.EnterMaximalSpread {
			price := spread.OrderBook.OpenBidVWAP
			price = math.Floor(price/swapTickSize) * swapTickSize

			quantity := entryValue / price
			quantity = math.Round(quantity/swapStepSize) * swapStepSize

			//不及一个0.8*EntryValue, 不操作
			if quantity*price < entryValue*0.8 {
				if time.Now().Sub(bnOpenLogSilentTimes[symbol]) > 0 {
					logger.Debugf(
						"%s FAILED LONG OPEN, ENTRY VALUE %f < 0.8*ENTRY_VALUE %f",
						symbol,
						price*quantity,
						entryValue*0.8,
					)
					bnOpenLogSilentTimes[symbol] = time.Now().Add(time.Minute * 5)
				}
				continue
			}
			if quantity*price > 0.8**bnswapUSDTAsset.AvailableBalance {
				if time.Now().Sub(bnOpenLogSilentTimes[symbol]) > 0 {
					logger.Debugf(
						"%s FAILED LONG OPEN, ENTRY VALUE %f > 0.8*AvailableBalance %f",
						symbol,
						price*quantity,
						0.8**bnswapUSDTAsset.AvailableBalance,
					)
					bnOpenLogSilentTimes[symbol] = time.Now().Add(time.Minute * 5)
				}
				continue
			}
			if quantity*price < swapMinNotional {
				if time.Now().Sub(bnOpenLogSilentTimes[symbol]) > 0 {
					logger.Debugf(
						"%s FAILED LONG OPEN, ENTRY VALUE %f < swapMinNotional %f",
						symbol,
						price*quantity,
						swapMinNotional,
					)
					bnOpenLogSilentTimes[symbol] = time.Now().Add(time.Minute * 5)
				}
				continue
			}
			//logger.Debugf(
			//	"%s LONG OPEN %f  %f SIZE %f",
			//	symbol,
			//	spread.LastLong,
			//	spread.MedianLong,
			//	quantity,
			//)
			id, _ := common.GenerateShortId()
			clOrdID := fmt.Sprintf(
				"%sLL%dML%d",
				id,
				int(spread.LastLong*10000),
				int(spread.MedianLong*10000),
			)
			clOrdID = strings.ReplaceAll(clOrdID, ".", "_")
			order := bnswap.NewOrderParams{
				ReduceOnly: false,
				Symbol:     symbol,
				Price:      price,
				Quantity:   quantity,

				TimeInForce:      common.OrderTimeInForceGTC,
				Side:             common.OrderSideBuy,
				Type:             common.OrderTypeLimit,
				NewClientOrderId: clOrdID,
			}
			bnswapOrderSilentTimes[symbol] = time.Now().Add(*bnConfig.OrderSilent)
			bnswapOrderCancelCounts[symbol] = 0
			bnswapOpenOrders[symbol] = order
			bnswapLastOrderTimes[symbol] = time.Now()
			bnswapOrderRequestChs[symbol] <- SwapOrderRequest{New: &order}
			return
		} else if markPrice.FundingRate < *bnConfig.MaximalShortFundingRate &&
			spread.MedianShort > *bnConfig.EnterMinimalSpread &&
			spread.MedianShort < *bnConfig.EnterMaximalSpread &&
			spread.LastShort > *bnConfig.EnterMinimalSpread &&
			spread.LastShort < *bnConfig.EnterMaximalSpread {
			price := spread.OrderBook.OpenAskVWAP
			price = math.Ceil(price/swapTickSize) * swapTickSize

			quantity := entryValue / price
			quantity = math.Round(quantity/swapStepSize) * swapStepSize

			//不及一个0.8*EntryValue, 不操作
			if quantity*price < entryValue*0.8 {
				if time.Now().Sub(bnOpenLogSilentTimes[symbol]) > 0 {
					logger.Debugf(
						"%s FAILED LONG OPEN, ENTRY VALUE %f < 0.8*ENTRY_VALUE %f",
						symbol,
						price*quantity,
						entryValue*0.8,
					)
					bnOpenLogSilentTimes[symbol] = time.Now().Add(time.Minute * 5)
				}
				continue
			}
			if quantity*price > 0.8**bnswapUSDTAsset.AvailableBalance {
				if time.Now().Sub(bnOpenLogSilentTimes[symbol]) > 0 {
					logger.Debugf(
						"%s FAILED LONG OPEN, ENTRY VALUE %f > 0.8*AvailableBalance %f",
						symbol,
						price*quantity,
						0.8**bnswapUSDTAsset.AvailableBalance,
					)
					bnOpenLogSilentTimes[symbol] = time.Now().Add(time.Minute * 5)
				}
				continue
			}
			if quantity*price < swapMinNotional {
				if time.Now().Sub(bnOpenLogSilentTimes[symbol]) > 0 {
					logger.Debugf(
						"%s FAILED LONG OPEN, ENTRY VALUE %f < swapMinNotional %f",
						symbol,
						price*quantity,
						swapMinNotional,
					)
					bnOpenLogSilentTimes[symbol] = time.Now().Add(time.Minute * 5)
				}
				continue
			}
			//logger.Debugf(
			//	"%s LONG OPEN %f  %f SIZE %f",
			//	symbol,
			//	spread.LastLong,
			//	spread.MedianLong,
			//	quantity,
			//)
			id, _ := common.GenerateShortId()
			clOrdID := fmt.Sprintf(
				"%sLL%dML%d",
				id,
				int(spread.LastLong*10000),
				int(spread.MedianLong*10000),
			)
			clOrdID = strings.ReplaceAll(clOrdID, ".", "_")
			order := bnswap.NewOrderParams{
				ReduceOnly: false,
				Symbol:     symbol,
				Price:      price,
				Quantity:   quantity,

				TimeInForce:      common.OrderTimeInForceGTC,
				Side:             common.OrderSideSell,
				Type:             common.OrderTypeLimit,
				NewClientOrderId: clOrdID,
			}
			bnswapOrderSilentTimes[symbol] = time.Now().Add(*bnConfig.OrderSilent)
			bnswapOrderCancelCounts[symbol] = 0
			bnswapOpenOrders[symbol] = order
			bnswapLastOrderTimes[symbol] = time.Now()
			bnswapOrderRequestChs[symbol] <- SwapOrderRequest{New: &order}
			return
		}
	}
}
