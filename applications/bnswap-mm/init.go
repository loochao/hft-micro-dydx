package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/getsentry/raven-go"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"time"
)

var bnInternalInfluxWriter *common.InfluxWriter
var bnExternalInfluxWriter *common.InfluxWriter

var bnswapAPI *bnswap.API
var bnswapUserWebsocket *bnswap.UserWebsocket

var bnswapOrderSilentTimes = make(map[string]time.Time)
var bnswapPositionsUpdateTimes = make(map[string]time.Time)
var bnswapCancelSilentTimes = make(map[string]time.Time)

var bnswapOrderNewErrorCh = make(chan SwapOrderNewError, 10)
var bnswapOrderFinishCh = make(chan bnswap.Order, 100)
var bnswapLastOrderTimes = make(map[string]time.Time)

var bnSymbols = make([]string, 0)
var bnSymbolsMap = make(map[string]bool, 0)

var bnswapAccountCh = make(chan bnswap.Account, 10)
var bnswapUSDTAsset *bnswap.Asset
var bnswapBNBAsset *bnswap.Asset

var bnswapTickSizes = make(map[string]float64)
var bnswapStepSizes = make(map[string]float64)
var bnswapMinNotional = make(map[string]float64)

var bnGlobalCtx context.Context
var bnGlobalCancel context.CancelFunc
var bnswapPositionCh = make(chan []bnswap.Position, 10)
var bnswapPositions = make(map[string]bnswap.Position)

var bnswapOrderRequestChs = make(map[string]chan SwapOrderRequest)
var bnswapNewOrderResponseCh chan bnswap.Order
var bnswapLastFilledOrders = make(map[string]bnswap.Order)
var bnswapCancelOrderResponsesCh chan bnswap.CancelAllOrderResponse
var bnswapNewOrderErrorCh chan SwapOrderNewError
var bnSignal Signal

var bnswapOpenOrders = make(map[string]bnswap.NewOrderParams)
var bnswapOrderCancelCounts = make(map[string]int)

var bnswapMarkPrices = make(map[string]bnswap.MarkPrice)
var bnOpenLogSilentTimes = make(map[string]time.Time)

var bnRealisedPnl = make(map[string]float64)
var bnSpreads = make(map[string]Spread)

var bnConfig *Config

const bnBNBSymbol = "BNBUSDT"

func init() {

	logger.Debug("####  BUILD @ 20210410 13:25:42  ####")

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

	hostname, err := os.Hostname()
	if err != nil {
		logger.Fatal(err)
	}

	if !common.StringDataContains(bnConfig.Symbols, bnBNBSymbol) {
		bnConfig.Symbols = append(bnConfig.Symbols, bnBNBSymbol)
	}

	//symbol输入的顺序，先写的合约比较重要，RANK的话是从小到大，所以得Reverse
	for i := len(bnConfig.Symbols) - 1; i >= 0; i-- {
		symbol := bnConfig.Symbols[i]
		bnSymbols = append(bnSymbols, symbol)
		bnSymbolsMap[symbol] = true
		bnswapPositionsUpdateTimes[symbol] = time.Unix(0, 0)
		bnswapOrderSilentTimes[symbol] = time.Now()

		bnswapOrderCancelCounts[symbol] = 0
		bnswapLastOrderTimes[symbol] = time.Unix(0, 0)
		bnOpenLogSilentTimes[symbol] = time.Unix(0, 0)
	}

	err = raven.SetDSN("https://5c318e0f10a349308d2ff86f51de31d8:fa0a8f90a8244c6ea762130cdd6d1bb9@sentry.jilinchen.com/12")

	raven.SetTagsContext(map[string]string{
		"influxAddress":     *bnConfig.InternalInflux.Address,
		"influxDatabase":    *bnConfig.InternalInflux.Address,
		"influxMeasurement": *bnConfig.InternalInflux.Address,
		"BnApiKey":          *bnConfig.InternalInflux.Address,
		"symbols":           fmt.Sprintf("%s", bnSymbols),
		"hostname":          hostname,
		"name":              *bnConfig.Name,
	})

	if err != nil {
		logger.Fatal(err)
	}
}
