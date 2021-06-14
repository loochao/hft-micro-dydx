package main

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"
	"time"
)

func main() {

	if xyConfig.CpuProfile != "" {
		f, err := os.Create(xyConfig.CpuProfile)
		if err != nil {
			logger.Debugf("os.Create error %v", err)
			return
		}
		err = pprof.StartCPUProfile(f)
		if err != nil {
			logger.Debugf("pprof.StartCPUProfile error %v", err)
			return
		}
		defer pprof.StopCPUProfile()
	}

	xyGlobalCtx, xyGlobalCancel = context.WithCancel(context.Background())
	defer xyGlobalCancel()

	var err error
	err = xExchange.Setup(xyGlobalCtx, xyConfig.XExchange)
	if err != nil {
		logger.Debugf("xExchange.Setup(xyGlobalCtx, xyConfig.XExchange) error %v", err)
		return
	}
	err = yExchange.Setup(xyGlobalCtx, xyConfig.YExchange)
	if err != nil {
		logger.Debugf("yExchange.Setup(xyGlobalCtx, xyConfig.YExchange) error %v", err)
		return
	}
	for _, ySymbol := range ySymbols {
		yStepSizes[ySymbol], err = yExchange.GetStepSize(ySymbol)
		if err != nil {
			logger.Debugf("yExchange.GetStepSize(ySymbol) error %v", err)
		}
		yMinNotionals[ySymbol], err = yExchange.GetMinNotional(ySymbol)
		if err != nil {
			logger.Debugf("yExchange.GetMinNotional(ySymbol) error %v", err)
		}
		yMultipliers[ySymbol], err = yExchange.GetMultiplier(ySymbol)
		if err != nil {
			logger.Debugf("yExchange.GetMultiplier(ySymbol) error %v", err)
		}
	}
	logger.Debugf("y stepSizes %v", yStepSizes)
	logger.Debugf("y minNotional %v", yMinNotionals)
	logger.Debugf("y multipliers %v", yMultipliers)
	for _, xSymbol := range xSymbols {
		xStepSizes[xSymbol], err = xExchange.GetStepSize(xSymbol)
		if err != nil {
			logger.Debugf("xExchange.GetStepSize(xSymbol) error %v", err)
		}
		xMinNotionals[xSymbol], err = xExchange.GetMinNotional(xSymbol)
		if err != nil {
			logger.Debugf("xExchange.GetMinNotional(xSymbol) error %v", err)
		}
		xMultipliers[xSymbol], err = xExchange.GetMultiplier(xSymbol)
		if err != nil {
			logger.Debugf("xExchange.GetMultiplier(xSymbol) error %v", err)
		}
	}
	logger.Debugf("x stepSizes %v", xStepSizes)
	logger.Debugf("x minNotional %v", xMinNotionals)
	logger.Debugf("x multipliers %v", xMultipliers)

	for xSymbol, xStepSize := range xStepSizes {
		ySymbol := xySymbolsMap[xSymbol]
		yStepSize := yStepSizes[ySymbol]
		xMultiplier := xMultipliers[xSymbol]
		yMultiplier := yMultipliers[ySymbol]
		xyUsdStepSizes[xSymbol] = common.MergedStepSize(xStepSize*xMultiplier, yStepSize*yMultiplier)
	}
	logger.Debugf("merged step sizes: %v", xyUsdStepSizes)

	if xyConfig.InternalInflux.Address != "" {
		xyInfluxWriter, err = common.NewInfluxWriter(
			xyGlobalCtx,
			xyConfig.InternalInflux.Address,
			xyConfig.InternalInflux.Username,
			xyConfig.InternalInflux.Password,
			xyConfig.InternalInflux.Database,
			xyConfig.InternalInflux.BatchSize,
		)
		if err != nil {
			logger.Debugf("common.NewInfluxWriter error %v", err)
			return
		}
		defer xyInfluxWriter.Stop()
	}

	if xyConfig.ExternalInflux.Address != "" {
		xyExternalInfluxWriter, err = common.NewInfluxWriter(
			xyGlobalCtx,
			xyConfig.ExternalInflux.Address,
			xyConfig.ExternalInflux.Username,
			xyConfig.ExternalInflux.Password,
			xyConfig.ExternalInflux.Database,
			xyConfig.ExternalInflux.BatchSize,
		)
		if err != nil {
			logger.Debugf("common.NewInfluxWriter error %v", err)
			return
		}
		defer xyExternalInfluxWriter.Stop()
	}

	influxSaveTimer := time.NewTimer(
		time.Now().Truncate(
			xyConfig.InternalInflux.SaveInterval,
		).Add(
			xyConfig.InternalInflux.SaveInterval * 3,
		).Sub(time.Now()),
	)
	defer influxSaveTimer.Stop()

	xyLoopTimer = time.NewTimer(time.Second)
	defer xyLoopTimer.Stop()

	xPositionChMap := make(map[string]chan common.Position)
	xOrderChMap := make(map[string]chan common.Order)
	xFundingRateChMap := make(map[string]chan common.FundingRate)
	xDepthChMap := make(map[string]chan common.Depth)
	xNewOrderErrorChMap := make(map[string]chan common.OrderError)
	xAccountChMap := make(map[string]chan common.Balance)
	for _, xSymbol := range xSymbols {
		xPositionChMap[xSymbol] = xPositionCh
		xOrderChMap[xSymbol] = xOrderCh
		xFundingRateChMap[xSymbol] = xFundingRateCh
		xDepthChMap[xSymbol] = make(chan common.Depth, 200)
		xOrderRequestChMap[xSymbol] = make(chan common.OrderRequest, 200)
		xNewOrderErrorChMap[xSymbol] = xNewOrderErrorCh
		xAccountChMap[xyConfig.XSymbolAssetMap[xSymbol]] = xAccountCh
	}
	go xExchange.StreamBasic(
		xyGlobalCtx,
		xSystemStatusCh,
		xAccountChMap,
		xPositionChMap,
		xOrderChMap,
	)
	go xExchange.StreamFundingRate(
		xyGlobalCtx,
		xFundingRateChMap,
		xyConfig.BatchSize,
	)
	go xExchange.StreamDepth(
		xyGlobalCtx,
		xDepthChMap,
		xyConfig.BatchSize,
	)
	go xExchange.WatchOrders(
		xyGlobalCtx,
		xOrderRequestChMap,
		xOrderChMap,
		xNewOrderErrorChMap,
	)

	yPositionChMap := make(map[string]chan common.Position)
	yOrderChMap := make(map[string]chan common.Order)
	yFundingRateChMap := make(map[string]chan common.FundingRate)
	yDepthChMap := make(map[string]chan common.Depth)
	yNewOrderErrorChMap := make(map[string]chan common.OrderError)
	yAccountChMap := make(map[string]chan common.Balance)
	for _, ySymbol := range ySymbols {
		yPositionChMap[ySymbol] = yPositionCh
		yOrderChMap[ySymbol] = yOrderCh
		yFundingRateChMap[ySymbol] = yFundingRateCh
		yDepthChMap[ySymbol] = make(chan common.Depth, 200)
		yOrderRequestChMap[ySymbol] = make(chan common.OrderRequest, 200)
		yNewOrderErrorChMap[ySymbol] = yNewOrderErrorCh
		yAccountChMap[xyConfig.YSymbolAssetMap[ySymbol]] = yAccountCh
	}
	go yExchange.StreamBasic(
		xyGlobalCtx,
		ySystemStatusCh,
		yAccountChMap,
		yPositionChMap,
		yOrderChMap,
	)
	go yExchange.StreamFundingRate(
		xyGlobalCtx,
		yFundingRateChMap,
		xyConfig.BatchSize,
	)
	go yExchange.StreamDepth(
		xyGlobalCtx,
		yDepthChMap,
		xyConfig.BatchSize,
	)
	go yExchange.WatchOrders(
		xyGlobalCtx,
		yOrderRequestChMap,
		yOrderChMap,
		yNewOrderErrorChMap,
	)

	spreadReportCh := make(chan SpreadReport, 10000)
	go reportsSaveLoop(
		xyGlobalCtx,
		xyInfluxWriter,
		xyConfig.InternalInflux,
		spreadReportCh,
	)

	spreadCh := make(chan *XYSpread, len(xSymbols)*100)
	for xSymbol, ySymbol := range xyConfig.XYPairs {
		go watchXYSpread(
			xyGlobalCtx,
			xSymbol, ySymbol,
			xMultipliers[xSymbol],
			yMultipliers[ySymbol],
			xyConfig.DepthTakerImpact,
			xyConfig.DepthXDecay,
			xyConfig.DepthXBias,
			xyConfig.DepthYDecay,
			xyConfig.DepthYBias,
			xyConfig.DepthTimeDeltaMin,
			xyConfig.DepthTimeDeltaMax,
			xyConfig.DepthMaxAgeDiffBias,
			xyConfig.ReportCount,
			xyConfig.SpreadLookback,
			xDepthChMap[xSymbol],
			yDepthChMap[ySymbol],
			spreadReportCh,
			spreadCh,
		)
	}

	if xyConfig.CpuProfile != "" {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			sig := <-sigs
			logger.Debugf("catch exit signal %v", sig)
			xyGlobalCancel()
		}()
	}

	logger.Debugf("start main loop")
	fundingInterval := time.Hour * 8
	fundingSilent := time.Minute * 5

	restartTimer := time.NewTimer(xyConfig.RestartInterval)
	defer restartTimer.Stop()

