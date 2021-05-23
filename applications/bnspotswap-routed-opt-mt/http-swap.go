package main

import (
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func handleSwapHttpPositions(positions []bnswap.Position) {
	for _, nextPos := range positions {
		if _, ok := bnspotOffsets[nextPos.Symbol]; !ok {
			return
		}
		if nextPos.PositionSide != "BOTH" {
			continue
		}
		//if bnswapHttpPositionUpdateSilentTimes[nextPos.Symbol].Sub(nextPos.ParseTime) > 0 {
		//	continue
		//}
		var lastPosition *bnswap.Position
		if currentPosition, ok := bnswapPositions[nextPos.Symbol]; ok {
			if currentPosition.EventTime.Sub(nextPos.EventTime) > 0 {
				logger.Debugf("%s nextPos EventTime is older %v < %v", nextPos.EventTime, currentPosition.EventTime)
				continue
			}
			lastPosition = &bnswap.Position{}
			*lastPosition = currentPosition
		}
		bnswapPositions[nextPos.Symbol] = nextPos
		bnswapPositionsUpdateTimes[nextPos.Symbol] = time.Now()
		if lastPosition == nil ||
			lastPosition.PositionAmt != nextPos.PositionAmt ||
			lastPosition.EntryPrice != nextPos.EntryPrice {
			//如果SPOT变仓，立刻调SWAP，如果SWAP变仓，等ORDER SILENT TIMEOUT
			bnLoopTimer.Reset(time.Nanosecond)
			//bnswapOrderSilentTimes[nextPos.Symbol] = time.Now()
			logger.Debugf("SWAP HTTP POSITION %s", nextPos.ToString())
			if lastPosition != nil {
				logger.Debugf("SWAP %s POS OLD TIME %v NEW TIME %v", nextPos.Symbol, lastPosition.EventTime, nextPos.EventTime)
			}
		}
	}
}

func handleSwapHttpAccount(account bnswap.Account) {
	for _, asset := range account.Assets {
		if asset.Asset == "USDT" {
			asset := asset
			bnswapUSDTAsset = &asset
			bnswapAssetUpdatedForReBalance = true
			bnswapAssetUpdatedForInflux = true
			bnswapAssetUpdatedForExternalInflux = true
			bnLoopTimer.Reset(time.Nanosecond)
			continue
		}
		if asset.Asset == "BNB" {
			asset := asset
			bnswapBNBAsset = &asset
			continue
		}
	}
}

//func swapCreateOrder(
//	ctx context.Context,
//	api *bnswap.API,
//	timeout time.Duration,
//	params bnswap.NewOrderParams,
//) {
//	childCtx, _ := context.WithTimeout(ctx, timeout)
//	order, err := api.SubmitOrder(childCtx, params)
//	if err != nil {
//		logger.Debugf("SUBMIT ERROR %s  %v ", params.ToString(), err)
//		select {
//		case <-ctx.Done():
//		case bnswapOrderNewErrorCh <- TakerOrderNewError{
//			Error:  err,
//			Params: params,
//		}:
//		}
//	} else if order.Status == "FILLED" ||
//		order.Status == "CANCELED" ||
//		order.Status == "REJECTED" ||
//		order.Status == "EXPIRED" {
//		bnswapOrderResponseCh <- *order
//	}
//}
