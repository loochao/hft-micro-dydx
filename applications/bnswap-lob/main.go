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
	swapAPI, err = bnswap.NewAPI(
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

	spotAPI, err := bnspot.NewAPI(&common.Credentials{
		Key:    *mtConfig.BnApiKey,
		Secret: *mtConfig.BnApiSecret,
	}, *mtConfig.ProxyAddress)
	if err != nil {
		logger.Debugf("bnspot.NewAPI error %v", err)
		return
	}

	swapGlobalCtx, swapGlobalCancel = context.WithCancel(context.Background())
	defer swapGlobalCancel()

	if *mtConfig.ChangeLeverage {
		for _, takerSymbol := range swapSymbols {
			res, err := swapAPI.UpdateLeverage(swapGlobalCtx, bnswap.UpdateLeverageParams{
				Symbol:   takerSymbol,
				Leverage: int64(*mtConfig.Leverage),
			})
			if err != nil {
				logger.Debugf("UPDATE LEVERAGE FOR %s ERROR %v", takerSymbol, err)
			} else {
				logger.Debugf("UPDATE LEVERAGE FOR %s RESPONSE %v", takerSymbol, res)
			}
			time.Sleep(time.Second)
			res, err = swapAPI.UpdateMarginType(swapGlobalCtx, bnswap.UpdateMarginTypeParams{
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

	swapTickSizes, swapStepSizes, _, swapMinNotional, _, _, err = bnswap.GetOrderLimits(swapGlobalCtx, swapAPI, swapSymbols)
	if err != nil {
		logger.Debugf("bnswap.GetOrderLimits error %v", err)
		return
	}

	swapInternalInfluxWriter, err = common.NewInfluxWriter(
		swapGlobalCtx,
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
	defer swapInternalInfluxWriter.Stop()

	swapExternalInfluxWriter, err = common.NewInfluxWriter(
		swapGlobalCtx,
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
	defer swapExternalInfluxWriter.Stop()

	swapUserWebsocket, err = bnswap.NewUserWebsocket(
		swapGlobalCtx,
		swapAPI,
		*mtConfig.ProxyAddress,
	)
	if err != nil {
		logger.Debugf("bnswap.NewUserWebsocket error %v", err)
		return
	}
	defer swapUserWebsocket.Stop()

	internalInfluxSaveTimer := time.NewTimer(
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
	defer internalInfluxSaveTimer.Stop()
	defer mtLoopTimer.Stop()
	defer externalInfluxSaveTimer.Stop()

	go bnswap.WatchAccountFromHttp(
		swapGlobalCtx, swapAPI,
		*mtConfig.PullInterval, swapAccountCh,
	)
	go bnswap.WatchPositionsFromHttp(
		swapGlobalCtx, swapAPI,
		swapSymbols,
		*mtConfig.PullInterval, swapPositionsCh,
	)

	swapDepthReportCh := make(chan DepthReport, 10000)
	go swapReportsSaveLoop(
		swapGlobalCtx,
		swapInternalInfluxWriter,
		*mtConfig.InternalInflux,
		swapDepthReportCh,
	)

	spotDepthReportCh := make(chan DepthReport, 10000)
	go swapReportsSaveLoop(
		swapGlobalCtx,
		swapInternalInfluxWriter,
		*mtConfig.InternalInflux,
		spotDepthReportCh,
	)

	bnswapRawDepthChs := make(map[string]chan *common.DepthRawMessage)
	for start := 0; start < len(swapSymbols); start += *mtConfig.OrderBookBatchSize {
		end := start + *mtConfig.OrderBookBatchSize
		if end > len(swapSymbols) {
			end = len(swapSymbols)
		}
		subTakerRowDepthChs := make(map[string]chan *common.DepthRawMessage)
		for _, symbol := range swapSymbols[start:end] {
			bnswapRawDepthChs[symbol] = make(chan *common.DepthRawMessage, 100)
			subTakerRowDepthChs[symbol] = bnswapRawDepthChs[symbol]
		}
		go bnswapDepthLoop(
			swapGlobalCtx,
			swapGlobalCancel,
			*mtConfig.ProxyAddress,
			subTakerRowDepthChs,
		)
	}

	bnswapWalkedDepth20Ch := make(chan WalkedDepth20, len(swapSymbols)*100)
	for _, takerSymbol := range mtConfig.Symbols {
		go bnswapDepthWalkingLoop(
			swapGlobalCtx,
			takerSymbol,
			*mtConfig.OrderBookLevelDecay,
			*mtConfig.OrderBookTimeDecay,
			*mtConfig.OrderBookTimeBias,
			*mtConfig.DepthLookbackDuration,
			*mtConfig.ReportCount,
			bnswapRawDepthChs[takerSymbol],
			swapDepthReportCh,
			bnswapWalkedDepth20Ch,
		)
	}

	bnspotRawDepthChs := make(map[string]chan *common.DepthRawMessage)
	for start := 0; start < len(swapSymbols); start += *mtConfig.OrderBookBatchSize {
		end := start + *mtConfig.OrderBookBatchSize
		if end > len(swapSymbols) {
			end = len(swapSymbols)
		}
		subTakerRowDepthChs := make(map[string]chan *common.DepthRawMessage)
		for _, symbol := range swapSymbols[start:end] {
			bnspotRawDepthChs[symbol] = make(chan *common.DepthRawMessage, 100)
			subTakerRowDepthChs[symbol] = bnspotRawDepthChs[symbol]
		}
		go bnspotDepthLoop(
			swapGlobalCtx,
			swapGlobalCancel,
			*mtConfig.ProxyAddress,
			subTakerRowDepthChs,
		)
	}

	bnspotWalkedDepth20Ch := make(chan WalkedDepth20, len(swapSymbols)*100)
	for _, takerSymbol := range mtConfig.Symbols {
		go bnspotDepthWalkingLoop(
			swapGlobalCtx,
			takerSymbol,
			*mtConfig.OrderBookLevelDecay,
			*mtConfig.OrderBookTimeDecay,
			*mtConfig.OrderBookTimeBias,
			*mtConfig.DepthLookbackDuration,
			*mtConfig.ReportCount,
			bnspotRawDepthChs[takerSymbol],
			spotDepthReportCh,
			bnspotWalkedDepth20Ch,
		)
	}

	swapNewOrderErrorCh = make(chan TakerOrderNewError, len(swapSymbols)*2)
	for _, takerSymbol := range swapSymbols {
		swapOrderRequestChs[takerSymbol] = make(chan TakerOrderRequest, 2)
		go watchTakerOrderRequest(
			swapGlobalCtx,
			swapAPI,
			*mtConfig.OrderTimeout,
			*mtConfig.DryRun,
			swapOrderRequestChs[takerSymbol],
			swapNewOrderErrorCh,
		)
	}

	go bnswap.SystemStatusLoop(
		swapGlobalCtx,
		swapAPI,
		*mtConfig.PullInterval/2,
		swapSystemStatusCh,
	)

	go bnspot.HttpPingLoop(
		swapGlobalCtx,
		spotAPI,
		*mtConfig.PullInterval/2,
		spotSystemStatusCh,
	)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		logger.Debugf("CATCH EXIT SIGNAL %v", sig)
		swapGlobalCancel()
	}()

	go func() {
		for _, takerSymbol := range swapSymbols {
			select {
			case <-swapGlobalCtx.Done():
				return
			case <-time.After(*mtConfig.RequestInterval):
				logger.Debugf("INITIAL CANCEL ALL %s", takerSymbol)
				select {
				case <-swapGlobalCtx.Done():
					return
				case swapOrderRequestChs[takerSymbol] <- TakerOrderRequest{
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
		case <-swapGlobalCtx.Done():
			logger.Debugf("GLOBAL CTX DONE, EXIT MAIN LOOP")
			return
		case <-swapUserWebsocket.Done():
			logger.Debugf("MAKER USER WS DONE, EXIT MAIN LOOP")
			return

		case swapSystemReady = <-swapSystemStatusCh:
			if !swapSystemReady {
				mtGlobalSilent = time.Now().Add(*mtConfig.RestartSilent)
			}
			break
		case spotSystemReady = <-swapSystemStatusCh:
			if !spotSystemReady {
				mtGlobalSilent = time.Now().Add(*mtConfig.RestartSilent)
			}
			break
		case <-swapUserWebsocket.RestartCh:
			logger.Debugf("<-swapUserWebsocket.RestartCh restart silent %v", *mtConfig.RestartSilent)
			mtGlobalSilent = time.Now().Add(*mtConfig.RestartSilent)
			break
		case account := <-swapAccountCh:
			handleTakerHttpAccount(account)
			break
		case ps := <-swapPositionsCh:
			handleTakerHttpPositions(ps)
			break
		case msg := <-swapUserWebsocket.BalanceAndPositionUpdateEventCh:
			handleTakerWSAccount(msg)
			break
		case takerOrderEvent := <-swapUserWebsocket.OrderUpdateEventCh:
			takerOrder := takerOrderEvent.Order
			if takerOrder.Status == "REJECTED" ||
				takerOrder.Status == "EXPIRED" ||
				takerOrder.Status == "CANCELED" {
				if openOrder, ok := swapOpenOrders[takerOrder.Symbol]; ok && openOrder.NewClientOrderId == takerOrder.ClientOrderId {
					tOrderSilentTimes[takerOrder.Symbol] = time.Now()
					swapOrderCancelSilentTimes[takerOrder.Symbol] = time.Now()
					swapPositionsUpdateTimes[takerOrder.Symbol] = time.Now()
					delete(swapOpenOrders, takerOrder.Symbol)
				}
			} else if takerOrder.Status == "FILLED" {
				if openOrder, ok := swapOpenOrders[takerOrder.Symbol]; ok && openOrder.NewClientOrderId == takerOrder.ClientOrderId {
					delete(swapOpenOrders, takerOrder.Symbol)
				}
				tOrderSilentTimes[takerOrder.Symbol] = time.Now()
				swapOrderCancelSilentTimes[takerOrder.Symbol] = time.Now()
				swapHttpPositionUpdateSilentTimes[takerOrder.Symbol] = time.Now().Add(*mtConfig.HttpSilent)
				if _, ok := swapSymbolsMap[takerOrder.Symbol]; ok {
					if takerOrder.Side == common.OrderSideSell &&
						!takerOrder.ReduceOnly {
						swapLastFilledSellPrices[takerOrder.Symbol] = takerOrder.AveragePrice
					} else if takerOrder.Side == common.OrderSideBuy &&
						!takerOrder.ReduceOnly {
						swapLastFilledBuyPrices[takerOrder.Symbol] = takerOrder.AveragePrice
					} else if takerOrder.Side == common.OrderSideSell &&
						takerOrder.ReduceOnly {
						if buyPrice, ok := swapLastFilledBuyPrices[takerOrder.Symbol]; ok {
							swapRealisedSpread[takerOrder.Symbol] = (takerOrder.AveragePrice - buyPrice) / buyPrice
							logger.Debugf("%s CLOSE LONG SPREAD %f", takerOrder.Symbol, swapRealisedSpread[takerOrder.Symbol])
						}
					} else if takerOrder.Side == common.OrderSideBuy &&
						takerOrder.ReduceOnly {
						if sellPrice, ok := swapLastFilledSellPrices[takerOrder.Symbol]; ok {
							swapRealisedSpread[takerOrder.Symbol] = (sellPrice - takerOrder.AveragePrice) / sellPrice
							logger.Debugf("%s CLOSE SHORT SPREAD %f", takerOrder.Symbol, swapRealisedSpread[takerOrder.Symbol])
						}
					}
				}
			}
			break
		case depth := <-bnswapWalkedDepth20Ch:
			swapWalkedDepths[depth.Symbol] = depth
			break
		case depth := <-bnspotWalkedDepth20Ch:
			spotWalkedDepths[depth.Symbol] = depth
			break
		case <-internalInfluxSaveTimer.C:
			handleSave()
			internalInfluxSaveTimer.Reset(
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
		case takerNewError := <-swapNewOrderErrorCh:
			tOrderSilentTimes[takerNewError.Params.Symbol] = time.Now().Add(*mtConfig.OrderSilent * 5)
			break
		case <-mtLoopTimer.C:
			if spotSystemReady && swapSystemReady && time.Now().Sub(mtGlobalSilent) > 0 {
				updateTakerOldOrders()
				updateTakerNewOrders()
			} else {
				if len(swapOpenOrders) > 0 {
					for takerSymbol := range swapOpenOrders {
						select {
						case swapOrderRequestChs[takerSymbol] <- TakerOrderRequest{
							Cancel: &bnswap.CancelAllOrderParams{
								Symbol: takerSymbol,
							},
						}:
							delete(swapOpenOrders, takerSymbol)
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
