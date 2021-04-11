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
			logger.Fatal(err)
		}
		err = pprof.StartCPUProfile(f)
		if err != nil {
			logger.Fatal(err)
		}
		defer pprof.StopCPUProfile()
	}

	var err error
	bnswapAPI, err = bnswap.NewAPI(&common.Credentials{
		Key:    *bnConfig.ApiKey,
		Secret: *bnConfig.ApiSecret,
	}, *bnConfig.ProxyAddress)
	if err != nil {
		logger.Fatal(err)
	}

	bnGlobalCtx, bnGlobalCancel = context.WithCancel(context.Background())
	defer bnGlobalCancel()

	bnswapTickSizes, bnswapStepSizes, _, bnswapMinNotional, _, _, err = bnswap.GetOrderLimits(bnGlobalCtx, bnswapAPI, bnSymbols)
	if err != nil {
		logger.Fatal(err)
	}
	//bnspotTickSizes, bnspotStepSizes, _, bnspotMinNotional, err = bnspot.GetOrderLimits(bnGlobalCtx, bnspotAPI, bnSymbols)
	//if err != nil {
	//	logger.Fatal(err)
	//}

	bnInternalInfluxWriter, err = common.NewInfluxWriter(
		*bnConfig.InternalInflux.Address,
		*bnConfig.InternalInflux.Username,
		*bnConfig.InternalInflux.Password,
		*bnConfig.InternalInflux.Database,
		*bnConfig.InternalInflux.BatchSize,
	)
	if err != nil {
		logger.Fatal(err)
	}

	bnExternalInfluxWriter, err = common.NewInfluxWriter(
		*bnConfig.ExternalInflux.Address,
		*bnConfig.ExternalInflux.Username,
		*bnConfig.ExternalInflux.Password,
		*bnConfig.ExternalInflux.Database,
		*bnConfig.ExternalInflux.BatchSize,
	)
	if err != nil {
		logger.Fatal(err)
	}

	defer func() {
		err := bnInternalInfluxWriter.Stop()
		if err != nil {
			logger.Warnf("stop influx writer error %v", err)
		}
	}()

	if *bnConfig.ChangeLeverage {
		for _, symbol := range bnSymbols {
			res, err := bnswapAPI.UpdateLeverage(bnGlobalCtx, bnswap.UpdateLeverageParams{
				Symbol:   symbol,
				Leverage: int64(*bnConfig.Leverage),
			})
			if err != nil {
				logger.Debugf("UPDATE LEVERAGE FOR %s ERROR %v", symbol, err)
			} else {
				logger.Debugf("UPDATE LEVERAGE FOR %s RESPONSE %v", symbol, res)
			}
			time.Sleep(time.Second)
			res, err = bnswapAPI.UpdateMarginType(bnGlobalCtx, bnswap.UpdateMarginTypeParams{
				Symbol:     symbol,
				MarginType: *bnConfig.MarginType,
			})
			if err != nil {
				logger.Debugf("UPDATE MARGIN TYPE FOR %s ERROR %v", symbol, err)
			} else {
				logger.Debugf("UPDATE MARGIN TYPE FOR %s RESPONSE %v", symbol, res)
			}
			time.Sleep(time.Second)
		}
	}

	//bnspotUserWebsocket = bnspot.NewUserWebsocket(
	//	bnGlobalCtx,
	//	bnspotAPI,
	//	*bnConfig.ProxyAddress,
	//)
	//defer bnspotUserWebsocket.Stop()

	bnswapUserWebsocket = bnswap.NewUserWebsocket(
		bnGlobalCtx,
		bnswapAPI,
		*bnConfig.ProxyAddress,
	)
	defer bnswapUserWebsocket.Stop()

	influxSaveTimer := time.NewTimer(
		time.Now().Truncate(
			*bnConfig.InternalInflux.SaveInterval,
		).Add(
			*bnConfig.InternalInflux.SaveInterval * 3,
		).Sub(time.Now()),
	)
	loopTimer := time.NewTimer(time.Second)
	frRankUpdatedTimer := time.NewTimer(time.Second * 60)


	defer influxSaveTimer.Stop()
	defer loopTimer.Stop()
	defer frRankUpdatedTimer.Stop()

	go bnswap.WatchPositionsFromHttp(
		bnGlobalCtx, bnswapAPI,
		bnSymbols, *bnConfig.PullInterval, bnswapPositionCh,
	)
	go bnswap.WatchAccountFromHttp(
		bnGlobalCtx, bnswapAPI,
		*bnConfig.PullInterval, bnswapAccountCh,
	)

	swapMarkPriceCh := make(chan *bnswap.MarkPrice, len(bnSymbols))
	for start := 0; start < len(bnSymbols); start += *bnConfig.SymbolsBatchSize {
		end := start + *bnConfig.SymbolsBatchSize
		if end > len(bnSymbols) {
			end = len(bnSymbols)
		}
		go watchMarkPrice(
			bnGlobalCtx, *bnConfig.ProxyAddress,
			bnSymbols[start:end],
			swapMarkPriceCh,
		)
	}


	bnswapNewOrderResponseCh = make(chan bnswap.Order, len(bnSymbols)*2)
	bnswapNewOrderErrorCh = make(chan SwapOrderNewError, len(bnSymbols)*2)
	for _, symbol := range bnSymbols {
		bnswapOrderNewChs[symbol] = make(chan bnswap.NewOrderParams, 2)
		go watchSwapOrderRequest(
			bnGlobalCtx,
			bnswapAPI,
			*bnConfig.OrderTimeout,
			*bnConfig.DryRun,
			bnswapOrderNewChs[symbol],
			bnswapNewOrderErrorCh,
			bnswapNewOrderResponseCh,
		)
	}

	done := make(chan bool, 1)
	if *bnConfig.CpuProfile != "" {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			sig := <-sigs
			logger.Debugf("Exit with sig %d, clean *.tmp files", sig)
			done <- true
		}()
	}

	for {
		select {
		case <-done:
			logger.Debugf("Exit")
			return
		case p := <-bnswapPositionCh:
			handleSwapHttpPositions(p)
			break
		case account := <-bnswapAccountCh:
			handleSwapHttpAccount(account)
			break
		case msg := <-bnswapUserWebsocket.BalanceAndPositionUpdateEventCh:
			handleWSAccountEvent(msg)
			break
		case msg := <-bnswapUserWebsocket.OrderUpdateEventCh:
			handleWSOrder(&msg.Order)
			break
		case markPrice := <-swapMarkPriceCh:
			bnswapMarkPrices[markPrice.Symbol] = *markPrice
			break
		case <-frRankUpdatedTimer.C:
			frs := make([]float64, len(bnSymbols))
			for i, symbol := range bnSymbols {
				if markPrice, ok := bnswapMarkPrices[symbol]; ok {
					frs[i] = markPrice.FundingRate
				} else {
					logger.Debugf("MISS MARK PRICE %s", symbol)
					return
				}
			}
			bnRankSymbolMap, err = common.RankSymbols(bnSymbols, frs)
			if err != nil {
				logger.Debugf("RankSymbols error %v", err)
			}
			logger.Debugf("SYMBOLS FR RANK %v", bnRankSymbolMap)
			frRankUpdatedTimer.Reset(time.Minute)
		case <-influxSaveTimer.C:
			handleSave()
			handleExternalInfluxSave()
			influxSaveTimer.Reset(
				time.Now().Truncate(
					*bnConfig.InternalInflux.SaveInterval,
				).Add(
					*bnConfig.InternalInflux.SaveInterval,
				).Sub(time.Now()),
			)
			break
		case newError := <-bnswapOrderNewErrorCh:
			order := newError.Params
			bnswapOrderSilentTimes[order.Symbol] = time.Now()
			break
		case order := <-bnswapOrderFinishCh:
			if order.Status == common.OrderStatusReject || order.Status == common.OrderStatusExpired {
				bnswapOrderSilentTimes[order.Symbol] = time.Now().Add(time.Second)
				bnswapPositionsUpdateTimes[order.Symbol] = time.Unix(0, 0)
			}
			break
		case order := <-bnswapNewOrderResponseCh:
			if order.Status == common.OrderStatusReject ||
				order.Status == common.OrderStatusExpired ||
				order.Status == common.OrderStatusCancelled {
				bnswapOrderSilentTimes[order.Symbol] = time.Now()
				bnswapPositionsUpdateTimes[order.Symbol] = time.Unix(0, 0)
			} else if order.Status == common.OrderStatusFilled {
			}
		case order := <-bnswapNewOrderErrorCh:
			bnswapOrderSilentTimes[order.Params.Symbol] = time.Now().Add(*bnConfig.OrderSilent * 3)
			break
		case <-loopTimer.C:
			updateSwapPosition()
			loopTimer.Reset(
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
