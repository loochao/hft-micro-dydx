package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/ftxperp"
	"github.com/geometrybase/hft-micro/kcperp"
	"github.com/geometrybase/hft-micro/logger"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"time"
)

var xSymbols = make([]string, 0)
var ySymbols = make([]string, 0)
var yxSymbolsMap = make(map[string]string, 0)
var xySymbolsMap = make(map[string]string, 0)

var xyGlobalCtx context.Context
var xyGlobalCancel context.CancelFunc
var xyInfluxWriter *common.InfluxWriter
var xyExternalInfluxWriter *common.InfluxWriter

var xStepSizes = make(map[string]float64)
var xMinNotionals = make(map[string]float64)
var yStepSizes = make(map[string]float64)
var yMinNotionals = make(map[string]float64)
var xyStepSizes = make(map[string]float64)

var xAccount common.Account
var xAccountCh = make(chan common.Account, 200)
var xPositionCh = make(chan common.Position, 200)
var xOrderCh = make(chan common.Order, 200)
var xPositions = make(map[string]common.Position)
var xPositionsUpdateTimes = make(map[string]time.Time)
var xOrderRequestChMap = make(map[string]chan common.OrderRequest)
var xNewOrderErrorCh = make(chan common.OrderError, 200)
var xOrderSilentTimes = make(map[string]time.Time)

var yPositionCh = make(chan common.Position, 200)
var yOrderCh = make(chan common.Order, 200)
var yPositions = make(map[string]common.Position)
var yPositionsUpdateTimes = make(map[string]time.Time)
var yAccount common.Account
var yAccountCh = make(chan common.Account, 200)
var yNewOrderErrorCh = make(chan common.OrderError, 200)
var yOrderRequestChMap = make(map[string]chan common.OrderRequest)
var yOrderSilentTimes = make(map[string]time.Time)

var xFundingRates = make(map[string]common.FundingRate)
var xFundingRateCh = make(chan common.FundingRate, 200)
var yFundingRates = make(map[string]common.FundingRate)
var yFundingRateCh = make(chan common.FundingRate, 200)
var xyFundingRates = make(map[string]float64)
var xyRankSymbolMap map[int]string

var xySpreads = make(map[string]*XYSpread)

var xLastFilledBuyPrices = make(map[string]float64)
var xLastFilledSellPrices = make(map[string]float64)
var yLastFilledBuyPrices = make(map[string]float64)
var yLastFilledSellPrices = make(map[string]float64)
var xyRealisedSpread = make(map[string]float64)

var xyUnHedgeValue float64
var xyLogSilentTimes = make(map[string]time.Time)
var xyLoopTimer *time.Timer
var xyDirResetTimer *time.Timer
var xyDualEnds []int
var xSystemStatus = common.SystemStatusNotReady
var ySystemStatus = common.SystemStatusNotReady
var xSystemStatusCh = make(chan common.SystemStatus, 100)
var ySystemStatusCh = make(chan common.SystemStatus, 100)

var yHedgeMarkPrices = make(map[string]float64)
var xHedgeMarkPrices = make(map[string]float64)
var xTargetPositionSizes = make(map[string]float64)
var yTargetPositionSizes = make(map[string]float64)
var xyTargetPositionUpdateSilentTimes = make(map[string]time.Time)

var xyConfig *Config

var xExchange common.Exchange
var yExchange common.Exchange

var xyEnterTradeOrders = make(map[string]EnterTradeOrder)
var xyMergedDirs = make(map[string]float64)
var xyEnterTimes = make(map[string]time.Time)

func init() {

	logger.Debug("####  BUILD @ 20210522 00:58:47  ####")

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

	configStr, err := yaml.Marshal(xyConfig)
	if err != nil {
		logger.Fatal(err)
	}
	fmt.Printf("CONFIG:\n\n%s\n\n", configStr)

	switch xyConfig.XExchange.Name {
	case "ftxperp":
		xExchange = &ftxperp.Ftxperp{}
	case "bnswap":
		xExchange = &bnswap.Bnswap{}
	case "kcperp":
		xExchange = &kcperp.Kcperp{}
	default:
		logger.Fatal("unsupported exchange %s", xyConfig.XExchange.Name)
	}

	switch xyConfig.YExchange.Name {
	case "ftxperp":
		yExchange = &ftxperp.Ftxperp{}
	case "bnswap":
		yExchange = &bnswap.Bnswap{}
	case "kcperp":
		yExchange = &kcperp.Kcperp{}
	default:
		logger.Fatal("unsupported exchange %s", xyConfig.YExchange.Name)
	}
	for xSymbol, ySymbol := range xyConfig.XYPairs {
		xSymbols = append(xSymbols, xSymbol)
		ySymbols = append(ySymbols, ySymbol)
		yxSymbolsMap[ySymbol] = xSymbol
		xySymbolsMap[xSymbol] = ySymbol

		xyEnterTimes[xSymbol] = time.Unix(0, 0)
		xyEnterTradeOrders[xSymbol] = EnterTradeOrderUnknown
		xyMergedDirs[xSymbol] = 0.0
		xyTargetPositionUpdateSilentTimes[xSymbol] = time.Now()

		xOrderSilentTimes[xSymbol] = time.Now().Add(xyConfig.RestartSilent)
		xyLogSilentTimes[xSymbol] = time.Now()
		xPositionsUpdateTimes[xSymbol] = time.Unix(0, 0)

		yOrderSilentTimes[ySymbol] = time.Now().Add(xyConfig.RestartSilent)
		yPositionsUpdateTimes[ySymbol] = time.Unix(0, 0)
	}
	xyConfig.XExchange.Symbols = xSymbols
	xyConfig.YExchange.Symbols = ySymbols
	xyDualEnds = make([]int, 0)
	for i := 0; i < len(xSymbols)/2; i++ {
		xyDualEnds = append(xyDualEnds, i)
		xyDualEnds = append(xyDualEnds, len(xSymbols)-1-i)
	}
	if len(xSymbols)%2 == 1 {
		xyDualEnds = append(xyDualEnds, len(xSymbols)/2)
	}
	logger.Debugf("dual end ranks %d", xyDualEnds)
}
