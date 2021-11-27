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
	kcuf "github.com/geometrybase/hft-micro/kucoin-usdtfuture"
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
	case "binanceUsdtSpotWithDepth5":
		xExchange = &bnus.BinanceUsdtSpotWithDepth5{}
		break
	case "binanceUsdtSpotWithDepth20":
		xExchange = &bnus.BinanceUsdtSpotWithDepth20{}
		break
	case "binanceBusdSpotWithDepth5":
		xExchange = &bnbs.BinanceBusdSpotWithDepth5{}
		break
	case "binanceBusdSpotWithDepth20":
		xExchange = &bnbs.BinanceBusdSpotWithDepth20{}
		break
	case "binanceUsdcSpotWithDepth5":
		xExchange = &bncs.BinanceUsdcSpotWithDepth5{}
		break
	case "binanceUsdcSpotWithDepth20":
		xExchange = &bncs.BinanceUsdcSpotWithDepth20{}
		break
	case "binanceUsdtFutureWithDepth5":
		xExchange = &bnuf.BinanceUsdtFutureWidthDepth5{}
		break
	case "binanceUsdtFutureWithDepth20":
		xExchange = &bnuf.BinanceUsdtFutureWidthDepth20{}
		break
	case "kucoinUsdtFutureWithDepth5":
		xExchange = &kcuf.KucoinUsdtFutureWithDepth5{}
		break
	case "ftxUsdtFuture":
		xExchange = &ftxuf.FtxUsdFuture{}
		break
	case "okexUsdtSpot":
		xExchange = &okut.OkexUsdtSpot{}
		break
	default:
		logger.Fatalf("unsupported exchange %s", xyConfig.XExchange.Name)
	}

	switch xyConfig.YExchange.Name {
	case "binanceUsdtSpotWithDepth5":
		yExchange = &bnus.BinanceUsdtSpotWithDepth5{}
		break
	case "binanceUsdtSpotWithDepth20":
		yExchange = &bnus.BinanceUsdtSpotWithDepth20{}
		break
	case "binanceBusdSpotWithDepth5":
		yExchange = &bnbs.BinanceBusdSpotWithDepth5{}
		break
	case "binanceBusdSpotWithDepth20":
		yExchange = &bnbs.BinanceBusdSpotWithDepth20{}
		break
	case "binanceUsdcSpotWithDepth5":
		yExchange = &bncs.BinanceUsdcSpotWithDepth5{}
		break
	case "binanceUsdcSpotWithDepth20":
		yExchange = &bncs.BinanceUsdcSpotWithDepth20{}
		break
	case "binanceUsdtFutureWithDepth5":
		yExchange = &bnuf.BinanceUsdtFutureWidthDepth5{}
		break
	case "binanceUsdtFutureWithDepth20":
		yExchange = &bnuf.BinanceUsdtFutureWidthDepth20{}
		break
	case "kucoinUsdtFutureWithDepth5":
		yExchange = &kcuf.KucoinUsdtFutureWithDepth5{}
		break
	case "ftxUsdtFuture":
		yExchange = &ftxuf.FtxUsdFuture{}
		break
	case "okexUsdtSpot":
		yExchange = &okut.OkexUsdtSpot{}
		break
	default:
		logger.Fatalf("unsupported exchange %s", xyConfig.YExchange.Name)
	}

	orderOffsets := make(map[string]Offset)
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
		orderOffsets[xSymbol], err = NewOffset(xyConfig.OrderOffsets[xSymbol])
		if err != nil {
			logger.Debugf("NewOffset error %s %v", xSymbol, err)
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

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		logger.Debugf("catch exit signal %v", sig)
		xyGlobalCancel()
	}()

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

	xDepthChMap := make(map[string]chan common.Depth)
	yDepthChMap := make(map[string]chan common.Depth)

	//此处的原则，chan越短，发送速度越快, 对于时效性高的，立即发出的要短
	for _, xSymbol := range xSymbols {
		xPositionChMap[xSymbol] = make(chan common.Position, 8)
		xOrderChMap[xSymbol] = make(chan common.Order, 32)
		xFundingRateChMap[xSymbol] = make(chan common.FundingRate, 1)
		xOrderRequestChMap[xSymbol] = make(chan common.OrderRequest, 1)
		xNewOrderErrorChMap[xSymbol] = make(chan common.OrderError, 1)
		xAccountChMap[xSymbol] = make(chan common.Balance, 8)
		xSystemStatusChMap[xSymbol] = make(chan common.SystemStatus, 1)

		//depth channel是共用的，以防出现顺序错乱问题
		xDepthChMap[xSymbol] = make(chan common.Depth, 64)
		yDepthChMap[config.XYPairs[xSymbol]] = xDepthChMap[xSymbol]
	}

	yPositionChMap := make(map[string]chan common.Position)
	yOrderChMap := make(map[string]chan common.Order)
	yFundingRateChMap := make(map[string]chan common.FundingRate)
	yNewOrderErrorChMap := make(map[string]chan common.OrderError)
	yAccountChMap := make(map[string]chan common.Balance)
	ySystemStatusChMap := make(map[string]chan common.SystemStatus)
	for _, ySymbol := range ySymbols {
		yPositionChMap[ySymbol] = make(chan common.Position, 8)
		yOrderChMap[ySymbol] = make(chan common.Order, 32)
		yFundingRateChMap[ySymbol] = make(chan common.FundingRate, 1)
		yOrderRequestChMap[ySymbol] = make(chan common.OrderRequest, 1)
		yNewOrderErrorChMap[ySymbol] = make(chan common.OrderError, 1)
		yAccountChMap[ySymbol] = make(chan common.Balance, 8)
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
			orderOffsets[xSymbol],
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
			xDepthChMap[xSymbol],
			saveCh,
		)
		if err != nil {
			logger.Debugf("startXYStrategy %s %s error %v", xSymbol, ySymbol, err)
			return
		}
	}

	//需要等待一会再stream数据，读取quantile需要时间
	select {
	case <-xyGlobalCtx.Done():
		return
	case <-time.After(time.Minute):
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
	go xExchange.StreamDepth(
		xyGlobalCtx,
		xDepthChMap,
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
	go yExchange.StreamDepth(
		xyGlobalCtx,
		yDepthChMap,
		xyConfig.BatchSize,
	)
	go yExchange.WatchOrders(
		xyGlobalCtx,
		yOrderRequestChMap,
		yOrderChMap,
		yNewOrderErrorChMap,
	)

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
		case xcv := <-xCommissionAssetValueCh:
			xCommissionAssetValue = &xcv
			//logger.Debugf("xCommissionAssetValue %f", *xCommissionAssetValue)
			break
		case ycv := <-yCommissionAssetValueCh:
			yCommissionAssetValue = &ycv
			//logger.Debugf("yCommissionAssetValue %f", *yCommissionAssetValue)
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
		case account := <-yAccountCh:
			if yAccount == account {
				logger.Debugf("bad  yAccount == account pass same pointer")
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
	logger.Fatal("exit after 15s")
}
