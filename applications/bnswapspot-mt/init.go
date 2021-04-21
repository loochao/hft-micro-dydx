package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/geometrybase/hft-micro/bnspot"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/getsentry/raven-go"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"time"
)

var bnInfluxWriter *common.InfluxWriter
var bnExternalInfluxWriter *common.InfluxWriter
var bnLoopTimer *time.Timer

var bnswapAPI *bnswap.API
var bnspotAPI *bnspot.API

var bnswapUserWebsocket *bnswap.UserWebsocket
var bnspotUserWebsocket *bnspot.UserWebsocket

var bnswapOrderSilentTimes = make(map[string]time.Time)
var bnswapPositionsUpdateTimes = make(map[string]time.Time)

var bnspotOrderSilentTimes = make(map[string]time.Time)
var bnspotCancelSilentTimes = make(map[string]time.Time)
var bnspotSilentTimes = make(map[string]time.Time)

var bnspotBalancesUpdateTimes = make(map[string]time.Time)
var bnswapOrderNewErrorCh = make(chan TakerOrderNewError, 10)
var bnswapOrderFinishCh = make(chan bnswap.Order, 10000)

var bnspotHttpBalanceUpdateSilentTimes = make(map[string]time.Time)
var bnswapHttpPositionUpdateSilentTimes = make(map[string]time.Time)

var bnSymbols = make([]string, 0)
var bnSymbolsMap = make(map[string]bool, 0)

var bnOpenLogSilentTimes = make(map[string]time.Time)

var bnswapAccountCh = make(chan bnswap.Account, 10)

var bnswapUSDTAsset *bnswap.Asset
var bnswapBNBAsset *bnswap.Asset

var bnswapTickSizes = make(map[string]float64)
var bnswapStepSizes = make(map[string]float64)
var bnswapMinNotional = make(map[string]float64)

var bnspotTickSizes = make(map[string]float64)
var bnspotStepSizes = make(map[string]float64)
var bnspotMinNotional = make(map[string]float64)

var bnMergedStepSizes = make(map[string]float64)

var bnGlobalCtx context.Context
var bnGlobalCancel context.CancelFunc
var bnswapPositionCh = make(chan []bnswap.Position, 10)
var bnswapPositions = make(map[string]bnswap.Position)

var bnspotBalances = make(map[string]bnspot.Balance)
var bnspotUSDTBalance *bnspot.Balance

var bnswapAssetUpdatedForReBalance = false
var bnspotBalanceUpdatedForReBalance = false
var bnswapAssetUpdatedForInflux = false
var bnspotBalanceUpdatedForInflux = false
var bnswapAssetUpdatedForExternalInflux = false
var bnspotBalanceUpdatedForExternalInflux = false

var bnSaveSilentTime = time.Now()

var bnspotAccountCh = make(chan bnspot.Account, 10)

var bnspotOrderRequestChs = make(map[string]chan SpotOrderRequest)
var bnspotNewOrderResponseCh chan bnspot.NewOrderResponse
var bnspotCancelOrderResponsesCh chan []bnspot.CancelOrderResponse
var bnspotNewOrderErrorCh chan MakerOrderNewError

var bnswapOrderRequestChs = make(map[string]chan bnswap.NewOrderParams)
var bnswapNewOrderErrorCh chan TakerOrderNewError

var bnspotOpenOrders = make(map[string]bnspot.NewOrderParams)
var bnspotOrderCancelCounts = make(map[string]int)

var bnswapPremiumIndexes = make(map[string]bnswap.PremiumIndex)
var bnswapPremiumIndexesCh = make(chan map[string]bnswap.PremiumIndex, 10)
var bnRankSymbolMap map[int]string

var bnswapBarsMapCh = make(chan common.KLinesMap)
var bnswapBarsMap = make(common.KLinesMap)
var bnspotBarsMapCh = make(chan common.KLinesMap)
var bnspotBarsMap = make(common.KLinesMap)
var bnBarsMapUpdated = make(map[string]bool)
var bnBarsMapCh = make(chan [2]common.KLinesMap, 10)
var bnQuantilesCh = make(chan map[string]Quantile)
var bnQuantiles = make(map[string]Quantile)
var bnspotLastFilledBuyPrices = make(map[string]float64)
var bnspotLastFilledSellPrices = make(map[string]float64)
var bnRealisedSpread = make(map[string]float64)
var bnSpreads = make(map[string]Spread)

var bnConfig *Config

const bnBNBSymbol = "BNBUSDT"

func init() {

	logger.Debug("####  BUILD @ 20210421 15:42:24  ####")

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
		bnswapOrderSilentTimes[symbol] = time.Now()
		bnswapPositionsUpdateTimes[symbol] = time.Unix(0, 0)
		bnspotOrderSilentTimes[symbol] = time.Now()
		bnspotBalancesUpdateTimes[symbol] = time.Unix(0, 0)

		bnspotOrderCancelCounts[symbol] = 0

		bnOpenLogSilentTimes[symbol] = time.Now()
		bnspotSilentTimes[symbol] = time.Now().Add(time.Minute)
		bnspotHttpBalanceUpdateSilentTimes[symbol] = time.Now()
		bnswapHttpPositionUpdateSilentTimes[symbol] = time.Now()
	}

	bnBarsMapUpdated["swap"] = false
	bnBarsMapUpdated["spot"] = false

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
