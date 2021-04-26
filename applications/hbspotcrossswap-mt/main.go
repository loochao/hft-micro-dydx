package main

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/hbcrossswap"
	"github.com/geometrybase/hft-micro/hbspot"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"os/signal"
	"runtime/pprof"
	"strings"
	"syscall"
	"time"
)

func main() {

	if *hbConfig.CpuProfile != "" {
		f, err := os.Create(*hbConfig.CpuProfile)
		if err != nil {
			logger.Debugf("os.Create %v", err)
			return
		}
		err = pprof.StartCPUProfile(f)
		if err != nil {
			logger.Debugf("pprof.StartCPUProfile %v", err)
			return
		}
		defer pprof.StopCPUProfile()
	}

	var err error
	hbcrossswapAPI, err = hbcrossswap.NewAPI(
		*hbConfig.ApiKey,
		*hbConfig.ApiSecret,
		*hbConfig.ProxyAddress,
	)
	if err != nil {
		logger.Debugf("hbcrossswap.NewAPI %v", err)
		return
	}
	hbspotAPI, err = hbspot.NewAPI(
		*hbConfig.ApiKey,
		*hbConfig.ApiSecret,
		*hbConfig.ProxyAddress,
	)
	if err != nil {
		logger.Debugf("hbspot.NewAPI %v", err)
		return
	}

	hbGlobalCtx, hbGlobalCancel = context.WithCancel(context.Background())
	defer hbGlobalCancel()

	accounts, err := hbspotAPI.GetAccounts(hbGlobalCtx)
	if err != nil {
		logger.Debugf("hbspotAPI.GetAccounts %v", err)
		return
	}
	for _, a := range accounts {
		if a.Type == "spot" {
			hbspotAccountID = a.ID
		}
	}
	if hbspotAccountID == 0 {
		logger.Debug("HB SPOT ACCOUNT ID NOT EXISTS!!!")
		return
	}

	_, hbcrossswapContractSizes, err = hbcrossswap.GetOrderLimits(hbGlobalCtx, hbcrossswapAPI, hbcrossswapSymbols)
	if err != nil {
		logger.Debugf("hbcrossswap.GetOrderLimits %v", err)
		return
	}
	hbspotTickSizes, hbspotStepSizes, _, hbspotMinNotional, hbspotPricePrecisions, hbspotAmountPrecisions, err = hbspot.GetOrderLimits(hbGlobalCtx, hbspotAPI, hbspotSymbols)
	if err != nil {
		logger.Debugf("hbspot.GetOrderLimits %v", err)
		return
	}

	for spotSymbol, spotStepSize := range hbspotStepSizes {
		swapStepSize := hbcrossswapContractSizes[hbSwapSpotSymbolsMap[spotSymbol]]
		hbMergedStepSizes[spotSymbol] = common.MergedStepSize(spotStepSize, swapStepSize)
	}

	hbInternalInfluxWriter, err = common.NewInfluxWriter(
		*hbConfig.InternalInflux.Address,
		*hbConfig.InternalInflux.Username,
		*hbConfig.InternalInflux.Password,
		*hbConfig.InternalInflux.Database,
		*hbConfig.InternalInflux.BatchSize,
	)
	if err != nil {
		logger.Debugf("common.NewInfluxWriter %v", err)
		return
	}

	hbExternalInfluxWriter, err = common.NewInfluxWriter(
		*hbConfig.ExternalInflux.Address,
		*hbConfig.ExternalInflux.Username,
		*hbConfig.ExternalInflux.Password,
		*hbConfig.ExternalInflux.Database,
		*hbConfig.ExternalInflux.BatchSize,
	)
	if err != nil {
		logger.Debugf("common.NewInfluxWriter %v", err)
		return
	}

	defer func() {
		err := hbInternalInfluxWriter.Stop()
		if err != nil {
			logger.Warnf("stop influx writer error %v", err)
		}
	}()

	spreadReportCh := make(chan common.SpreadReport, 10000)
	go reportsSaveLoop(
		hbGlobalCtx,
		hbInternalInfluxWriter,
		*hbConfig.InternalInflux,
		spreadReportCh,
	)

	hbspotUserWebsocket = hbspot.NewUserWebsocket(
		hbGlobalCtx,
		*hbConfig.ApiKey,
		*hbConfig.ApiSecret,
		hbspotSymbols,
		*hbConfig.ProxyAddress,
	)
	defer hbspotUserWebsocket.Stop()

	hbcrossswapUserWebsocket = hbcrossswap.NewUserWebsocket(
		hbGlobalCtx,
		*hbConfig.ApiKey,
		*hbConfig.ApiSecret,
		hbcrossswapSymbols,
		*hbConfig.ProxyAddress,
	)
	defer hbcrossswapUserWebsocket.Stop()

	influxSaveTimer := time.NewTimer(
		time.Now().Truncate(
			*hbConfig.InternalInflux.SaveInterval,
		).Add(
			*hbConfig.InternalInflux.SaveInterval * 3,
		).Sub(time.Now()),
	)
	externalInfluxSaveTimer := time.NewTimer(
		time.Now().Truncate(
			*hbConfig.ExternalInflux.SaveInterval,
		).Add(
			*hbConfig.ExternalInflux.SaveInterval * 3,
		).Sub(time.Now()),
	)
	hbLoopTimer = time.NewTimer(time.Second) //先等1分钟
	frRankUpdatedTimer := time.NewTimer(time.Second * 15)

	defer influxSaveTimer.Stop()
	defer hbLoopTimer.Stop()
	defer frRankUpdatedTimer.Stop()
	defer externalInfluxSaveTimer.Stop()

	go hbcrossswap.WatchPositionsFromHttp(
		hbGlobalCtx, hbcrossswapAPI,
		hbcrossswapSymbols, *hbConfig.PullInterval,
		hbcrossswapPositionCh,
	)
	go hbcrossswap.WatchAccountFromHttp(
		hbGlobalCtx, hbcrossswapAPI,
		*hbConfig.PullInterval, hbcrossswapAccountCh,
	)
	go hbspot.WatchAccountFromHttp(
		hbGlobalCtx, hbspotAPI, hbspotAccountID,
		*hbConfig.PullInterval, hbspotAccountCh,
	)
	go hbcrossswap.WatchFundingRate(
		hbGlobalCtx, hbcrossswapAPI,
		hbcrossswapSymbols,
		*hbConfig.PullInterval*10,
		hbcrossswapFundingRatesCh,
	)

	go watchSwapBars(
		hbGlobalCtx,
		hbcrossswapAPI,
		hbcrossswapSymbols,
		*hbConfig.BarsLookback,
		*hbConfig.PullBarsInterval,
		*hbConfig.PullBarsRetryInterval,
		*hbConfig.RequestInterval,
		hbcrossswapBarsMapCh,
	)

	go watchSpotBars(
		hbGlobalCtx,
		hbspotAPI,
		hbspotSymbols,
		*hbConfig.BarsLookback,
		*hbConfig.PullBarsInterval,
		*hbConfig.PullBarsRetryInterval,
		*hbConfig.RequestInterval,
		hbspotBarsMapCh,
	)

	go watchDeltaQuantile(
		hbGlobalCtx,
		hbspotSymbols,
		hbSpotSwapSymbolsMap,
		*hbConfig.BotQuantile,
		*hbConfig.TopQuantile,
		*hbConfig.TopBandScale,
		*hbConfig.BotBandScale,
		*hbConfig.MinimalEnterDelta,
		*hbConfig.MaximalExitDelta,
		*hbConfig.MinimalBandOffset,
		hbBarsMapCh,
		hbQuantilesCh,
	)

	makerRowDepthChs := make(map[string]chan *common.DepthRawMessage)
	for start := 0; start < len(hbspotSymbols); start += *hbConfig.OrderBookBatchSize {
		end := start + *hbConfig.OrderBookBatchSize
		if end > len(hbspotSymbols) {
			end = len(hbspotSymbols)
		}
		subMakerRowDepthChs := make(map[string]chan *common.DepthRawMessage)
		for _, symbol := range hbspotSymbols[start:end] {
			makerRowDepthChs[symbol] = make(chan *common.DepthRawMessage, 100)
			subMakerRowDepthChs[symbol] = makerRowDepthChs[symbol]
		}
		go makerDepthWSLoop(
			hbGlobalCtx,
			hbGlobalCancel,
			*hbConfig.ProxyAddress,
			subMakerRowDepthChs,
		)
	}

	takerRowDepthChs := make(map[string]chan *common.DepthRawMessage)
	for start := 0; start < len(hbcrossswapSymbols); start += *hbConfig.OrderBookBatchSize {
		end := start + *hbConfig.OrderBookBatchSize
		if end > len(hbcrossswapSymbols) {
			end = len(hbcrossswapSymbols)
		}
		subTakerRowDepthChs := make(map[string]chan *common.DepthRawMessage)
		for _, symbol := range hbcrossswapSymbols[start:end] {
			takerRowDepthChs[symbol] = make(chan *common.DepthRawMessage, 100)
			subTakerRowDepthChs[symbol] = takerRowDepthChs[symbol]
		}
		go takerDepthWebsocketLoop(
			hbGlobalCtx,
			hbGlobalCancel,
			*hbConfig.ProxyAddress,
			subTakerRowDepthChs,
		)
	}

	spreadCh := make(chan *common.MakerTakerSpread, len(hbspotSymbols)*100)
	for makerSymbol, takerSymbol := range hbConfig.SpotSwapPairs {
		go watchMakerTakerSpread(
			hbGlobalCtx,
			makerSymbol, takerSymbol,
			hbcrossswapContractSizes[takerSymbol],
			*hbConfig.OrderBookMakerImpact,
			*hbConfig.OrderBookTakerImpact,
			*hbConfig.OrderBookMakerDecay,
			*hbConfig.OrderBookMakerBias,
			*hbConfig.OrderBookTakerDecay,
			*hbConfig.OrderBookTakerBias,
			*hbConfig.OrderBookMaxAgeDiffBias,
			*hbConfig.ReportCount,
			*hbConfig.SpreadLookbackDuration,
			*hbConfig.SpreadLookbackMinimalWindow,
			makerRowDepthChs[makerSymbol],
			takerRowDepthChs[takerSymbol],
			spreadReportCh,
			spreadCh,
		)
	}

	hbspotNewOrderErrorCh = make(chan SpotOrderNewError, len(hbspotSymbols)*2)
	for _, spotSymbol := range hbspotSymbols {
		hbspotOrderRequestChs[spotSymbol] = make(chan SpotOrderRequest, 2)
		go watchSpotOrderRequest(
			hbGlobalCtx,
			hbspotAPI,
			*hbConfig.OrderTimeout,
			*hbConfig.DryRun,
			hbspotOrderRequestChs[spotSymbol],
			hbspotNewOrderErrorCh,
		)
		hbspotOrderRequestChs[spotSymbol] <- SpotOrderRequest{
			Cancel: &hbspot.CancelAllParam{Symbol: spotSymbol},
		}
	}

	hbcrossswapNewOrderErrorCh = make(chan SwapOrderNewError, len(hbspotSymbols)*2)
	for _, swapSymbol := range hbcrossswapSymbols {
		hbcrossswapOrderRequestChs[swapSymbol] = make(chan hbcrossswap.NewOrderParam, 2)
		go watchSwapOrderRequest(
			hbGlobalCtx,
			hbcrossswapAPI,
			*hbConfig.OrderTimeout,
			*hbConfig.DryRun,
			hbcrossswapOrderRequestChs[swapSymbol],
			hbcrossswapNewOrderErrorCh,
		)
	}

	go hbcrossswap.SystemStatusLoop(
		hbGlobalCtx,
		hbcrossswapAPI,
		*hbConfig.PullInterval/2,
		hbcrossswapSystemStatusCh,
	)
	go hbspot.SystemStatusLoop(
		hbGlobalCtx,
		hbspotAPI,
		*hbConfig.PullInterval/2,
		hbspotSystemStatusCh,
	)

	if *hbConfig.CpuProfile != "" {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			sig := <-sigs
			logger.Debugf("CATCH EXIT SIGNAL %v", sig)
			hbGlobalCancel()
		}()
	}


	go func() {
		for _, spotSymbol := range hbspotSymbols {
			select {
			case <-hbGlobalCtx.Done():
				return
			case <-time.After(*hbConfig.RequestInterval):
				logger.Debugf("INITIAL CANCEL ALL %s", spotSymbol)
				select {
				case <-hbGlobalCtx.Done():
					return
				case hbspotOrderRequestChs[spotSymbol] <- SpotOrderRequest{
					Cancel: &hbspot.CancelAllParam{
						Symbol: spotSymbol,
					},
				}:
				}
			}
		}
	}()


	logger.Debugf("MAIN LOOP START")
	fundingInterval := time.Hour * 8
	fundingSilent := time.Minute * 5
	for {
		select {
		case <-hbGlobalCtx.Done():
			logger.Debug("MAIN LOOP EXIT")
			return

		case hbcrossswapSystemReady = <-hbcrossswapSystemStatusCh:
			if !hbcrossswapSystemReady {
				//logger.Debugf("hbcrossswapSystemReady restart silent %v", *hbConfig.RestartSilent)
				hbGlobalSilent = time.Now().Add(*hbConfig.RestartSilent)
			}
		case hbspotSystemReady = <-hbspotSystemStatusCh:
			if !hbspotSystemReady {
				//logger.Debugf("hbspotSystemReady restart silent %v", *hbConfig.RestartSilent)
				hbGlobalSilent = time.Now().Add(*hbConfig.RestartSilent)
			}
		case <-hbspotUserWebsocket.RestartCh:
			logger.Debugf("hbspotUserWebsocket restart silent %v", *hbConfig.RestartSilent)
			hbGlobalSilent = time.Now().Add(*hbConfig.RestartSilent)

		case <-hbcrossswapUserWebsocket.RestartCh:
			logger.Debugf("hbcrossswapUserWebsocket restart silent %v", *hbConfig.RestartSilent)
			hbGlobalSilent = time.Now().Add(*hbConfig.RestartSilent)
		case p := <-hbcrossswapPositionCh:
			handleSwapHttpPositions(p)
			break
		case account := <-hbcrossswapAccountCh:
			handleSwapHttpAccount(account)
			break
		case account := <-hbspotAccountCh:
			handleSpotHttpAccount(account)
			break
		case msg := <-hbspotUserWebsocket.BalanceCh:
			handleSpotWSBalance(msg)
			break
		case spotOrder := <-hbspotUserWebsocket.OrderCh:
			hbLoopTimer.Reset(time.Nanosecond)
			if spotOrder.OrderStatus != nil {
				if *spotOrder.OrderStatus == hbspot.OrderStatusFilled {
					hbspotHttpBalanceUpdateSilentTimes[spotOrder.Symbol] = time.Now().Add(*hbConfig.HttpSilent)
					hbcrossswapOrderSilentTimes[hbSpotSwapSymbolsMap[spotOrder.Symbol]] = time.Now()
					if spotOrder.TradeVolume != nil && spotOrder.TradePrice != nil && spotOrder.Type != nil {
						if strings.Contains(*spotOrder.Type, "buy") {
							hbspotLastFilledBuyPrices[spotOrder.Symbol] = *spotOrder.TradePrice
						} else {
							hbspotLastFilledSellPrices[spotOrder.Symbol] = *spotOrder.TradePrice
						}
						logger.Debugf("SPOT WS ORDER FILLED %s %s SIZE %f PRICE %f", spotOrder.Symbol, *spotOrder.Type, *spotOrder.TradeVolume, *spotOrder.TradePrice)
					}
					if openOrder, ok := hbspotOpenOrders[spotOrder.Symbol]; ok && openOrder.ClientOrderID == spotOrder.ClientOrderID {
						delete(hbspotOpenOrders, spotOrder.Symbol)
					}
				} else if *spotOrder.OrderStatus == hbspot.OrderStatusCanceled {
					if openOrder, ok := hbspotOpenOrders[spotOrder.Symbol]; ok && openOrder.ClientOrderID == spotOrder.ClientOrderID {
						delete(hbspotOpenOrders, spotOrder.Symbol)
					}
				} else if *spotOrder.OrderStatus == hbspot.OrderStatusRejected {
					if openOrder, ok := hbspotOpenOrders[spotOrder.Symbol]; ok && openOrder.ClientOrderID == spotOrder.ClientOrderID {
						delete(hbspotOpenOrders, spotOrder.Symbol)
					}
				}
			}
			break
		case msg := <-hbcrossswapUserWebsocket.PositionCh:
			handleWSPosition(msg)
			break
		case msg := <-hbcrossswapUserWebsocket.AccountCh:
			handleWSAccount(msg)
			break
		case swapOrder := <-hbcrossswapUserWebsocket.OrderCh:
			hbLoopTimer.Reset(time.Nanosecond)
			if swapOrder.Status == hbcrossswap.OrderStatusFilled ||
				swapOrder.Status == hbcrossswap.OrderStatusCancelled ||
				swapOrder.Status == hbcrossswap.OrderStatusPartiallyFilledButCancelledByClient {
				if swapOrder.Status == hbcrossswap.OrderStatusCancelled {
					logger.Debugf("SWAP WS ORDER CANCELED %v ", swapOrder)
					hbcrossswapOrderSilentTimes[swapOrder.Symbol] = time.Now().Add(time.Second)
					hbcrossswapPositionsUpdateTimes[swapOrder.Symbol] = time.Unix(0, 0)
				} else {
					logger.Debugf(
						"SWAP WS ORDER FILLED %s SIDE %s TRADE SIZE %v TRADE PRICE %f",
						swapOrder.Symbol, swapOrder.Direction, swapOrder.TradeVolume, swapOrder.TradeAvgPrice,
					)
					hbcrossswapHttpPositionUpdateSilentTimes[swapOrder.Symbol] = time.Now().Add(*hbConfig.PullInterval * 3)
					if swapOrder.Direction == hbcrossswap.OrderDirectionSell {
						if spotSymbol, ok := hbSwapSpotSymbolsMap[swapOrder.Symbol]; ok {
							if spotPrice, ok := hbspotLastFilledBuyPrices[spotSymbol]; ok {
								hbRealisedSpread[spotSymbol] = (swapOrder.TradeAvgPrice - spotPrice) / spotPrice
								logger.Debugf("%s %s REALISED OPEN SPREAD %f", spotSymbol, swapOrder.Symbol, hbRealisedSpread[spotSymbol])
							}
						}
					} else if swapOrder.Direction == hbcrossswap.OrderDirectionBuy {
						if spotSymbol, ok := hbSwapSpotSymbolsMap[swapOrder.Symbol]; ok {
							if spotPrice, ok := hbspotLastFilledSellPrices[spotSymbol]; ok {
								hbRealisedSpread[spotSymbol] = (swapOrder.TradeAvgPrice - spotPrice) / spotPrice
								logger.Debugf("%s %s REALISED CLOSE SPREAD %f", spotSymbol, swapOrder.Symbol, hbRealisedSpread[spotSymbol])
							}
						}
					}
				}
			}
			break
		case spread := <-spreadCh:
			hbSpreads[spread.MakerSymbol] = spread
			//hbLoopTimer.Reset(time.Nanosecond)
			break
		case hbcrossswapFundingRates = <-hbcrossswapFundingRatesCh:
			//logger.Debugf("FRS %v", hbcrossswapFundingRates)
			break
		case hbcrossswapBarsMap = <-hbcrossswapBarsMapCh:
			if hbBarsMapUpdated["spot"] {
				hbBarsMapCh <- [2]common.KLinesMap{hbspotBarsMap, hbcrossswapBarsMap}
				hbBarsMapUpdated["spot"] = false
				hbBarsMapUpdated["swap"] = false
			} else {
				hbBarsMapUpdated["swap"] = true
			}
			break
		case hbspotBarsMap = <-hbspotBarsMapCh:
			if hbBarsMapUpdated["swap"] {
				hbBarsMapCh <- [2]common.KLinesMap{hbspotBarsMap, hbcrossswapBarsMap}
				hbBarsMapUpdated["spot"] = false
				hbBarsMapUpdated["swap"] = false
			} else {
				hbBarsMapUpdated["spot"] = true
			}
			break
		case qs := <-hbQuantilesCh:
			if hbQuantiles == nil {
				logger.Debugf("QUANTILES %v", qs)
			}
			hbQuantiles = qs
			hbLoopTimer.Reset(time.Millisecond)
			break
		case <-influxSaveTimer.C:
			handleSave()
			influxSaveTimer.Reset(
				time.Now().Truncate(
					*hbConfig.InternalInflux.SaveInterval,
				).Add(
					*hbConfig.InternalInflux.SaveInterval,
				).Sub(time.Now()),
			)
			break
		case <-externalInfluxSaveTimer.C:
			handleExternalInfluxSave()
			externalInfluxSaveTimer.Reset(
				time.Now().Truncate(
					*hbConfig.ExternalInflux.SaveInterval,
				).Add(
					*hbConfig.ExternalInflux.SaveInterval,
				).Sub(time.Now()),
			)
			break

		case newError := <-hbcrossswapNewOrderErrorCh:
			hbcrossswapPositionsUpdateTimes[newError.Params.Symbol] = time.Now()
			hbcrossswapOrderSilentTimes[newError.Params.Symbol] = time.Now().Add(time.Second * 15)
			break

		case order := <-hbspotNewOrderErrorCh:
			if openOrder, ok := hbspotOpenOrders[order.Params.Symbol]; ok && openOrder.ClientOrderID == order.Params.ClientOrderID {
				delete(hbspotOpenOrders, order.Params.Symbol)
			}
			hbspotOrderSilentTimes[order.Params.Symbol] = time.Now().Add(*hbConfig.OrderSilent * 3)
		case <-frRankUpdatedTimer.C:
			frs := make([]float64, len(hbcrossswapSymbols))
			for i, symbol := range hbcrossswapSymbols {
				if fr, ok := hbcrossswapFundingRates[symbol]; ok {
					frs[i] = fr.FundingRate
				} else {
					logger.Debugf("MISS FUNDING RATE %s", symbol)
					break
				}
			}
			if len(hbRankSymbolMap) == 0 {
				logger.Debugf("RANK FR...")
			}
			hbRankSymbolMap, err = common.RankSymbols(hbcrossswapSymbols, frs)
			if err != nil {
				logger.Debugf("RankSymbols error %v", err)
			}
			//logger.Debugf("SYMBOLS FR RANK %v", hbRankSymbolMap)
			frRankUpdatedTimer.Reset(time.Minute)
			break
		case <-hbLoopTimer.C:
			if hbcrossswapSystemReady && hbspotSystemReady && time.Now().Sub(hbGlobalSilent) > 0 {
				updatePerpPositions()
				updateSpotOldOrders()
				if time.Now().Sub(time.Now().Truncate(fundingInterval)) > fundingSilent &&
					time.Now().Truncate(fundingInterval).Add(fundingInterval).Sub(time.Now()) > fundingSilent {
					updateSpotNewOrders()
				}
			} else {
				if len(hbspotOpenOrders) > 0 {
					for spotSymbol := range hbspotOpenOrders {
						select {
						case hbspotOrderRequestChs[spotSymbol] <- SpotOrderRequest{
							Cancel: &hbspot.CancelAllParam{Symbol: spotSymbol},
						}:
							delete(hbspotOpenOrders, spotSymbol)
						default:
						}
					}
				}
			}
			hbLoopTimer.Reset(
				time.Now().Truncate(
					*hbConfig.LoopInterval,
				).Add(
					*hbConfig.LoopInterval,
				).Sub(time.Now()),
			)
			break
		}
	}
}
