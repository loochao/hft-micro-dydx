package main

import (
	"context"
	"flag"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"time"
)

var swapSymbols = make([]string, 0)
var swapSymbolsMap = make(map[string]string, 0)

var swapGlobalCtx context.Context
var swapGlobalCancel context.CancelFunc
var swapInternalInfluxWriter *common.InfluxWriter
var swapExternalInfluxWriter *common.InfluxWriter

var swapAPI *bnswap.API
var swapUserWebsocket *bnswap.UserWebsocket
var swapHttpPositionUpdateSilentTimes = make(map[string]time.Time)

var swapTickSizes = make(map[string]float64)
var swapStepSizes = make(map[string]float64)
var swapMinNotional = make(map[string]float64)

var swapOpenOrders = make(map[string]TakerOpenOrder)
var swapOrderCancelSilentTimes = make(map[string]time.Time)
var swapPositionsCh = make(chan []bnswap.Position, 10)
var swapPositions = make(map[string]*bnswap.Position)
var swapPositionsUpdateTimes = make(map[string]time.Time)
var swapAccount *bnswap.Asset
var swapAccountCh = make(chan bnswap.Account, 10)
var swapNewOrderErrorCh = make(chan TakerOrderNewError, 10)
var swapOrderRequestChs = make(map[string]chan TakerOrderRequest)
var swapWalkedDepths = make(map[string]WalkedDepth5)

var swapOrderSilentTimes = make(map[string]time.Time)

var swapLogSilentTimes = make(map[string]time.Time)
var swapLoopTimer *time.Timer
var swapSystemReady = false
var swapSystemStatusCh = make(chan bool, 10)
var swapGlobalSilent = time.Now()

var swapLastEnterPrices = make(map[string]float64)
var swapEnterOffset = make(map[string]float64)
var swapEnterSilentTimes = make(map[string]time.Time)

var swapMergedSignalCh = make(chan MergedSignal, 10000)
var swapMergedSignals = make(map[string]MergedSignal)

var mtConfig *Config

func init() {

	logger.Debug("####  BUILD @ 20210501 11:39:28  ####")

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
	logger.Debugf("\n\nYAML CONFIG:\n\n%s", config.ToString())
	valid, reason := config.IsValid()
	if !valid {
		logger.Fatalf("CONFIG IS NOT VALID:\n%s\n", reason)
	}
	mtConfig = &config

	for symbol := range mtConfig.SymbolsMap {
		swapSymbols = append(swapSymbols, symbol)
		swapSymbolsMap[symbol] = symbol
		swapLogSilentTimes[symbol] = time.Now()

		swapOrderSilentTimes[symbol] = time.Now()
		swapPositionsUpdateTimes[symbol] = time.Unix(0, 0)
		swapGlobalSilent = time.Now().Add(*mtConfig.RestartSilent)
		swapHttpPositionUpdateSilentTimes[symbol] = time.Now()
	}
}
