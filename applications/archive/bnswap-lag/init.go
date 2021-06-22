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

var bnSymbols = make([]string, 0)

var bnGlobalCtx context.Context
var bnGlobalCancel context.CancelFunc
var bnInternalInfluxWriter *common.InfluxWriter
var bnExternalInfluxWriter *common.InfluxWriter
var bnAPI *bnswap.API
var bnUserWebsocket *bnswap.UserWebsocket
var bnHttpPositionUpdateSilentTimes = make(map[string]time.Time)
var bnTickSizes = make(map[string]float64)
var bnStepSizes = make(map[string]float64)
var bnMinNotional = make(map[string]float64)
var bnPositionsCh = make(chan []bnswap.Position, 10)
var bnPositions = make(map[string]*bnswap.Position)
var bnPositionsUpdateTimes = make(map[string]time.Time)
var bnAccount *bnswap.Asset
var bnAccountCh = make(chan bnswap.Account, 10)
var bnNewOrderErrorCh = make(chan OrderNewError, 10)
var bnOrderRequestChs = make(map[string]chan OrderRequest)
var bnOrderSilentTimes = make(map[string]time.Time)
var bnOpenOrders = make(map[string]bnswap.NewOrderParams)

var bnQuantilesCh = make(chan HighLowQuantile, 100)
var bnQuantiles =make(map[string]HighLowQuantile)
var bnBidPrices = make(map[string]BidPrice)
var bnBidPriceCh = make(chan BidPrice, 10000)
var bnTimeEmaDelta *float64
var bnSystemOverHeated = false
var bnTimeEmaDeltaCh = make(chan float64, 10)
var bnLastBuyCosts = make(map[string]float64)
var bnRealisedProfitPcts = make(map[string]float64)
var bnNextLoopTimes = make(map[string]time.Time)
var bnEnterSilentTimes = make(map[string]time.Time)
var bnCollectedBidPrices = make(map[string][]float64)

var bnSymbolsMap = make(map[string]string)

var bnLogSilentTimes = make(map[string]time.Time)
var bnLoopTimer *time.Timer

var bnConfig *Config

func init() {

	logger.Debug("####  BUILD @ 20210423 09:37:03  ####")

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
	bnConfig = &config

	for _, symbol := range bnConfig.Symbols {
		bnSymbols = append(bnSymbols, symbol)
		bnOrderSilentTimes[symbol] = time.Now()
		bnLogSilentTimes[symbol] = time.Now()
		bnEnterSilentTimes[symbol] = time.Now().Add(*bnConfig.SymbolLoopInterval)
		bnPositionsUpdateTimes[symbol] = time.Unix(0, 0)
		bnOrderSilentTimes[symbol] = time.Now()
		bnPositionsUpdateTimes[symbol] = time.Unix(0, 0)
		bnHttpPositionUpdateSilentTimes[symbol] = time.Now()
		bnNextLoopTimes[symbol] = time.Now().Add(*bnConfig.SymbolLoopInterval)
		bnSymbolsMap[symbol] = symbol
		bnCollectedBidPrices[symbol] = make([]float64, 0)
	}
}
