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

	if *bnConfig.CpuProfile != "" {
		f, err := os.Create(*bnConfig.CpuProfile)
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
	bnAPI, err = bnswap.NewAPI(
		&common.Credentials{
			Key:    *bnConfig.ApiKey,
			Secret: *bnConfig.ApiSecret,
		},
		*bnConfig.ProxyAddress,
	)
	if err != nil {
		logger.Debugf("bnswap.NewAPI error %v", err)
		return
	}

	bnGlobalCtx, bnGlobalCancel = context.WithCancel(context.Background())
	defer bnGlobalCancel()

	if *bnConfig.ChangeLeverage {
		for _, takerSymbol := range bnSymbols[:*bnConfig.TradeSymbolIndex] {
			res, err := bnAPI.UpdateLeverage(bnGlobalCtx, bnswap.UpdateLeverageParams{
				Symbol:   takerSymbol,
				Leverage: int64(*bnConfig.Leverage),
			})
			if err != nil {
				logger.Debugf("UPDATE LEVERAGE FOR %s ERROR %v", takerSymbol, err)
			} else {
				logger.Debugf("UPDATE LEVERAGE FOR %s RESPONSE %v", takerSymbol, res)
			}
			time.Sleep(time.Second)
			res, err = bnAPI.UpdateMarginType(bnGlobalCtx, bnswap.UpdateMarginTypeParams{
				Symbol:     takerSymbol,
				MarginType: *bnConfig.MarginType,
			})
			if err != nil {
				logger.Debugf("UPDATE MARGIN TYPE FOR %s ERROR %v", takerSymbol, err)
			} else {
				logger.Debugf("UPDATE MARGIN TYPE FOR %s RESPONSE %v", takerSymbol, res)
			}
			time.Sleep(time.Second)
		}
	}

	bnTickSizes, bnStepSizes, _, bnMinNotional, _, _, err = bnswap.GetOrderLimits(bnGlobalCtx, bnAPI, bnSymbols[:*bnConfig.TradeSymbolIndex])
	if err != nil {
		logger.Debugf("bnswap.GetOrderLimits error %v", err)
		return
	}

	bnInternalInfluxWriter, err = common.NewInfluxWriter(
		*bnConfig.InternalInflux.Address,
		*bnConfig.InternalInflux.Username,
		*bnConfig.InternalInflux.Password,
		*bnConfig.InternalInflux.Database,
		*bnConfig.InternalInflux.BatchSize,
	)
	if err != nil {
		logger.Debugf("common.NewInfluxWriter error %v", err)
		return
	}
	defer func() {
		err := bnInternalInfluxWriter.Stop()
		if err != nil {
			logger.Warnf("stop influx writer error %v", err)
		}
	}()

	bnExternalInfluxWriter, err = common.NewInfluxWriter(
		*bnConfig.ExternalInflux.Address,
		*bnConfig.ExternalInflux.Username,
		*bnConfig.ExternalInflux.Password,
		*bnConfig.ExternalInflux.Database,
		*bnConfig.ExternalInflux.BatchSize,
	)
	if err != nil {
		logger.Debugf("common.NewInfluxWriter error %v", err)
		return
	}
	defer func() {
		err := bnExternalInfluxWriter.Stop()
		if err != nil {
			logger.Warnf("stop influx writer error %v", err)
		}
	}()

	bnUserWebsocket, err = bnswap.NewUserWebsocket(
		bnGlobalCtx,
		bnAPI,
		*bnConfig.ProxyAddress,
	)
	if err != nil {
		logger.Debugf("bnswap.NewUserWebsocket error %v", err)
		return
	}
	defer bnUserWebsocket.Stop()

	influxSaveTimer := time.NewTimer(
		time.Now().Truncate(
			*bnConfig.InternalInflux.SaveInterval,
		).Add(
			*bnConfig.InternalInflux.SaveInterval * 3,
		).Sub(time.Now()),
	)
	externalInfluxSaveTimer := time.NewTimer(
		time.Now().Truncate(
			*bnConfig.ExternalInflux.SaveInterval,
		).Add(
			*bnConfig.ExternalInflux.SaveInterval * 3,
		).Sub(time.Now()),
	)
	bnLoopTimer = time.NewTimer(time.Second) //先等1分钟
	defer influxSaveTimer.Stop()
	defer bnLoopTimer.Stop()
	defer externalInfluxSaveTimer.Stop()

	go bnswap.WatchAccountFromHttp(
		bnGlobalCtx, bnAPI,
		*bnConfig.PullInterval, bnAccountCh,
	)
	go bnswap.WatchPositionsFromHttp(
		bnGlobalCtx, bnAPI,
		bnSymbols[:*bnConfig.TradeSymbolIndex],
		*bnConfig.PullInterval, bnPositionsCh,
	)

	go watchHighLowQuantile(
		bnGlobalCtx,
		bnAPI,
		bnSymbols[:*bnConfig.TradeSymbolIndex],
		*bnConfig.BarsLookback,
		*bnConfig.PullBarsInterval,
		*bnConfig.PullBarsRetryInterval,
		*bnConfig.RequestInterval,
		*bnConfig.QuantileOffset,
		*bnConfig.DirWindow,
		bnQuantilesCh,
	)

	for start := 0; start < len(bnSymbols); start += *bnConfig.DepthBatchSize {
		end := start + *bnConfig.DepthBatchSize
		if end > len(bnSymbols) {
			end = len(bnSymbols)
		}
		go watchDepthWebsocket(
			bnGlobalCtx,
			bnGlobalCancel,
			*bnConfig.TimeDecay,
			*bnConfig.TimeBias,
			*bnConfig.ProxyAddress,
			bnSymbols[start:end],
			bnSymbols[:*bnConfig.TradeSymbolIndex],
			bnBidPriceCh,
			bnTimeEmaDeltaCh,
		)
	}

	bnNewOrderErrorCh = make(chan OrderNewError, len(bnSymbols)*2)
	for _, symbol := range bnSymbols[:*bnConfig.TradeSymbolIndex] {
		bnOrderRequestChs[symbol] = make(chan OrderRequest, 2)
		go watchTakerOrderRequest(
			bnGlobalCtx,
			bnAPI,
			*bnConfig.OrderTimeout,
			*bnConfig.DryRun,
			bnOrderRequestChs[symbol],
			bnNewOrderErrorCh,
		)
	}

	if *bnConfig.CpuProfile != "" {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			sig := <-sigs
			logger.Debugf("CATCH EXIT SIGNAL %v", sig)
			bnGlobalCancel()
		}()
	}

	go func() {
		for _, symbol := range bnSymbols[:*bnConfig.TradeSymbolIndex] {
			select {
			case <-bnGlobalCtx.Done():
				return
			case <-time.After(*bnConfig.RequestInterval):
				logger.Debugf("INITIAL CANCEL ALL %s", symbol)
				select {
				case <-bnGlobalCtx.Done():
					return
				case bnOrderRequestChs[symbol] <- OrderRequest{
					Cancel: &bnswap.CancelAllOrderParams{
						Symbol: symbol,
					},
				}:
				}
			}
		}
	}()

	logger.Debugf("START MAIN LOOP")
	for {
		select {
		case <-bnGlobalCtx.Done():
			logger.Debugf("GLOBAL CTX DONE, EXIT MAIN LOOP")
			return
		case ema := <-bnTimeEmaDeltaCh:
			logger.Debugf("EMA %v", ema)
			bnTimeEmaDelta = &ema
			if ema > *bnConfig.EnterThreshold {
				bnSystemOverHeated = true
			} else if ema < *bnConfig.EnterThreshold {
				bnSystemOverHeated = false
			}
		case <-bnUserWebsocket.Done():
			logger.Debugf("MAKER USER WS DONE, EXIT MAIN LOOP")
			return
		case account := <-bnAccountCh:
			handleTakerHttpAccount(account)
			break
		case ps := <-bnPositionsCh:
			handleTakerHttpPositions(ps)
			break
		case msg := <-bnUserWebsocket.BalanceAndPositionUpdateEventCh:
			handleTakerWSAccount(msg)
			break
		case oderEvent := <-bnUserWebsocket.OrderUpdateEventCh:
			order := oderEvent.Order
			if order.Status == "REJECTED" || order.Status == "EXPIRED" {
				logger.Debugf("BNSWAP WS ORDER %s %s", order.Symbol, order.Status)
				bnOrderSilentTimes[order.Symbol] = time.Now().Add(time.Second)
				bnPositionsUpdateTimes[order.Symbol] = time.Unix(0, 0)
				if openOrder, ok := bnOpenOrders[order.Symbol]; ok && openOrder.NewClientOrderId == order.ClientOrderId {
					delete(bnOpenOrders, order.Symbol)
				}
			} else if order.Status == "FILLED" {
				logger.Debugf("BNSWAP WS ORDER %s %s %f %f", order.Symbol, order.Status, order.FilledAccumulatedQuantity, order.AveragePrice)
				bnHttpPositionUpdateSilentTimes[order.Symbol] = time.Now().Add(*bnConfig.HttpSilent)
				if symbol, ok := bnSymbolsMap[order.Symbol]; ok {
					if order.Side == "SELL" {
						if buyPrice, ok := bnLastBuyCosts[symbol]; ok {
							bnRealisedProfitPcts[symbol] = (order.AveragePrice - buyPrice) / buyPrice
							logger.Debugf("%s REALISED SHORT SPREAD %f", symbol, bnRealisedProfitPcts[symbol])
						}
					} else if order.Side == common.OrderSideBuy {
						bnLastBuyCosts[order.Symbol] = order.AveragePrice
					}
				}
				if openOrder, ok := bnOpenOrders[order.Symbol]; ok && openOrder.NewClientOrderId == order.ClientOrderId {
					delete(bnOpenOrders, order.Symbol)
				}
			}
			break
		case qs := <-bnQuantilesCh:
			logger.Debugf("%v", qs)
			bnQuantiles[qs.Symbol] = qs
			break
		case <-influxSaveTimer.C:
			handleSave()
			influxSaveTimer.Reset(
				time.Now().Truncate(
					*bnConfig.InternalInflux.SaveInterval,
				).Add(
					*bnConfig.InternalInflux.SaveInterval,
				).Sub(time.Now()),
			)
			break
		case <-externalInfluxSaveTimer.C:
			handleExternalInfluxSave()
			externalInfluxSaveTimer.Reset(
				time.Now().Truncate(
					*bnConfig.ExternalInflux.SaveInterval,
				).Add(
					*bnConfig.ExternalInflux.SaveInterval,
				).Sub(time.Now()),
			)
			break
		case takerNewError := <-bnNewOrderErrorCh:
			bnOrderSilentTimes[takerNewError.Params.Symbol] = time.Now().Add(*bnConfig.OrderSilent * 5)
			break
		case <-bnLoopTimer.C:
			updatePositions()
			bnLoopTimer.Reset(
				time.Now().Truncate(
					*bnConfig.LoopInterval,
				).Add(
					*bnConfig.LoopInterval,
				).Sub(time.Now()),
			)
			break
		}
	}
}
