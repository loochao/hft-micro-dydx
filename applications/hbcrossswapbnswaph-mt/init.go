package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/hbcrossswap"
	"github.com/geometrybase/hft-micro/hbspot"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/getsentry/raven-go"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"time"
)

var hSymbols = make([]string, 0)
var bSymbols = make([]string, 0)
var bhSymbolsMap = make(map[string]string, 0)
var hbSymbolsMap = make(map[string]string, 0)

var hbGlobalCtx context.Context
var hbGlobalCancel context.CancelFunc
var hbInfluxWriter *common.InfluxWriter
var hbExternalInfluxWriter *common.InfluxWriter

var hAPI *hbcrossswap.API
var bAPI *bnswap.API

var hUserWebsocket *hbcrossswap.UserWebsocket
var bUserWebsocket *bnswap.UserWebsocket

var hOrderSilentTimes = make(map[string]time.Time)
var hPositionsUpdateTimes = make(map[string]time.Time)
var hbspotCancelSilentTimes = make(map[string]time.Time)
var hSilentTimes = make(map[string]time.Time)
var hHttpPositionUpdateSilentTimes = make(map[string]time.Time)
var hLastOrderTimes = make(map[string]time.Time)

var hbLoopTimer *time.Timer

var hNewOrderErrorCh = make(chan SwapOrderNewError, 10)
var hOrderRequestChs = make(map[string]chan hbcrossswap.NewOrderParam)

var bOrderSilentTimes = make(map[string]time.Time)

var bPositionsUpdateTimes = make(map[string]time.Time)
var bLastOrderTimes = make(map[string]time.Time)


var hOpenLogSilentTimes = make(map[string]time.Time)

var hbcrossswapAccountCh = make(chan hbcrossswap.Account, 10)

var hbcrossswapAccount *hbcrossswap.Account

var hbcrossswapTickSizes = make(map[string]float64)
var hbcrossswapContractSizes = make(map[string]float64)
var hbspotPricePrecisions = make(map[string]int)
var hbspotAmountPrecisions = make(map[string]int)

var hbspotTickSizes = make(map[string]float64)
var hbspotStepSizes = make(map[string]float64)
var hbspotMinSizes = make(map[string]float64)
var hbspotMinNotional = make(map[string]float64)

var hPositionCh = make(chan []hbcrossswap.Position, 10)
var hPositions = make(map[string]hbcrossswap.Position)

var hbspotBalances = make(map[string]*hbspot.Balance)
var hbspotUSDTBalance *hbspot.Balance
var hbcrossswapAssetUpdatedForReBalance = false
var hbspotBalanceUpdatedForReBalance = false
var hbcrossswapAssetUpdatedForInflux = false
var hbspotBalanceUpdatedForInflux = false
var hbcrossswapAssetUpdatedForExternalInflux = false
var hbspotBalanceUpdatedForExternalInflux = false
var hbSaveSilentTime = time.Now()

var hbspotAccountCh = make(chan hbspot.Account, 10)

var hbspotOrderRequestChs = make(map[string]chan SpotOrderRequest)
var hbspotNewOrderErrorCh chan SpotOrderNewError

var hOpenOrders = make(map[string]hbcrossswap.NewOrderParam)
var hOrderCancelCounts = make(map[string]int)

var hFundingRates = make(map[string]hbcrossswap.FundingRate)
var hFundingRatesCh = make(chan map[string]hbcrossswap.FundingRate, 10)

var hFundingRates = make(map[string]hbcrossswap.FundingRate)
var hFundingRatesCh = make(chan map[string]hbcrossswap.FundingRate, 10)

var hbRankSymbolMap map[int]string

var hbcrossswapBarsMapCh = make(chan common.KLinesMap)
var hbcrossswapBarsMap = make(common.KLinesMap)
var hbspotBarsMapCh = make(chan common.KLinesMap)
var hbspotBarsMap = make(common.KLinesMap)
var hBarsMapUpdated = make(map[string]bool)
var hbBarsMapCh = make(chan [2]common.KLinesMap, 10)
var hbQuantilesCh = make(chan map[string]Quantile)
var hbQuantiles = make(map[string]Quantile)
var hLastFilledBuyPrices = make(map[string]float64)
var hLastFilledSellPrices = make(map[string]float64)
var hbRealisedSpread = make(map[string]float64)
var hbSpreads = make(map[string]Spread)
var hbUnHedgeValue float64

var hbConfig *Config

func init() {

	logger.Debug("####  BUILD @ 20210416 04:32:19  ####")

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
	hbConfig = &config

	hostname, err := os.Hostname()
	if err != nil {
		logger.Fatal(err)
	}

	for hSymbol, bSymbol := range hbConfig.SymbolsMap {
		hSymbols = append(hSymbols, hSymbol)
		bSymbols = append(bSymbols, bSymbol)
		bhSymbolsMap[bSymbol] = hSymbol
		hbSymbolsMap[hSymbol] = bSymbol

		hOrderSilentTimes[hSymbol] = time.Now()
		hOrderCancelCounts[hSymbol] = 0
		hOpenLogSilentTimes[hSymbol] = time.Now()
		hSilentTimes[hSymbol] = time.Now().Add(time.Minute)
		hHttpPositionUpdateSilentTimes[hSymbol] = time.Now()
		hPositionsUpdateTimes[hSymbol] = time.Unix(0, 0)
		hLastOrderTimes[hSymbol] = time.Unix(0, 0)

		bOrderSilentTimes[bSymbol] = time.Now()
		bPositionsUpdateTimes[bSymbol] = time.Unix(0, 0)
		bLastOrderTimes[bSymbol] = time.Unix(0, 0)
	}

	hBarsMapUpdated["huobi"] = false
	hBarsMapUpdated["binance"] = false

	err = raven.SetDSN("https://5c318e0f10a349308d2ff86f51de31d8:fa0a8f90a8244c6ea762130cdd6d1bb9@sentry.jilinchen.com/12")

	raven.SetTagsContext(map[string]string{
		"influxAddress":     *hbConfig.InternalInflux.Address,
		"influxDatabase":    *hbConfig.InternalInflux.Address,
		"influxMeasurement": *hbConfig.InternalInflux.Address,
		"BnApiKey":          *hbConfig.InternalInflux.Address,
		"symbols":           fmt.Sprintf("%s", hSymbols),
		"hostname":          hostname,
		"name":              *hbConfig.Name,
	})

	if err != nil {
		logger.Fatal(err)
	}
}
