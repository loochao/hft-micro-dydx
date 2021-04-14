package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/kcperp"
	"github.com/geometrybase/hft-micro/kcspot"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/getsentry/raven-go"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"time"
)

var kcInfluxWriter *common.InfluxWriter
var kcExternalInfluxWriter *common.InfluxWriter

var kcperpAPI *kcperp.API
var kcspotAPI *kcspot.API

var kcperpUserWebsocket *kcperp.UserWebsocket
var kcspotUserWebsocket *kcspot.UserWebsocket

var kcperpOrderSilentTimes = make(map[string]time.Time)
var kcperpPositionsUpdateTimes = make(map[string]time.Time)

var kcspotOrderSilentTimes = make(map[string]time.Time)
var kcspotCancelSilentTimes = make(map[string]time.Time)
var kcspotSilentTimes = make(map[string]time.Time)

var kcspotBalancesUpdateTimes = make(map[string]time.Time)
var kcperpNewOrderErrorCh = make(chan PerpOrderNewError, 10)
var kcperpOrderRequestChs = make(map[string]chan kcperp.NewOrderParam)

var kcspotHttpBalanceUpdateSilentTimes = make(map[string]time.Time)
var kcspotLastOrderTimes = make(map[string]time.Time)
var kcperpLastOrderTimes = make(map[string]time.Time)

var kcspotSymbols = make([]string, 0)
var kcperpSymbols = make([]string, 0)
var kcspSymbolsMap = make(map[string]string, 0)
var kcpsSymbolsMap = make(map[string]string, 0)

var kcOpenLogSilentTimes = make(map[string]time.Time)

var kcperpAccountCh = make(chan kcperp.Account, 10)

var kcperpUSDTAccount *kcperp.Account

var kcperpTickSizes = make(map[string]float64)
var kcperpLotSizes = make(map[string]float64)
var kcperpMultipliers = make(map[string]float64)

var kcspotTickSizes = make(map[string]float64)
var kcspotStepSizes = make(map[string]float64)
var kcspotMinNotional = make(map[string]float64)

var kcGlobalCtx context.Context
var kcGlobalCancel context.CancelFunc
var kcperpPositionCh = make(chan []kcperp.Position, 10)
var kcperpPositions = make(map[string]kcperp.Position)

var kcspotBalances = make(map[string]kcspot.Account)
var kcspotUSDTBalance *kcspot.Account
var kcperpAssetUpdatedForReBalance = false
var kcspotBalanceUpdatedForReBalance = false
var kcperpAssetUpdatedForInflux = false
var kcspotBalanceUpdatedForInflux = false
var kcperpAssetUpdatedForExternalInflux = false
var kcspotBalanceUpdatedForExternalInflux = false
var kcSaveSilentTime = time.Now()

var kcspotAccountCh = make(chan []kcspot.Account, 10)

var kcspotOrderRequestChs = make(map[string]chan SpotOrderRequest)
var kcspotNewOrderErrorCh chan SpotOrderNewError

var kcspotOpenOrders = make(map[string]kcspot.NewOrderParam)
var kcspotOrderCancelCounts = make(map[string]int)

var kcperpMarkPrices = make(map[string]*kcperp.MarkPrice)
var kcperpFundingRates = make(map[string]*kcperp.FundingRate)
var kcRankSymbolMap map[int]string

var kcperpBarsMapCh = make(chan common.KLinesMap)
var kcperpBarsMap = make(common.KLinesMap)
var kcspotBarsMapCh = make(chan common.KLinesMap)
var kcspotBarsMap = make(common.KLinesMap)
var kcBarsMapUpdated = make(map[string]bool)
var kcBarsMapCh = make(chan [2]common.KLinesMap, 10)
var kcQuantilesCh = make(chan map[string]Quantile)
var kcQuantiles = make(map[string]Quantile)
var kcspotLastFilledBuyPrices = make(map[string]float64)
var kcspotLastFilledSellPrices = make(map[string]float64)
var kcRealisedSpread = make(map[string]float64)
var kcSpreads = make(map[string]Spread)
var kcUnHedgeValue float64

var kcConfig *Config

func init() {

	logger.Debug("####  BUILD @ 20210414 03:35:38  ####")

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
	kcConfig = &config

	hostname, err := os.Hostname()
	if err != nil {
		logger.Fatal(err)
	}

	for spotSymbol, perpSymbol := range kcConfig.SpotPerpPairs {
		kcspotSymbols = append(kcspotSymbols, spotSymbol)
		kcperpSymbols = append(kcperpSymbols, perpSymbol)
		kcspSymbolsMap[spotSymbol] = perpSymbol
		kcpsSymbolsMap[perpSymbol] = spotSymbol

		kcperpOrderSilentTimes[perpSymbol] = time.Now()
		kcperpPositionsUpdateTimes[perpSymbol] = time.Unix(0, 0)

		kcspotOrderSilentTimes[spotSymbol] = time.Now()
		kcspotBalancesUpdateTimes[spotSymbol] = time.Unix(0, 0)

		kcspotOrderCancelCounts[spotSymbol] = 0

		kcOpenLogSilentTimes[spotSymbol] = time.Now()
		kcspotSilentTimes[spotSymbol] = time.Now().Add(time.Minute)
		kcspotHttpBalanceUpdateSilentTimes[spotSymbol] = time.Now()

		kcperpLastOrderTimes[perpSymbol] = time.Unix(0, 0)
		kcspotLastOrderTimes[spotSymbol] = time.Unix(0, 0)
	}

	kcBarsMapUpdated["swap"] = false
	kcBarsMapUpdated["spot"] = false

	err = raven.SetDSN("https://5c318e0f10a349308d2ff86f51de31d8:fa0a8f90a8244c6ea762130cdd6d1bb9@sentry.jilinchen.com/12")

	raven.SetTagsContext(map[string]string{
		"influxAddress":     *kcConfig.InternalInflux.Address,
		"influxDatabase":    *kcConfig.InternalInflux.Address,
		"influxMeasurement": *kcConfig.InternalInflux.Address,
		"BnApiKey":          *kcConfig.InternalInflux.Address,
		"symbols":           fmt.Sprintf("%s", kcspotSymbols),
		"hostname":          hostname,
		"name":              *kcConfig.Name,
	})

	if err != nil {
		logger.Fatal(err)
	}
}
