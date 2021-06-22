package main

import (
	"fmt"
	"github.com/geometrybase/hft-micro/bnspot"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"strings"
	"time"
)

func updateSwapPositions() {
	unHedgeValue := 0.0
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
		swapOrderBook := spread.SwapOrderBook

		swapStepSize := bnswapStepSizes[symbol]
		swapTickSize := bnswapTickSizes[symbol]
		swapMinNotional := bnswapMinNotional[symbol]

		swapSize := -(spotBalance.Locked + spotBalance.Free) - swapPosition.PositionAmt
		unHedgeValue += math.Abs(swapSize*(swapOrderBook.AskPrice+swapOrderBook.BidPrice)*0.5)
		swapSize = math.Round(swapSize/swapStepSize) * swapStepSize

		//只做空SWAP，所以开空是加仓，开多是减仓，减仓大小受当前空仓大小限制, 加仓受MinNotional限制
		if swapSize <= 0 && -swapSize*swapOrderBook.BidPrice*(1.0-*bnConfig.EnterSlippage) < swapMinNotional {
			continue
		}
		if swapSize > 0 && swapPosition.PositionAmt >= 0 {
			logger.Debugf("%s SWAP POSITION ERROR, CAN'T ADD %f TO POS %f", swapSize, swapPosition.PositionAmt)
			continue
		}
		if swapSize > 0 && swapSize > -swapPosition.PositionAmt {
			swapSize = -swapPosition.PositionAmt
		}

		logger.Debugf("updateSwapPositions %s SIZE %f POS %f -> %f", symbol, swapSize, swapPosition.PositionAmt, -(spotBalance.Locked + spotBalance.Free))

		reduceOnly := false
		if swapSize*swapPosition.PositionAmt < 0 && math.Abs(swapSize) <= math.Abs(swapPosition.PositionAmt) {
			reduceOnly = true
		}
		price := math.Round(swapOrderBook.AskPrice*(1.0+*bnConfig.EnterSlippage)/swapTickSize) * swapTickSize
		side := "BUY"
		id, _ := common.GenerateShortId()
		clOrdID := fmt.Sprintf(
			"%s-H%.6f",
			id,
			spotBalance.Locked+spotBalance.Free,
		)
		clOrdID = strings.ReplaceAll(clOrdID, ".", "_")
		if swapSize < 0 {
			side = "SELL"
			swapSize = -swapSize
			price = math.Round(swapOrderBook.BidPrice*(1.0-*bnConfig.EnterSlippage)/swapTickSize) * swapTickSize
		}
		order := bnswap.NewOrderParams{
			Symbol:           symbol,
			Side:             side,
			Type:             "LIMIT",
			Price:            price,
			TimeInForce:      "FOK",
			Quantity:         swapSize,
			ReduceOnly:       reduceOnly,
			NewClientOrderId: clOrdID,
		}
		bnswapOrderSilentTimes[symbol] = time.Now().Add(*bnConfig.OrderSilent)
		bnswapPositionsUpdateTimes[symbol] = time.Unix(0, 0)
		bnswapLastOrderTimes[symbol] = time.Now()
		if !*bnConfig.DryRun {
			go swapCreateOrder(bnGlobalCtx, bnswapAPI, *bnConfig.OrderTimeout, order)
		}
	}
	bnUnHedgeValue = unHedgeValue
}

