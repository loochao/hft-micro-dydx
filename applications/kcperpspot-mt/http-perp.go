package main

import (
	"context"
	"github.com/geometrybase/hft-micro/kcperp"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func handlePerpHttpPositions(positions []kcperp.Position) {
	for _, nextPos := range positions {
		if _, ok := kcpsSymbolsMap[nextPos.Symbol]; !ok {
			continue
		}
		if nextPos.EventTime.Sub(kcperpLastOrderTimes[nextPos.Symbol]) < *kcConfig.PullInterval {
			return
		}
		var lastPosition *kcperp.Position
		if p, ok := kcperpPositions[nextPos.Symbol]; ok {
			p := p
			lastPosition = &p
		}
		kcperpPositions[nextPos.Symbol] = nextPos
		kcperpPositionsUpdateTimes[nextPos.Symbol] = time.Now()
		if lastPosition == nil ||
			lastPosition.CurrentQty != nextPos.CurrentQty ||
			lastPosition.AvgEntryPrice != nextPos.AvgEntryPrice {
			//如果SPOT变仓，立刻调PERP，如果PERP变仓，等ORDER SILENT TIMEOUT
			kcperpOrderSilentTimes[nextPos.Symbol] = time.Now()
			logger.Debugf("PERP HTTP POSITION %v", nextPos)
		}
	}
}

func handlePerpHttpAccount(account kcperp.Account) {
	if account.Currency == "USDT" {
		if kcperpUSDTAccount == nil ||
			kcperpUSDTAccount.AvailableBalance != account.AvailableBalance {
			logger.Debugf("PERP HTTP USDT ACCOUNT %v", account)
		}
		kcperpUSDTAccount = &account
		kcperpAssetUpdatedForReBalance = true
		kcperpAssetUpdatedForInflux = true
		kcperpAssetUpdatedForExternalInflux = true
	}
}

func swapCreateOrder(
	ctx context.Context,
	api *kcperp.API,
	timeout time.Duration,
	params kcperp.NewOrderParam,
) {
	childCtx, _ := context.WithTimeout(ctx, timeout)
	_, err := api.SubmitOrder(childCtx, params)
	if err != nil {
		select {
		case <-ctx.Done():
		case kcperpNewOrderErrorCh <- PerpOrderNewError{
			Error:  err,
			Params: params,
		}:
		}
	//} else if order.Status == "FILLED" ||
	//	order.Status == "CANCELED" ||
	//	order.Status == "REJECTED" ||
	//	order.Status == "EXPIRED" {
	//	kcperpOrderFinishCh <- *order
	}
}
