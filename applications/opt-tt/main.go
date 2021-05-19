package main

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
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
		yMinNotional[ySymbol], err = yExchange.GetMinNotional(ySymbol)
		if err != nil {
			logger.Debugf("yExchange.GetMinNotional(ySymbol) error %v", err)
		}
	}
	logger.Debugf("y stepSizes %v", yStepSizes)
	logger.Debugf("y minNotional %v", yMinNotional)
	for _, xSymbol := range xSymbols {
		xStepSizes[xSymbol], err = xExchange.GetStepSize(xSymbol)
		if err != nil {
			logger.Debugf("xExchange.GetStepSize(xSymbol) error %v", err)
		}
		xMinNotional[xSymbol], err = xExchange.GetMinNotional(xSymbol)
		if err != nil {
			logger.Debugf("xExchange.GetMinNotional(xSymbol) error %v", err)
		}
	}
	logger.Debugf("x stepSizes %v", xStepSizes)
	logger.Debugf("x minNotional %v", xMinNotional)

	for xSymbol, xStepSize := range xStepSizes {
		if yStepSize, ok := yStepSizes[xySymbolsMap[xSymbol]]; !ok {
			logger.Debugf("y step size not exists for %s - %s", xSymbol, xySymbolsMap[xSymbol])
			return
		} else {
			xyStepSizes[xSymbol] = common.MergedStepSize(xStepSize, yStepSize)
		}
	}
	logger.Debugf("merged step sizes: %v", xyStepSizes)

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

	internalInfluxSaveTimer := time.NewTimer(
		time.Now().Truncate(
			xyConfig.InternalInflux.SaveInterval,
		).Add(
			xyConfig.InternalInflux.SaveInterval * 3,
		).Sub(time.Now()),
	)
	externalInfluxSaveTimer := time.NewTimer(
		time.Now().Truncate(
			xyConfig.ExternalInflux.SaveInterval,
		).Add(
			xyConfig.ExternalInflux.SaveInterval * 3,
		).Sub(time.Now()),
	)
	defer internalInfluxSaveTimer.Stop()
	defer externalInfluxSaveTimer.Stop()

	xyLoopTimer = time.NewTimer(time.Second)
	defer xyLoopTimer.Stop()
	xyDirResetTimer = time.NewTimer(time.Second)
	defer xyDirResetTimer.Stop()

	xPositionChMap := make(map[string]chan common.Position)
	xOrderChMap := make(map[string]chan common.Order)
	xFundingRateChMap := make(map[string]chan common.FundingRate)
	xDepthChMap := make(map[string]chan common.Depth)
	xNewOrderErrorChMap := make(map[string]chan common.OrderError)
	for _, xSymbol := range xSymbols {
		xPositionChMap[xSymbol] = xPositionCh
		xOrderChMap[xSymbol] = xOrderCh
		xFundingRateChMap[xSymbol] = xFundingRateCh
		xDepthChMap[xSymbol] = make(chan common.Depth, 200)
		xOrderRequestChMap[xSymbol] = make(chan common.OrderRequest, 200)
		xNewOrderErrorChMap[xSymbol] = xNewOrderErrorCh
	}
	go xExchange.StreamBasic(
		xyGlobalCtx,
		xSystemStatusCh,
		xAccountCh,
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
	for _, ySymbol := range ySymbols {
		yPositionChMap[ySymbol] = yPositionCh
		yOrderChMap[ySymbol] = yOrderCh
		yFundingRateChMap[ySymbol] = yFundingRateCh
		yDepthChMap[ySymbol] = make(chan common.Depth, 200)
		yOrderRequestChMap[ySymbol] = make(chan common.OrderRequest, 200)
		yNewOrderErrorChMap[ySymbol] = yNewOrderErrorCh
	}
	go yExchange.StreamBasic(
		xyGlobalCtx,
		ySystemStatusCh,
		yAccountCh,
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

	spreadReportCh := make(chan common.SpreadReport, 10000)
	go reportsSaveLoop(
		xyGlobalCtx,
		xyInfluxWriter,
		xyConfig.InternalInflux,
		spreadReportCh,
	)

	spreadCh := make(chan *common.MakerTakerSpread, len(xSymbols)*100)
	for xSymbol, ySymbol := range xyConfig.XYPairs {
		go watchXYSpread(
			xyGlobalCtx,
			xSymbol, ySymbol,
			xyConfig.DepthMakerImpact,
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
			xyConfig.DepthDirLookback,
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

	if !xyConfig.DryRun {
		go func() {
			for _, xSymbol := range xSymbols {
				select {
				case <-xyGlobalCtx.Done():
					return
				case <-time.After(xyConfig.RequestInterval):
					logger.Debugf("initial cancel all %s", xSymbol)
					select {
					case <-xyGlobalCtx.Done():
						return
					case xOrderRequestChMap[xSymbol] <- common.OrderRequest{
						Cancel: &common.CancelOrderParam{
							Symbol: xSymbol,
						},
					}:
					}
				}
			}
		}()
	}

	logger.Debugf("start main loop")
	fundingInterval := time.Hour * 8
	fundingSilent := time.Minute * 5
	for {
		select {
		case <-xyGlobalCtx.Done():
			logger.Debugf("global ctx done, exit main loop")
			return
		case <-xExchange.Done():
			logger.Debugf("x exchange done, exit main loop")
			return
		case <-yExchange.Done():
			logger.Debugf("y exchange done, exit main loop")
			return
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
		case <-xyDirResetTimer.C:
			for xSymbol := range xySymbolsMap {
				if time.Now().Sub(xyEnterTimes[xSymbol]) < 0 && xyEnterTradeOrders[xSymbol] != EnterTradeOrderUnknown{
					continue
				}
				if spread, ok := xySpreads[xSymbol]; ok {
					xyMergedDirs[xSymbol]  = spread.XDir*xyConfig.XYDirRatio + spread.YDir*(1.0 - xyConfig.XYDirRatio)
				}
			}
		case nextPos := <-xPositionCh:
			//logger.Debugf("x position %s %v %f %f", nextPos.GetSymbol(), nextPos.GetTime(), nextPos.GetPrice(), nextPos.GetSize())
			if prevPos, ok := xPositions[nextPos.GetSymbol()]; ok {
				if nextPos.GetTime().Sub(prevPos.GetTime()) >= 0 {
					xPositions[nextPos.GetSymbol()] = nextPos
					xPositionsUpdateTimes[nextPos.GetSymbol()] = nextPos.GetTime()
					if prevPos.GetSize() != nextPos.GetSize() {
						logger.Debugf("%s x position change %f -> %f", nextPos.GetSymbol(), prevPos.GetSize(), nextPos.GetSize())

						changeSize := -(nextPos.GetSize() - prevPos.GetSize())

						if xyConfig.HedgeInstantly {
							yOrderSilentTimes[xySymbolsMap[nextPos.GetSymbol()]] = time.Now()
							xyLoopTimer.Reset(time.Nanosecond)
						} else {
							if spread, ok := xySpreads[nextPos.GetSymbol()]; ok && ((changeSize < 0 && spread.TakerDir > 0) ||
								(changeSize > 0 && spread.TakerDir < 0)) {
								if changeSize < 0 {
									yHedgeMarkPrices[xySymbolsMap[nextPos.GetSymbol()]] = spread.TakerDepth.BestBidPrice
								} else {
									yHedgeMarkPrices[xySymbolsMap[nextPos.GetSymbol()]] = spread.TakerDepth.BestAskPrice
								}
								logger.Debugf(
									"%s y change size %f dir %f mark price %f",
									xySymbolsMap[nextPos.GetSymbol()], changeSize, spread.TakerDir,
									yHedgeMarkPrices[xySymbolsMap[nextPos.GetSymbol()]],
								)
								yOrderSilentTimes[xySymbolsMap[nextPos.GetSymbol()]] = time.Now().Add(xyConfig.HedgeCheckInterval)
								xOrderSilentTimes[nextPos.GetSymbol()] = time.Now().Add(xyConfig.HedgeCheckInterval)
								yPositionsUpdateTimes[xySymbolsMap[nextPos.GetSymbol()]] = time.Now()
								xPositionsUpdateTimes[nextPos.GetSymbol()] = time.Now()
							} else {
								if ok {
									logger.Debugf("%s y dir %f", xySymbolsMap[nextPos.GetSymbol()], spread.TakerDir)
								}
								yOrderSilentTimes[xySymbolsMap[nextPos.GetSymbol()]] = time.Now()
								xyLoopTimer.Reset(time.Nanosecond)
							}
						}

						if nextPos.GetSize() != 0 {
							xEnterSilentTimes[nextPos.GetSymbol()] = time.Now().Add(xyConfig.EnterSilent)
						}
					}
				}
			} else {
				xPositions[nextPos.GetSymbol()] = nextPos
				xPositionsUpdateTimes[nextPos.GetSymbol()] = nextPos.GetTime()
			}
			break
		case xAccount = <-xAccountCh:
			break
		case nextPos := <-yPositionCh:
			//logger.Debugf("y position %s %v %f %f", nextPos.GetSymbol(), nextPos.GetTime(), nextPos.GetPrice(), nextPos.GetSize())
			if prevPos, ok := yPositions[nextPos.GetSymbol()]; ok {
				if nextPos.GetTime().Sub(prevPos.GetTime()) >= 0 {
					yPositions[nextPos.GetSymbol()] = nextPos
					yPositionsUpdateTimes[nextPos.GetSymbol()] = nextPos.GetTime()
					if prevPos.GetSize() != nextPos.GetSize() {
						logger.Debugf("%s y position change %f -> %f", nextPos.GetSymbol(), prevPos.GetSize(), nextPos.GetSize())
					}
				}
			} else {
				yPositions[nextPos.GetSymbol()] = nextPos
				yPositionsUpdateTimes[nextPos.GetSymbol()] = nextPos.GetTime()
			}
			break
		case yAccount = <-yAccountCh:
			break
		case xOrder := <-xOrderCh:
			if xOrder.GetStatus() == common.OrderStatusExpired ||
				xOrder.GetStatus() == common.OrderStatusReject ||
				xOrder.GetStatus() == common.OrderStatusCancelled ||
				xOrder.GetStatus() == common.OrderStatusFilled {
				if openOrder, ok := xOpenOrders[xOrder.GetSymbol()]; ok && openOrder.ClientID == xOrder.GetClientID() {
					delete(xOpenOrders, xOrder.GetSymbol())
				}
				if xOrder.GetStatus() == common.OrderStatusFilled {
					logger.Debugf(
						"MAKER ORDER FILLED %s SIDE %s TRADE SIZE %v TRADE PRICE %f",
						xOrder.GetSymbol(), xOrder.GetSide(), xOrder.GetFilledSize(), xOrder.GetFilledPrice(),
					)

					if xyConfig.HedgeInstantly {
						yOrderSilentTimes[xySymbolsMap[xOrder.GetSymbol()]] = time.Now()
						xyLoopTimer.Reset(time.Nanosecond)
					} else {
						if spread, ok := xySpreads[xOrder.GetSymbol()]; ok && ((xOrder.GetSide() == common.OrderSideBuy && spread.TakerDir > 0) ||
							(xOrder.GetSide() == common.OrderSideSell && spread.TakerDir < 0)) {
							if xOrder.GetSide() == common.OrderSideBuy {
								yHedgeMarkPrices[xySymbolsMap[xOrder.GetSymbol()]] = spread.TakerDepth.BestBidPrice
								logger.Debugf(
									"%s y change size %f dir %f mark price %f",
									xySymbolsMap[xOrder.GetSymbol()], -xOrder.GetSize(), spread.TakerDir,
									yHedgeMarkPrices[xySymbolsMap[xOrder.GetSymbol()]],
								)
							} else {
								yHedgeMarkPrices[xySymbolsMap[xOrder.GetSymbol()]] = spread.TakerDepth.BestAskPrice
								logger.Debugf(
									"%s y change size %f dir %f mark price %f",
									xySymbolsMap[xOrder.GetSymbol()], xOrder.GetSize(), spread.TakerDir,
									yHedgeMarkPrices[xySymbolsMap[xOrder.GetSymbol()]],
								)
							}
							yOrderSilentTimes[xySymbolsMap[xOrder.GetSymbol()]] = time.Now().Add(xyConfig.HedgeCheckInterval)
							xOrderSilentTimes[xOrder.GetSymbol()] = time.Now().Add(xyConfig.HedgeCheckInterval)
							yPositionsUpdateTimes[xySymbolsMap[xOrder.GetSymbol()]] = time.Now()
							xPositionsUpdateTimes[xOrder.GetSymbol()] = time.Now()
						} else {
							if ok {
								logger.Debugf("%s y dir %f", xySymbolsMap[xOrder.GetSymbol()], spread.TakerDir)
							}
							yOrderSilentTimes[xySymbolsMap[xOrder.GetSymbol()]] = time.Now()
							xyLoopTimer.Reset(time.Nanosecond)
						}
					}

					if xOrder.GetSide() == common.OrderSideSell {
						xLastFilledSellPrices[xOrder.GetSymbol()] = xOrder.GetFilledPrice()
					} else if xOrder.GetSide() == common.OrderSideBuy {
						xLastFilledBuyPrices[xOrder.GetSymbol()] = xOrder.GetFilledPrice()
					}
					xEnterSilentTimes[xOrder.GetSymbol()] = time.Now().Add(xyConfig.EnterSilent)
				} else {
					logger.Debugf("MAKER ORDER %s %s", xOrder.GetSymbol(), xOrder.GetStatus())
					logger.Debugf("MAKER WS ORDER CANCELED %v ", xOrder)
					xOrderSilentTimes[xOrder.GetSymbol()] = time.Now().Add(time.Second)
					xPositionsUpdateTimes[xOrder.GetSymbol()] = time.Now()
				}
			}
			break
		case yOrder := <-yOrderCh:
			if yOrder.GetStatus() == common.OrderStatusExpired ||
				yOrder.GetStatus() == common.OrderStatusReject ||
				yOrder.GetStatus() == common.OrderStatusCancelled ||
				yOrder.GetStatus() == common.OrderStatusFilled {
				if yOrder.GetStatus() != common.OrderStatusFilled {
					logger.Debugf("TAKER ORDER %s %s", yOrder.GetSymbol(), yOrder.GetStatus())
					yOrderSilentTimes[yOrder.GetSymbol()] = time.Now().Add(time.Second)
					yPositionsUpdateTimes[yOrder.GetSymbol()] = time.Unix(0, 0)
				} else {
					logger.Debugf("TAKER ORDER %s %s %f %f", yOrder.GetSymbol(), yOrder.GetStatus(), yOrder.GetFilledSize(), yOrder.GetFilledPrice())
					if xSymbol, ok := yxSymbolsMap[yOrder.GetSymbol()]; ok {
						if yOrder.GetSide() == common.OrderSideSell {
							if xPrice, ok := xLastFilledBuyPrices[xSymbol]; ok {
								xyRealisedSpread[xSymbol] = (yOrder.GetFilledPrice() - xPrice) / xPrice
								logger.Debugf("%s REALISED SHORT SPREAD %f", xSymbol, xyRealisedSpread[xSymbol])
								delete(xLastFilledBuyPrices, xSymbol)
							}
						} else if yOrder.GetSide() == common.OrderSideBuy {
							if xPrice, ok := xLastFilledSellPrices[xSymbol]; ok {
								xyRealisedSpread[xSymbol] = (yOrder.GetFilledPrice() - xPrice) / xPrice
								logger.Debugf("%s REALISED LONG SPREAD %f", xSymbol, xyRealisedSpread[xSymbol])
								delete(xLastFilledSellPrices, xSymbol)
							}
						}
					}
				}

			}
			break
		case spread := <-spreadCh:
			xySpreads[spread.MakerSymbol] = spread
			break
		case fr := <-xFundingRateCh:
			xFundingRates[fr.GetSymbol()] = fr
			handleUpdateFundingRates()
			break
		case fr := <-yFundingRateCh:
			yFundingRates[fr.GetSymbol()] = fr
			handleUpdateFundingRates()
			break
		case <-internalInfluxSaveTimer.C:
			handleInternalSave()
			internalInfluxSaveTimer.Reset(
				time.Now().Truncate(
					xyConfig.InternalInflux.SaveInterval,
				).Add(
					xyConfig.InternalInflux.SaveInterval + time.Second*15,
				).Sub(time.Now()),
			)
			break
		case <-externalInfluxSaveTimer.C:
			handleExternalInfluxSave()
			externalInfluxSaveTimer.Reset(
				time.Now().Truncate(
					xyConfig.ExternalInflux.SaveInterval,
				).Add(
					xyConfig.ExternalInflux.SaveInterval + time.Second*15,
				).Sub(time.Now()),
			)
			break
		case yNewError := <-yNewOrderErrorCh:
			if yNewError.Cancel != nil {
				yOrderSilentTimes[yNewError.Cancel.Symbol] = time.Now().Add(xyConfig.OrderSilent)
			} else if yNewError.New != nil {
				yOrderSilentTimes[yNewError.New.Symbol] = time.Now().Add(xyConfig.OrderSilent)
			}
			break
		case xNewError := <-xNewOrderErrorCh:
			if xNewError.Cancel != nil {
				xOrderSilentTimes[xNewError.Cancel.Symbol] = time.Now().Add(xyConfig.OrderSilent)
			} else if xNewError.New != nil {
				xOrderSilentTimes[xNewError.New.Symbol] = time.Now().Add(xyConfig.OrderSilent)
			}
			break

		case <-xyLoopTimer.C:
			if xSystemStatus == common.SystemStatusReady && ySystemStatus == common.SystemStatusReady {
				updateTakerPositions()
				if time.Now().Sub(time.Now().Truncate(fundingInterval)) > fundingSilent &&
					time.Now().Truncate(fundingInterval).Add(fundingInterval).Sub(time.Now()) > fundingSilent {
					updateMakerOldOrders()
					updateTargetPositionSizes()
				} else {
					if len(xOpenOrders) > 0 && !xyConfig.DryRun {
						cancelAllMakerOpenOrders()
					}
				}
			} else {
				if time.Now().Sub(time.Now().Truncate(time.Second*15)) < xyConfig.LoopInterval {
					logger.Debugf(
						"system not ready xSystemStatus %v ySystemStatus %v",
						xSystemStatus, ySystemStatus,
					)
				}
				if len(xOpenOrders) > 0 && !xyConfig.DryRun {
					cancelAllMakerOpenOrders()
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
}
