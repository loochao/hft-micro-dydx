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
			logger.Fatal(err)
		}
		err = pprof.StartCPUProfile(f)
		if err != nil {
			logger.Fatal(err)
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
		logger.Fatal(err)
	}
	hbspotAPI, err = hbspot.NewAPI(
		*hbConfig.ApiKey,
		*hbConfig.ApiSecret,
		*hbConfig.ProxyAddress,
	)
	if err != nil {
		logger.Fatal(err)
	}

	hbGlobalCtx, hbGlobalCancel = context.WithCancel(context.Background())
	defer hbGlobalCancel()

	accounts, err := hbspotAPI.GetAccounts(hbGlobalCtx)
	if err != nil {
		logger.Fatal(err)
	}
	for _, a := range accounts {
		if a.Type == "spot" {
			hbspotAccountID = a.ID
		}
	}
	if hbspotAccountID == 0 {
		logger.Fatal("HB SPOT ACCOUNT ID NOT EXISTS!!!")
	}

	hbcrossswapTickSizes, hbcrossswapContractSizes, err = hbcrossswap.GetOrderLimits(hbGlobalCtx, hbcrossswapAPI, hbcrossswapSymbols)
	if err != nil {
		logger.Fatal(err)
	}
	hbspotTickSizes, hbspotStepSizes, hbspotMinSizes, hbspotMinNotional, hbspotPricePrecisions, hbspotAmountPrecisions, err = hbspot.GetOrderLimits(hbGlobalCtx, hbspotAPI, hbspotSymbols)
	if err != nil {
		logger.Fatal(err)
	}

	hbInfluxWriter, err = common.NewInfluxWriter(
		*hbConfig.InternalInflux.Address,
		*hbConfig.InternalInflux.Username,
		*hbConfig.InternalInflux.Password,
		*hbConfig.InternalInflux.Database,
		*hbConfig.InternalInflux.BatchSize,
	)
	if err != nil {
		logger.Fatal(err)
	}

	kcExternalInfluxWriter, err = common.NewInfluxWriter(
		*hbConfig.ExternalInflux.Address,
		*hbConfig.ExternalInflux.Username,
		*hbConfig.ExternalInflux.Password,
		*hbConfig.ExternalInflux.Database,
		*hbConfig.ExternalInflux.BatchSize,
	)
	if err != nil {
		logger.Fatal(err)
	}

	defer func() {
		err := hbInfluxWriter.Stop()
		if err != nil {
			logger.Warnf("stop influx writer error %v", err)
		}
	}()

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
	loopTimer := time.NewTimer(time.Second) //先等1分钟
	frRankUpdatedTimer := time.NewTimer(time.Second * 15)

	defer influxSaveTimer.Stop()
	defer loopTimer.Stop()
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
		kcspSymbolsMap,
		*hbConfig.BotQuantile,
		*hbConfig.TopQuantile,
		*hbConfig.TopBandScale,
		*hbConfig.BotBandScale,
		*hbConfig.MinimalEnterDelta,
		*hbConfig.MaximalExitDelta,
		*hbConfig.MinimalBandOffset,
		kcBarsMapCh,
		kcQuantilesCh,
	)

	walkedOrderBookCh := make(chan WalkedOrderBook, len(hbspotSymbols)*10)
	for start := 0; start < len(hbspotSymbols); start += *hbConfig.OrderBookBatchSize {
		end := start + *hbConfig.OrderBookBatchSize
		if end > len(hbspotSymbols) {
			end = len(hbspotSymbols)
		}
		go watchSpotWalkedOrderBooks(
			hbGlobalCtx,
			*hbConfig.ProxyAddress,
			*hbConfig.OrderBookTakerImpact,
			*hbConfig.OrderBookMakerImpact,
			hbspotSymbols[start:end],
			walkedOrderBookCh,
		)
	}

	for start := 0; start < len(hbcrossswapSymbols); start += *hbConfig.OrderBookBatchSize {
		end := start + *hbConfig.OrderBookBatchSize
		if end > len(hbspotSymbols) {
			end = len(hbspotSymbols)
		}
		go watchSwapWalkedOrderBooks(
			hbGlobalCtx,
			*hbConfig.ProxyAddress,
			hbcrossswapContractSizes,
			*hbConfig.OrderBookTakerImpact,
			*hbConfig.OrderBookMakerImpact,
			hbcrossswapSymbols[start:end],
			walkedOrderBookCh,
		)
	}

	spreadCh := make(chan Spread, len(hbspotSymbols)*100)
	go watchSpread(
		hbGlobalCtx,
		hbspotSymbols,
		kcpsSymbolsMap,
		*hbConfig.OrderBookMaxAgeDiff,
		*hbConfig.OrderBookMaxAge,
		*hbConfig.SpreadLookbackDuration,
		*hbConfig.SpreadLookbackMinimalWindow,
		walkedOrderBookCh,
		spreadCh,
	)

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

	done := make(chan bool, 1)
	if *hbConfig.CpuProfile != "" {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			sig := <-sigs
			logger.Debugf("Exit with sig %d, clean *.tmp files", sig)
			done <- true
		}()
	}

	logger.Debugf("START")

	for {
		select {
		case <-done:
			logger.Debugf("Exit")
			return
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
			if spotOrder.OrderStatus != nil {
				if *spotOrder.OrderStatus == hbspot.OrderStatusFilled {
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
					if swapOrder.Direction == hbcrossswap.OrderDirectionSell {
						if spotSymbol, ok := kcpsSymbolsMap[swapOrder.Symbol]; ok {
							if spotPrice, ok := hbspotLastFilledBuyPrices[spotSymbol]; ok {
								kcRealisedSpread[spotSymbol] = (swapOrder.TradeAvgPrice - spotPrice) / spotPrice
								logger.Debugf("%s %s REALISED OPEN SPREAD %f", spotSymbol, swapOrder.Symbol, kcRealisedSpread[spotSymbol])
							}
						}
					} else if swapOrder.Direction == hbcrossswap.OrderDirectionBuy {
						if spotSymbol, ok := kcpsSymbolsMap[swapOrder.Symbol]; ok {
							if spotPrice, ok := hbspotLastFilledSellPrices[spotSymbol]; ok {
								kcRealisedSpread[spotSymbol] = (swapOrder.TradeAvgPrice - spotPrice) / spotPrice
								logger.Debugf("%s %s REALISED CLOSE SPREAD %f", spotSymbol, swapOrder.Symbol, kcRealisedSpread[spotSymbol])
							}
						}
					}
				}
			}
			break
		case spread := <-spreadCh:
			hbSpreads[spread.Symbol] = spread
			loopTimer.Reset(time.Nanosecond)
			break
		case hbcrossswapFundingRates = <-hbcrossswapFundingRatesCh:
			//logger.Debugf("FRS %v", hbcrossswapFundingRates)
			break
		case hbcrossswapBarsMap = <-hbcrossswapBarsMapCh:
			if kcBarsMapUpdated["spot"] {
				kcBarsMapCh <- [2]common.KLinesMap{hbspotBarsMap, hbcrossswapBarsMap}
				kcBarsMapUpdated["spot"] = false
				kcBarsMapUpdated["swap"] = false
			} else {
				kcBarsMapUpdated["swap"] = true
			}
			break
		case hbspotBarsMap = <-hbspotBarsMapCh:
			if kcBarsMapUpdated["swap"] {
				kcBarsMapCh <- [2]common.KLinesMap{hbspotBarsMap, hbcrossswapBarsMap}
				kcBarsMapUpdated["spot"] = false
				kcBarsMapUpdated["swap"] = false
			} else {
				kcBarsMapUpdated["spot"] = true
			}
			break
		case qs := <-kcQuantilesCh:
			if kcQuantiles == nil {
				logger.Debugf("QUANTILES %v", qs)
			}
			kcQuantiles = qs
			loopTimer.Reset(time.Millisecond)
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
			if len(kcRankSymbolMap) == 0 {
				logger.Debugf("RANK FR...")
			}
			kcRankSymbolMap, err = common.RankSymbols(hbcrossswapSymbols, frs)
			if err != nil {
				logger.Debugf("RankSymbols error %v", err)
			}
			//logger.Debugf("SYMBOLS FR RANK %v", kcRankSymbolMap)
			frRankUpdatedTimer.Reset(time.Minute)
			break
		case <-loopTimer.C:
			updatePerpPositions()
			updateSpotOldOrders()
			updateSpotNewOrders()
			loopTimer.Reset(
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
