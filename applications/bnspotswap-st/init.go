package main

import (
	"context"
	"flag"
	"github.com/geometrybase/hft-micro/bnspot"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"time"
)

var mSymbols = make([]string, 0)
var tSymbols = make([]string, 0)
var tmSymbolsMap = make(map[string]string, 0)
var mtSymbolsMap = make(map[string]string, 0)

var mtGlobalCtx context.Context
var mtGlobalCancel context.CancelFunc
var mtInfluxWriter *common.InfluxWriter
var mtExternalInfluxWriter *common.InfluxWriter

var mAPI *bnspot.API
var tAPI *bnswap.API

var mUserWebsocket *bnspot.UserWebsocket
var tUserWebsocket *bnswap.UserWebsocket

var mHttpPositionUpdateSilentTimes = make(map[string]time.Time)
var tHttpPositionUpdateSilentTimes = make(map[string]time.Time)

var mTickSizes = make(map[string]float64)
var mMultipliers = make(map[string]float64)
var tTickSizes = make(map[string]float64)
var tStepSizes = make(map[string]float64)
var tMinNotional = make(map[string]float64)
var mtStepSizes = make(map[string]float64)

var tOpenOrders = make(map[string]TakerOpenOrder)
var tOrderCancelCounts = make(map[string]int)
var tOrderCancelSilentTimes = make(map[string]time.Time)
var tOpenOrderCh = make(chan TakerOpenOrder, 10000)
var tPositionsCh = make(chan []bnswap.Position, 10)
var tPositions = make(map[string]*bnswap.Position)
var tPositionsUpdateTimes = make(map[string]time.Time)
var tAccount *bnswap.Asset
var tAccountCh = make(chan bnswap.Account, 10)
var tNewOrderErrorCh = make(chan TakerOrderNewError, 10)
var tOrderRequestChs = make(map[string]chan TakerOrderRequest)
var tOrderSilentTimes = make(map[string]time.Time)
var mtEnterSilentTimes = make(map[string]time.Time)

var mtSpreads = make(map[string]*common.ShortSpread)

var mLastFilledBuyPrices = make(map[string]float64)
var mLastFilledSellPrices = make(map[string]float64)
var mtRealisedSpread = make(map[string]float64)

var mtCloseTimeouts = make(map[string]time.Time)
var mtEnterTimeouts = make(map[string]time.Time)

var mtLogSilentTimes = make(map[string]time.Time)
var mtLoopTimer *time.Timer
var mSystemReady = false
var tSystemReady = false
var mSystemStatusCh = make(chan bool, 10)
var tSystemStatusCh = make(chan bool, 10)
var mtGlobalSilent = time.Now()
var mtTriggeredDirection = make(map[string]int)

var mtConfig *Config

func init() {

	logger.Debug("####  BUILD @ 20210427 04:34:05  ####")

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

	for makerSymbol, takerSymbol := range mtConfig.MakerTakerSymbolsMap {
		mSymbols = append(mSymbols, makerSymbol)
		tSymbols = append(tSymbols, takerSymbol)
		tmSymbolsMap[takerSymbol] = makerSymbol
		mtSymbolsMap[makerSymbol] = takerSymbol
		mtLogSilentTimes[makerSymbol] = time.Now()

		tOrderSilentTimes[takerSymbol] = time.Now()
		tPositionsUpdateTimes[takerSymbol] = time.Unix(0, 0)

		mHttpPositionUpdateSilentTimes[makerSymbol] = time.Now()
		tHttpPositionUpdateSilentTimes[makerSymbol] = time.Now()

		mtCloseTimeouts[takerSymbol] = time.Now()
		mtEnterSilentTimes[takerSymbol] = time.Now()
		mtEnterTimeouts[takerSymbol] = time.Now()
	}
}