func updateSpotPositions() {

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
		return
	}

	entryStep := (*bnswapUSDTAsset.AvailableBalance + bnspotUSDTBalance.Free) * *bnConfig.EnterFreePct
	if entryStep < *bnConfig.EnterMinimalStep {
		entryStep = *bnConfig.EnterMinimalStep
	}
	entryTarget := entryStep * *bnConfig.EnterTargetFactor

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
		if time.Now().Sub(bnspotOrderSilentTimes[symbol]) < 0 {
			continue
		}
		if time.Now().Sub(bnspotSilentTimes[symbol]) < 0 {
			continue
		}
		quantile, okQuantile := bnQuantiles[symbol]
		spread, okSpread := bnSpreads[symbol]
		spotBalance, okSpotBalance := bnspotBalances[symbol]
		markPrice, okMarkPrice := bnswapMarkPrices[symbol]
		if !okSpread || !okQuantile || !okSpotBalance || !okMarkPrice {
			continue
		}
		if time.Now().Sub(spread.LastUpdateTime) > *bnConfig.SpreadTimeToLive {
			continue
		}
		swapStepSize := bnswapStepSizes[symbol]
		swapMinNotional := bnswapMinNotional[symbol]
		spotStepSize := bnspotStepSizes[symbol]
		spotTickSize := bnspotTickSizes[symbol]
		spotMinNotional := bnspotMinNotional[symbol]

		currentSpotSize := spotBalance.Locked + spotBalance.Free
		if spread.LastEnter > quantile.Top &&
			spread.MedianEnter > quantile.Top &&
			markPrice.FundingRate > *bnConfig.MinimalEnterFundingRate &&
			rank >= len(bnSymbols)-*bnConfig.TradeCount {
			price := spread.SpotOrderBook.TakerAskVWAP * (1.0 + *bnConfig.EnterSlippage)
			price = math.Floor(price/spotTickSize) * spotTickSize
			targetValue := currentSpotSize*price + entryStep
			if targetValue > entryTarget {
				targetValue = entryTarget
			}
			entryValue := targetValue - currentSpotSize*price

			if entryValue > bnspotUSDTBalance.Free*0.8 {
				entryValue = bnspotUSDTBalance.Free * 0.8
			}

			entryValue = math.Max(entryValue, swapMinNotional)
			entryValue = math.Max(entryValue, spotMinNotional)

			quantity := entryValue / price
			quantity = math.Round(quantity/spotStepSize) * spotStepSize
			quantity = math.Round(quantity/swapStepSize) * swapStepSize

			//不及一个0.8*EntryStep, 不操作
			if entryValue < entryStep*0.8 {
				if time.Now().Sub(bnOpenLogSilentTimes[symbol]) > 0 {
					logger.Debugf(
						"FAILED TOP OPEN, ENTRY VALUE %f LESS THAN 0.8*ENTRY_STEP %f, %s %f > %f, %f > %f, SIZE %f",
						entryValue,
						entryStep*0.8,
						symbol,
						spread.LastEnter, quantile.Top,
						spread.MedianEnter, quantile.Top,
						quantity,
					)
					bnOpenLogSilentTimes[symbol] = time.Now().Add(time.Minute * 5)
				}
				continue
			}
			if entryValue > bnspotUSDTBalance.Free {
				if time.Now().Sub(bnOpenLogSilentTimes[symbol]) > 0 {
					logger.Debugf(
						"FAILED TOP OPEN, ENTRY VALUE %f MORE THAN FREE USDT %f, %s %f > %f, %f > %f, SIZE %f",
						entryValue,
						bnspotUSDTBalance.Free,
						symbol,
						spread.LastEnter, quantile.Top,
						spread.MedianEnter, quantile.Top,
						quantity,
					)
					bnOpenLogSilentTimes[symbol] = time.Now().Add(time.Minute * 5)
				}
				continue
			}
			if quantity*price > bnspotUSDTBalance.Free {
				if time.Now().Sub(bnOpenLogSilentTimes[symbol]) > 0 {
					logger.Debugf(
						"FAILED TOP OPEN, ORDER VALUE %f MORE THAN FREE USDT %f, %s %f > %f, %f > %f, SIZE %f",
						quantity*price,
						bnspotUSDTBalance.Free,
						symbol,
						spread.LastEnter, quantile.Top,
						spread.MedianEnter, quantile.Top,
						quantity,
					)
					bnOpenLogSilentTimes[symbol] = time.Now().Add(time.Minute * 5)
				}
				continue
			}
			if quantity*price < spotMinNotional ||
				quantity*price < swapMinNotional {
				if time.Now().Sub(bnOpenLogSilentTimes[symbol]) > 0 {
					logger.Debugf(
						"FAILED TOP OPEN, ORDER VALUE %f LESS THAN NOTIONAL %f, %s %f > %f, %f > %f, SIZE %f",
						quantity*price,
						spotMinNotional,
						symbol,
						spread.LastEnter, quantile.Top,
						spread.MedianEnter, quantile.Top,
						quantity,
					)
					bnOpenLogSilentTimes[symbol] = time.Now().Add(time.Minute * 5)
				}
				continue
			}
			bnOpenLogSilentTimes[symbol] = time.Now()
			logger.Debugf(
				"TOP OPEN %s %f > %f, %f > %f, SIZE %f",
				symbol,
				spread.LastEnter, quantile.Top,
				spread.MedianEnter, quantile.Top,
				quantity,
			)
			id, _ := common.GenerateShortId()
			clOrdID := fmt.Sprintf(
				"%sLE%dME%dT%d",
				id,
				int(spread.LastEnter*10000),
				int(spread.MedianEnter*10000),
				int(quantile.Top*10000),
			)
			order := bnspot.NewOrderParams{
				Symbol:           symbol,
				Price:            price,
				Quantity:         quantity,
				TimeInForce:      bnspot.OrderTimeInForceFOK,
				Side:             bnspot.OrderSideBuy,
				Type:             bnspot.OrderTypeLimit,
				NewClientOrderID: clOrdID,
			}
			bnspotOrderSilentTimes[symbol] = time.Now().Add(*bnConfig.OrderSilent)
			if !*bnConfig.DryRun {
				go spotCreateOrder(bnGlobalCtx, bnspotAPI, *bnConfig.OrderTimeout, order)
			}
			return
		} else if spread.LastExit < quantile.Bot &&
			spread.MedianExit < quantile.Bot &&
			markPrice.FundingRate < *bnConfig.MinimalKeepFundingRate {
			price := spread.SpotOrderBook.TakerAskVWAP * (1.0 - *bnConfig.EnterSlippage)
			price = math.Ceil(price/spotTickSize) * spotTickSize
			if spotBalance.Free*price > spotMinNotional {
				entryValue := math.Min(-4*entryStep, -spotBalance.Free*price*0.5)
				if markPrice.FundingRate > *bnConfig.MinimalKeepFundingRate/2 {
					entryValue = math.Min(-2*entryStep, -spotBalance.Free*price*0.5)
				}
				quantity := entryValue / price
				quantity = math.Round(quantity/spotStepSize) * spotStepSize
				quantity = math.Round(quantity/swapStepSize) * swapStepSize
				if spotBalance.Free*price-entryValue < entryStep {
					quantity = -math.Floor(spotBalance.Free/spotStepSize) * spotStepSize
					//quantity = math.Ceil(quantity/swapStepSize) * swapStepSize
				}
				logger.Debugf(
					"BOT REDUCE %s %f < %f, %f < %f, SIZE %f",
					symbol,
					spread.LastExit, quantile.Bot,
					spread.MedianExit, quantile.Bot,
					quantity,
				)
				if quantity < 0 {
					id, _ := common.GenerateShortId()
					clOrdID := fmt.Sprintf(
						"%sLE%dME%dB%d",
						id,
						int(spread.LastExit*10000),
						int(spread.MedianExit*10000),
						int(quantile.Bot*10000),
					)
					order := bnspot.NewOrderParams{
						Symbol:           symbol,
						Price:            price,
						Quantity:         -quantity,
						TimeInForce:      bnspot.OrderTimeInForceFOK,
						Side:             bnspot.OrderSideSell,
						Type:             bnspot.OrderTypeLimit,
						NewClientOrderID: clOrdID,
					}
					bnspotOrderSilentTimes[symbol] = time.Now().Add(*bnConfig.OrderSilent)
					if !*bnConfig.DryRun {
						go spotCreateOrder(bnGlobalCtx, bnspotAPI, *bnConfig.OrderTimeout, order)
					}
					return
				}
			}
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
	swapMarkPrice, okSwapMarkPrice := bnswapMarkPrices[symbol]
	if !okSwapPosition || !okSpotBalance ||
		!okSwapMarkPrice || bnswapBNBAsset == nil ||
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
	if swapSize < 0 && -swapSize*swapMarkPrice.MarkPrice*(1.0-*bnConfig.EnterSlippage) < swapMinNotional {
		return
	}
	if swapSize > 0 && swapSize*swapMarkPrice.MarkPrice*(1.0+*bnConfig.EnterSlippage) < swapMinNotional {
		return
	}
	logger.Debugf("hedgeBnb %s SIZE %f POS %f -> %f", symbol, swapSize, swapPosition.PositionAmt, targetSize)

	reduceOnly := false
	if swapSize*swapPosition.PositionAmt < 0 && math.Abs(swapSize) <= math.Abs(swapPosition.PositionAmt) {
		reduceOnly = true
	}
	price := math.Round(swapMarkPrice.MarkPrice*(1.0+*bnConfig.EnterSlippage)/swapTickSize) * swapTickSize
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
		price = math.Round(swapMarkPrice.MarkPrice*(1.0-*bnConfig.EnterSlippage)/swapTickSize) * swapTickSize
	}
	order := bnswap.NewOrderParams{
		Symbol:           symbol,
		Side:             side,
		Type:             "LIMIT",
		Price:            price,
		TimeInForce:      "FOK",
		Quantity:         swapSize,
		ReduceOnly:       reduceOnly,
		NewClientOrderId: clOrdID,
	}
	bnswapOrderSilentTimes[symbol] = time.Now().Add(*bnConfig.OrderSilent)
	bnswapPositionsUpdateTimes[symbol] = time.Unix(0, 0)
	bnswapLastOrderTimes[symbol] = time.Now()
	if !*bnConfig.DryRun {
		go swapCreateOrder(bnGlobalCtx, bnswapAPI, *bnConfig.OrderTimeout, order)
	}
}
