package main

import (
	"context"
	"fmt"
	"github.com/geometrybase/hft/bnswap"
	"github.com/geometrybase/hft/common"
	"github.com/geometrybase/hft/logger"
	"github.com/geometrybase/hft/okspot"
	"strings"
	"time"
)

func main() {

	var err error
	bnswapAPI, err = bnswap.NewAPI(*boConfig.ProxyAddress)
	if err != nil {
		logger.Fatal(err)
	}
	okspotAPI, err = okspot.NewAPI(*boConfig.OkApiUrl, *boConfig.ProxyAddress)
	if err != nil {
		logger.Fatal(err)
	}

	boGlobalCtx, boGlobalCancel = context.WithCancel(context.Background())
	defer boGlobalCancel()

	bnswapTickSizes, bnswapStepSizes, _, bnswapMinNotional, _, _, err = bnswap.GetOrderLimits(boGlobalCtx, bnswapAPI, boSymbols)
	if err != nil {
		logger.Fatal(err)
	}
	boSymbols, okspotTickSizes, okspotStepSizes, okspotMinSizes, err = getOkOrderLimits(boGlobalCtx, okspotAPI, boSymbolsMap)
	if err != nil {
		logger.Fatal(err)
	}

	boSymbolsMap = make(map[string]bool)
	for _, symbol := range boSymbols {
		boSymbolsMap[symbol] = true
		bnswapOrderSilentTimes[symbol] = time.Now()
		bnswapPositionsUpdated[symbol] = false
		okspotOrderSilentTimes[symbol] = time.Now()
		okspotBalancesUpdated[symbol] = false
		boEnterDeltaWindows[symbol] = make([]float64, 0)
		boExitDeltaWindows[symbol] = make([]float64, 0)
		boArrivalTimes[symbol] = make([]time.Time, 0)
		boEnterDeltaSortedSlices[symbol] = common.SortedFloatSlice{}
		boExitDeltaSortedSlices[symbol] = common.SortedFloatSlice{}
		bnSymbolReady[symbol] = false
		bnswapOrderBooksReady[symbol] = false
		okspotOrderBooksReady[symbol] = false
		bnswapOrderBookTimestamps[symbol] = time.Unix(0, 0)
		okspotOrderBookTimestamps[symbol] = time.Unix(0, 0)
		bnOpenLogSilentTimes[symbol] = time.Now()
		okspotEnterSilentTimes[symbol] = time.Now().Add(time.Minute * 5)
		okspotExitSilentTimes[symbol] = time.Now().Add(time.Minute * 5)
	}

	boInfluxWriter, err = common.NewInfluxWriter(
		*boConfig.InternalInflux.Address,
		*boConfig.InternalInflux.Username,
		*boConfig.InternalInflux.Password,
		*boConfig.InternalInflux.Database,
		*boConfig.InternalInflux.BatchSize,
	)
	if err != nil {
		logger.Fatal(err)
	}

	boExternalInfluxWriter, err = common.NewInfluxWriter(
		*boConfig.ExternalInflux.Address,
		*boConfig.ExternalInflux.Username,
		*boConfig.ExternalInflux.Password,
		*boConfig.ExternalInflux.Database,
		*boConfig.ExternalInflux.BatchSize,
	)
	if err != nil {
		logger.Fatal(err)
	}

	defer func() {
		err := boInfluxWriter.Stop()
		if err != nil {
			logger.Warnf("stop influx writer error %v", err)
		}
	}()

	bnswapCredentials = &bnswap.Credentials{
		Key:    *boConfig.BnApiKey,
		Secret: *boConfig.BnApiSecret,
	}

	if *boConfig.ChangeLeverage {
		for _, symbol := range boSymbols {
			res, err := bnswapAPI.UpdateLeverage(boGlobalCtx, bnswapCredentials, bnswap.UpdateLeverageParams{
				Symbol:   symbol,
				Leverage: int64(*boConfig.Leverage),
			})
			if err != nil {
				logger.Debugf("UPDATE LEVERAGE FOR %s ERROR %v", symbol, err)
			} else {
				logger.Debugf("UPDATE LEVERAGE FOR %s RESPONSE %v", symbol, res)
			}
			time.Sleep(time.Second)
			res, err = bnswapAPI.UpdateMarginType(boGlobalCtx, bnswapCredentials, bnswap.UpdateMarginTypeParams{
				Symbol:     symbol,
				MarginType: *boConfig.MarginType,
			})
			if err != nil {
				logger.Debugf("UPDATE MARGIN TYPE FOR %s ERROR %v", symbol, err)
			} else {
				logger.Debugf("UPDATE MARGIN TYPE FOR %s RESPONSE %v", symbol, res)
			}
			time.Sleep(time.Second)
		}
	}

	okspotCredentials = &okspot.Credentials{
		Key:        *boConfig.OkApiKey,
		Secret:     *boConfig.OkApiSecret,
		Passphrase: *boConfig.OkPassphrase,
	}

	okChannels := []string{
		"spot/account:USDT",
	}
	for _, symbol := range boSymbols {
		okChannels = append(
			okChannels,
			fmt.Sprintf(
				"spot/account:%s",
				symbol[:len(symbol)-4],
			),
		)
		okChannels = append(
			okChannels,
			fmt.Sprintf(
				"spot/order:%s",
				okspot.SymbolToInstrumentId(symbol),
			),
		)
		okChannels = append(
			okChannels,
			fmt.Sprintf(
				"spot/depth5:%s",
				okspot.SymbolToInstrumentId(symbol),
			),
		)
	}
	okspotWebsocket = okspot.NewWebsocket(boGlobalCtx, *boConfig.OkWsUrl, okspotCredentials, okChannels, *boConfig.ProxyAddress, 300)
	logger.Debugf("OKSPOT WS URL: %s", okspotWebsocket.Url)

	bnswapStreams := fmt.Sprintf("%s@markPrice@1s/", strings.ToLower(bnBNBSymbol))
	for _, symbol := range boSymbols {
		bnswapStreams += fmt.Sprintf(
			"%s@%s/%s@markPrice@1s/",
			strings.ToLower(symbol),
			*boConfig.OrderBookType,
			strings.ToLower(symbol),
		)
	}
	bnswapWebsocket = bnswap.NewWebsocket(boGlobalCtx, bnswapStreams[:len(bnswapStreams)-1], *boConfig.ProxyAddress, 300)
	logger.Debugf("BNSWAP MARKET URL: %s", bnswapWebsocket.Url)
	defer bnswapWebsocket.Stop()
	bnswapUserWebsocket = bnswap.NewUserDataWebsocket(
		boGlobalCtx,
		*boConfig.ProxyAddress,
		bnswapCredentials,
		bnswapAPI,
		1,
	)
	logger.Debugf("BNSWAP USERDATA URL: %s", bnswapUserWebsocket.Url)
	defer bnswapUserWebsocket.Stop()

	influxSaveTimer := time.NewTimer(
		time.Now().Truncate(
			*boConfig.InternalInflux.SaveInterval,
		).Add(
			*boConfig.InternalInflux.SaveInterval * 3,
		).Sub(time.Now()),
	)
	externalInfluxSaveTimer := time.NewTimer(
		time.Now().Truncate(
			*boConfig.ExternalInflux.SaveInterval,
		).Add(
			*boConfig.ExternalInflux.SaveInterval * 3,
		).Sub(time.Now()),
	)
	loopTimer := time.NewTimer(time.Hour * 24) //先等1分钟
	frRankUpdatedTimer := time.NewTimer(time.Second * 30)

	defer frRankUpdatedTimer.Stop()
	defer influxSaveTimer.Stop()
	defer loopTimer.Stop()

	go bnswap.WatchPositionsFromHttp(
		boGlobalCtx, bnswapAPI, bnswapCredentials,
		boSymbols, *boConfig.PullInterval, bnswapPositionCh,
	)
	go bnswap.WatchAccountFromHttp(
		boGlobalCtx, bnswapAPI, bnswapCredentials,
		*boConfig.PullInterval, bnswapAccountCh,
	)
	go okspot.WatchBalancesFromHttp(
		boGlobalCtx, okspotAPI, okspotCredentials,
		*boConfig.PullInterval, okspotBalancesCh,
	)

	go watchBnswapBars(
		boGlobalCtx,
		bnswapAPI,
		boSymbols,
		*boConfig.BarsLookback,
		*boConfig.PullBarsInterval,
		*boConfig.PullBarsRetryInterval,
		bnswapBarsMapCh,
	)

	go watchOkspotBars(
		boGlobalCtx,
		okspotAPI,
		boSymbols,
		*boConfig.BarsLookback,
		*boConfig.PullBarsInterval,
		*boConfig.PullBarsRetryInterval,
		okspotBarsMapCh,
	)

	go watchDeltaQuantile(
		boGlobalCtx,
		boSymbols,
		*boConfig.BotQuantile,
		*boConfig.TopQuantile,
		*boConfig.TopBandScale,
		*boConfig.BotBandScale,
		*boConfig.MinimalEnterDelta,
		*boConfig.MaximalExitDelta,
		*boConfig.MinimalBandOffset,
		bnBarsMapCh,
		bnQuantilesCh,
	)

	for {
		select {
		case <-bnswapWebsocket.Done():
			return
		case p := <-bnswapPositionCh:
			handleSwapHttpPosition(p)
			break
		case account := <-bnswapAccountCh:
			handleSwapHttpAccount(account)
			break
		case balances := <-okspotBalancesCh:
			handleSpotHttpAccount(balances)
			break
		case msg := <-bnswapUserWebsocket.DataCh:
			if msg != nil {
				switch data := msg.(type) {
				case bnswap.WSListenKeyExpiredEvent:
					logger.Debugf("SWAP WS LISTEN KEY EXPIRED, EXIT!")
					return
				case bnswap.WSAccountEvent:
					handleWSAccountEvent(&data)
				case bnswap.WSOrderEvent:
					handleWSOrder(&data.Order)
				default:
					logger.Debugf("SWAP WS OTHER %v", data)
				}
			}
			break
		case msg := <-bnswapWebsocket.DataCh:
			if msg != nil {
				switch data := msg.(type) {
				case bnswap.MarkPriceUpdate:
					bnswapMarkPrices[data.Symbol] = data
				case bnswap.PartialBookDepthStream:
					if lastOB, ok := bnswapOrderBooks[data.Symbol]; ok {
						if data.ArrivalTime.Sub(lastOB.ArrivalTime).Seconds() > 0.15 {
							bnswapOrderBooksReady[data.Symbol] = false
						} else {
							bnswapOrderBooksReady[data.Symbol] = true
						}
					}
					bnswapOrderBooks[data.Symbol] = data
				default:
					logger.Debugf("UNKNOWN SWAP MARKET DATA %v", data)
				}
			}
			break
		case msg := <-okspotWebsocket.DataCh:
			if msg != nil {
				switch data := msg.(type) {
				case []okspot.WSDepth5:
					for _, d := range data {
						symbol := okspot.InstrumentIdToSymbol(d.InstrumentID)
						if lastOB, ok := okspotOrderBooks[symbol]; ok {
							if d.Timestamp.Sub(lastOB.Timestamp).Seconds() > 0.25 {
								//logger.Debugf(
								//	"BAD SPOT DEPTH TIME DIFF %fs",
								//	d.Timestamp.Sub(lastOB.Timestamp).Seconds(),
								//)
								okspotOrderBooksReady[symbol] = false
							} else {
								okspotOrderBooksReady[symbol] = true
							}
						}
						okspotOrderBooks[symbol] = d
					}
				case []okspot.Balance:
					handleSpotWSBalances(data)
				case []okspot.WSOrder:
					handleSpotWSOrder(data)
				case okspot.ErrorEvent:
					logger.Debugf("OKSPOT WS ERROR %v", data)
					break
				case okspot.SubscribeEvent, okspot.LoginEvent:
					//logger.Debugf("OKSPOT WS EVENT %v", data)
					break
				default:
					logger.Debugf("UNKNOWN SPOT MARKET DATA %v", data)
				}
			}
			break
		case bnswapBarsMap = <-bnswapBarsMapCh:
			if bnBarsMapUpdated["spot"] {
				bnBarsMapCh <- [2]common.OhlcvsMap{okspotBarsMap, bnswapBarsMap}
				bnBarsMapUpdated["spot"] = false
				bnBarsMapUpdated["swap"] = false
			} else {
				bnBarsMapUpdated["swap"] = true
			}
			break
		case okspotBarsMap = <-okspotBarsMapCh:
			if bnBarsMapUpdated["swap"] {
				bnBarsMapCh <- [2]common.OhlcvsMap{okspotBarsMap, bnswapBarsMap}
				bnBarsMapUpdated["spot"] = false
				bnBarsMapUpdated["swap"] = false
			} else {
				bnBarsMapUpdated["spot"] = true
			}
			break
		case bnQuantiles = <-bnQuantilesCh:
			loopTimer.Reset(time.Second)
			break
		case <-influxSaveTimer.C:
			handleInternalSave()
			influxSaveTimer.Reset(
				time.Now().Truncate(
					*boConfig.InternalInflux.SaveInterval,
				).Add(
					*boConfig.InternalInflux.SaveInterval,
				).Sub(time.Now()),
			)
			break
		case <-externalInfluxSaveTimer.C:
			handleExternalSave()
			externalInfluxSaveTimer.Reset(
				time.Now().Truncate(
					*boConfig.ExternalInflux.SaveInterval,
				).Add(
					*boConfig.ExternalInflux.SaveInterval,
				).Sub(time.Now()),
			)
			break

		case newError := <-bnswapOrderNewErrorCh:
			bnswapOrderSilentTimes[newError.Params.Symbol] = time.Now().Add(time.Second * 15)
			break

		case newError := <-okspotOrderNewErrorCh:
			okspotOrderSilentTimes[okspot.InstrumentIdToSymbol(newError.Params.InstrumentId)] = time.Now().Add(time.Second * 15)
			break

		case order := <-bnswapOrderFinishCh:
			logStr := fmt.Sprintf("SWAP ORDER %s", order.ToString())
			if order.Status == "REJECTED" || order.Status == "EXPIRED" {
				logStr = fmt.Sprintf("%s RESET TIMEOUT", logStr)
				bnswapOrderSilentTimes[order.Symbol] = time.Now().Add(time.Second)
				bnswapPositionsUpdated[order.Symbol] = true
			}
			if spotPrice, ok := okspotLastFilledPrices[order.Symbol]; ok && order.Status == "FILLED" && spotPrice != 0 {
				bnRealisedDelta[order.Symbol] = (order.CumQuote - spotPrice) / spotPrice
			}
			logger.Debug(logStr)
			break

		case order := <-okspotOrderFinishCh:
			logStr := fmt.Sprintf("SPOT ORDER %v", order)
			if order.State == okspot.OrderStateFailed || order.State == okspot.OrderStateCanceled {
				logStr = fmt.Sprintf("%s RESET TIMEOUT", logStr)
				okspotOrderSilentTimes[okspot.InstrumentIdToSymbol(order.InstrumentId)] = time.Now().Add(time.Second)
				okspotBalancesUpdated[okspot.InstrumentIdToSymbol(order.InstrumentId)] = true
			} else if order.State == okspot.OrderStateFullyFilled {
				lastFilledPrice := 0.0
				if order.FilledNotional != 0 && order.FilledSize != 0 {
					lastFilledPrice = order.FilledNotional / order.FilledSize
					okspotLastFilledPrices[okspot.InstrumentIdToSymbol(order.InstrumentId)] = lastFilledPrice
				}
				logStr = fmt.Sprintf("%s FILLED PRICE %f", logStr, lastFilledPrice)
			}
			logger.Debug(logStr)
			break
		case <-frRankUpdatedTimer.C:
			frs := make([]float64, len(boSymbols))
			for i, symbol := range boSymbols {
				if markPrice, ok := bnswapMarkPrices[symbol]; ok {
					frs[i] = markPrice.FundingRate
				} else {
					logger.Debugf("MISS MARK PRICE %s", symbol)
					return
				}
			}
			bnswapFundingRateRanks = common.Rank(frs)
			frRankUpdatedTimer.Reset(time.Minute)
		case <-loopTimer.C:
			updateSwapPositions()
			updateSpotPositions()
			loopTimer.Reset(
				time.Now().Truncate(
					*boConfig.LoopInterval,
				).Add(
					*boConfig.LoopInterval,
				).Sub(time.Now()),
			)
			break
		}
	}
}
