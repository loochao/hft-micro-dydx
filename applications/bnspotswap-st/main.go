package main

import (
	"context"
	"github.com/geometrybase/hft-micro/bnspot"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"
	"time"
)

func main() {

	if *mtConfig.CpuProfile != "" {
		f, err := os.Create(*mtConfig.CpuProfile)
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

	var err error
	mAPI, err = bnspot.NewAPI(
		&common.Credentials{
			Key:    *mtConfig.BnApiKey,
			Secret: *mtConfig.BnApiSecret,
		},
		*mtConfig.ProxyAddress,
	)
	if err != nil {
		logger.Debugf("bnspot.NewAPI error %v", err)
		return
	}
	tAPI, err = bnswap.NewAPI(
		&common.Credentials{
			Key:    *mtConfig.BnApiKey,
			Secret: *mtConfig.BnApiSecret,
		},
		*mtConfig.ProxyAddress,
	)
	if err != nil {
		logger.Debugf("bnswap.NewAPI error %v", err)
		return
	}

	mtGlobalCtx, mtGlobalCancel = context.WithCancel(context.Background())
	defer mtGlobalCancel()

	if *mtConfig.ChangeLeverage {
		for _, takerSymbol := range tSymbols {
			res, err := tAPI.UpdateLeverage(mtGlobalCtx, bnswap.UpdateLeverageParams{
				Symbol:   takerSymbol,
				Leverage: int64(*mtConfig.Leverage),
			})
			if err != nil {
				logger.Debugf("UPDATE LEVERAGE FOR %s ERROR %v", takerSymbol, err)
			} else {
				logger.Debugf("UPDATE LEVERAGE FOR %s RESPONSE %v", takerSymbol, res)
			}
			time.Sleep(time.Second)
			res, err = tAPI.UpdateMarginType(mtGlobalCtx, bnswap.UpdateMarginTypeParams{
				Symbol:     takerSymbol,
				MarginType: *mtConfig.MarginType,
			})
			if err != nil {
				logger.Debugf("UPDATE MARGIN TYPE FOR %s ERROR %v", takerSymbol, err)
			} else {
				logger.Debugf("UPDATE MARGIN TYPE FOR %s RESPONSE %v", takerSymbol, res)
			}
			time.Sleep(time.Second)
		}
	}
	_, _, _, _, err = bnspot.GetOrderLimits(mtGlobalCtx, mAPI, tSymbols)
	if err != nil {
		logger.Debugf("bnspot.GetOrderLimits error %v", err)
		return
	}
	tTickSizes, tStepSizes, _, tMinNotional, _, _, err = bnswap.GetOrderLimits(mtGlobalCtx, tAPI, tSymbols)
	if err != nil {
		logger.Debugf("bnswap.GetOrderLimits error %v", err)
		return
	}

	mtInfluxWriter, err = common.NewInfluxWriter(
		*mtConfig.InternalInflux.Address,
		*mtConfig.InternalInflux.Username,
		*mtConfig.InternalInflux.Password,
		*mtConfig.InternalInflux.Database,
		*mtConfig.InternalInflux.BatchSize,
	)
	if err != nil {
		logger.Debugf("common.NewInfluxWriter error %v", err)
		return
	}
	defer func() {
		err := mtInfluxWriter.Stop()
		if err != nil {
			logger.Warnf("stop influx writer error %v", err)
		}
	}()

	mtExternalInfluxWriter, err = common.NewInfluxWriter(
		*mtConfig.ExternalInflux.Address,
		*mtConfig.ExternalInflux.Username,
		*mtConfig.ExternalInflux.Password,
		*mtConfig.ExternalInflux.Database,
		*mtConfig.ExternalInflux.BatchSize,
	)
	if err != nil {
		logger.Debugf("common.NewInfluxWriter error %v", err)
		return
	}
	defer func() {
		err := mtExternalInfluxWriter.Stop()
		if err != nil {
			logger.Warnf("stop influx writer error %v", err)
		}
	}()

	tUserWebsocket, err = bnswap.NewUserWebsocket(
		mtGlobalCtx,
		tAPI,
		*mtConfig.ProxyAddress,
	)
	if err != nil {
		logger.Debugf("bnswap.NewUserWebsocket error %v", err)
		return
	}
	defer tUserWebsocket.Stop()

	influxSaveTimer := time.NewTimer(
		time.Now().Truncate(
			*mtConfig.InternalInflux.SaveInterval,
		).Add(
			*mtConfig.InternalInflux.SaveInterval * 3,
		).Sub(time.Now()),
	)
	externalInfluxSaveTimer := time.NewTimer(
		time.Now().Truncate(
			*mtConfig.ExternalInflux.SaveInterval,
		).Add(
			*mtConfig.ExternalInflux.SaveInterval * 3,
		).Sub(time.Now()),
	)
	mtLoopTimer = time.NewTimer(time.Second) //先等1分钟
	defer influxSaveTimer.Stop()
	defer mtLoopTimer.Stop()
	defer externalInfluxSaveTimer.Stop()

	go bnswap.WatchAccountFromHttp(
		mtGlobalCtx, tAPI,
		*mtConfig.PullInterval, tAccountCh,
	)
	go bnswap.WatchPositionsFromHttp(
		mtGlobalCtx, tAPI,
		tSymbols,
		*mtConfig.PullInterval, tPositionsCh,
	)

	spreadReportCh := make(chan common.SpreadReport, 10000)
	go reportsSaveLoop(
		mtGlobalCtx,
		mtInfluxWriter,
		*mtConfig.InternalInflux,
		spreadReportCh,
	)

	makerRowDepthChs := make(map[string]chan *common.DepthRawMessage)
	for start := 0; start < len(tSymbols); start += *mtConfig.OrderBookBatchSize {
		end := start + *mtConfig.OrderBookBatchSize
		if end > len(tSymbols) {
			end = len(tSymbols)
		}
		subMakerRowDepthChs := make(map[string]chan *common.DepthRawMessage)
		for _, symbol := range tSymbols[start:end] {
			makerRowDepthChs[symbol] = make(chan *common.DepthRawMessage, 100)
			subMakerRowDepthChs[symbol] = makerRowDepthChs[symbol]
		}
		go makerRoutedDepthLoop(
			mtGlobalCtx,
			mtGlobalCancel,
			*mtConfig.ProxyAddress,
			subMakerRowDepthChs,
		)
	}

	takerRowDepthChs := make(map[string]chan *common.DepthRawMessage)
	for start := 0; start < len(tSymbols); start += *mtConfig.OrderBookBatchSize {
		end := start + *mtConfig.OrderBookBatchSize
		if end > len(tSymbols) {
			end = len(tSymbols)
		}
		subTakerRowDepthChs := make(map[string]chan *common.DepthRawMessage)
		for _, symbol := range tSymbols[start:end] {
			takerRowDepthChs[symbol] = make(chan *common.DepthRawMessage, 100)
			subTakerRowDepthChs[symbol] = takerRowDepthChs[symbol]
		}
		go takerRoutedDepthLoop(
			mtGlobalCtx,
			mtGlobalCancel,
			*mtConfig.ProxyAddress,
			subTakerRowDepthChs,
		)
	}

	spreadCh := make(chan *common.ShortSpread, len(tSymbols)*100)
	for _, takerSymbol := range mtConfig.Symbols {
		makerSymbol := takerSymbol
		go watchMakerTakerSpread(
			mtGlobalCtx,
			makerSymbol, takerSymbol,
			*mtConfig.OrderBookTakerImpact,
			*mtConfig.OrderBookMakerDecay,
			*mtConfig.OrderBookMakerBias,
			*mtConfig.OrderBookTakerDecay,
			*mtConfig.OrderBookTakerBias,
			*mtConfig.OrderBookMaxAgeDiffBias,
			*mtConfig.ReportCount,
			*mtConfig.SpreadLookbackDuration,
			*mtConfig.SpreadLookbackMinimalWindow,
			makerRowDepthChs[makerSymbol],
			takerRowDepthChs[takerSymbol],
			spreadReportCh,
			spreadCh,
		)
	}

	tNewOrderErrorCh = make(chan TakerOrderNewError, len(tSymbols)*2)
	for _, takerSymbol := range tSymbols {
		tOrderRequestChs[takerSymbol] = make(chan TakerOrderRequest, 2)
		go watchTakerOrderRequest(
			mtGlobalCtx,
			tAPI,
			*mtConfig.OrderTimeout,
			*mtConfig.DryRun,
			tOrderRequestChs[takerSymbol],
			tNewOrderErrorCh,
		)
	}

	go bnspot.HttpPingLoop(
		mtGlobalCtx,
		mAPI,
		*mtConfig.PullInterval/2,
		mSystemStatusCh,
	)

	go bnswap.SystemStatusLoop(
		mtGlobalCtx,
		tAPI,
		*mtConfig.PullInterval/2,
		tSystemStatusCh,
	)

	if *mtConfig.CpuProfile != "" {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			sig := <-sigs
			logger.Debugf("CATCH EXIT SIGNAL %v", sig)
			mtGlobalCancel()
		}()
	}

	go func() {
		for _, takerSymbol := range tSymbols {
			select {
			case <-mtGlobalCtx.Done():
				return
			case <-time.After(*mtConfig.RequestInterval):
				logger.Debugf("INITIAL CANCEL ALL %s", takerSymbol)
				select {
				case <-mtGlobalCtx.Done():
					return
				case tOrderRequestChs[takerSymbol] <- TakerOrderRequest{
					Cancel: &bnswap.CancelAllOrderParams{
						Symbol: takerSymbol,
					},
				}:
				}
			}
		}
	}()

	logger.Debugf("START MAIN LOOP")
	for {
		select {
		case <-mtGlobalCtx.Done():
			logger.Debugf("GLOBAL CTX DONE, EXIT MAIN LOOP")
			return
		case <-tUserWebsocket.Done():
			logger.Debugf("MAKER USER WS DONE, EXIT MAIN LOOP")
			return
		case mSystemReady = <-mSystemStatusCh:
			if !mSystemReady {
				mtGlobalSilent = time.Now().Add(*mtConfig.RestartSilent)
			}
			break
		case tSystemReady = <-tSystemStatusCh:
			if !tSystemReady {
				mtGlobalSilent = time.Now().Add(*mtConfig.RestartSilent)
			}
			break
		case <-tUserWebsocket.RestartCh:
			logger.Debugf("<-tUserWebsocket.RestartCh restart silent %v", *mtConfig.RestartSilent)
			mtGlobalSilent = time.Now().Add(*mtConfig.RestartSilent)
			break
		case account := <-tAccountCh:
			handleTakerHttpAccount(account)
			break
		case ps := <-tPositionsCh:
			handleTakerHttpPositions(ps)
			break
		case msg := <-tUserWebsocket.BalanceAndPositionUpdateEventCh:
			handleTakerWSAccount(msg)
			break
		case takerOrderEvent := <-tUserWebsocket.OrderUpdateEventCh:
			takerOrder := takerOrderEvent.Order
			if takerOrder.Status == "REJECTED" ||
				takerOrder.Status == "EXPIRED" ||
				takerOrder.Status == "CANCELED" {
				if openOrder, ok := tOpenOrders[takerOrder.Symbol]; ok && openOrder.NewClientOrderId == takerOrder.ClientOrderId {
					tOrderSilentTimes[takerOrder.Symbol] = time.Now()
					tOrderCancelSilentTimes[takerOrder.Symbol] = time.Now()
					tPositionsUpdateTimes[takerOrder.Symbol] = time.Now()
					logger.Debugf("%s %s %s", takerOrder.Status, takerOrder.Symbol, takerOrder.ClientOrderId)
					delete(tOpenOrders, takerOrder.Symbol)
				}
			} else if takerOrder.Status == "FILLED" {
				if openOrder, ok := tOpenOrders[takerOrder.Symbol]; ok && openOrder.NewClientOrderId == takerOrder.ClientOrderId {
					delete(tOpenOrders, takerOrder.Symbol)
					logger.Debugf("%s %s %s", takerOrder.Status, takerOrder.Symbol, takerOrder.ClientOrderId)
				}
				tOrderSilentTimes[takerOrder.Symbol] = time.Now()
				tOrderCancelSilentTimes[takerOrder.Symbol] = time.Now()
				tHttpPositionUpdateSilentTimes[takerOrder.Symbol] = time.Now().Add(*mtConfig.HttpSilent)
				if _, ok := mtSymbolsMap[takerOrder.Symbol]; ok {
					if takerOrder.Side == common.OrderSideSell &&
						!takerOrder.ReduceOnly {
						mLastFilledSellPrices[takerOrder.Symbol] = takerOrder.AveragePrice
						tEnterTimeouts[takerOrder.Symbol] = time.Now()
						tCloseTimeouts[takerOrder.Symbol] = time.Now().Add(*mtConfig.CloseTimeout)
						tEnterSilentTimes[takerOrder.Symbol] = time.Now().Add(*mtConfig.EnterSilent)
						logger.Debugf("SET TIMEOUTS %v", tCloseTimeouts[takerOrder.Symbol])
					} else if takerOrder.Side == common.OrderSideBuy &&
						!takerOrder.ReduceOnly {
						mLastFilledBuyPrices[takerOrder.Symbol] = takerOrder.AveragePrice
						tEnterTimeouts[takerOrder.Symbol] = time.Now()
						tCloseTimeouts[takerOrder.Symbol] = time.Now().Add(*mtConfig.CloseTimeout)
						tEnterSilentTimes[takerOrder.Symbol] = time.Now().Add(*mtConfig.EnterSilent)
					} else if takerOrder.Side == common.OrderSideSell &&
						takerOrder.ReduceOnly {
						if buyPrice, ok := mLastFilledBuyPrices[takerOrder.Symbol]; ok {
							mtRealisedSpread[takerOrder.Symbol] = (takerOrder.AveragePrice - buyPrice) / buyPrice
							logger.Debugf("%s REALISED CLOSE LONG SPREAD %f", takerOrder.Symbol, mtRealisedSpread[takerOrder.Symbol])
						}
					} else if takerOrder.Side == common.OrderSideBuy &&
						takerOrder.ReduceOnly {
						if sellPrice, ok := mLastFilledSellPrices[takerOrder.Symbol]; ok {
							mtRealisedSpread[takerOrder.Symbol] = (sellPrice - takerOrder.AveragePrice) / sellPrice
							logger.Debugf("%s REALISED CLOSE SHORT SPREAD %f", takerOrder.Symbol, mtRealisedSpread[takerOrder.Symbol])
						}
					}
				}
			}
			break
		case spread := <-spreadCh:
			if lastSpread, ok := mtSpreads[spread.TakerSymbol]; ok {
				if takerPosition, ok := tPositions[spread.TakerSymbol]; ok {
					if lastSpread.MedianEnter*spread.MedianEnter <= 0 {
						if spread.MedianEnter > 0 {
							//if tEnterSilentTimes[spread.TakerSymbol].Sub(time.Now()) > 0 {
							//	logger.Debugf("TRIGGER LONG %s IN SILENT", spread.TakerSymbol)
							//} else {
							//	tEnterTimeouts[spread.TakerSymbol] = time.Now().Add(*mtConfig.EnterTimeout)
							//	logger.Debugf("TRIGGER LONG %s", spread.TakerSymbol)
							//}
							if takerPosition.PositionAmt <= 0 {
								tEnterSilentTimes[spread.TakerSymbol] = time.Now()
							}
							logger.Debugf("TRIGGER LONG %s", spread.TakerSymbol)
							tEnterTimeouts[spread.TakerSymbol] = time.Now().Add(*mtConfig.EnterTimeout)
							tOrderCancelSilentTimes[spread.TakerSymbol] = time.Now().Add(*mtConfig.OrderSilent)
							mtTriggeredDirection[spread.TakerSymbol] = 1
						} else if spread.MedianEnter < 0 {
							//if tEnterSilentTimes[spread.TakerSymbol].Sub(time.Now()) > 0 {
							//	logger.Debugf("TRIGGER SHORT %s IN SILENT", spread.TakerSymbol)
							//} else {
							//	tEnterTimeouts[spread.TakerSymbol] = time.Now().Add(*mtConfig.EnterTimeout)
							//	logger.Debugf("TRIGGER SHORT %s", spread.TakerSymbol)
							//}
							logger.Debugf("TRIGGER SHORT %s", spread.TakerSymbol)
							if takerPosition.PositionAmt >= 0 {
								tEnterSilentTimes[spread.TakerSymbol] = time.Now()
							}
							tEnterTimeouts[spread.TakerSymbol] = time.Now().Add(*mtConfig.EnterTimeout)
							tOrderCancelSilentTimes[spread.TakerSymbol] = time.Now().Add(*mtConfig.OrderSilent)
							mtTriggeredDirection[spread.TakerSymbol] = -1
						}
					} else {
						//logger.Debugf("%s %f", spread.TakerSymbol, spread.MedianEnter)
					}
				}
			}
			mtSpreads[spread.TakerSymbol] = spread
			break
		case <-influxSaveTimer.C:
			handleSave()
			influxSaveTimer.Reset(
				time.Now().Truncate(
					*mtConfig.InternalInflux.SaveInterval,
				).Add(
					*mtConfig.InternalInflux.SaveInterval,
				).Sub(time.Now()),
			)
			break
		case <-externalInfluxSaveTimer.C:
			handleExternalInfluxSave()
			externalInfluxSaveTimer.Reset(
				time.Now().Truncate(
					*mtConfig.ExternalInflux.SaveInterval,
				).Add(
					*mtConfig.ExternalInflux.SaveInterval,
				).Sub(time.Now()),
			)
			break
		case takerNewError := <-tNewOrderErrorCh:
			tOrderSilentTimes[takerNewError.Params.Symbol] = time.Now().Add(*mtConfig.OrderSilent * 5)
			break
		case <-mtLoopTimer.C:
			if mSystemReady && tSystemReady && time.Now().Sub(mtGlobalSilent) > 0 {
				updateTakerOldOrders()
				updateTakerNewOrders()
			} else {
				if len(tOpenOrders) > 0 {
					for takerSymbol := range tOpenOrders {
						select {
						case tOrderRequestChs[takerSymbol] <- TakerOrderRequest{
							Cancel: &bnswap.CancelAllOrderParams{
								Symbol: takerSymbol,
							},
						}:
							delete(tOpenOrders, takerSymbol)
						default:
						}
					}
				}
			}
			mtLoopTimer.Reset(
				time.Now().Truncate(
					*mtConfig.LoopInterval,
				).Add(
					*mtConfig.LoopInterval,
				).Sub(time.Now()),
			)
			break
		}
	}
}
