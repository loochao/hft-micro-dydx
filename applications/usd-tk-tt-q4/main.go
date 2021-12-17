package main

import (
	"context"
	"flag"
	"fmt"
	bnbs "github.com/geometrybase/hft-micro/binance-busdspot"
	bnuf "github.com/geometrybase/hft-micro/binance-usdtfuture"
	bnus "github.com/geometrybase/hft-micro/binance-usdtspot"
	"github.com/geometrybase/hft-micro/common"
	dduf "github.com/geometrybase/hft-micro/dydx-usdfuture"
	ftx_usdfuture "github.com/geometrybase/hft-micro/ftx-usdfuture"
	kucoin_usdtfuture "github.com/geometrybase/hft-micro/kucoin-usdtfuture"
	"github.com/geometrybase/hft-micro/logger"
	okexv5_usdtspot "github.com/geometrybase/hft-micro/okexv5-usdtspot"
	okexv5_usdtswap "github.com/geometrybase/hft-micro/okexv5-usdtswap"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"os"
	"os/signal"
	"path"
	"runtime"
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
	var startTime = time.Now()

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
		var cpuProfFile *os.File
		cpuProfFile, err = os.Create(xyConfig.CpuProfile + time.Now().Format("-060102.cpu.prof"))
		if err != nil {
			logger.Warnf("os.Create error %v", err)
			return
		}
		err = pprof.StartCPUProfile(cpuProfFile)
		if err != nil {
			logger.Warnf("pprof.StartCPUProfile error %v", err)
			return
		}
		defer pprof.StopCPUProfile()
	}

	switch xyConfig.XExchange.Name {
	case "okexV5UsdtSpot":
		xExchange = &okexv5_usdtspot.OkexV5UsdtSpot{}
	case "okexV5UsdtSwap":
		xExchange = &okexv5_usdtswap.OkexV5UsdtSwap{}
	case "dydxUsdFuture":
		xExchange = &dduf.DydxUsdFuture{}
		break
	case "binanceUsdtFuture":
		xExchange = &bnuf.BinanceUsdtFuture{}
		break
	case "binanceUsdtFutureWithMergedTicker":
		xExchange = &bnuf.BinanceUsdtFutureWithMergedTicker{}
		break
	case "ftxUsdFutureWithWalkedDepth":
		xExchange = &ftx_usdfuture.FtxUsdFutureWithWalkedDepth{}
		break

	case "binanceUsdtSpotWithMergedTicker":
		xExchange = &bnus.BinanceUsdtSpotWithMergedTicker{}
		break
	case "binanceUsdtFutureWithWalkedDepth5":
		xExchange = &bnuf.BinanceUsdtFutureWithWalkedDepth5{}
		break
	case "binanceBusdSpotWithMergedTicker":
		xExchange = &bnbs.BinanceBusdSpotWithMergedTicker{}
		break

	case "okexV5UsdtSpotWithWalkedDepth5":
		xExchange = &okexv5_usdtspot.OkexV5UsdtSpotWithWalkedDepth5{}
		break
	case "okexV5UsdtSwapWithWalkedDepth5":
		xExchange = &okexv5_usdtswap.OkexV5UsdtSwapWithWalkedDepth5{}
		break

	case "kucoinUsdtFutureWithMergedTicker":
		xExchange = &kucoin_usdtfuture.KucoinUsdtFutureWithMergedTicker{}
		break
	case "kucoinUsdtFutureWithWalkedDepth5":
		xExchange = &kucoin_usdtfuture.KucoinUsdtFutureWithWalkedDepth5{}
		break
	//case "ftxUsdFuture":
	//	xExchange = &ftxuf.FtxUsdFuture{}
	//	break
	//case "ftxUsdSpot":
	//	xExchange = &ftxus.FtxUsdSpot{}
	//	break
	//case "kucoinUsdtFuture":
	//	xExchange = &kcut.KucoinUsdtFuture{}
	//	break
	//case "binanceBusdFutureWithMergedTicker":
	//	xExchange = &bnbf.BinanceBusdFutureWidthMergedTicker{}
	//	break
	//case "binanceUsdtSpot":
	//	xExchange = &bnus.BinanceUsdtSpot{}
	//	break
	//case "binanceTusdSpotWithMergedTicker":
	//	xExchange = &binance_tusdspot.BinanceTusdSpotWithMergedTicker{}
	//	break
	//case "binanceBusdSpot":
	//	xExchange = &bnbs.BinanceBusdSpot{}
	//	break
	//case "binanceUsdcSpotWithMergedTicker":
	//	xExchange = &bncs.BinanceUsdcSpotWithMergedTicker{}
	//	break
	//case "huobiUsdtFutureWithMergedTicker":
	//	xExchange = &hbuf.HuobiUsdtFutureWithMergedTicker{}
	//	break
	//case "bybitUsdtFuture":
	//	xExchange = &bbuf.BybitUsdtFuture{}
	//	break
	//case "kucoinUsdtSpotWithMergedTicker":
	//	xExchange = &kcus.KucoinUsdtSpotWithMergedTicker{}
	//	break
	default:
		logger.Warnf("unsupported exchange %s", xyConfig.XExchange.Name)
		return
	}

	switch xyConfig.YExchange.Name {
	case "okexV5UsdtSpot":
		yExchange = &okexv5_usdtspot.OkexV5UsdtSpot{}
	case "okexV5UsdtSwap":
		yExchange = &okexv5_usdtswap.OkexV5UsdtSwap{}
	case "dydxUsdFuture":
		yExchange = &dduf.DydxUsdFuture{}
		break
	case "binanceUsdtFuture":
		yExchange = &bnuf.BinanceUsdtFuture{}
		break
	case "binanceUsdtFutureWithMergedTicker":
		yExchange = &bnuf.BinanceUsdtFutureWithMergedTicker{}
		break

	case "binanceUsdtSpotWithMergedTicker":
		yExchange = &bnus.BinanceUsdtSpotWithMergedTicker{}
		break
	case "binanceUsdtFutureWithWalkedDepth5":
		yExchange = &bnuf.BinanceUsdtFutureWithWalkedDepth5{}
		break
	case "binanceBusdSpotWithMergedTicker":
		yExchange = &bnbs.BinanceBusdSpotWithMergedTicker{}
		break

	case "ftxUsdFutureWithWalkedDepth":
		yExchange = &ftx_usdfuture.FtxUsdFutureWithWalkedDepth{}
		break

	case "okexV5UsdtSpotWithWalkedDepth5":
		yExchange = &okexv5_usdtspot.OkexV5UsdtSpotWithWalkedDepth5{}
		break
	case "okexV5UsdtSwapWithWalkedDepth5":
		yExchange = &okexv5_usdtswap.OkexV5UsdtSwapWithWalkedDepth5{}
		break

	case "kucoinUsdtFutureWithMergedTicker":
		yExchange = &kucoin_usdtfuture.KucoinUsdtFutureWithMergedTicker{}
		break
	case "kucoinUsdtFutureWithWalkedDepth5":
		yExchange = &kucoin_usdtfuture.KucoinUsdtFutureWithWalkedDepth5{}
		break
	//case "ftxUsdFuture":
	//	yExchange = &ftxuf.FtxUsdFuture{}
	//	break
	//case "ftxUsdSpot":
	//	yExchange = &ftxus.FtxUsdSpot{}
	//	break
	//case "kucoinUsdtFuture":
	//	yExchange = &kcut.KucoinUsdtFuture{}
	//	break
	//case "kucoinUsdtFutureWithMergedTicker":
	//	yExchange = &kcut.KucoinUsdtFutureWithMergedTicker{}
	//	break
	//case "binanceUsdtSpot":
	//	yExchange = &bnus.BinanceUsdtSpot{}
	//	break
	//case "binanceUsdtSpotWithMergedTicker":
	//	yExchange = &bnus.BinanceUsdtSpotWithMergedTicker{}
	//	break
	//case "binanceBusdSpot":
	//	yExchange = &bnbs.BinanceBusdSpot{}
	//	break
	//case "binanceBusdSpotWithMergedTicker":
	//	yExchange = &bnbs.BinanceBusdSpotWithMergedTicker{}
	//	break
	//case "binanceUsdcSpotWithMergedTicker":
	//	yExchange = &bncs.BinanceUsdcSpotWithMergedTicker{}
	//	break
	//case "huobiUsdtFutureWithMergedTicker":
	//	yExchange = &hbuf.HuobiUsdtFutureWithMergedTicker{}
	//	break
	//case "bybitUsdtFuture":
	//	yExchange = &bbuf.BybitUsdtFuture{}
	//	break
	//case "kucoinUsdtSpotWithMergedTicker":
	//	yExchange = &kcus.KucoinUsdtSpotWithMergedTicker{}
	//	break
	//case "binanceBusdFutureWithMergedTicker":
	//	yExchange = &bnbf.BinanceBusdFutureWidthMergedTicker{}
	//	break
	//case "binanceTusdSpotWithMergedTicker":
	//	yExchange = &binance_tusdspot.BinanceTusdSpotWithMergedTicker{}
	//	break
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

	files := make([]string, 0)
	for xSymbol, ySymbol := range xyConfig.XYPairs {
		files = append(files, "%s-%s.json", xSymbol, ySymbol)
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

		if xyConfig.CpuProfile != "" {
			var heapProfFile *os.File
			runtime.GC() // profile all outstanding allocations
			if heapProfFile, err = os.Create(xyConfig.HeapProfile + time.Now().Format(fmt.Sprintf("-060102-%v.heap.prof", time.Now().Sub(startTime)))); err != nil {
				logger.Warnf("os.Create %s error %v", xyConfig.HeapProfile, err)
			} else if err = pprof.WriteHeapProfile(heapProfFile); err != nil {
				logger.Warnf("pprof.WriteHeapProfile error %v", err)
			}
		}

		logger.Debugf("catch exit signal %v", sig)
		xyGlobalCancel()
	}()

	logger.Debugf("start main loop")
	restartTimer := time.NewTimer(xyConfig.RestartInterval)
	defer restartTimer.Stop()

	influxSaveTimer := time.NewTimer(config.RestartSilent)
	defer influxSaveTimer.Stop()

	archiveTimer := time.NewTimer(config.RestartSilent)
	defer archiveTimer.Stop()

	lastExternalSaveTime := &time.Time{}

	archive1hFolder := path.Join(config.StatsRootPath, "1h")
	archive4hFolder := path.Join(config.StatsRootPath, "4h")
	archive8hFolder := path.Join(config.StatsRootPath, "8h")
	archive24hFolder := path.Join(config.StatsRootPath, "24h")
	archive48hFolder := path.Join(config.StatsRootPath, "48h")
	archive120hFolder := path.Join(config.StatsRootPath, "120h")
	archive240hFolder := path.Join(config.StatsRootPath, "240h")

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
			if xSystemStatus != common.SystemStatusReady {
				runtime.GC()
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
		case <-archiveTimer.C:
			hour1Time := time.Now().Truncate(time.Hour)
			hour4Time := time.Now().Truncate(time.Hour*4)
			hour8Time := time.Now().Truncate(time.Hour*8)
			hour24Time := time.Now().Truncate(time.Hour*24)
			hour48Time := time.Now().Truncate(time.Hour*48)
			hour120Time := time.Now().Truncate(time.Hour*120)
			hour240Time := time.Now().Truncate(time.Hour*240)
			archiveFiles(files, config.StatsRootPath, archive1hFolder)
			if hour1Time.Sub(hour4Time) == 0 {
				archiveFiles(files, config.StatsRootPath, archive4hFolder)
			}
			if hour1Time.Sub(hour8Time) == 0 {
				archiveFiles(files, config.StatsRootPath, archive8hFolder)
			}
			if hour1Time.Sub(hour24Time) == 0 {
				archiveFiles(files, config.StatsRootPath, archive24hFolder)
			}
			if hour1Time.Sub(hour48Time) == 0 {
				archiveFiles(files, config.StatsRootPath, archive48hFolder)
			}
			if hour1Time.Sub(hour120Time) == 0 {
				archiveFiles(files, config.StatsRootPath, archive120hFolder)
			}
			if hour1Time.Sub(hour240Time) == 0 {
				archiveFiles(files, config.StatsRootPath, archive240hFolder)
			}
			archiveTimer.Reset(time.Hour)
			break
		}
	}
	logger.Debugf("stop waiting 5s")
	<-time.After(time.Second * 30)
	logger.Debugf("exit 0")
}

func copyFile(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}

func archiveFiles(files []string, sourceFolder, targetFolder string) {
	err := os.MkdirAll(targetFolder, 0775)
	if err != nil {
		logger.Debugf("ARCHIVE os.MkdirAll error %v", err)
		return
	}
	for _, file := range files {
		srcFile := path.Join(sourceFolder, file)
		targetFile := path.Join(targetFolder, file)
		_, err = copyFile(srcFile, targetFile)
		if err != nil {
			logger.Debugf("copyFile(srcFile, targetFile) %s -> %s error %v", srcFile, targetFile, err)
		}
	}
}
