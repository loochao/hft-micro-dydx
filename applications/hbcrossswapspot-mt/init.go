package main

import (
	"context"
	"flag"
	"fmt"
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

var hbInfluxWriter *common.InfluxWriter
var kcExternalInfluxWriter *common.InfluxWriter

var hbcrossswapAPI *hbcrossswap.API
var hbspotAPI *hbspot.API
var hbspotAccountID int64

var hbcrossswapUserWebsocket *hbcrossswap.UserWebsocket
var hbspotUserWebsocket *hbspot.UserWebsocket

var hbcrossswapOrderSilentTimes = make(map[string]time.Time)
var hbcrossswapPositionsUpdateTimes = make(map[string]time.Time)

var hbspotOrderSilentTimes = make(map[string]time.Time)
var hbspotCancelSilentTimes = make(map[string]time.Time)
var hbspotSilentTimes = make(map[string]time.Time)

var hbspotBalancesUpdateTimes = make(map[string]time.Time)
var hbcrossswapNewOrderErrorCh = make(chan SwapOrderNewError, 10)
var hbcrossswapOrderRequestChs = make(map[string]chan hbcrossswap.NewOrderParam)

var hbspotHttpBalanceUpdateSilentTimes = make(map[string]time.Time)
var hbspotLastOrderTimes = make(map[string]time.Time)
var hbcrossswapLastOrderTimes = make(map[string]time.Time)

var hbspotSymbols = make([]string, 0)
var hbcrossswapSymbols = make([]string, 0)
var kcspSymbolsMap = make(map[string]string, 0)
var kcpsSymbolsMap = make(map[string]string, 0)

var kcOpenLogSilentTimes = make(map[string]time.Time)

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

var hbGlobalCtx context.Context
var hbGlobalCancel context.CancelFunc
var hbcrossswapPositionCh = make(chan []hbcrossswap.Position, 10)
var hbcrossswapPositions = make(map[string]hbcrossswap.Position)

var hbspotBalances = make(map[string]*hbspot.Balance)
var hbspotUSDTBalance *hbspot.Balance
var hbcrossswapAssetUpdatedForReBalance = false
var hbspotBalanceUpdatedForReBalance = false
var hbcrossswapAssetUpdatedForInflux = false
var hbspotBalanceUpdatedForInflux = false
var hbcrossswapAssetUpdatedForExternalInflux = false
var hbspotBalanceUpdatedForExternalInflux = false
var kcSaveSilentTime = time.Now()

//var hbspotAccountCh = make(chan hbspot.Account, 10)

var hbspotOrderRequestChs = make(map[string]chan SpotOrderRequest)
var hbspotNewOrderErrorCh chan SpotOrderNewError

var hbspotOpenOrders = make(map[string]hbspot.NewOrderParam)
var hbspotOrderCancelCounts = make(map[string]int)

var hbcrossswapFundingRates = make(map[string]hbcrossswap.FundingRate)
var hbcrossswapFundingRatesCh = make(chan map[string]hbcrossswap.FundingRate, 10)
var kcRankSymbolMap map[int]string

var hbcrossswapBarsMapCh = make(chan common.KLinesMap)
var hbcrossswapBarsMap = make(common.KLinesMap)
var hbspotBarsMapCh = make(chan common.KLinesMap)
var hbspotBarsMap = make(common.KLinesMap)
var kcBarsMapUpdated = make(map[string]bool)
var kcBarsMapCh = make(chan [2]common.KLinesMap, 10)
var kcQuantilesCh = make(chan map[string]Quantile)
var kcQuantiles = make(map[string]Quantile)
var hbspotLastFilledBuyPrices = make(map[string]float64)
var hbspotLastFilledSellPrices = make(map[string]float64)
var kcRealisedSpread = make(map[string]float64)
var hbSpreads = make(map[string]Spread)
var kcUnHedgeValue float64

var hbConfig *Config

func init() {

	logger.Debug("####  BUILD @ 20210415 17:37:30  ####")

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

	for spotSymbol, swapSymbol := range hbConfig.SpotSwapPairs {
		hbspotSymbols = append(hbspotSymbols, spotSymbol)
		hbcrossswapSymbols = append(hbcrossswapSymbols, swapSymbol)
		kcspSymbolsMap[spotSymbol] = swapSymbol
		kcpsSymbolsMap[swapSymbol] = spotSymbol

		hbcrossswapOrderSilentTimes[swapSymbol] = time.Now()
		hbcrossswapPositionsUpdateTimes[swapSymbol] = time.Unix(0, 0)

		hbspotOrderSilentTimes[spotSymbol] = time.Now()
		hbspotBalancesUpdateTimes[spotSymbol] = time.Unix(0, 0)

		hbspotOrderCancelCounts[spotSymbol] = 0

		kcOpenLogSilentTimes[spotSymbol] = time.Now()
		hbspotSilentTimes[spotSymbol] = time.Now().Add(time.Minute)
		hbspotHttpBalanceUpdateSilentTimes[spotSymbol] = time.Now()

		hbcrossswapLastOrderTimes[swapSymbol] = time.Unix(0, 0)
		hbspotLastOrderTimes[spotSymbol] = time.Unix(0, 0)
	}

	kcBarsMapUpdated["swap"] = false
	kcBarsMapUpdated["spot"] = false

	err = raven.SetDSN("https://5c318e0f10a349308d2ff86f51de31d8:fa0a8f90a8244c6ea762130cdd6d1bb9@sentry.jilinchen.com/12")

	raven.SetTagsContext(map[string]string{
		"influxAddress":     *hbConfig.InternalInflux.Address,
		"influxDatabase":    *hbConfig.InternalInflux.Address,
		"influxMeasurement": *hbConfig.InternalInflux.Address,
		"BnApiKey":          *hbConfig.InternalInflux.Address,
		"symbols":           fmt.Sprintf("%s", hbspotSymbols),
		"hostname":          hostname,
		"name":              *hbConfig.Name,
	})

	if err != nil {
		logger.Fatal(err)
	}
}
