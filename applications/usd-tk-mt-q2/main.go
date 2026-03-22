package main

import (
	"context"
	"flag"
	binance_usdtfuture "github.com/geometrybase/hft-micro/binance-usdtfuture"
	"github.com/geometrybase/hft-micro/common"
	kucoin_usdtfuture "github.com/geometrybase/hft-micro/kucoin-usdtfuture"
	"github.com/geometrybase/hft-micro/logger"
	okexv5_usdtspot "github.com/geometrybase/hft-micro/okexv5-usdtspot"
	okexv5_usdtswap "github.com/geometrybase/hft-micro/okexv5-usdtswap"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"math"
	"os"
	"os/signal"
	"runtime/pprof"
	"strings"
	"syscall"
	"time"
)

func main() {
	var xSymbols = make([]string, 0)
	var ySymbols = make([]string, 0)

	var xyGlobalCtx context.Context
	var xyGlobalCancel context.CancelFunc
	var xyInternalInfluxWriter *common.InfluxWriter
	var xyExternalInfluxWriter *common.InfluxWriter

	var xAccount common.Balance
	var xAccountCh = make(chan common.Balance, 4)
	var xOrderRequestChMap = make(map[string]chan common.OrderRequest)
	var yAccount common.Balance
	var yAccountCh = make(chan common.Balance, 4)
	var yOrderRequestChMap = make(map[string]chan common.OrderRequest)

	var xyConfig *Config
	var xExchange common.UsdExchange
	var yExchange common.UsdExchange

	var xSystemStatus = common.SystemStatusNotReady
	var ySystemStatus = common.SystemStatusNotReady
	var xSystemStatusCh = make(chan common.SystemStatus, 4)
	var ySystemStatusCh = make(chan common.SystemStatus, 4)

	configPath := flag.String("config", "", "config path")
	flag.Parse()

	if *configPath == "" {
		logger.Warn("config is empty")
		return
	}

	configFile, err := ioutil.ReadFile(*configPath)
	if err != nil {
		logger.Warn(err)
		return
	}
	var config Config
	err = yaml.Unmarshal(configFile, &config)
	if err != nil {
		logger.Warn(err)
		return
	}
	err = config.SetDefaultIfNotSet()
	if err != nil {
		logger.Warn(err)
		return
	}
	xyConfig = &config

	if xyConfig.CpuProfile != "" {
		f, err := os.Create(xyConfig.CpuProfile)
		if err != nil {
			logger.Warnf("os.Create error %v", err)
			return
		}
		err = pprof.StartCPUProfile(f)
		if err != nil {
			logger.Warnf("pprof.StartCPUProfile error %v", err)
			return
		}
		defer pprof.StopCPUProfile()
	}

	switch xyConfig.XExchange.Name {
	case "okexV5UsdtSpotWithWalkedDepth5":
		xExchange = &okexv5_usdtspot.OkexV5UsdtSpotWithWalkedDepth5{}
	case "okexV5UsdtSwapWithWalkedDepth5":
		xExchange = &okexv5_usdtswap.OkexV5UsdtSwapWithWalkedDepth5{}
	case "kucoinUsdtFutureWithWalkedDepth5":
		xExchange = &kucoin_usdtfuture.KucoinUsdtFutureWithWalkedDepth5{}
	case "binanceUsdtFutureWithWalkedDepth5":
		xExchange = &binance_usdtfuture.BinanceUsdtFutureWithWalkedDepth5{}
	default:
		logger.Warnf("unsupported exchange %s", xyConfig.XExchange.Name)
		return
	}

	switch xyConfig.YExchange.Name {
	case "okexV5UsdtSpotWithWalkedDepth5":
		yExchange = &okexv5_usdtspot.OkexV5UsdtSpotWithWalkedDepth5{}
	case "okexV5UsdtSwapWithWalkedDepth5":
		yExchange = &okexv5_usdtswap.OkexV5UsdtSwapWithWalkedDepth5{}
	case "kucoinUsdtFutureWithWalkedDepth5":
		yExchange = &kucoin_usdtfuture.KucoinUsdtFutureWithWalkedDepth5{}
	case "binanceUsdtFutureWithWalkedDepth5":
		yExchange = &binance_usdtfuture.BinanceUsdtFutureWithWalkedDepth5{}
	default:
		logger.Warnf("unsupported exchange %s", xyConfig.YExchange.Name)
		return
	}

	for xSymbol, ySymbol := range xyConfig.XYPairs {
		xSymbols = append(xSymbols, xSymbol)
		ySymbols = append(ySymbols, ySymbol)
		if _, ok := xyConfig.MaxPosSizes[xSymbol]; !ok {
			logger.Warnf("miss maximal position value for %s", xSymbol)
			return
		}
	}
	xyConfig.XExchange.Symbols = xSymbols
	xyConfig.YExchange.Symbols = ySymbols

	configStr, err := yaml.Marshal(xyConfig)
	if err != nil {
		logger.Warn(err)
		return
	}
	logger.Debug("\n\nCONFIG:")
	for _, l := range strings.Split(string(configStr), "\n") {
		logger.Debugf("%s", l)
	}
	logger.Debug("\n\n")

	logger.Debugf("X EXCHANGE ID %d", xExchange.GetExchange())
	logger.Debugf("Y EXCHANGE ID %d", yExchange.GetExchange())

	logger.Debug("\n\n")

	xyGlobalCtx, xyGlobalCancel = context.WithCancel(context.Background())
	defer xyGlobalCancel()

	err = xExchange.Setup(xyGlobalCtx, xyConfig.XExchange)
	if err != nil {
		logger.Warnf("xExchange.Setup(xyGlobalCtx, xyConfig.XExchange) error %v", err)
		return
	}
	err = yExchange.Setup(xyGlobalCtx, xyConfig.YExchange)
	if err != nil {
		logger.Warnf("yExchange.Setup(xyGlobalCtx, xyConfig.YExchange) error %v", err)
		return
	}

	if xyConfig.InternalInflux.Address != "" {
		xyInternalInfluxWriter, err = common.NewInfluxWriter(
			xyGlobalCtx,
			xyConfig.InternalInflux.Address,
			xyConfig.InternalInflux.Username,
			xyConfig.InternalInflux.Password,
			xyConfig.InternalInflux.Database,
			xyConfig.InternalInflux.BatchSize,
		)
		if err != nil {
			logger.Warnf("common.NewInfluxWriter error %v", err)
			return
		}
		defer xyInternalInfluxWriter.Stop()
	}

	if xyConfig.ExternalInflux.Address != "" {
		xyExternalInfluxWriter, err = common.NewInfluxWriter(
			xyGlobalCtx,
			xyConfig.ExternalInflux.Address,
			xyConfig.ExternalInflux.Username,
			xyConfig.ExternalInflux.Password,
			xyConfig.ExternalInflux.Database,
			xyConfig.ExternalInflux.BatchSize,
		)
		if err != nil {
			logger.Warnf("common.NewInfluxWriter error %v", err)
			return
		}
		defer xyExternalInfluxWriter.Stop()
	}

	xPositionChMap := make(map[string]chan common.Position)
	xOrderChMap := make(map[string]chan common.Order)
	xFundingRateChMap := make(map[string]chan common.FundingRate)
	xNewOrderErrorChMap := make(map[string]chan common.OrderError)
	xAccountChMap := make(map[string]chan common.Balance)
	xSystemStatusChMap := make(map[string]chan common.SystemStatus)

	xTickerChMap := make(map[string]chan common.Ticker)
	yTickerChMap := make(map[string]chan common.Ticker)

	for _, xSymbol := range xSymbols {
		xPositionChMap[xSymbol] = make(chan common.Position, 16)
		xOrderChMap[xSymbol] = make(chan common.Order, 32)
		xFundingRateChMap[xSymbol] = make(chan common.FundingRate, 16)
		xOrderRequestChMap[xSymbol] = make(chan common.OrderRequest, 16)
		xNewOrderErrorChMap[xSymbol] = make(chan common.OrderError, 16)
		xAccountChMap[xSymbol] = make(chan common.Balance, 16)
		xSystemStatusChMap[xSymbol] = make(chan common.SystemStatus, 16)

		xTickerChMap[xSymbol] = make(chan common.Ticker, 256)
		yTickerChMap[config.XYPairs[xSymbol]] = xTickerChMap[xSymbol]
	}

	yPositionChMap := make(map[string]chan common.Position)
	yOrderChMap := make(map[string]chan common.Order)
	yFundingRateChMap := make(map[string]chan common.FundingRate)
	yNewOrderErrorChMap := make(map[string]chan common.OrderError)
	yAccountChMap := make(map[string]chan common.Balance)
	ySystemStatusChMap := make(map[string]chan common.SystemStatus)
	for _, ySymbol := range ySymbols {
		yPositionChMap[ySymbol] = make(chan common.Position, 16)
		yOrderChMap[ySymbol] = make(chan common.Order, 32)
		yFundingRateChMap[ySymbol] = make(chan common.FundingRate, 16)
		yOrderRequestChMap[ySymbol] = make(chan common.OrderRequest, 16)
		yNewOrderErrorChMap[ySymbol] = make(chan common.OrderError, 16)
		yAccountChMap[ySymbol] = make(chan common.Balance, 16)
		ySystemStatusChMap[ySymbol] = make(chan common.SystemStatus, 16)
	}

	strategyMap := make(map[string]*XYStrategy)

	var xCommissionAssetValue, yCommissionAssetValue *float64
	var xCommissionAssetValueCh = make(chan float64, 4)
	var yCommissionAssetValueCh = make(chan float64, 4)

	for xSymbol, ySymbol := range xyConfig.XYPairs {
		strategyMap[xSymbol], err = startXYStrategy(
			xyGlobalCtx,
			xSymbol, ySymbol,
			*xyConfig,
			xExchange,
			yExchange,
			xAccountChMap[xSymbol],
			yAccountChMap[ySymbol],
			xPositionChMap[xSymbol],
			yPositionChMap[ySymbol],
			xFundingRateChMap[xSymbol],
			yFundingRateChMap[ySymbol],
			xOrderRequestChMap[xSymbol],
			yOrderRequestChMap[ySymbol],
			xOrderChMap[xSymbol],
			yOrderChMap[ySymbol],
			xNewOrderErrorChMap[xSymbol],
			yNewOrderErrorChMap[ySymbol],
			xSystemStatusChMap[xSymbol],
			ySystemStatusChMap[ySymbol],
			xTickerChMap[xSymbol],
		)
		if err != nil {
			logger.Debugf("startXYStrategy %s %s error %v", xSymbol, ySymbol, err)
			return
		}
	}

	go xExchange.StreamBasic(
		xyGlobalCtx,
		xSystemStatusCh,
		xAccountCh,
		xCommissionAssetValueCh,
		xPositionChMap,
		xOrderChMap,
	)
	go xExchange.StreamFundingRate(
		xyGlobalCtx,
		xFundingRateChMap,
		xyConfig.StreamBatchSize,
	)
	go xExchange.StreamTicker(
		xyGlobalCtx,
		xTickerChMap,
		xyConfig.StreamBatchSize,
	)
	go xExchange.WatchOrders(
		xyGlobalCtx,
		xOrderRequestChMap,
		xOrderChMap,
		xNewOrderErrorChMap,
	)

	go yExchange.StreamBasic(
		xyGlobalCtx,
		ySystemStatusCh,
		yAccountCh,
		yCommissionAssetValueCh,
		yPositionChMap,
		yOrderChMap,
	)
	go yExchange.StreamFundingRate(
		xyGlobalCtx,
		yFundingRateChMap,
		xyConfig.StreamBatchSize,
	)
	go yExchange.StreamTicker(
		xyGlobalCtx,
		yTickerChMap,
		xyConfig.StreamBatchSize,
	)
	go yExchange.WatchOrders(
		xyGlobalCtx,
		yOrderRequestChMap,
		yOrderChMap,
		yNewOrderErrorChMap,
	)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		logger.Debugf("catch exit signal %v", sig)
		xyGlobalCancel()
	}()

	logger.Debugf("start main loop")
	restartTimer := time.NewTimer(xyConfig.RestartInterval)
	defer restartTimer.Stop()

	targetWeightUpdateTimer := time.NewTimer(xyConfig.InternalInflux.SaveInterval / 2)
	defer targetWeightUpdateTimer.Stop()

	influxSaveTimer := time.NewTimer(config.RestartSilent)
	defer influxSaveTimer.Stop()

	lastExternalSaveTime := &time.Time{}
