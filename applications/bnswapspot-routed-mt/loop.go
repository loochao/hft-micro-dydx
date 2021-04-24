package main

import (
	"fmt"
	"github.com/geometrybase/hft-micro/bnspot"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"math/rand"
	"strings"
	"time"
)

func updateSwapPositions() {
	unHedgedValue := 0.0
	for _, symbol := range bnSymbols {
		if symbol == bnBNBSymbol {
			hedgeBnb()
			continue
		}

		if time.Now().Sub(bnspotBalancesUpdateTimes[symbol]) > *bnConfig.BalancePositionMaxAge {
			continue
		}

		if time.Now().Sub(bnswapPositionsUpdateTimes[symbol]) > *bnConfig.BalancePositionMaxAge {
			continue
		}

		if bnswapOrderSilentTimes[symbol].Sub(time.Now()).Seconds() > 0 {
			continue
		}

		swapPosition, okSwapPosition := bnswapPositions[symbol]
		spotBalance, okSpotBalance := bnspotBalances[symbol]
		spread, okSpread := bnSpreads[symbol]
		if !okSwapPosition || !okSpotBalance || !okSpread {
			continue
		}
		swapOrderBook := spread.TakerDepth

		swapStepSize := bnswapStepSizes[symbol]
		swapTickSize := bnswapTickSizes[symbol]
		swapMinNotional := bnswapMinNotional[symbol]

		swapSize := -(spotBalance.Locked + spotBalance.Free) - swapPosition.PositionAmt
		swapSize = math.Round(swapSize/swapStepSize) * swapStepSize

		//只做空SWAP，所以开空是加仓，开多是减仓，减仓大小受当前空仓大小限制, 加仓受MinNotional限制
		if swapSize <= 0 && -swapSize*swapOrderBook.TakerBid*(1.0-*bnConfig.EnterSlippage) < swapMinNotional {
			continue
		}
		if swapSize > 0 && swapPosition.PositionAmt >= 0 {
			logger.Debugf("%s SWAP POSITION ERROR, CAN'T ADD %f TO POS %f", swapSize, swapPosition.PositionAmt)
			continue
		}
		if swapSize > 0 && swapSize > -swapPosition.PositionAmt {
			swapSize = -swapPosition.PositionAmt
		}

		unHedgedValue += math.Abs(swapSize * (spread.TakerDepth.MakerAsk + spread.TakerDepth.TakerBid) * 0.5)

		logger.Debugf("updateSwapPositions %s SIZE %f POS %f -> %f", symbol, swapSize, swapPosition.PositionAmt, -(spotBalance.Locked + spotBalance.Free))

		reduceOnly := false
		if swapSize*swapPosition.PositionAmt < 0 && math.Abs(swapSize) <= math.Abs(swapPosition.PositionAmt) {
			reduceOnly = true
		}
		price := math.Round(swapOrderBook.TakerAsk*(1.0+*bnConfig.EnterSlippage)/swapTickSize) * swapTickSize
		side := "BUY"
		if swapSize < 0 {
			side = "SELL"
			swapSize = -swapSize
			price = math.Round(swapOrderBook.TakerBid*(1.0-*bnConfig.EnterSlippage)/swapTickSize) * swapTickSize
		}
		bnswapOrderSilentTimes[symbol] = time.Now().Add(*bnConfig.OrderSilent)
		bnswapPositionsUpdateTimes[symbol] = time.Unix(0, 0)
		bnswapHttpPositionUpdateSilentTimes[symbol] = time.Now().Add(*bnConfig.HttpSilent)
		bnswapOrderRequestChs[symbol] <- bnswap.NewOrderParams{
			Symbol:           symbol,
			Side:             side,
			Type:             "LIMIT",
			Price:            price,
			TimeInForce:      "FOK",
			Quantity:         swapSize,
			ReduceOnly:       reduceOnly,
			NewClientOrderId: fmt.Sprintf("%d", time.Now().Unix()*10000+int64(rand.Intn(10000))),
		}
	}
	bnUnHedgeValue = unHedgedValue
}

