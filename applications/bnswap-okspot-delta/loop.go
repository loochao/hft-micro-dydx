package main

import (
	"fmt"
	"github.com/geometrybase/hft/bnswap"
	"github.com/geometrybase/hft/common"
	"github.com/geometrybase/hft/logger"
	"github.com/geometrybase/hft/okspot"
	"math"
	"strings"
	"time"
)

//func handleResetPnl() {
//
//	if bnswapUSDTAsset == nil || bnswapUSDTAsset.WalletBalance == nil || *bnswapUSDTAsset.WalletBalance == 0 {
//		return
//	}
//
//	if bnswapUSDTAsset.UnrealizedProfit == nil ||
//		*bnswapUSDTAsset.UnrealizedProfit / *bnswapUSDTAsset.WalletBalance > *boConfig.ResetUnrealisedTriggerPct {
//		return
//	}
//
//	//重置 先平仓FundingRate最小的
//
//	closableSymbols := make([]string, 0)
//	deltas := make([]float64, 0)
//	for symbol, position := range bnswapPositions {
//		if symbol == bnBNBSymbol {
//			continue
//		}
//		if position.PositionAmt == 0 {
//			continue
//		}
//
//		if !bnswapPositionsUpdated[symbol] {
//			continue
//		}
//
//		if bnswapOrderSilentTimes[symbol].Sub(time.Now()).Seconds() > 0 {
//			continue
//		}
//
//		//要保证现货能被平仓, 现货亏太多到最小开仓大小之后只能手动平仓
//		if math.Abs(position.PositionAmt*position.EntryPrice)-position.UnRealizedProfit > 1.2*okspotMinSizes[symbol] {
//			if exitDelta, ok := boMedianExitDeltas[symbol]; ok {
//				if quantile, ok := bnQuantiles[symbol]; ok {
//					deltas = append(deltas, quantile.ShortBot-exitDelta)
//					closableSymbols = append(closableSymbols, symbol)
//				}
//			} else {
//				logger.Debugf(" %s MARK PRICE NOT READY", symbol)
//				return
//			}
//		} else {
//			logger.Debugf("WARNING %s SWAP PNL %f, SPOT ESTIMATE VALUE %f < MIN NOTIONAL %f, SPOT NOT CLOSABLE",
//				symbol,
//				position.UnRealizedProfit,
//				math.Abs(position.PositionAmt*position.EntryPrice)-position.UnRealizedProfit,
//				okspotMinSizes[symbol],
//			)
//		}
//	}
//
//	deltasRanks := common.Rank(deltas)
//
//	for i, rank := range deltasRanks {
//		if int(rank) >= len(deltasRanks)-*boConfig.ResetCount {
//			symbol := closableSymbols[i]
//			logger.Debugf("TRIGGER RESET SWAP URPNL %f > %f, CLOSE SYMBOL %s CURRENT FR %f",
//				*bnswapUSDTAsset.UnrealizedProfit / *bnswapUSDTAsset.WalletBalance,
//				*boConfig.ResetUnrealisedTriggerPct,
//				symbol,
//				deltas[i],
//			)
//			spotOrderBook, okSpotOrderBook := okspotOrderBooks[symbol]
//			spotBalance, okSpotBalance := okspotBalances[symbol]
//
//			swapStepSize := bnswapStepSizes[symbol]
//			spotStepSize := okspotStepSizes[symbol]
//			spotTickSize := okspotTickSizes[symbol]
//
//			if !okSpotOrderBook || !okSpotBalance {
//				continue
//			}
//			entryValue := - *boConfig.EntryStep
//			entrySize := entryValue / ((spotOrderBook.Bids[0][0] + spotOrderBook.Asks[0][0]) * 0.5)
//			entrySize = math.Round(entrySize/spotStepSize) * spotStepSize
//			entrySize = math.Round(entrySize/swapStepSize) * swapStepSize
//			if spotBalance.Free*(spotOrderBook.Bids[0][0]+spotOrderBook.Asks[0][0])*0.5+entryValue < *boConfig.EntryStep {
//				entrySize = -spotBalance.Free
//			}
//			logger.Debugf("RESET %s RANK %.0f FR %.4f", symbol, rank, deltas[i])
//			id, _ := common.GenerateShortId()
//			clOrdID := fmt.Sprintf(
//				"%s-RESET-RK%.0f-FR%.0f",
//				id,
//				rank,
//				deltas[i]*10000,
//			)
//			changeSpotPosition(symbol, entrySize, clOrdID, spotTickSize, spotOrderBook.Bids[0][0], spotBalance, *okspotUSDTBalance)
//		}
//	}
//
//}

