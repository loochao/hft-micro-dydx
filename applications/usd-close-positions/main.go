package main

import (
	"context"
	"flag"
	bnbs "github.com/geometrybase/hft-micro/binance-busdspot"
	bncs "github.com/geometrybase/hft-micro/binance-usdcspot"
	bnuf "github.com/geometrybase/hft-micro/binance-usdtfuture"
	bnus "github.com/geometrybase/hft-micro/binance-usdtspot"
	"github.com/geometrybase/hft-micro/common"
	ftxuf "github.com/geometrybase/hft-micro/ftx-usdfuture"
	kcut "github.com/geometrybase/hft-micro/kucoin-usdtfuture"
	"github.com/geometrybase/hft-micro/logger"
	//okut "github.com/geometrybase/hft-micro/okex-usdtspot"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"os/signal"
	"runtime/pprof"
	"strings"
	"syscall"
	"time"
)

func main() {
	var xSymbols = make([]string, 0)

	var xyGlobalCtx context.Context
	var xyGlobalCancel context.CancelFunc

	var xyConfig *Config
	var xExchange common.UsdExchange

	var xSystemStatus = common.SystemStatusNotReady
	var xSystemStatusCh = make(chan common.SystemStatus, 4)

	configPath := flag.String("config", "", "config path")
	flag.Parse()

	if *configPath == "" {
		logger.Fatal("config is empty")
	}

	configFile, err := ioutil.ReadFile(*configPath)
	if err != nil {
		logger.Fatal(err)
	}
	var config Config
	err = yaml.Unmarshal(configFile, &config)
	if err != nil {
		logger.Fatal(err)
	}
	config.SetDefaultIfNotSet()
	xyConfig = &config

	if xyConfig.CpuProfile != "" {
		f, err := os.Create(xyConfig.CpuProfile)
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

	switch xyConfig.XExchange.Name {
	case "binanceUsdtFuture":
		xExchange = &bnuf.BinanceUsdtFuture{}
		break
	case "ftxUsdtFuture":
		xExchange = &ftxuf.FtxUsdFuture{}
		break
	//case "okexUsdtSpot":
	//	xExchange = &okut.OkexUsdtSpot{}
	//	break
	case "kucoinUsdtFuture":
		xExchange = &kcut.KucoinUsdtFuture{}
		break
	case "kucoinUsdtFutureWithMergedTicker":
		xExchange = &kcut.KucoinUsdtFutureWithMergedTicker{}
		break
	case "binanceUsdtFutureWithMergedTicker":
		xExchange = &bnuf.BinanceUsdtFutureWithMergedTicker{}
		break
	case "binanceUsdtSpot":
		xExchange = &bnus.BinanceUsdtSpot{}
		break
	case "binanceUsdtSpotWithMergedTicker":
		xExchange = &bnus.BinanceUsdtSpotWithMergedTicker{}
		break
	case "binanceBusdSpot":
		xExchange = &bnbs.BinanceBusdSpot{}
		break
	case "binanceBusdSpotWithMergedTicker":
		xExchange = &bnbs.BinanceBusdSpotWithMergedTicker{}
		break
	case "binanceUsdcSpotWithMergedTicker":
		xExchange = &bncs.BinanceUsdcSpotWithMergedTicker{}
		break
	default:
		logger.Fatalf("unsupported exchange %s", xyConfig.XExchange.Name)
	}

	for _, xSymbol := range xyConfig.Symbols {
		xSymbols = append(xSymbols, xSymbol)
	}
	xyConfig.XExchange.Symbols = xSymbols

	configStr, err := yaml.Marshal(xyConfig)
	if err != nil {
		logger.Fatal(err)
	}
	logger.Debug("\n\nCONFIG:")
	for _, l := range strings.Split(string(configStr), "\n") {
		logger.Debugf("%s", l)
	}
	logger.Debug("\n\n")

	xyGlobalCtx, xyGlobalCancel = context.WithCancel(context.Background())
	defer xyGlobalCancel()

	err = xExchange.Setup(xyGlobalCtx, xyConfig.XExchange)
	if err != nil {
		logger.Debugf("xExchange.Setup(xyGlobalCtx, xyConfig.XExchange) error %v", err)
		return
	}

	xAccountChMap := make(map[string]chan common.Balance)

	xPositionChMap := make(map[string]chan common.Position)

	xOrderChMap := make(map[string]chan common.Order)

	xNewOrderErrorChMap := make(map[string]chan common.OrderError)

	xSystemStatusChMap := make(map[string]chan common.SystemStatus)

	xTickerChMap := make(map[string]chan common.Ticker)

	var xAccount common.Balance

	var xAccountCh = make(chan common.Balance, 4)

	var xOrderRequestChMap = make(map[string]chan common.OrderRequest)

	for _, xSymbol := range xSymbols {
		xAccountChMap[xSymbol] = make(chan common.Balance, 4)
		xPositionChMap[xSymbol] = make(chan common.Position, 4)
		xOrderChMap[xSymbol] = make(chan common.Order, 32)
		xTickerChMap[xSymbol] = make(chan common.Ticker, 256)
		xOrderRequestChMap[xSymbol] = make(chan common.OrderRequest, 4)
		xNewOrderErrorChMap[xSymbol] = make(chan common.OrderError, 1)
		xSystemStatusChMap[xSymbol] = make(chan common.SystemStatus, 1)
	}


	var xCommissionAssetValueCh = make(chan float64, 4)

	for _, xSymbol := range xyConfig.Symbols {
		err = startXYStrategy(
			xyGlobalCtx,

			xSymbol,

			*xyConfig,

			xExchange,

			xAccountChMap[xSymbol],

			xPositionChMap[xSymbol],

			xOrderRequestChMap[xSymbol],

			xOrderChMap[xSymbol],

			xNewOrderErrorChMap[xSymbol],

			xSystemStatusChMap[xSymbol],

			xTickerChMap[xSymbol],
		)
		if err != nil {
			logger.Debugf("startXYStrategy %s error %v", xSymbol, err)
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

	go xExchange.StreamTicker(
		xyGlobalCtx,
		xTickerChMap,
		xyConfig.BatchSize,
	)

	go xExchange.WatchOrders(
		xyGlobalCtx,
		xOrderRequestChMap,
		xOrderChMap,
		xNewOrderErrorChMap,
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
		case _ = <-xCommissionAssetValueCh:
			break

		case account := <-xAccountCh:
			if xAccount == account {
				logger.Debugf("bad xAccount == account pass same pointer")
			}
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
		}
	}
}