func updateMakerNewOrders() {

	if bnspotUSDTBalance == nil {
		return
	}

	if bnswapUSDTAsset == nil || bnswapUSDTAsset.AvailableBalance == nil {
		return
	}

	if len(bnRankSymbolMap) == 0 {
		return
	}

	if bnUnHedgeValue > *bnConfig.MaxUnHedgeValue {
		if time.Now().Sub(bnUnHedgeLogSilentTime) > 0 {
			bnUnHedgeLogSilentTime = time.Now().Add(*bnConfig.LogInterval)
			logger.Debugf("UN HEDGE VALUE %f > %f", bnUnHedgeValue, *bnConfig.MaxUnHedgeValue)
		}
		return
	}

	entryStep := (*bnswapUSDTAsset.AvailableBalance + bnspotUSDTBalance.Free) * *bnConfig.EnterFreePct
	if entryStep < *bnConfig.EnterMinimalStep {
		entryStep = *bnConfig.EnterMinimalStep
	}
	entryTarget := entryStep * *bnConfig.EnterTargetFactor

	usdtAvailable := bnspotUSDTBalance.Free


	//遍历合约 从最大的rank 开始，能保证FR强的先下单
	for rank := len(bnSymbols) - 1; rank >= 0; rank-- {
		symbol := bnRankSymbolMap[rank]
		if symbol == bnBNBSymbol {
			continue
		}
		//需要保证期货和现货都有仓位更新，才调整现货仓位
		if time.Now().Sub(bnspotBalancesUpdateTimes[symbol]) > *bnConfig.BalancePositionMaxAge {
			continue
		}
		if time.Now().Sub(bnswapPositionsUpdateTimes[symbol]) > *bnConfig.BalancePositionMaxAge {
			continue
		}
		if _, ok := bnspotOpenOrders[symbol]; ok {
			//如果还有订单不操作
			continue
		}
		if time.Now().Sub(bnspotOrderSilentTimes[symbol]) < 0 {
			continue
		}
		if time.Now().Sub(bnspotSilentTimes[symbol]) < 0 {
			continue
		}
		quantile, okQuantile := bnQuantiles[symbol]
		spread, okSpread := bnSpreads[symbol]
		spotBalance, okSpotBalance := bnspotBalances[symbol]
		premiumIndex, okPremiumIndex := bnswapPremiumIndexes[symbol]
		if !okSpread || !okQuantile || !okSpotBalance || !okPremiumIndex {
			continue
		}
		if time.Now().Sub(spread.Time) > *bnConfig.SpreadTimeToLive {
			continue
		}
		swapMinNotional := bnswapMinNotional[symbol]
		spotStepSize := bnspotStepSizes[symbol]
		spotTickSize := bnspotTickSizes[symbol]
		spotMinNotional := bnspotMinNotional[symbol]
		mergedStepSize := bnMergedStepSizes[symbol]

		currentSpotSize := spotBalance.Locked + spotBalance.Free

		if spread.ShortLastLeave < quantile.Bot &&
			spread.ShortMedianLeave < quantile.Bot &&
			premiumIndex.FundingRate < *bnConfig.MinimalKeepFundingRate {
			price := spread.MakerDepth.MakerAsk
			price = math.Ceil(price/spotTickSize) * spotTickSize
			if spotBalance.Free*price > spotMinNotional {
				entryValue := math.Min(-4*entryStep, -spotBalance.Free*price*0.5)
				if premiumIndex.FundingRate > *bnConfig.MinimalKeepFundingRate/2 {
					entryValue = math.Min(-2*entryStep, -spotBalance.Free*price*0.5)
				}
				quantity := entryValue / price
				quantity = math.Round(quantity/mergedStepSize) * mergedStepSize
				if spotBalance.Free*price-entryValue < entryStep {
					quantity = -math.Floor(spotBalance.Free/spotStepSize) * spotStepSize
				}
				if quantity < 0 {
					logger.Debugf(
						"BOT REDUCE %s %f < %f, %f < %f, SIZE %f",
						symbol,
						spread.ShortLastLeave, quantile.Bot,
						spread.ShortMedianLeave, quantile.Bot,
						quantity,
					)
					order := bnspot.NewOrderParams{
						Symbol:           symbol,
						Price:            price,
						Quantity:         -quantity,
						TimeInForce:      bnspot.OrderTimeInForceGTC,
						Side:             bnspot.OrderSideSell,
						Type:             bnspot.OrderTypeLimitMarker,
						NewClientOrderID: fmt.Sprintf("%d", time.Now().Unix()*10000+int64(rand.Intn(10000))),
					}
					bnspotOrderSilentTimes[symbol] = time.Now().Add(*bnConfig.OrderSilent)
					bnspotOrderCancelCounts[symbol] = 0
					bnspotOpenOrders[symbol] = order
					bnspotHttpBalanceUpdateSilentTimes[symbol] = time.Now().Add(*bnConfig.HttpSilent)
					bnspotOrderRequestChs[symbol] <- SpotOrderRequest{New: &order}
				}
			}
		} else if spread.ShortLastEnter > quantile.Top &&
			spread.ShortMedianEnter > quantile.Top &&
			premiumIndex.FundingRate > *bnConfig.MinimalEnterFundingRate &&
			rank >= len(bnSymbols)-*bnConfig.TradeCount {
			price := spread.MakerDepth.MakerBid
			price = math.Floor(price/spotTickSize) * spotTickSize
			targetValue := currentSpotSize*price + entryStep
			if targetValue > entryTarget {
				targetValue = entryTarget
			}
			entryValue := targetValue - currentSpotSize*price

			if entryValue > usdtAvailable*0.8 {
				entryValue = usdtAvailable* 0.8
			}

			entryValue = math.Max(entryValue, swapMinNotional)
			entryValue = math.Max(entryValue, spotMinNotional)

			quantity := entryValue / price
			quantity = math.Round(quantity/mergedStepSize) * mergedStepSize

			entryValue = quantity * price

			//不及一个0.8*EntryStep, 不操作
			if entryValue < entryStep*0.8 {
				if time.Now().Sub(bnOpenLogSilentTimes[symbol]) > 0 {
					logger.Debugf(
						"FAILED TOP OPEN, ENTRY VALUE %f LESS THAN 0.8*ENTRY_STEP %f, %s %f > %f, %f > %f, SIZE %f",
						entryValue,
						entryStep*0.8,
						symbol,
						spread.ShortLastEnter, quantile.Top,
						spread.ShortMedianEnter, quantile.Top,
						quantity,
					)
					bnOpenLogSilentTimes[symbol] = time.Now().Add(*bnConfig.LogInterval)
				}
				continue
			}
			if entryValue > usdtAvailable {
				if time.Now().Sub(bnOpenLogSilentTimes[symbol]) > 0 {
					logger.Debugf(
						"FAILED TOP OPEN, ENTRY VALUE %f MORE THAN FREE USDT %f, %s %f > %f, %f > %f, SIZE %f",
						entryValue,
						usdtAvailable,
						symbol,
						spread.ShortLastEnter, quantile.Top,
						spread.ShortMedianEnter, quantile.Top,
						quantity,
					)
					bnOpenLogSilentTimes[symbol] = time.Now().Add(*bnConfig.LogInterval)
				}
				continue
			}
			bnOpenLogSilentTimes[symbol] = time.Now()
			logger.Debugf(
				"TOP OPEN %s %f > %f, %f > %f, SIZE %f",
				symbol,
				spread.ShortLastEnter, quantile.Top,
				spread.ShortMedianEnter, quantile.Top,
				quantity,
			)
			usdtAvailable -= entryValue
			order := bnspot.NewOrderParams{
				Symbol:           symbol,
				Price:            price,
				Quantity:         quantity,
				TimeInForce:      bnspot.OrderTimeInForceGTC,
				Side:             bnspot.OrderSideBuy,
				Type:             bnspot.OrderTypeLimitMarker,
				NewClientOrderID: fmt.Sprintf("%d", time.Now().Unix()*10000+int64(rand.Intn(10000))),
			}
			bnspotOrderSilentTimes[symbol] = time.Now().Add(*bnConfig.OrderSilent)
			bnspotOrderCancelCounts[symbol] = 0
			bnspotOpenOrders[symbol] = order
			bnspotHttpBalanceUpdateSilentTimes[symbol] = time.Now().Add(*bnConfig.HttpSilent)
			bnspotOrderRequestChs[symbol] <- SpotOrderRequest{New: &order}
		}
	}
}

