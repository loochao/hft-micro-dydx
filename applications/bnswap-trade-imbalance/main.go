package main

import (
	"context"
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

	if *swapConfig.CpuProfile != "" {
		f, err := os.Create(*swapConfig.CpuProfile)
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
			Key:    *swapConfig.BnApiKey,
			Secret: *swapConfig.BnApiSecret,
		},
		*swapConfig.ProxyAddress,
	)
	if err != nil {
		logger.Debugf("bnswap.NewAPI error %v", err)
		return
	}

	swapGlobalCtx, swapGlobalCancel = context.WithCancel(context.Background())
	defer swapGlobalCancel()

	if *swapConfig.ChangeLeverage {
		for _, takerSymbol := range swapSymbols {
			res, err := swapAPI.UpdateLeverage(swapGlobalCtx, bnswap.UpdateLeverageParams{
				Symbol:   takerSymbol,
				Leverage: int64(*swapConfig.Leverage),
			})
			if err != nil {
				logger.Debugf("UPDATE LEVERAGE FOR %s ERROR %v", takerSymbol, err)
			} else {
				logger.Debugf("UPDATE LEVERAGE FOR %s RESPONSE %v", takerSymbol, res)
			}
			time.Sleep(time.Second)
			res, err = swapAPI.UpdateMarginType(swapGlobalCtx, bnswap.UpdateMarginTypeParams{
				Symbol:     takerSymbol,
				MarginType: *swapConfig.MarginType,
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
		*swapConfig.InternalInflux.Address,
		*swapConfig.InternalInflux.Username,
		*swapConfig.InternalInflux.Password,
		*swapConfig.InternalInflux.Database,
		*swapConfig.InternalInflux.BatchSize,
	)
	if err != nil {
		logger.Debugf("common.NewInfluxWriter error %v", err)
		return
	}
	defer swapInternalInfluxWriter.Stop()

	swapExternalInfluxWriter, err = common.NewInfluxWriter(
		swapGlobalCtx,
		*swapConfig.ExternalInflux.Address,
		*swapConfig.ExternalInflux.Username,
		*swapConfig.ExternalInflux.Password,
		*swapConfig.ExternalInflux.Database,
		*swapConfig.ExternalInflux.BatchSize,
	)
	if err != nil {
		logger.Debugf("common.NewInfluxWriter error %v", err)
		return
	}
	defer swapExternalInfluxWriter.Stop()

	swapUserWebsocket, err = bnswap.NewUserWebsocket(
		swapGlobalCtx,
		swapAPI,
		*swapConfig.ProxyAddress,
	)
	if err != nil {
		logger.Debugf("bnswap.NewUserWebsocket error %v", err)
		return
	}
	defer swapUserWebsocket.Stop()

	internalInfluxSaveTimer := time.NewTimer(
		time.Now().Truncate(
			*swapConfig.InternalInflux.SaveInterval,
		).Add(
			*swapConfig.InternalInflux.SaveInterval * 3,
		).Sub(time.Now()),
	)
	externalInfluxSaveTimer := time.NewTimer(
		time.Now().Truncate(
			*swapConfig.ExternalInflux.SaveInterval,
		).Add(
			*swapConfig.ExternalInflux.SaveInterval * 3,
		).Sub(time.Now()),
	)
	swapLoopTimer = time.NewTimer(time.Second) //先等1分钟
	defer internalInfluxSaveTimer.Stop()
	defer swapLoopTimer.Stop()
	defer externalInfluxSaveTimer.Stop()

	go bnswap.WatchAccountFromHttp(
		swapGlobalCtx, swapAPI,
		*swapConfig.PullInterval, swapAccountCh,
	)
	go bnswap.WatchPositionsFromHttp(
		swapGlobalCtx, swapAPI,
		swapSymbols,
		*swapConfig.PullInterval, swapPositionsCh,
	)

	swapDepthReportCh := make(chan DepthReport, 10000)
	go swapReportsSaveLoop(
		swapGlobalCtx,
		swapInternalInfluxWriter,
		*swapConfig.InternalInflux,
		swapDepthReportCh,
	)

	spotDepthReportCh := make(chan DepthReport, 10000)
	go swapReportsSaveLoop(
		swapGlobalCtx,
		swapInternalInfluxWriter,
		*swapConfig.InternalInflux,
		spotDepthReportCh,
	)

	bnswapRawDepthChs := make(map[string]chan *common.DepthRawMessage)
	for start := 0; start < len(swapSymbols); start += *swapConfig.OrderBookBatchSize {
		end := start + *swapConfig.OrderBookBatchSize
		if end > len(swapSymbols) {
			end = len(swapSymbols)
		}
		subTakerRowDepthChs := make(map[string]chan *common.DepthRawMessage)
		for _, symbol := range swapSymbols[start:end] {
			bnswapRawDepthChs[symbol] = make(chan *common.DepthRawMessage, 100)
			subTakerRowDepthChs[symbol] = bnswapRawDepthChs[symbol]
		}
		go StreamDepth5(
			swapGlobalCtx,
			swapGlobalCancel,
			*swapConfig.ProxyAddress,
			subTakerRowDepthChs,
		)
	}

	bnswapWalkedDepth5Ch := make(chan common.WalkedTakerDepth, len(swapSymbols)*100)
	for takerSymbol := range swapConfig.SymbolsMap {
		go StreamWalkedDepth(
			swapGlobalCtx,
			takerSymbol,
			*swapConfig.OrderBookTimeDecay,
			*swapConfig.OrderBookTimeBias,
			*swapConfig.OrderBookTakerImpact,
			*swapConfig.ReportCount,
			bnswapRawDepthChs[takerSymbol],
			swapDepthReportCh,
			bnswapWalkedDepth5Ch,
		)
	}

	swapNewOrderErrorCh = make(chan TakerOrderNewError, len(swapSymbols)*2)
	for _, takerSymbol := range swapSymbols {
		swapOrderRequestChs[takerSymbol] = make(chan bnswap.NewOrderParams, 2)
		go watchTakerOrderRequest(
			swapGlobalCtx,
			swapAPI,
			*swapConfig.OrderTimeout,
			*swapConfig.DryRun,
			swapOrderRequestChs[takerSymbol],
			swapNewOrderErrorCh,
		)
	}

	go bnswap.SystemStatusLoop(
		swapGlobalCtx,
		swapAPI,
		*swapConfig.PullInterval/2,
		swapSystemStatusCh,
	)

	go StreamMergedSignals(
		swapGlobalCtx,
		swapGlobalCancel,
		*swapConfig.ProxyAddress,
		swapConfig.Exchanges,
		swapConfig.SymbolsMap,
		*swapConfig.ImbalanceLookback,
		*swapConfig.ImbalanceTimeToLive,
		*swapConfig.ImbalanceUpdateInterval,
		swapMergedSignalCh,
	)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		logger.Debugf("CATCH EXIT SIGNAL %v", sig)
		swapGlobalCancel()
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
				swapGlobalSilent = time.Now().Add(*swapConfig.RestartSilent)
			}
			break
		case <-swapUserWebsocket.RestartCh:
			logger.Debugf("<-swapUserWebsocket.RestartCh restart silent %v", *swapConfig.RestartSilent)
			swapGlobalSilent = time.Now().Add(*swapConfig.RestartSilent)
			break

		case s := <-swapMergedSignalCh:
			swapMergedSignals[s.Symbol] = s

		case account := <-swapAccountCh:
			handleTakerHttpAccount(account)
			break
		case ps := <-swapPositionsCh:
			handleTakerHttpPositions(ps)
			break
		case msg := <-swapUserWebsocket.BalanceAndPositionUpdateEventCh:
			handleTakerWSAccount(msg)
			break
		case swapOrderEvent := <-swapUserWebsocket.OrderUpdateEventCh:
			swapOrder := swapOrderEvent.Order
			if swapOrder.Status == "REJECTED" ||
				swapOrder.Status == "EXPIRED" ||
				swapOrder.Status == "CANCELED" {
				logger.Debugf("%s %s %s %s", swapOrder.Symbol, swapOrder.Status, swapOrder.Side, swapOrder.ClientOrderId)
			} else if swapOrder.Status == "FILLED" {
				swapOrderSilentTimes[swapOrder.Symbol] = time.Now()
				swapHttpPositionUpdateSilentTimes[swapOrder.Symbol] = time.Now().Add(*swapConfig.HttpSilent)
				if _, ok := swapSymbolsMap[swapOrder.Symbol]; ok {
					logger.Debugf("%s FILLED %s %f %f", swapOrder.Symbol, swapOrder.Side, swapOrder.FilledAccumulatedQuantity, swapOrder.AveragePrice)
					if lastEnterPrice, ok := swapLastEnterPrices[swapOrder.Symbol]; ok {
						swapEnterOffset[swapOrder.Symbol] = (lastEnterPrice - swapOrder.AveragePrice)/ lastEnterPrice
						logger.Debugf("%s ENTER OFFSET %f", swapOrder.Symbol, swapEnterOffset[swapOrder.Symbol])
					}
					swapLastEnterPrices[swapOrder.Symbol] = swapOrder.AveragePrice
				}
			}
			break
		case depth := <-bnswapWalkedDepth5Ch:
			swapWalkedDepths[depth.Symbol] = depth
			break
		case <-internalInfluxSaveTimer.C:
			handleSave()
			internalInfluxSaveTimer.Reset(
				time.Now().Truncate(
					*swapConfig.InternalInflux.SaveInterval,
				).Add(
					*swapConfig.InternalInflux.SaveInterval,
				).Sub(time.Now()),
			)
			break
		case <-externalInfluxSaveTimer.C:
			handleExternalInfluxSave()
			externalInfluxSaveTimer.Reset(
				time.Now().Truncate(
					*swapConfig.ExternalInflux.SaveInterval,
				).Add(
					*swapConfig.ExternalInflux.SaveInterval,
				).Sub(time.Now()),
			)
			break
		case takerNewError := <-swapNewOrderErrorCh:
			swapOrderSilentTimes[takerNewError.Params.Symbol] = time.Now().Add(*swapConfig.OrderSilent * 5)
			break
		case <-swapLoopTimer.C:
			if swapSystemReady && time.Now().Sub(swapGlobalSilent) > 0 {
				updateNewOrders()
			} else {
				if time.Now().Truncate(time.Second*15).Add(*swapConfig.LoopInterval).Sub(time.Now()) > 0 {
					logger.Debugf("SYSTEM NOT READY SWAP %v SILENT TIME %v",
						swapSystemReady,  time.Now().Sub(swapGlobalSilent),
					)
				}
			}
			swapLoopTimer.Reset(
				time.Now().Truncate(
					*swapConfig.LoopInterval,
				).Add(
					*swapConfig.LoopInterval,
				).Sub(time.Now()),
			)
			break
		}
	}
}