mainLoop:
	for {
		select {
		case <-xyGlobalCtx.Done():
			logger.Debugf("global ctx done, exit main loop")
			xyGlobalCancel()
			break mainLoop
		case <-xExchange.Done():
			logger.Debugf("x exchange done, exit main loop")
			xyGlobalCancel()
			break mainLoop
		case <-yExchange.Done():
			logger.Debugf("y exchange done, exit main loop")
			xyGlobalCancel()
			break mainLoop
		case <-restartTimer.C:
			logger.Debugf("timed restart in %v", xyConfig.RestartInterval)
			xyGlobalCancel()
			break mainLoop
		case xSystemStatus = <-xSystemStatusCh:
			if xSystemStatus != common.SystemStatusReady {
				logger.Debugf("xSystemStatus %v", xSystemStatus)
			}
			for xSymbol, ch := range xSystemStatusChMap {
				select {
				case ch <- xSystemStatus:
				default:
					logger.Debugf("ch <- xSystemStatus failed %s ch len %d", xSymbol, len(ch))
				}
			}
			break
		case ySystemStatus = <-ySystemStatusCh:
			if ySystemStatus != common.SystemStatusReady {
				logger.Debugf("ySystemStatus %v", ySystemStatus)
			}
			for ySymbol, ch := range ySystemStatusChMap {
				select {
				case ch <- ySystemStatus:
				default:
					logger.Debugf("ch <- ySystemStatus failed %s ch len %d", ySymbol, len(ch))
				}
			}
			break
		case <-targetWeightUpdateTimer.C:
			totalLiquidity := 0.0
			liquidityMap := make(map[string]float64)
			for xSymbol, st := range strategyMap {
				offset := math.Max(st.stats.YAskOffset.Load(), st.stats.YBidOffset.Load())
				if st.stats.YMiddlePrice.Load() > 0 && offset > 0 {
					liquidityMap[xSymbol] = st.yMultiplier * math.Min(st.stats.YBidSize.Load(), st.stats.YAskSize.Load()) * st.stats.YMiddlePrice.Load()
					liquidityMap[xSymbol] /= offset
					liquidityMap[xSymbol] = math.Sqrt(liquidityMap[xSymbol])
					totalLiquidity += liquidityMap[xSymbol]
				}
			}
			if len(liquidityMap) > len(strategyMap)/2 {
				meanLiquidity := totalLiquidity / float64(len(liquidityMap))
				for xSymbol, liquidity := range liquidityMap {
					st := strategyMap[xSymbol]
					weight := liquidity / meanLiquidity
					weight = math.Sqrt(weight)
					if weight > 1.0 {
						weight = 1.0
					} else if weight < 0.25 {
						weight = 0.25
					}
					st.targetWeight.Set(weight)
					if !st.targetWeightUpdated.True() {
						st.targetWeightUpdated.Set(true)
						logger.Debugf("%10s TARGET WEIGHT UPDATE %f", xSymbol, weight)
					}
				}
			}
			if len(liquidityMap) == len(strategyMap) {
				targetWeightUpdateTimer.Reset(config.TargetWeightUpdateInterval)
			} else {
				targetWeightUpdateTimer.Reset(xyConfig.InternalInflux.SaveInterval / 2)
			}
			break
		case xcv := <-xCommissionAssetValueCh:
			xCommissionAssetValue = &xcv
			break
		case ycv := <-yCommissionAssetValueCh:
			yCommissionAssetValue = &ycv
			break
		case account := <-xAccountCh:
			if xAccount == nil || account.GetTime().Sub(xAccount.GetTime()) >= 0 {
				xAccount = account
				for xSymbol, ch := range xAccountChMap {
					select {
					case ch <- xAccount:
					default:
						logger.Debugf("ch <- xAccount failed %s ch len %d", xSymbol, len(ch))
					}
				}
			}
			break
		case account := <-yAccountCh:
			if yAccount == nil || account.GetTime().Sub(yAccount.GetTime()) >= 0 {
				yAccount = account
				for ySymbol, ch := range yAccountChMap {
					select {
					case ch <- yAccount:
					default:
						logger.Debugf("ch <- yAccount failed %s ch len %d", ySymbol, len(ch))
					}
				}
			}
			break
		case <-influxSaveTimer.C:
			if xyConfig.InternalInflux.Address != "" {
				handleSave(
					xAccount, yAccount,
					xExchange, yExchange,
					strategyMap,
					xSymbols,
					xSystemStatus, ySystemStatus,
					xyConfig,
					xCommissionAssetValue, yCommissionAssetValue,
					xyInternalInfluxWriter, xyExternalInfluxWriter,
					lastExternalSaveTime,
				)
				influxSaveTimer.Reset(
					time.Now().Truncate(
						xyConfig.InternalInflux.SaveInterval,
					).Add(
						xyConfig.InternalInflux.SaveInterval,
					).Sub(time.Now()),
				)
			}
			break
		}
	}
	logger.Debugf("stop waiting 15s")
	<-time.After(time.Second * 15)
	logger.Debugf("exit 0")
}