func hedgeBnb() {
	symbol := bnBNBSymbol
	if time.Now().Sub(bnspotBalancesUpdateTimes[symbol]) > *bnConfig.BalancePositionMaxAge {
		return
	}

	if time.Now().Sub(bnswapPositionsUpdateTimes[symbol]) > *bnConfig.BalancePositionMaxAge {
		return
	}

	if bnswapOrderSilentTimes[symbol].Sub(time.Now()).Seconds() > 0 {
		return
	}

	swapPosition, okSwapPosition := bnswapPositions[symbol]
	spotBalance, okSpotBalance := bnspotBalances[symbol]
	swapPremiumIndex, okSwapPremiumIndex := bnswapPremiumIndexes[symbol]
	if !okSwapPosition || !okSpotBalance ||
		!okSwapPremiumIndex || bnswapBNBAsset == nil ||
		bnswapBNBAsset.MarginBalance == nil {
		return
	}

	swapStepSize := bnswapStepSizes[symbol]
	swapTickSize := bnswapTickSizes[symbol]
	swapMinNotional := bnswapMinNotional[symbol]

	targetSize := -(spotBalance.Free + *bnswapBNBAsset.MarginBalance)

	swapSize := targetSize - swapPosition.PositionAmt
	swapSize = math.Round(swapSize/swapStepSize) * swapStepSize

	if math.Abs(swapSize) < swapStepSize {
		return
	}
	if swapSize < 0 && -swapSize*swapPremiumIndex.MarkPrice*(1.0-*bnConfig.EnterSlippage) < swapMinNotional {
		return
	}
	if swapSize > 0 && swapSize*swapPremiumIndex.MarkPrice*(1.0+*bnConfig.EnterSlippage) < swapMinNotional {
		return
	}
	logger.Debugf("hedgeBnb %s SIZE %f POS %f -> %f", symbol, swapSize, swapPosition.PositionAmt, targetSize)

	reduceOnly := false
	if swapSize*swapPosition.PositionAmt < 0 && math.Abs(swapSize) <= math.Abs(swapPosition.PositionAmt) {
		reduceOnly = true
	}
	price := math.Round(swapPremiumIndex.MarkPrice*(1.0+*bnConfig.EnterSlippage)/swapTickSize) * swapTickSize
	side := "BUY"
	id, _ := common.GenerateShortId()
	clOrdID := fmt.Sprintf(
		"%s-H%.6f",
		id,
		spotBalance.Free+*bnswapBNBAsset.MarginBalance,
	)
	clOrdID = strings.ReplaceAll(clOrdID, ".", "_")
	if swapSize < 0 {
		side = "SELL"
		swapSize = -swapSize
		price = math.Round(swapPremiumIndex.MarkPrice*(1.0-*bnConfig.EnterSlippage)/swapTickSize) * swapTickSize
	}
	bnswapOrderSilentTimes[symbol] = time.Now().Add(*bnConfig.OrderSilent)
	bnswapPositionsUpdateTimes[symbol] = time.Unix(0, 0)
	bnswapHttpPositionUpdateSilentTimes[symbol] = time.Now()
	bnswapOrderRequestChs[symbol] <- bnswap.NewOrderParams{
		Symbol:           symbol,
		Side:             side,
		Type:             "LIMIT",
		Price:            price,
		TimeInForce:      "FOK",
		Quantity:         swapSize,
		ReduceOnly:       reduceOnly,
		NewClientOrderId: clOrdID,
	}
}
