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

	if mtConfig.CpuProfile != "" {
		f, err := os.Create(mtConfig.CpuProfile)
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

	mtGlobalCtx, mtGlobalCancel = context.WithCancel(context.Background())
	defer mtGlobalCancel()

	var err error
	err = mExchange.Setup(mtGlobalCtx, mtConfig.MakerExchange)
	if err != nil {
		logger.Debugf("mExchange.Setup(mtGlobalCtx, mtConfig.MakerExchange) error %v", err)
		return
	}
	err = tExchange.Setup(mtGlobalCtx, mtConfig.TakerExchange)
	if err != nil {
		logger.Debugf("tExchange.Setup(mtGlobalCtx, mtConfig.TakerExchange) error %v", err)
		return
	}
	for _, tSymbol := range tSymbols {
		tStepSizes[tSymbol], err = tExchange.GetStepSize(tSymbol)
		if err != nil {
			logger.Debugf("tExchange.GetStepSize(tSymbol) error %v", err)
		}
		tMinNotional[tSymbol], err = tExchange.GetMinNotional(tSymbol)
		if err != nil {
			logger.Debugf("tExchange.GetMinNotional(tSymbol) error %v", err)
		}
	}
	logger.Debugf("taker stepSizes %v", tStepSizes)
	logger.Debugf("taker minNotional %v", tMinNotional)
	for _, mSymbol := range mSymbols {
		mStepSizes[mSymbol], err = mExchange.GetStepSize(mSymbol)
		if err != nil {
			logger.Debugf("mExchange.GetStepSize(mSymbol) error %v", err)
		}
		mTickSizes[mSymbol], err = mExchange.GetTickSize(mSymbol)
		if err != nil {
			logger.Debugf("mExchange.GetTickSize(mSymbol) error %v", err)
		}
		mMinNotional[mSymbol], err = mExchange.GetMinNotional(mSymbol)
		if err != nil {
			logger.Debugf("mExchange.GetMinNotional(mSymbol) error %v", err)
		}
	}
	logger.Debugf("maker tickSizes %v", mTickSizes)
	logger.Debugf("maker stepSizes %v", mStepSizes)
	logger.Debugf("maker minNotional %v", mMinNotional)

	for makerSymbol, makerStepSize := range mStepSizes {
		if takerStepSize, ok := tStepSizes[mtSymbolsMap[makerSymbol]]; !ok {
			logger.Debugf("taker step size not exists for maker %s - %s", makerSymbol, mtSymbolsMap[makerSymbol])
			return
		} else {
			mtStepSizes[makerSymbol] = common.MergedStepSize(makerStepSize, takerStepSize)
		}
	}
	logger.Debugf("merged step sizes: %v", mtStepSizes)

	mtInfluxWriter, err = common.NewInfluxWriter(
		mtGlobalCtx,
		mtConfig.InternalInflux.Address,
		mtConfig.InternalInflux.Username,
		mtConfig.InternalInflux.Password,
		mtConfig.InternalInflux.Database,
		mtConfig.InternalInflux.BatchSize,
	)
	if err != nil {
		logger.Debugf("common.NewInfluxWriter error %v", err)
		return
	}
	defer mtInfluxWriter.Stop()

	mtExternalInfluxWriter, err = common.NewInfluxWriter(
		mtGlobalCtx,
		mtConfig.ExternalInflux.Address,
		mtConfig.ExternalInflux.Username,
		mtConfig.ExternalInflux.Password,
		mtConfig.ExternalInflux.Database,
		mtConfig.ExternalInflux.BatchSize,
	)
	if err != nil {
		logger.Debugf("common.NewInfluxWriter error %v", err)
		return
	}
	defer mtExternalInfluxWriter.Stop()

	internalInfluxSaveTimer := time.NewTimer(
		time.Now().Truncate(
			mtConfig.InternalInflux.SaveInterval,
		).Add(
			mtConfig.InternalInflux.SaveInterval * 3,
		).Sub(time.Now()),
	)
	externalInfluxSaveTimer := time.NewTimer(
		time.Now().Truncate(
			mtConfig.ExternalInflux.SaveInterval,
		).Add(
			mtConfig.ExternalInflux.SaveInterval * 3,
		).Sub(time.Now()),
	)
	mtLoopTimer = time.NewTimer(time.Second) //先等1分钟
	defer internalInfluxSaveTimer.Stop()
	defer mtLoopTimer.Stop()
	defer externalInfluxSaveTimer.Stop()

	makerPositionChMap := make(map[string]chan common.Position)
	makerOrderChMap := make(map[string]chan common.Order)
	makerFundingRateChMap := make(map[string]chan common.FundingRate)
	makerDepthChMap := make(map[string]chan common.Depth)
	makerNewOrderErrorChMap := make(map[string]chan common.OrderError)
	for _, makerSymbol := range mSymbols {
		makerPositionChMap[makerSymbol] = mPositionCh
		makerOrderChMap[makerSymbol] = mOrderCh
		makerFundingRateChMap[makerSymbol] = mFundingRateCh
		makerDepthChMap[makerSymbol] = make(chan common.Depth, 100)
		mOrderRequestChMap[makerSymbol] = make(chan common.OrderRequest, 2)
		makerNewOrderErrorChMap[makerSymbol] = mNewOrderErrorCh
	}
	go mExchange.StreamBasic(
		mtGlobalCtx,
		mSystemStatusCh,
		mAccountCh,
		makerPositionChMap,
		makerOrderChMap,
	)
	go mExchange.StreamFundingRate(
		mtGlobalCtx,
		makerFundingRateChMap,
		mtConfig.BatchSize,
	)
	go mExchange.StreamDepth(
		mtGlobalCtx,
		makerDepthChMap,
		mtConfig.BatchSize,
	)
	go mExchange.WatchOrders(
		mtGlobalCtx,
		mOrderRequestChMap,
		makerOrderChMap,
		makerNewOrderErrorChMap,
	)

	takerPositionChMap := make(map[string]chan common.Position)
	takerOrderChMap := make(map[string]chan common.Order)
	takerFundingRateChMap := make(map[string]chan common.FundingRate)
	takerDepthChMap := make(map[string]chan common.Depth)
	takerNewOrderErrorChMap := make(map[string]chan common.OrderError)
	for _, takerSymbol := range tSymbols {
		takerPositionChMap[takerSymbol] = tPositionCh
		takerOrderChMap[takerSymbol] = tOrderCh
		takerFundingRateChMap[takerSymbol] = tFundingRateCh
		takerDepthChMap[takerSymbol] = make(chan common.Depth, 100)
		tOrderRequestChMap[takerSymbol] = make(chan common.OrderRequest, 2)
		takerNewOrderErrorChMap[takerSymbol] = tNewOrderErrorCh
	}
	go tExchange.StreamBasic(
		mtGlobalCtx,
		tSystemStatusCh,
		tAccountCh,
		takerPositionChMap,
		takerOrderChMap,
	)
	go tExchange.StreamFundingRate(
		mtGlobalCtx,
		takerFundingRateChMap,
		mtConfig.BatchSize,
	)
	go tExchange.StreamDepth(
		mtGlobalCtx,
		takerDepthChMap,
		mtConfig.BatchSize,
	)
	go tExchange.WatchOrders(
		mtGlobalCtx,
		tOrderRequestChMap,
		takerOrderChMap,
		takerNewOrderErrorChMap,
	)

	spreadReportCh := make(chan common.SpreadReport, 10000)
	go reportsSaveLoop(
		mtGlobalCtx,
		mtInfluxWriter,
		mtConfig.InternalInflux,
		spreadReportCh,
	)

	spreadCh := make(chan *common.MakerTakerSpread, len(mSymbols)*100)
	for makerSymbol, takerSymbol := range mtConfig.MakerTakerPairs {
		go watchMakerTakerSpread(
			mtGlobalCtx,
			makerSymbol, takerSymbol,
			mtConfig.DepthMakerImpact,
			mtConfig.DepthTakerImpact,
			mtConfig.DepthMakerDecay,
			mtConfig.DepthMakerBias,
			mtConfig.DepthTakerDecay,
			mtConfig.DepthTakerBias,
			mtConfig.DepthTimeDeltaMin,
			mtConfig.DepthTimeDeltaMax,
			mtConfig.DepthMaxAgeDiffBias,
			mtConfig.ReportCount,
			mtConfig.SpreadLookback,
			mtConfig.DepthDirLookback,
			makerDepthChMap[makerSymbol],
			takerDepthChMap[takerSymbol],
			spreadReportCh,
			spreadCh,
		)
	}

	if mtConfig.CpuProfile != "" {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			sig := <-sigs
			logger.Debugf("catch exit signal %v", sig)
			mtGlobalCancel()
		}()
	}

	if !mtConfig.DryRun {
		go func() {
			for _, makerSymbol := range mSymbols {
				select {
				case <-mtGlobalCtx.Done():
					return
				case <-time.After(mtConfig.RequestInterval):
					logger.Debugf("initial cancel all %s", makerSymbol)
					select {
					case <-mtGlobalCtx.Done():
						return
					case mOrderRequestChMap[makerSymbol] <- common.OrderRequest{
						Cancel: &common.CancelOrderParam{
							Symbol: makerSymbol,
						},
					}:
					}
				}
			}
		}()

		go func() {
			for _, takerSymbol := range tSymbols {
				select {
				case <-mtGlobalCtx.Done():
					return
				case <-time.After(mtConfig.RequestInterval):
					logger.Debugf("initial cancel all %s", takerSymbol)
					select {
					case <-mtGlobalCtx.Done():
						return
					case tOrderRequestChMap[takerSymbol] <- common.OrderRequest{
						Cancel: &common.CancelOrderParam{
							Symbol: takerSymbol,
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
		case <-mtGlobalCtx.Done():
			logger.Debugf("global ctx done, exit main loop")
			return
		case <-mExchange.Done():
			logger.Debugf("maker exchange done, exit main loop")
			return
		case <-tExchange.Done():
			logger.Debugf("taker exchange done, exit main loop")
			return
		case mSystemStatus = <-mSystemStatusCh:
			if mSystemStatus != common.SystemStatusReady {
				logger.Debugf("mSystemStatus %v", mSystemStatus)
			}
			break
		case tSystemStatus = <-tSystemStatusCh:
			if tSystemStatus != common.SystemStatusReady {
				logger.Debugf("tSystemStatus %v", tSystemStatus)
			}
			break
		case nextPos := <-mPositionCh:
			//logger.Debugf("maker position %s %v %f %f", nextPos.GetSymbol(), nextPos.GetTime(), nextPos.GetPrice(), nextPos.GetSize())
			if prevPos, ok := mPositions[nextPos.GetSymbol()]; ok {
				if nextPos.GetTime().Sub(prevPos.GetTime()) >= 0 {
					mPositions[nextPos.GetSymbol()] = nextPos
					mPositionsUpdateTimes[nextPos.GetSymbol()] = nextPos.GetTime()
					if prevPos.GetSize() != nextPos.GetSize() {
						logger.Debugf("%s maker position change %f -> %f", nextPos.GetSymbol(), prevPos.GetSize(), nextPos.GetSize())

						changeSize := -(nextPos.GetSize() - prevPos.GetSize())

						if mtConfig.HedgeInstantly {
							tOrderSilentTimes[mtSymbolsMap[nextPos.GetSymbol()]] = time.Now()
							mtLoopTimer.Reset(time.Nanosecond)
						} else {
							if spread, ok := mtSpreads[nextPos.GetSymbol()]; ok && ((changeSize < 0 && spread.TakerDir > 0) ||
								(changeSize > 0 && spread.TakerDir < 0)) {
								if changeSize < 0 {
									tHedgeMarkPrices[mtSymbolsMap[nextPos.GetSymbol()]] = spread.TakerDepth.BestBidPrice
								} else {
									tHedgeMarkPrices[mtSymbolsMap[nextPos.GetSymbol()]] = spread.TakerDepth.BestAskPrice
								}
								logger.Debugf(
									"%s taker change size %f dir %f mark price %f",
									mtSymbolsMap[nextPos.GetSymbol()], changeSize, spread.TakerDir,
									tHedgeMarkPrices[mtSymbolsMap[nextPos.GetSymbol()]],
								)
								tOrderSilentTimes[mtSymbolsMap[nextPos.GetSymbol()]] = time.Now().Add(mtConfig.HedgeCheckInterval)
								mOrderSilentTimes[nextPos.GetSymbol()] = time.Now().Add(mtConfig.HedgeCheckInterval)
								tPositionsUpdateTimes[mtSymbolsMap[nextPos.GetSymbol()]] = time.Now()
								mPositionsUpdateTimes[nextPos.GetSymbol()] = time.Now()
							} else {
								if ok {
									logger.Debugf("%s taker dir %f", mtSymbolsMap[nextPos.GetSymbol()], spread.TakerDir)
								}
								tOrderSilentTimes[mtSymbolsMap[nextPos.GetSymbol()]] = time.Now()
								mtLoopTimer.Reset(time.Nanosecond)
							}
						}

						if nextPos.GetSize() != 0 {
							mEnterSilentTimes[nextPos.GetSymbol()] = time.Now().Add(mtConfig.EnterSilent)
						}
					}
				}
			} else {
				mPositions[nextPos.GetSymbol()] = nextPos
				mPositionsUpdateTimes[nextPos.GetSymbol()] = nextPos.GetTime()
			}
			break
		case mAccount = <-mAccountCh:
			break
		case nextPos := <-tPositionCh:
			//logger.Debugf("taker position %s %v %f %f", nextPos.GetSymbol(), nextPos.GetTime(), nextPos.GetPrice(), nextPos.GetSize())
			if prevPos, ok := tPositions[nextPos.GetSymbol()]; ok {
				if nextPos.GetTime().Sub(prevPos.GetTime()) >= 0 {
					tPositions[nextPos.GetSymbol()] = nextPos
					tPositionsUpdateTimes[nextPos.GetSymbol()] = nextPos.GetTime()
					if prevPos.GetSize() != nextPos.GetSize() {
						logger.Debugf("%s taker position change %f -> %f", nextPos.GetSymbol(),prevPos.GetSize(), nextPos.GetSize())
					}
				}
			} else {
				tPositions[nextPos.GetSymbol()] = nextPos
				tPositionsUpdateTimes[nextPos.GetSymbol()] = nextPos.GetTime()
			}
			break
		case tAccount = <-tAccountCh:
			break
		case makerOrder := <-mOrderCh:
			if makerOrder.GetStatus() == common.OrderStatusExpired ||
				makerOrder.GetStatus() == common.OrderStatusReject ||
				makerOrder.GetStatus() == common.OrderStatusCancelled ||
				makerOrder.GetStatus() == common.OrderStatusFilled {
				if openOrder, ok := mOpenOrders[makerOrder.GetSymbol()]; ok && openOrder.ClientID == makerOrder.GetClientID() {
					delete(mOpenOrders, makerOrder.GetSymbol())
				}
				if makerOrder.GetStatus() == common.OrderStatusFilled {
					logger.Debugf(
						"MAKER ORDER FILLED %s SIDE %s TRADE SIZE %v TRADE PRICE %f",
						makerOrder.GetSymbol(), makerOrder.GetSide(), makerOrder.GetFilledSize(), makerOrder.GetFilledPrice(),
					)

					if mtConfig.HedgeInstantly {
						tOrderSilentTimes[mtSymbolsMap[makerOrder.GetSymbol()]] = time.Now()
						mtLoopTimer.Reset(time.Nanosecond)
					} else {
						if spread, ok := mtSpreads[makerOrder.GetSymbol()]; ok && ((makerOrder.GetSide() == common.OrderSideBuy && spread.TakerDir > 0) ||
							(makerOrder.GetSide() == common.OrderSideSell && spread.TakerDir < 0)) {
							if makerOrder.GetSide() == common.OrderSideBuy {
								tHedgeMarkPrices[mtSymbolsMap[makerOrder.GetSymbol()]] = spread.TakerDepth.BestBidPrice
								logger.Debugf(
									"%s taker change size %f dir %f mark price %f",
									mtSymbolsMap[makerOrder.GetSymbol()], -makerOrder.GetSize(), spread.TakerDir,
									tHedgeMarkPrices[mtSymbolsMap[makerOrder.GetSymbol()]],
								)
							} else {
								tHedgeMarkPrices[mtSymbolsMap[makerOrder.GetSymbol()]] = spread.TakerDepth.BestAskPrice
								logger.Debugf(
									"%s taker change size %f dir %f mark price %f",
									mtSymbolsMap[makerOrder.GetSymbol()], makerOrder.GetSize(), spread.TakerDir,
									tHedgeMarkPrices[mtSymbolsMap[makerOrder.GetSymbol()]],
								)
							}
							tOrderSilentTimes[mtSymbolsMap[makerOrder.GetSymbol()]] = time.Now().Add(mtConfig.HedgeCheckInterval)
							mOrderSilentTimes[makerOrder.GetSymbol()] = time.Now().Add(mtConfig.HedgeCheckInterval)
							tPositionsUpdateTimes[mtSymbolsMap[makerOrder.GetSymbol()]] = time.Now()
							mPositionsUpdateTimes[makerOrder.GetSymbol()] = time.Now()
						} else {
							if ok {
								logger.Debugf("%s taker dir %f", mtSymbolsMap[makerOrder.GetSymbol()], spread.TakerDir)
							}
							tOrderSilentTimes[mtSymbolsMap[makerOrder.GetSymbol()]] = time.Now()
							mtLoopTimer.Reset(time.Nanosecond)
						}
					}

					if makerOrder.GetSide() == common.OrderSideSell {
						mLastFilledSellPrices[makerOrder.GetSymbol()] = makerOrder.GetFilledPrice()
					} else if makerOrder.GetSide() == common.OrderSideBuy {
						mLastFilledBuyPrices[makerOrder.GetSymbol()] = makerOrder.GetFilledPrice()
					}
					mEnterSilentTimes[makerOrder.GetSymbol()] = time.Now().Add(mtConfig.EnterSilent)
				} else {
					logger.Debugf("MAKER ORDER %s %s", makerOrder.GetSymbol(), makerOrder.GetStatus())
					logger.Debugf("MAKER WS ORDER CANCELED %v ", makerOrder)
					mOrderSilentTimes[makerOrder.GetSymbol()] = time.Now().Add(time.Second)
					mPositionsUpdateTimes[makerOrder.GetSymbol()] = time.Now()
				}
			}
			break
		case takerOrder := <-tOrderCh:
			if takerOrder.GetStatus() == common.OrderStatusExpired ||
				takerOrder.GetStatus() == common.OrderStatusReject ||
				takerOrder.GetStatus() == common.OrderStatusCancelled ||
				takerOrder.GetStatus() == common.OrderStatusFilled {
				if takerOrder.GetStatus() != common.OrderStatusFilled {
					logger.Debugf("TAKER ORDER %s %s", takerOrder.GetSymbol(), takerOrder.GetStatus())
					tOrderSilentTimes[takerOrder.GetSymbol()] = time.Now().Add(time.Second)
					tPositionsUpdateTimes[takerOrder.GetSymbol()] = time.Unix(0, 0)
				} else {
					logger.Debugf("TAKER ORDER %s %s %f %f", takerOrder.GetSymbol(), takerOrder.GetStatus(), takerOrder.GetFilledSize(), takerOrder.GetFilledPrice())
					if makerSymbol, ok := tmSymbolsMap[takerOrder.GetSymbol()]; ok {
						if takerOrder.GetSide() == common.OrderSideSell {
							if makerPrice, ok := mLastFilledBuyPrices[makerSymbol]; ok {
								mtRealisedSpread[makerSymbol] = (takerOrder.GetFilledPrice() - makerPrice) / makerPrice
								logger.Debugf("%s REALISED SHORT SPREAD %f", makerSymbol, mtRealisedSpread[makerSymbol])
								delete(mLastFilledBuyPrices, makerSymbol)
							}
						} else if takerOrder.GetSide() == common.OrderSideBuy {
							if makerPrice, ok := mLastFilledSellPrices[makerSymbol]; ok {
								mtRealisedSpread[makerSymbol] = (takerOrder.GetFilledPrice() - makerPrice) / makerPrice
								logger.Debugf("%s REALISED LONG SPREAD %f", makerSymbol, mtRealisedSpread[makerSymbol])
								delete(mLastFilledSellPrices, makerSymbol)
							}
						}
					}
				}

			}
			break
		case spread := <-spreadCh:
			mtSpreads[spread.MakerSymbol] = spread
			break
		case fr := <-mFundingRateCh:
			mFundingRates[fr.GetSymbol()] = fr
			handleUpdateFundingRates()
			break
		case fr := <-tFundingRateCh:
			tFundingRates[fr.GetSymbol()] = fr
			handleUpdateFundingRates()
			break
		case <-internalInfluxSaveTimer.C:
			handleInternalSave()
			internalInfluxSaveTimer.Reset(
				time.Now().Truncate(
					mtConfig.InternalInflux.SaveInterval,
				).Add(
					mtConfig.InternalInflux.SaveInterval+time.Second*15,
				).Sub(time.Now()),
			)
			break
		case <-externalInfluxSaveTimer.C:
			handleExternalInfluxSave()
			externalInfluxSaveTimer.Reset(
				time.Now().Truncate(
					mtConfig.ExternalInflux.SaveInterval,
				).Add(
					mtConfig.ExternalInflux.SaveInterval+time.Second*15,
				).Sub(time.Now()),
			)
			break
		case takerNewError := <-tNewOrderErrorCh:
			if takerNewError.Cancel != nil {
				tOrderSilentTimes[takerNewError.Cancel.Symbol] = time.Now().Add(mtConfig.OrderSilent)
			} else if takerNewError.New != nil {
				tOrderSilentTimes[takerNewError.New.Symbol] = time.Now().Add(mtConfig.OrderSilent)
			}
			break
		case makerNewError := <-mNewOrderErrorCh:
			if makerNewError.Cancel != nil {
				mOrderSilentTimes[makerNewError.Cancel.Symbol] = time.Now().Add(mtConfig.OrderSilent)
			} else if makerNewError.New != nil {
				mOrderSilentTimes[makerNewError.New.Symbol] = time.Now().Add(mtConfig.OrderSilent)
			}
			break

		case <-mtLoopTimer.C:
			if mSystemStatus == common.SystemStatusReady && tSystemStatus == common.SystemStatusReady {
				updateTakerPositions()
				if time.Now().Sub(time.Now().Truncate(fundingInterval)) > fundingSilent &&
					time.Now().Truncate(fundingInterval).Add(fundingInterval).Sub(time.Now()) > fundingSilent {
					updateMakerOldOrders()
					updateMakerNewOrders()
				} else {
					if len(mOpenOrders) > 0 && !mtConfig.DryRun {
						cancelAllMakerOpenOrders()
					}
				}
			} else {
				if time.Now().Sub(time.Now().Truncate(time.Second*15)) < mtConfig.LoopInterval {
					logger.Debugf(
						"system not ready mSystemStatus %v tSystemStatus %v",
						mSystemStatus, tSystemStatus,
					)
				}
				if len(mOpenOrders) > 0 && !mtConfig.DryRun {
					cancelAllMakerOpenOrders()
				}
			}
			mtLoopTimer.Reset(
				time.Now().Truncate(
					mtConfig.LoopInterval,
				).Add(
					mtConfig.LoopInterval,
				).Sub(time.Now()),
			)
			break
		}
	}
}
