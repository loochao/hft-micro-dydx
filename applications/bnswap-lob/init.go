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
var swapOrderCancelCounts = make(map[string]int)
var swapOrderCancelSilentTimes = make(map[string]time.Time)
var swapPositionsCh = make(chan []bnswap.Position, 10)
var swapPositions = make(map[string]*bnswap.Position)
var swapPositionsUpdateTimes = make(map[string]time.Time)
var swapAccount *bnswap.Asset
var swapAccountCh = make(chan bnswap.Account, 10)
var swapNewOrderErrorCh = make(chan TakerOrderNewError, 10)
var swapOrderRequestChs = make(map[string]chan TakerOrderRequest)
var swapWalkedDepths = make(map[string]WalkedDepth20)
var spotWalkedDepths = make(map[string]WalkedDepth20)

var swapLastFilledBuyPrices = make(map[string]float64)
var swapLastFilledSellPrices = make(map[string]float64)
var swapRealisedSpread = make(map[string]float64)

var tOrderSilentTimes = make(map[string]time.Time)

var mtLogSilentTimes = make(map[string]time.Time)
var mtLoopTimer *time.Timer
var swapSystemReady = false
var spotSystemReady = false
var swapSystemStatusCh = make(chan bool, 10)
var spotSystemStatusCh = make(chan bool, 10)
var mtGlobalSilent = time.Now()

var mtConfig *Config

func init() {

	logger.Debug("####  BUILD @ 20210429 13:37:14  ####")

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

	for _, symbol := range mtConfig.Symbols {
		swapSymbols = append(swapSymbols, symbol)
		swapSymbolsMap[symbol] = symbol
		mtLogSilentTimes[symbol] = time.Now()

		tOrderSilentTimes[symbol] = time.Now()
		swapPositionsUpdateTimes[symbol] = time.Unix(0, 0)
		mtGlobalSilent = time.Now().Add(*mtConfig.RestartSilent)
		swapHttpPositionUpdateSilentTimes[symbol] = time.Now()
	}
}
