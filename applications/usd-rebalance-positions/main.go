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
	ftxus "github.com/geometrybase/hft-micro/ftx-usdspot"
	kcut "github.com/geometrybase/hft-micro/kucoin-usdtfuture"
	kcus "github.com/geometrybase/hft-micro/kucoin-usdtspot"
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
	var ySymbols = make([]string, 0)

	var xyGlobalCtx context.Context
	var xyGlobalCancel context.CancelFunc

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
	case "ftxUsdFuture":
		xExchange = &ftxuf.FtxUsdFuture{}
		break
	case "ftxUsdSpot":
		xExchange = &ftxus.FtxUsdSpot{}
		break
	case "okexUsdtSpot":
		xExchange = &okut.OkexUsdtSpot{}
		break
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
	case "kucoinUsdtSpotWithMergedTicker":
		xExchange = &kcus.KucoinUsdtSpotWithMergedTicker{}
		break
	default:
		logger.Fatalf("unsupported exchange %s", xyConfig.XExchange.Name)
	}

	switch xyConfig.YExchange.Name {
	case "binanceUsdtFuture":
		yExchange = &bnuf.BinanceUsdtFuture{}
		break
	case "ftxUsdFuture":
		yExchange = &ftxuf.FtxUsdFuture{}
		break
	case "ftxUsdSpot":
		yExchange = &ftxus.FtxUsdSpot{}
		break
	case "okexUsdtSpot":
		yExchange = &okut.OkexUsdtSpot{}
		break
	case "kucoinUsdtFuture":
		yExchange = &kcut.KucoinUsdtFuture{}
		break
	case "kucoinUsdtFutureWithMergedTicker":
		yExchange = &kcut.KucoinUsdtFutureWithMergedTicker{}
		break
	case "binanceUsdtFutureWithMergedTicker":
		yExchange = &bnuf.BinanceUsdtFutureWithMergedTicker{}
		break
	case "binanceUsdtSpot":
		yExchange = &bnus.BinanceUsdtSpot{}
		break
	case "binanceUsdtSpotWithMergedTicker":
		yExchange = &bnus.BinanceUsdtSpotWithMergedTicker{}
		break
	case "binanceBusdSpot":
		yExchange = &bnbs.BinanceBusdSpot{}
		break
	case "binanceBusdSpotWithMergedTicker":
		yExchange = &bnbs.BinanceBusdSpotWithMergedTicker{}
		break
	case "binanceUsdcSpotWithMergedTicker":
		yExchange = &bncs.BinanceUsdcSpotWithMergedTicker{}
		break
	case "kucoinUsdtSpotWithMergedTicker":
		yExchange = &kcus.KucoinUsdtSpotWithMergedTicker{}
		break
	default:
		logger.Fatalf("unsupported exchange %s", xyConfig.YExchange.Name)
	}

	for xSymbol, ySymbol := range xyConfig.XYPairs {
		xSymbols = append(xSymbols, xSymbol)
		ySymbols = append(ySymbols, ySymbol)
	}
	xyConfig.XExchange.Symbols = xSymbols
	xyConfig.YExchange.Symbols = ySymbols

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
	err = yExchange.Setup(xyGlobalCtx, xyConfig.YExchange)
	if err != nil {
		logger.Debugf("yExchange.Setup(xyGlobalCtx, xyConfig.YExchange) error %v", err)
		return
	}

	xAccountChMap := make(map[string]chan common.Balance)
	yAccountChMap := make(map[string]chan common.Balance)

	xPositionChMap := make(map[string]chan common.Position)
	yPositionChMap := make(map[string]chan common.Position)

	xOrderChMap := make(map[string]chan common.Order)
	yOrderChMap := make(map[string]chan common.Order)

	xNewOrderErrorChMap := make(map[string]chan common.OrderError)
	yNewOrderErrorChMap := make(map[string]chan common.OrderError)

	xSystemStatusChMap := make(map[string]chan common.SystemStatus)
	ySystemStatusChMap := make(map[string]chan common.SystemStatus)

	xTickerChMap := make(map[string]chan common.Ticker)
	yTickerChMap := make(map[string]chan common.Ticker)

	var xAccount common.Balance
	var yAccount common.Balance

	var xAccountCh = make(chan common.Balance, 64)
	var yAccountCh = make(chan common.Balance, 64)

	var xOrderRequestChMap = make(map[string]chan common.OrderRequest)
	var yOrderRequestChMap = make(map[string]chan common.OrderRequest)

	for _, xSymbol := range xSymbols {
		xAccountChMap[xSymbol] = make(chan common.Balance, 64)
		xPositionChMap[xSymbol] = make(chan common.Position, 64)
		xOrderChMap[xSymbol] = make(chan common.Order, 32)
		xTickerChMap[xSymbol] = make(chan common.Ticker, 256)
		xOrderRequestChMap[xSymbol] = make(chan common.OrderRequest, 64)
		xNewOrderErrorChMap[xSymbol] = make(chan common.OrderError, 1)
		xSystemStatusChMap[xSymbol] = make(chan common.SystemStatus, 64)
	}



	for _, ySymbol := range ySymbols {
		yAccountChMap[ySymbol] = make(chan common.Balance, 64)
		yPositionChMap[ySymbol] = make(chan common.Position, 64)
		yOrderChMap[ySymbol] = make(chan common.Order, 32)
		yTickerChMap[ySymbol] = make(chan common.Ticker, 256)
		yOrderRequestChMap[ySymbol] = make(chan common.OrderRequest, 64)
		yNewOrderErrorChMap[ySymbol] = make(chan common.OrderError, 1)
		ySystemStatusChMap[ySymbol] = make(chan common.SystemStatus, 64)
	}

	var xCommissionAssetValueCh = make(chan float64, 64)
	var yCommissionAssetValueCh = make(chan float64, 64)

	for xSymbol, ySymbol := range xyConfig.XYPairs {
		err = startXYStrategy(
			xyGlobalCtx,

			xSymbol, ySymbol,

			*xyConfig,

			xExchange,
			yExchange,

			xAccountChMap[xSymbol],
			yAccountChMap[ySymbol],

			xPositionChMap[xSymbol],
			yPositionChMap[ySymbol],

			xOrderRequestChMap[xSymbol],
			yOrderRequestChMap[ySymbol],

			xOrderChMap[xSymbol],
			yOrderChMap[ySymbol],

			xNewOrderErrorChMap[xSymbol],
			yNewOrderErrorChMap[ySymbol],

			xSystemStatusChMap[xSymbol],
			ySystemStatusChMap[ySymbol],

			xTickerChMap[xSymbol],
			yTickerChMap[ySymbol],
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
	go yExchange.StreamBasic(
		xyGlobalCtx,
		ySystemStatusCh,
		yAccountCh,
		yCommissionAssetValueCh,
		yPositionChMap,
		yOrderChMap,
	)

	go xExchange.StreamTicker(
		xyGlobalCtx,
		xTickerChMap,
		xyConfig.BatchSize,
	)
	go yExchange.StreamTicker(
		xyGlobalCtx,
		yTickerChMap,
		xyConfig.BatchSize,
	)

	go xExchange.WatchOrders(
		xyGlobalCtx,
		xOrderRequestChMap,
		xOrderChMap,
		xNewOrderErrorChMap,
	)
	go yExchange.WatchOrders(
		xyGlobalCtx,
		yOrderRequestChMap,
		yOrderChMap,
		yNewOrderErrorChMap,
	)

	go xExchange.StreamSystemStatus(
		xyGlobalCtx,
		xSystemStatusCh,
	)
	go yExchange.StreamSystemStatus(
		xyGlobalCtx,
		ySystemStatusCh,
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
		case _ = <-xCommissionAssetValueCh:
			break
		case _ = <-yCommissionAssetValueCh:
			break
		case account := <-xAccountCh:
			//logger.Debugf("xAccount %v", account)
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
		case account := <-yAccountCh:
			//logger.Debugf("yAccount %v", account)
			if yAccount == account {
				logger.Debugf("bad yAccount == account pass same pointer")
			}
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
		}
	}
}
