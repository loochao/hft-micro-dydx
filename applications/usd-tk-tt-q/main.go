package main

import (
	"context"
	"flag"
	bnbf "github.com/geometrybase/hft-micro/binance-busdfuture"
	bnbs "github.com/geometrybase/hft-micro/binance-busdspot"
	bncs "github.com/geometrybase/hft-micro/binance-usdcspot"
	bnuf "github.com/geometrybase/hft-micro/binance-usdtfuture"
	bnus "github.com/geometrybase/hft-micro/binance-usdtspot"
	bbuf "github.com/geometrybase/hft-micro/bybit-usdtfuture"
	"github.com/geometrybase/hft-micro/common"
	ftxuf "github.com/geometrybase/hft-micro/ftx-usdfuture"
	hbuf "github.com/geometrybase/hft-micro/huobi-usdtfuture"
	kcut "github.com/geometrybase/hft-micro/kucoin-usdtfuture"
	kcus "github.com/geometrybase/hft-micro/kucoin-usdtspot"
	"github.com/geometrybase/hft-micro/logger"
	okut "github.com/geometrybase/hft-micro/okex-usdtspot"
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
	case "binanceBusdFutureWithMergedTicker":
		xExchange = &bnbf.BinanceBusdFutureWidthMergedTicker{}
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
	case "huobiUsdtFutureWithMergedTicker":
		xExchange = &hbuf.HuobiUsdtFutureWithMergedTicker{}
		break
	case "bybitUsdtFuture":
		xExchange = &bbuf.BybitUsdtFuture{}
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
	case "ftxUsdtFuture":
		yExchange = &ftxuf.FtxUsdFuture{}
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
	case "huobiUsdtFutureWithMergedTicker":
		yExchange = &hbuf.HuobiUsdtFutureWithMergedTicker{}
		break
	case "bybitUsdtFuture":
		yExchange = &bbuf.BybitUsdtFuture{}
		break
	case "kucoinUsdtSpotWithMergedTicker":
		yExchange = &kcus.KucoinUsdtSpotWithMergedTicker{}
		break
	case "binanceBusdFutureWithMergedTicker":
		yExchange = &bnbf.BinanceBusdFutureWidthMergedTicker{}
		break
	default:
		logger.Fatalf("unsupported exchange %s", xyConfig.YExchange.Name)
	}

	for xSymbol, ySymbol := range xyConfig.XYPairs {
		xSymbols = append(xSymbols, xSymbol)
		ySymbols = append(ySymbols, ySymbol)
		if _, ok := xyConfig.TargetWeights[xSymbol]; !ok {
			logger.Debugf("miss target weight for %s", xSymbol)
			return
		}
		if _, ok := xyConfig.MaxOrderValues[xSymbol]; !ok {
			logger.Debugf("miss max order value for %s", xSymbol)
			return
		}
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
			logger.Debugf("common.NewInfluxWriter error %v", err)
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
			logger.Debugf("common.NewInfluxWriter error %v", err)
			return
		}
		defer xyExternalInfluxWriter.Stop()
	}

	influxSaveTimer := time.NewTimer(
		time.Now().Truncate(
			xyConfig.InternalInflux.SaveInterval,
		).Add(
			xyConfig.InternalInflux.SaveInterval * 3,
		).Sub(time.Now()),
	)
	defer influxSaveTimer.Stop()

	xPositionChMap := make(map[string]chan common.Position)
	xOrderChMap := make(map[string]chan common.Order)
	xFundingRateChMap := make(map[string]chan common.FundingRate)
	xNewOrderErrorChMap := make(map[string]chan common.OrderError)
	xAccountChMap := make(map[string]chan common.Balance)
	xSystemStatusChMap := make(map[string]chan common.SystemStatus)
	//此处的原则，chan越短，发送速度越快, 对于时效性高的，立即发出的要短

	xTickerChMap := make(map[string]chan common.Ticker)
	yTickerChMap := make(map[string]chan common.Ticker)

	for _, xSymbol := range xSymbols {
		xPositionChMap[xSymbol] = make(chan common.Position, 4)
		xOrderChMap[xSymbol] = make(chan common.Order, 32)
		xFundingRateChMap[xSymbol] = make(chan common.FundingRate, 1)
		xTickerChMap[xSymbol] = make(chan common.Ticker, 256)
		yTickerChMap[config.XYPairs[xSymbol]] = xTickerChMap[xSymbol]
		xOrderRequestChMap[xSymbol] = make(chan common.OrderRequest, 1)
		xNewOrderErrorChMap[xSymbol] = make(chan common.OrderError, 1)
		xAccountChMap[xSymbol] = make(chan common.Balance, 4)
		xSystemStatusChMap[xSymbol] = make(chan common.SystemStatus, 1)
	}

	yPositionChMap := make(map[string]chan common.Position)
	yOrderChMap := make(map[string]chan common.Order)
	yFundingRateChMap := make(map[string]chan common.FundingRate)
	yNewOrderErrorChMap := make(map[string]chan common.OrderError)
	yAccountChMap := make(map[string]chan common.Balance)
	ySystemStatusChMap := make(map[string]chan common.SystemStatus)
	for _, ySymbol := range ySymbols {
		yPositionChMap[ySymbol] = make(chan common.Position, 4)
		yOrderChMap[ySymbol] = make(chan common.Order, 32)
		yFundingRateChMap[ySymbol] = make(chan common.FundingRate, 1)
		yOrderRequestChMap[ySymbol] = make(chan common.OrderRequest, 1)
		yNewOrderErrorChMap[ySymbol] = make(chan common.OrderError, 1)
		yAccountChMap[ySymbol] = make(chan common.Balance, 4)
		ySystemStatusChMap[ySymbol] = make(chan common.SystemStatus, 1)
	}

	saveCh := make(chan *XYStrategy, 2048)
	strategiesMap := make(map[string]*XYStrategy)

	var xCommissionAssetValue, yCommissionAssetValue *float64
	var xCommissionAssetValueCh = make(chan float64, 4)
	var yCommissionAssetValueCh = make(chan float64, 4)

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
			saveCh,
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
		xyConfig.BatchSize,
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
		xyConfig.BatchSize,
	)
	go yExchange.StreamTicker(
		xyGlobalCtx,
		yTickerChMap,
		xyConfig.BatchSize,
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
		case xcv := <-xCommissionAssetValueCh:
			xCommissionAssetValue = &xcv
			//logger.Debugf("xCommissionAssetValue %f", *xCommissionAssetValue)
		case ycv := <-yCommissionAssetValueCh:
			yCommissionAssetValue = &ycv
			//logger.Debugf("yCommissionAssetValue %f", *yCommissionAssetValue)
		case account := <-xAccountCh:
			if xAccount == account {
				logger.Debugf("bad xAccount == account pass same pointer")
			}
			//logger.Debugf("xAccount %f %f", account.GetBalance(), account.GetFree())
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
			if yAccount == account {
				logger.Debugf("bad  yAccount == account pass same pointer")
			}
			//logger.Debugf("yAccount %f %f", account.GetBalance(), account.GetFree())
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
		case st := <-saveCh:
			strategiesMap[st.xSymbol] = st
			break
		case <-influxSaveTimer.C:
			if xyConfig.InternalInflux.Address != "" {

				handleSave(
					xAccount, yAccount,
					xExchange, yExchange,
					strategiesMap,
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
	logger.Debugf("stop waiting 5s")
	<-time.After(time.Second * 5)
	logger.Debugf("exit 0")
}
