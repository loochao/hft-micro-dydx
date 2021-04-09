package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/geometrybase/hft/bnswap"
	"github.com/geometrybase/hft/common"
	"github.com/geometrybase/hft/logger"
	"github.com/geometrybase/hft/okspot"
	"github.com/getsentry/raven-go"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"time"
)

var boInfluxWriter *common.InfluxWriter
var boExternalInfluxWriter *common.InfluxWriter

var bnswapCredentials *bnswap.Credentials
var okspotCredentials *okspot.Credentials

var bnswapAPI *bnswap.API
var okspotAPI *okspot.API

var bnswapWebsocket *bnswap.Websocket
var bnswapUserWebsocket *bnswap.UserDataWebsocket

var okspotWebsocket *okspot.Websocket

var bnswapOrderSilentTimes = make(map[string]time.Time)
var bnswapPositionsUpdated = make(map[string]bool)

var bnswapOrderBooks = make(map[string]bnswap.PartialBookDepthStream)
var okspotOrderBooks = make(map[string]okspot.WSDepth5)

var bnswapOrderBooksReady = make(map[string]bool)
var okspotOrderBooksReady = make(map[string]bool)

var bnswapOrderBookTimestamps = make(map[string]time.Time)
var okspotOrderBookTimestamps = make(map[string]time.Time)

var okspotOrderSilentTimes = make(map[string]time.Time)

var okspotEnterSilentTimes = make(map[string]time.Time)
var okspotExitSilentTimes = make(map[string]time.Time)

var okspotBalancesUpdated = make(map[string]bool)

var bnswapOrderNewErrorCh = make(chan SwapOrderNewError, 10)
var bnswapOrderFinishCh = make(chan bnswap.Order, 100)

var okspotOrderNewErrorCh = make(chan SpotOrderNewError, 10)
var okspotOrderFinishCh = make(chan okspot.WSOrder, 100)

var boSymbols = make([]string, 0)
var boSymbolsMap = make(map[string]bool, 0)

var bnswapAccountCh = make(chan bnswap.Account, 10)

var bnswapUSDTAsset *bnswap.Asset
var bnswapBNBAsset *bnswap.Asset

var bnswapTickSizes = make(map[string]float64)
var bnswapStepSizes = make(map[string]float64)
var bnswapMinNotional = make(map[string]float64)

var okspotTickSizes = make(map[string]float64)
var okspotStepSizes = make(map[string]float64)
var okspotMinSizes = make(map[string]float64)

var boGlobalCtx context.Context
var boGlobalCancel context.CancelFunc
var bnswapPositionCh = make(chan bnswap.Position, 10)
var bnswapPositions = make(map[string]bnswap.Position)

var okspotBalances = make(map[string]okspot.Balance)
var okspotUSDTBalance *okspot.Balance

var bnswapAssetUpdatedForInflux = false
var okspotBalanceUpdatedForInflux = false
var bnswapAssetUpdatedForExternalInflux = false
var okspotBalanceUpdatedForExternalInflux = false
var bnSaveSilentTime = time.Now()

var okspotBalancesCh = make(chan []okspot.Balance, 10)

var boEnterDeltaWindows = make(map[string][]float64)
var boExitDeltaWindows = make(map[string][]float64)
var boEnterDeltaSortedSlices = make(map[string]common.SortedFloatSlice)
var boExitDeltaSortedSlices = make(map[string]common.SortedFloatSlice)
var boMedianEnterDeltas = make(map[string]float64)
var boMedianExitDeltas = make(map[string]float64)
var boLastEnterDeltas = make(map[string]float64)
var boLastExitDeltas = make(map[string]float64)
var boSwapSpotTimeDeltas = make(map[string]time.Duration)
var boSystemTimeDeltas = make(map[string]time.Duration)
var boArrivalTimes = make(map[string][]time.Time)

var bnswapMarkPrices = make(map[string]bnswap.MarkPriceUpdate)
var okspotBidFarPrices = make(map[string]float64)
var okspotAskFarPrices = make(map[string]float64)
var bnswapBidFarPrices = make(map[string]float64)
var bnswapAskFarPrices = make(map[string]float64)
var okspotBidVwaps = make(map[string]float64)
var okspotAskVwaps = make(map[string]float64)
var bnswapBidVwaps = make(map[string]float64)
var bnswapAskVwaps = make(map[string]float64)
var bnMidVwaps = make(map[string]float64)

var bnswapFundingRateRanks []float64

var bnOpenLogSilentTimes = make(map[string]time.Time)

var bnswapBarsMapCh = make(chan common.OhlcvsMap)
var bnswapBarsMap = make(common.OhlcvsMap)
var okspotBarsMapCh = make(chan common.OhlcvsMap)
var okspotBarsMap = make(common.OhlcvsMap)
var bnBarsMapUpdated = make(map[string]bool)
var bnBarsMapCh = make(chan [2]common.OhlcvsMap, 10)
var bnQuantilesCh = make(chan map[string]Quantile)
var bnQuantiles = make(map[string]Quantile)
var bnSymbolReady = make(map[string]bool)
var okspotLastFilledPrices = make(map[string]float64)
var bnRealisedDelta = make(map[string]float64)
var boConfig *Config

const bnBNBSymbol = "BNBUSDT"
const EPSILON = 1e-10

func init() {

	logger.Debug("####  BUILD @ 20210228 03:43:50  ####")

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
	boConfig = &config

	hostname, err := os.Hostname()
	if err != nil {
		logger.Fatal(err)
	}

	for _, symbol := range boConfig.Symbols {
		boSymbols = append(boSymbols, symbol)
		boSymbolsMap[symbol] = true
	}

	bnBarsMapUpdated["swap"] = false
	bnBarsMapUpdated["spot"] = false

	err = raven.SetDSN("https://5c318e0f10a349308d2ff86f51de31d8:fa0a8f90a8244c6ea762130cdd6d1bb9@sentry.jilinchen.com/12")

	raven.SetTagsContext(map[string]string{
		"influxAddress":     *boConfig.InternalInflux.Address,
		"influxDatabase":    *boConfig.InternalInflux.Address,
		"influxMeasurement": *boConfig.InternalInflux.Address,
		"BnApiKey":          *boConfig.InternalInflux.Address,
		"symbols":           fmt.Sprintf("%s", boSymbols),
		"hostname":          hostname,
		"name":              *boConfig.Name,
	})

	if err != nil {
		logger.Fatal(err)
	}
}