func updateSwapPositions() {
	for _, symbol := range boSymbols {
		if symbol == bnBNBSymbol {
			continue
		}

		if !bnswapPositionsUpdated[symbol] {
			continue
		}

		if bnswapOrderSilentTimes[symbol].Sub(time.Now()).Seconds() > 0 {
			continue
		}

		swapPosition, okSwapPosition := bnswapPositions[symbol]
		spotBalance, okSpotBalance := okspotBalances[symbol]
		swapOrderBook, okSwapOrderBook := bnswapOrderBooks[symbol]
		if !okSwapPosition || !okSpotBalance || !okSwapOrderBook {
			continue
		}

		swapStepSize := bnswapStepSizes[symbol]
		swapTickSize := bnswapTickSizes[symbol]
		swapMinNotional := bnswapMinNotional[symbol]

		swapSize := -spotBalance.Balance - swapPosition.PositionAmt
		swapSize = math.Round(swapSize/swapStepSize) * swapStepSize

		//只做空SWAP，所以开空是加仓，开多是减仓，减仓大小受当前空仓大小限制, 加仓受MinNotional限制
		if swapSize <= 0 && -swapSize*swapOrderBook.Bids[0][0]*(1.0-*boConfig.EnterSlippage) < swapMinNotional {
			continue
		}
		if swapSize > 0 && swapPosition.PositionAmt >= 0 {
			logger.Debugf("%s SWAP POSITION ERROR, CAN'T ADD %f TO POS %f", swapSize, swapPosition.PositionAmt)
			continue
		}
		if swapSize > 0 && swapSize > -swapPosition.PositionAmt {
			swapSize = -swapPosition.PositionAmt
		}

		logger.Debugf("updateSwapPositions %s SIZE %f POS %f -> %f", symbol, swapSize, swapPosition.PositionAmt, -spotBalance.Balance)

		reduceOnly := false
		if swapSize*swapPosition.PositionAmt < 0 && math.Abs(swapSize) <= math.Abs(swapPosition.PositionAmt) {
			reduceOnly = true
		}
		price := math.Round(swapOrderBook.Asks[0][0]*(1.0+*boConfig.EnterSlippage)/swapTickSize) * swapTickSize
		side := "BUY"
		id, _ := common.GenerateShortId()
		clOrdID := fmt.Sprintf(
			"%s-H%.6f",
			id,
			spotBalance.Balance,
		)
		clOrdID = strings.ReplaceAll(clOrdID, ".", "_")
		if swapSize < 0 {
			side = "SELL"
			swapSize = -swapSize
			price = math.Round(swapOrderBook.Bids[0][0]*(1.0-*boConfig.EnterSlippage)/swapTickSize) * swapTickSize
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
		bnswapOrderSilentTimes[symbol] = time.Now().Add(*boConfig.OrderSilent)
		bnswapPositionsUpdated[symbol] = false
		if !*boConfig.DryRun {
			go swapCreateOrder(boGlobalCtx, bnswapCredentials, bnswapAPI, *boConfig.OrderTimeout, order)
		}
	}
}

func updateSpotPositions() {

	if okspotUSDTBalance == nil {
		return
	}

	if len(bnswapFundingRateRanks) == 0 {
		return
	}

	for i, symbol := range boSymbols {

		if !okspotOrderBooksReady[symbol] || !bnswapOrderBooksReady[symbol] {
			continue
		}
		quantile, okQuantile := bnQuantiles[symbol]
		swapOrderBook, okSwapOrderBook := bnswapOrderBooks[symbol]
		spotOrderBook, okSpotOrderBook := okspotOrderBooks[symbol]
		spotBalance, okSpotBalance := okspotBalances[symbol]
		markPrice, okMarkPrice := bnswapMarkPrices[symbol]
		swapStepSize := bnswapStepSizes[symbol]
		if !okSpotOrderBook || !okSwapOrderBook || !okQuantile || !okSpotBalance || !okMarkPrice {
			continue
		}
		if okspotOrderBookTimestamps[symbol].Sub(spotOrderBook.Timestamp).Seconds() > 0 ||
			bnswapOrderBookTimestamps[symbol].Sub(swapOrderBook.EventTime).Seconds() > 0 {
			continue
		}
		okspotOrderBookTimestamps[symbol] = spotOrderBook.Timestamp
		bnswapOrderBookTimestamps[symbol] = swapOrderBook.EventTime

		swapSpotTimeDelta := swapOrderBook.EventTime.Sub(spotOrderBook.Timestamp)
		if swapSpotTimeDelta < 0 {
			swapSpotTimeDelta = -swapSpotTimeDelta
		}
		systemTimeDelta := (time.Now().Sub(swapOrderBook.EventTime) + time.Now().Sub(spotOrderBook.Timestamp)) / 2
		boSwapSpotTimeDeltas[symbol] = swapSpotTimeDelta
		boSystemTimeDeltas[symbol] = systemTimeDelta
		if swapSpotTimeDelta > *boConfig.SwapSpotTimeDeltaTolerance ||
			systemTimeDelta > *boConfig.SystemTimeDeltaTolerance {
			//logger.Debugf(
			//	"%s BAD TIME TOLERANCE, SWAP-SPOT TIMEDIFF %v SYSTEM TIMEDIFF %v",
			//	symbol, swapSpotTimeDelta, systemTimeDelta,
			//)
			continue
		}

		swapBidVwap, swapBidFarPrice, swapAskVwap, swapAskFarPrice := walkSwapOrderBook(swapOrderBook, *boConfig.MinimalOrderBookValue)
		spotBidVwap, spotBidFarPrice, spotAskVwap, spotAskFarPrice := walkSpotOrderBook(spotOrderBook, *boConfig.MinimalOrderBookValue)

		bnswapBidFarPrices[symbol] = swapBidFarPrice
		bnswapAskFarPrices[symbol] = swapAskFarPrice
		okspotBidFarPrices[symbol] = spotBidFarPrice
		okspotAskFarPrices[symbol] = spotAskFarPrice

		bnswapBidVwaps[symbol] = swapBidVwap
		bnswapAskVwaps[symbol] = swapAskVwap
		okspotBidVwaps[symbol] = spotBidVwap
		okspotAskVwaps[symbol] = spotAskVwap

		if spotBidVwap >= spotAskVwap*(1.0+EPSILON) {
			logger.Warnf("%s BAD ORDER BOOK  spotBidVwap %f >= spotAskVwap %f %v",
				symbol,
				spotBidVwap, spotAskVwap,
				spotOrderBook,
			)
			continue
		}
		if spotBidVwap < spotBidFarPrice*(1.0-EPSILON) {
			logger.Warnf("%s BAD ORDER BOOK  spotBidVwap %f < spotBidFarPrice %f %v",
				symbol,
				spotBidVwap, spotBidFarPrice,
				spotOrderBook,
			)
			continue
		}
		if spotAskVwap > spotAskFarPrice*(1.0+EPSILON) {
			logger.Warnf("%s BAD ORDER BOOK  spotAskVwap %f > spotAskFarPrice %f %v",
				symbol,
				spotAskVwap, spotAskVwap,
				spotOrderBook,
			)
			continue
		}

		if swapBidVwap >= swapAskVwap*(1.0+EPSILON) {
			logger.Warnf("%s BAD ORDER BOOK  swapBidVwap %f >= swapAskVwap %f %v",
				symbol,
				swapBidVwap, swapAskVwap,
				swapOrderBook,
			)
			continue
		}
		if swapBidVwap < swapBidFarPrice*(1.0-EPSILON) {
			logger.Warnf("%s BAD ORDER BOOK  swapBidVwap %f < swapBidFarPrice %f %v",
				symbol,
				swapBidVwap, swapBidFarPrice,
				swapOrderBook,
			)
			continue
		}
		if swapAskVwap > swapAskFarPrice*(1.0+EPSILON) {
			logger.Warnf("%s BAD ORDER BOOK  swapAskVwap %f > swapAskFarPrice %f %v",
				symbol,
				swapAskVwap, swapAskVwap,
				swapOrderBook,
			)
			continue
		}

		midVwap := (swapBidVwap + swapAskVwap + spotBidVwap + spotAskVwap) * 0.25

		bnMidVwaps[symbol] = midVwap

		lastEnterDelta := (swapBidVwap - spotAskVwap) / midVwap
		lastExitDelta := (swapAskVwap - spotBidVwap) / midVwap

		boLastEnterDeltas[symbol] = lastEnterDelta
		boLastExitDeltas[symbol] = lastExitDelta

		boEnterDeltaWindows[symbol] = append(boEnterDeltaWindows[symbol], lastEnterDelta)
		boExitDeltaWindows[symbol] = append(boExitDeltaWindows[symbol], lastExitDelta)
		boArrivalTimes[symbol] = append(boArrivalTimes[symbol], swapOrderBook.ArrivalTime)
		boEnterDeltaSortedSlices[symbol] = boEnterDeltaSortedSlices[symbol].Insert(lastEnterDelta)
		boExitDeltaSortedSlices[symbol] = boExitDeltaSortedSlices[symbol].Insert(lastExitDelta)

		cutIndex := 0
		for i, arrivalTime := range boArrivalTimes[symbol] {
			if swapOrderBook.ArrivalTime.Sub(arrivalTime) > *boConfig.DeltaLookback {
				cutIndex = i
			} else {
				break
			}
		}
		if cutIndex > 0 {
			for _, d := range boEnterDeltaWindows[symbol][:cutIndex] {
				boEnterDeltaSortedSlices[symbol] = boEnterDeltaSortedSlices[symbol].Delete(d)
			}
			for _, d := range boExitDeltaWindows[symbol][:cutIndex] {
				boExitDeltaSortedSlices[symbol] = boExitDeltaSortedSlices[symbol].Delete(d)
			}
			boEnterDeltaWindows[symbol] = boEnterDeltaWindows[symbol][cutIndex:]
			boExitDeltaWindows[symbol] = boExitDeltaWindows[symbol][cutIndex:]
			boArrivalTimes[symbol] = boArrivalTimes[symbol][cutIndex:]
		}

		if len(boEnterDeltaWindows[symbol]) < *boConfig.MinimalDeltaWindow ||
			len(boExitDeltaWindows[symbol]) < *boConfig.MinimalDeltaWindow {
			continue
		}

		arrivalTimeDiff := swapOrderBook.ArrivalTime.Sub(boArrivalTimes[symbol][0])
		if arrivalTimeDiff < *boConfig.DeltaLookback/2 {
			continue
		}

		medianEnterDelta := boEnterDeltaSortedSlices[symbol].Median()
		medianExitDelta := boExitDeltaSortedSlices[symbol].Median()

		boMedianEnterDeltas[symbol] = medianEnterDelta
		boMedianExitDeltas[symbol] = medianExitDelta

		if symbol == bnBNBSymbol {
			continue
		}

		if !bnSymbolReady[symbol] {
			logger.Debugf("%s INITIALIZED Enter %f %f Exit %f %f", symbol, lastEnterDelta, medianEnterDelta, lastExitDelta, medianExitDelta)
			bnSymbolReady[symbol] = true

		}

		if !okspotBalancesUpdated[symbol] {
			continue
		}

		if okspotOrderSilentTimes[symbol].Sub(time.Now()).Seconds() > 0 {
			continue
		}

		swapMinNotional := bnswapMinNotional[symbol]

		spotStepSize := okspotStepSizes[symbol]
		spotTickSize := okspotTickSizes[symbol]
		spotMinSize := okspotMinSizes[symbol]

		currentSpotSize := spotBalance.Available

		if medianEnterDelta > quantile.Top &&
			lastEnterDelta > quantile.Top &&
			markPrice.FundingRate > *boConfig.MinimalEnterFundingRate &&
			int(bnswapFundingRateRanks[i]) >= len(bnswapFundingRateRanks)-*boConfig.TradeCount &&
			okspotEnterSilentTimes[symbol].Sub(time.Now()) <= 0 {

			targetValue := currentSpotSize*midVwap + *boConfig.EnterStep
			if targetValue > *boConfig.EnterTarget {
				targetValue = *boConfig.EnterTarget
			}
			entryValue := targetValue - currentSpotSize*midVwap

			//logger.Debugf("entryValue %f", entryValue)
			//不及一个0.8*EntryStep, 不操作
			if entryValue < *boConfig.EnterStep*0.8 {
				continue
			}

			if entryValue > okspotUSDTBalance.Available*0.8 {
				entryValue = okspotUSDTBalance.Available * 0.8
			}

			entryValue = math.Max(entryValue, swapMinNotional)
			if entryValue > okspotUSDTBalance.Available {
				continue
			}

			entrySize := entryValue / (spotAskFarPrice * (1.0 + *boConfig.EnterSlippage))
			if entrySize < spotMinSize {
				entrySize = spotMinSize
			}
			entrySize = math.Round(entrySize/spotStepSize) * spotStepSize
			entrySize = math.Round(entrySize/swapStepSize) * swapStepSize

			if bnOpenLogSilentTimes[symbol].Sub(time.Now()).Seconds() < 0 &&
				entrySize*spotAskFarPrice*(1.0+*boConfig.EnterSlippage) > okspotUSDTBalance.Available {
				logger.Debugf(
					"FAILED OPEN %s %f > %f, %f > %f, SIZE %f",
					symbol,
					lastEnterDelta, quantile.Top,
					medianEnterDelta, quantile.Top,
					entrySize,
				)
				bnOpenLogSilentTimes[symbol] = time.Now().Add(time.Minute)
			}

			if entrySize*spotAskFarPrice*(1.0+*boConfig.EnterSlippage) > okspotUSDTBalance.Available {
				continue
			}
			logger.Debugf(
				"OPEN %s %f > %f, %f > %f, SIZE %f",
				symbol,
				lastEnterDelta, quantile.Top,
				medianEnterDelta, quantile.Top,
				entrySize,
			)
			id, _ := common.GenerateShortId()
			clOrdID := fmt.Sprintf(
				"%sLD%dMD%dQT%d",
				id,
				int(lastEnterDelta*10000),
				int(medianEnterDelta*10000),
				int(quantile.Top*10000),
			)
			changeSpotPosition(symbol, entrySize, clOrdID, spotTickSize, spotAskFarPrice, spotBalance, *okspotUSDTBalance)

		} else if lastExitDelta < quantile.Bot &&
			medianExitDelta < quantile.Bot &&
			markPrice.FundingRate < *boConfig.MinimalKeepFundingRate &&
			okspotExitSilentTimes[symbol].Sub(time.Now()) <= 0 {

			if spotBalance.Available > spotMinSize {
				entryValue := -4 * *boConfig.EnterStep
				if markPrice.FundingRate > *boConfig.MinimalKeepFundingRate/2 {
					entryValue = -2 * *boConfig.EnterStep
				}
				entrySize := entryValue / (spotBidFarPrice * (1.0 - *boConfig.EnterSlippage))
				entrySize = math.Round(entrySize/spotStepSize) * spotStepSize
				entrySize = math.Round(entrySize/swapStepSize) * swapStepSize
				if spotBalance.Available*spotBidFarPrice*(1.0-*boConfig.EnterSlippage)+entryValue < *boConfig.EnterStep {
					entrySize = -spotBalance.Available
				}
				logger.Debugf(
					"REDUCE %s %f < %f, %f < %f, SIZE %f",
					symbol,
					medianExitDelta, quantile.Bot,
					lastExitDelta, quantile.Bot,
					entrySize,
				)
				id, _ := common.GenerateShortId()
				clOrdID := fmt.Sprintf(
					"%sLE%dME%dQB%d",
					id,
					int(lastExitDelta*10000),
					int(medianExitDelta*10000),
					int(quantile.Bot*10000),
				)
				changeSpotPosition(symbol, entrySize, clOrdID, spotTickSize, spotBidFarPrice, spotBalance, *okspotUSDTBalance)
			}
		}
	}
}

func changeSpotPosition(
	symbol string,
	size float64,
	clOrdID string,
	tickSize float64,
	refPrice float64,
	balance okspot.Balance,
	usdtBalance okspot.Balance,
) {
	if size == 0 {
		logger.Debugf(
			"SPOT %s changeSpotPosition failed, bad size %f",
			symbol,
			size,
		)
	}else if size < 0 && size*balance.Available < 0 && -size <= balance.Available{
		logger.Debugf("PURE LONG REDUCE")
	} else if size > 0 && usdtBalance.Available < size*(refPrice*(1.0+*boConfig.EnterSlippage)) {
		logger.Debugf(
			"SPOT USDT BALANCE INSUFFICIENT NEED %.2f AVAILABLE %.2f, RETRY IN 15M",
			size*(refPrice*(1.0+*boConfig.EnterSlippage)),
			okspotUSDTBalance.Available,
		)
		okspotOrderSilentTimes[symbol] = time.Now().Add(time.Minute * 15)
		return
	}
	if size < 0 && balance.Available < -size {
		logger.Debugf("SPOT USDT BALANCE INSUFFICIENT NEED %.2f AVAILABLE %.2f, RETRY IN 15M",
			-size,
			balance.Available,
		)
		okspotOrderSilentTimes[symbol] = time.Now().Add(time.Minute * 15)
		return
	}
	clOrdID = strings.Replace(clOrdID, "-", "0", -1)
	if len(clOrdID) > 32 {
		clOrdID = clOrdID[:32]
	}
	price := math.Round(refPrice*(1.0+*boConfig.EnterSlippage)/tickSize) * tickSize
	params := okspot.NewOrderParams{
		InstrumentId: okspot.SymbolToInstrumentId(symbol),
		ClientOID:    clOrdID,
		Side:         okspot.OrderSideBuy,
		Type:         okspot.OrderLimit,
		OrderType:    okspot.OrderTypeFillOrKill,
		Price:        &price,
		Size:         &size,
	}
	if size < 0 {
		*params.Size = -size
		*params.Price = math.Round(refPrice*(1.0-*boConfig.EnterSlippage)/tickSize) * tickSize
		params.Side = okspot.OrderSideSell
	}
	okspotOrderSilentTimes[symbol] = time.Now().Add(*boConfig.OrderSilent)
	okspotBalancesUpdated[symbol] = false
	if !*boConfig.DryRun {
		go createSpotOrder(boGlobalCtx, okspotCredentials, okspotAPI, *boConfig.OrderTimeout, params)
	}
}

func walkSwapOrderBook(orderBook bnswap.PartialBookDepthStream, minimalValue float64) (bidVwap, bidFarPrice, askVwap, askFarPrice float64) {
	totalValue := 0.0
	totalQty := 0.0
	for _, bid := range orderBook.Bids {
		value := bid[0] * bid[1]
		bidFarPrice = bid[0]
		if totalValue+value > minimalValue {
			totalQty += (minimalValue - totalValue) / bid[0]
			totalValue = minimalValue
			break
		} else {
			totalQty += bid[1]
			totalValue += value
		}
	}
	bidVwap = totalValue / totalQty

	totalValue = 0.0
	totalQty = 0.0
	for _, ask := range orderBook.Asks {
		value := ask[0] * ask[1]
		askFarPrice = ask[0]
		if totalValue+value >= minimalValue {
			totalQty += (minimalValue - totalValue) / ask[0]
			totalValue = minimalValue
			break
		} else {
			totalQty += ask[1]
			totalValue += value
		}
	}
	askVwap = totalValue / totalQty
	return
}

func walkSpotOrderBook(orderBook okspot.WSDepth5, minimalValue float64) (bidVwap, bidFarPrice, askVwap, askFarPrice float64) {
	totalValue := 0.0
	totalQty := 0.0
	for _, bid := range orderBook.Bids {
		value := bid[0] * bid[1]
		bidFarPrice = bid[0]
		if totalValue+value >= minimalValue {
			totalQty += (minimalValue - totalValue) / bid[0]
			totalValue = minimalValue
			break
		} else {
			totalQty += bid[1]
			totalValue += value
		}
	}
	bidVwap = totalValue / totalQty

	totalValue = 0.0
	totalQty = 0.0
	for _, ask := range orderBook.Asks {
		value := ask[0] * ask[1]
		askFarPrice = ask[0]
		if totalValue+value >= minimalValue {
			totalQty += (minimalValue - totalValue) / ask[0]
			totalValue = minimalValue
			break
		} else {
			totalQty += ask[1]
			totalValue += value
		}
	}
	askVwap = totalValue / totalQty
	return
}