mainLoop:
	for {
		select {
		case <-xyGlobalCtx.Done():
			logger.Debugf("global ctx done, exit main loop")
			xyGlobalCancel()
			break mainLoop
		case <-xExchange.Done():
			logger.Debugf("x exchange done, exit main loop")
			xyGlobalCancel()
			break mainLoop
		case <-yExchange.Done():
			logger.Debugf("y exchange done, exit main loop")
			xyGlobalCancel()
			break mainLoop
		case <-restartTimer.C:
			logger.Debugf("timed restart in %v", xyConfig.RestartInterval)
			xyGlobalCancel()
			break mainLoop
		case xSystemStatus = <-xSystemStatusCh:
			if xSystemStatus != common.SystemStatusReady {
				logger.Debugf("xSystemStatus %v", xSystemStatus)
			}
			break
		case ySystemStatus = <-ySystemStatusCh:
			if ySystemStatus != common.SystemStatusReady {
				logger.Debugf("ySystemStatus %v", ySystemStatus)
			}
			break
		case nextPos := <-xPositionCh:
			//logger.Debugf("x position %s %v %v %f %f", nextPos.GetSymbol(), nextPos.GetEventTime(), nextPos.GetParseTime(), nextPos.GetPrice(), nextPos.GetSize())
			if _, ok := xySymbolsMap[nextPos.GetSymbol()]; !ok {
				break
			}
			if prevPos, ok := xPositions[nextPos.GetSymbol()]; ok {
				if prevPos == nextPos {
					logger.Debugf("bad prevPos == nextPos pass same pointer")
				}
				if nextPos.GetEventTime().Sub(prevPos.GetEventTime()) >= 0 {
					xTimedPositionChange.Insert(time.Now(), math.Abs(prevPos.GetSize()-nextPos.GetSize())*xMultipliers[nextPos.GetSymbol()])
					xPositions[nextPos.GetSymbol()] = nextPos
					if prevPos.GetSize() != nextPos.GetSize() {
						logger.Debugf("%s x position change %f -> %f %v", nextPos.GetSymbol(), prevPos.GetSize(), nextPos.GetSize(), nextPos.GetEventTime())
					}
				}
				xPositionsUpdateTimes[nextPos.GetSymbol()] = nextPos.GetParseTime()
			} else {
				xPositions[nextPos.GetSymbol()] = nextPos
				xPositionsUpdateTimes[nextPos.GetSymbol()] = nextPos.GetParseTime()
				logger.Debugf("%s x position change nil -> %f", nextPos.GetSymbol(), nextPos.GetSize())
			}
			break
		case account := <-xAccountCh:
			//logger.Debugf("x account %s %f %f %f", xAccount.GetCurrency(), xAccount.GetBalance(), xAccount.GetFree(), xAccount.GetUsed())
			if xAccount, ok := xBalances[account.GetCurrency()]; ok {
				if xAccount == account {
					logger.Debugf("bad xAccount == account pass same pointer")
				}else if account.GetTime().Sub(xAccount.GetTime()) >= 0 {
					xBalances[account.GetCurrency()] = account
					//logger.Debugf("xBalance %f", account.GetBalance())
				}
			}else{
				xBalances[account.GetCurrency()] = account
			}
			break
		case nextPos := <-yPositionCh:
			//logger.Debugf("y position %s %v %v %f %f", nextPos.GetSymbol(), nextPos.GetEventTime(), nextPos.GetParseTime(), nextPos.GetPrice(), nextPos.GetSize())
			if _, ok := yxSymbolsMap[nextPos.GetSymbol()]; !ok {
				break
			}
			if prevPos, ok := yPositions[nextPos.GetSymbol()]; ok {
				if prevPos == nextPos {
					logger.Debugf("bad prevPos == nextPos pass same pointer")
				}
				if nextPos.GetEventTime().Sub(prevPos.GetEventTime()) >= 0 {
					yPositions[nextPos.GetSymbol()] = nextPos
					if prevPos.GetSize() != nextPos.GetSize() {
						yTimedPositionChange.Insert(time.Now(), math.Abs(prevPos.GetSize()-nextPos.GetSize())*yMultipliers[nextPos.GetSymbol()])
						logger.Debugf("%s y position change %f -> %f %v", nextPos.GetSymbol(), prevPos.GetSize(), nextPos.GetSize(), nextPos.GetEventTime())
					}
				}
				yPositionsUpdateTimes[nextPos.GetSymbol()] = nextPos.GetParseTime()
			} else {
				yPositions[nextPos.GetSymbol()] = nextPos
				yPositionsUpdateTimes[nextPos.GetSymbol()] = nextPos.GetParseTime()
				logger.Debugf("%s y position change nil -> %f", nextPos.GetSymbol(), nextPos.GetSize())
			}
			break
		case account := <-yAccountCh:
			if yAccount, ok := yBalances[account.GetCurrency()]; ok {
				if yAccount == account {
					logger.Debugf("bad  yAccount == account pass same pointer")
				}else if account.GetTime().Sub(yAccount.GetTime()) >= 0 {
					yBalances[account.GetCurrency()] = account
					//logger.Debugf("yBalance %f", account.GetBalance())
				}
			}else{
				yBalances[account.GetCurrency()] = account
			}
			//logger.Debugf("y account %s %f %f %f", yAccount.GetCurrency(), yAccount.GetBalance(), yAccount.GetFree(), yAccount.GetUsed())
			break
		case xOrder := <-xOrderCh:
			if xOrder.GetStatus() == common.OrderStatusExpired ||
				xOrder.GetStatus() == common.OrderStatusReject ||
				xOrder.GetStatus() == common.OrderStatusCancelled ||
				xOrder.GetStatus() == common.OrderStatusFilled {

				xSymbol := xOrder.GetSymbol()
				if xOrder.GetStatus() != common.OrderStatusFilled {
					logger.Debugf("x order ended %s %s %s", xOrder.GetSymbol(), xOrder.GetStatus(), xOrder.GetSide())
					xOrderSilentTimes[xSymbol] = time.Now().Add(time.Second)
					xPositionsUpdateTimes[xSymbol] = time.Unix(0, 0)
				} else {
					logger.Debugf("x order filled %s %s %s size %f price %f", xOrder.GetSymbol(), xOrder.GetStatus(), xOrder.GetSide(), xOrder.GetFilledSize(), xOrder.GetFilledPrice())
					if ySymbol, ok := xySymbolsMap[xSymbol]; ok {
						if xOrder.GetSide() == common.OrderSideBuy {
							xBuyPrice := xOrder.GetFilledPrice()
							xLastFilledBuyPrices[xSymbol] = xBuyPrice
							if ySellPrice, ok := yLastFilledSellPrices[ySymbol]; ok {
								xyRealisedSpread[xSymbol] = (ySellPrice - xBuyPrice) / ySellPrice
								logger.Debugf("%s - %s realised short spread %f", xSymbol, ySymbol, xyRealisedSpread[xSymbol])
								delete(xLastFilledBuyPrices, xSymbol)
								delete(xLastFilledSellPrices, xSymbol)
								delete(yLastFilledBuyPrices, ySymbol)
								delete(yLastFilledSellPrices, ySymbol)
							}
						} else if xOrder.GetSide() == common.OrderSideSell {
							xSellPrice := xOrder.GetFilledPrice()
							xLastFilledSellPrices[xSymbol] = xSellPrice
							if yBuyPrice, ok := yLastFilledBuyPrices[ySymbol]; ok {
								xyRealisedSpread[xSymbol] = (yBuyPrice - xSellPrice) / yBuyPrice
								logger.Debugf("%s - %s realised long spread %f", ySymbol, xSymbol, xyRealisedSpread[xSymbol])
								delete(xLastFilledBuyPrices, xSymbol)
								delete(xLastFilledSellPrices, xSymbol)
								delete(yLastFilledBuyPrices, ySymbol)
								delete(yLastFilledSellPrices, ySymbol)
							}
						}
					}
				}
			}
			break
		case yOrder := <-yOrderCh:
			if yOrder.GetStatus() == common.OrderStatusExpired ||
				yOrder.GetStatus() == common.OrderStatusReject ||
				yOrder.GetStatus() == common.OrderStatusCancelled ||
				yOrder.GetStatus() == common.OrderStatusFilled {

				ySymbol := yOrder.GetSymbol()
				if yOrder.GetStatus() != common.OrderStatusFilled {
					logger.Debugf("y order ended %s %s %s", yOrder.GetSymbol(), yOrder.GetStatus(), yOrder.GetSide())
					yOrderSilentTimes[ySymbol] = time.Now().Add(time.Second)
					yPositionsUpdateTimes[ySymbol] = time.Unix(0, 0)
				} else {
					logger.Debugf("y order filled %s %s %s size %f price %f", yOrder.GetSymbol(), yOrder.GetStatus(), yOrder.GetSide(), yOrder.GetFilledSize(), yOrder.GetFilledPrice())
					if xSymbol, ok := yxSymbolsMap[ySymbol]; ok {
						if yOrder.GetSide() == common.OrderSideBuy {
							yBuyPrice := yOrder.GetFilledPrice()
							yLastFilledBuyPrices[ySymbol] = yBuyPrice
							//logger.Debugf("%s set y buy price %f dir x %s sell %f", ySymbol, yBuyPrice, xSymbol, xLastFilledSellPrices[xSymbol])
							if xSellPrice, ok := xLastFilledSellPrices[xSymbol]; ok {
								xyRealisedSpread[xSymbol] = (yBuyPrice - xSellPrice) / yBuyPrice
								logger.Debugf("%s - %s realised long spread %f", ySymbol, xSymbol, xyRealisedSpread[xSymbol])
								delete(xLastFilledBuyPrices, xSymbol)
								delete(xLastFilledSellPrices, xSymbol)
								delete(yLastFilledBuyPrices, ySymbol)
								delete(yLastFilledSellPrices, ySymbol)
							}
						} else if yOrder.GetSide() == common.OrderSideSell {
							ySellPrice := yOrder.GetFilledPrice()
							yLastFilledSellPrices[ySymbol] = ySellPrice
							//logger.Debugf("%s set y sell price %f dir x %s buy %f", ySymbol, ySellPrice, xSymbol, xLastFilledBuyPrices[xSymbol])
							if xBuyPrice, ok := xLastFilledBuyPrices[xSymbol]; ok {
								xyRealisedSpread[xSymbol] = (ySellPrice - xBuyPrice) / ySellPrice
								logger.Debugf("%s - %s realised short spread %f", ySymbol, xSymbol, xyRealisedSpread[xSymbol])
								delete(xLastFilledBuyPrices, xSymbol)
								delete(xLastFilledSellPrices, xSymbol)
								delete(yLastFilledBuyPrices, ySymbol)
								delete(yLastFilledSellPrices, ySymbol)
							}
						}
					}
				}
			}
			break
		case spread := <-spreadCh:
			xySpreads[spread.XSymbol] = spread
			break
		case fr := <-xFundingRateCh:
			xFundingRates[fr.GetSymbol()] = fr
			handleUpdateFundingRates()
			break
		case fr := <-yFundingRateCh:
			yFundingRates[fr.GetSymbol()] = fr
			handleUpdateFundingRates()
			break
		case <-influxSaveTimer.C:
			handleSave()
			influxSaveTimer.Reset(
				time.Now().Truncate(
					xyConfig.InternalInflux.SaveInterval,
				).Add(
					xyConfig.InternalInflux.SaveInterval + time.Second*15,
				).Sub(time.Now()),
			)
			break
		case yNewError := <-yNewOrderErrorCh:
			if yNewError.Cancel != nil {
				logger.Debugf("Cancel %v error %v", *yNewError.Cancel, yNewError.Error)
				yOrderSilentTimes[yNewError.Cancel.Symbol] = time.Now().Add(xyConfig.OrderSilent)
			} else if yNewError.New != nil {
				logger.Debugf("New %v error %v", *yNewError.New, yNewError.Error)
				yOrderSilentTimes[yNewError.New.Symbol] = time.Now().Add(xyConfig.OrderSilent)
			}
			break
		case xNewError := <-xNewOrderErrorCh:
			if xNewError.Cancel != nil {
				logger.Debugf("Cancel %v error %v", *xNewError.Cancel, xNewError.Error)
				xOrderSilentTimes[xNewError.Cancel.Symbol] = time.Now().Add(xyConfig.OrderSilent)
			} else if xNewError.New != nil {
				logger.Debugf("New %v error %v", *xNewError.New, xNewError.Error)
				xOrderSilentTimes[xNewError.New.Symbol] = time.Now().Add(xyConfig.OrderSilent)
			}
			break

		case <-xyLoopTimer.C:
			if xSystemStatus == common.SystemStatusReady && ySystemStatus == common.SystemStatusReady {
				if time.Now().Sub(time.Now().Truncate(fundingInterval)) > fundingSilent &&
					time.Now().Truncate(fundingInterval).Add(fundingInterval).Sub(time.Now()) > fundingSilent {
					updateTargetPositionSizes()
					updateXPositions()
					updateYPositions()
				}
			} else {
				if time.Now().Sub(time.Now().Truncate(time.Second*15)) < xyConfig.LoopInterval {
					logger.Debugf(
						"system not ready xSystemStatus %v ySystemStatus %v",
						xSystemStatus, ySystemStatus,
					)
				}
			}
			xyLoopTimer.Reset(
				time.Now().Truncate(
					xyConfig.LoopInterval,
				).Add(
					xyConfig.LoopInterval,
				).Sub(time.Now()),
			)
			break
		}
	}
	logger.Debugf("hedge all positions, and wait 30s")
	updateXPositions()
	updateYPositions()
	<-time.After(time.Second * 30)
}
