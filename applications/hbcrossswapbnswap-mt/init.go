package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/hbcrossswap"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/getsentry/raven-go"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
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

var mAPI *hbcrossswap.API
var tAPI *bnswap.API

var mUserWebsocket *hbcrossswap.UserWebsocket
var tUserWebsocket *bnswap.UserWebsocket

var mOrderSilentTimes = make(map[string]time.Time)
var mPositionsUpdateTimes = make(map[string]time.Time)
var hbspotCancelSilentTimes = make(map[string]time.Time)
var mSilentTimes = make(map[string]time.Time)
var hHttpPositionUpdateSilentTimes = make(map[string]time.Time)
var mLastOrderTimes = make(map[string]time.Time)

var mtLoopTimer *time.Timer

var tNewOrderErrorCh = make(chan TakerOrderNewError, 10)
var tOrderRequestChs = make(map[string]chan bnswap.NewOrderParams)

var tOrderSilentTimes = make(map[string]time.Time)

var mPositionsUpdateTimes = make(map[string]time.Time)
var tLastOrderTimes = make(map[string]time.Time)

var mLogSilentTimes = make(map[string]time.Time)
var mtUnHedgeLogSilentTimes = time.Unix(0, 0)

var mAccountCh = make(chan hbcrossswap.Account, 10)
var mAccount *hbcrossswap.Account

var mTickSizes = make(map[string]float64)
var mContractSizes = make(map[string]float64)
var hbspotPricePrecisions = make(map[string]int)
var hbspotAmountPrecisions = make(map[string]int)

var mtStepSizes = make(map[string]float64)

var tTickSizes = make(map[string]float64)
var tStepSizes = make(map[string]float64)
var bMinSizes = make(map[string]float64)
var tMinNotional = make(map[string]float64)

var mPositionCh = make(chan []hbcrossswap.Position, 10)
var mPositions = make(map[string]hbcrossswap.Position)

var tPositions = make(map[string]*bnswap.Position)
var tPositionsCh = make(chan []bnswap.Position, 10)
var tPositionsUpdateTimes = make(map[string]time.Time)
var tUSDTAsset *bnswap.Asset
var tBNBAsset *bnswap.Asset
var bAccountCh = make(chan bnswap.Account, 10)
var hbcrossswapAssetUpdatedForReBalance = false
var hbspotBalanceUpdatedForReBalance = false
var bAssetUpdatedForInflux = false
var hAccountUpdatedForInflux = false
var hbcrossswapAssetUpdatedForExternalInflux = false
var hbspotBalanceUpdatedForExternalInflux = false
var hbSaveSilentTime = time.Now()

var tAccountCh = make(chan bnswap.Account, 10)

var mOrderRequestChs = make(map[string]chan MakerOrderRequest)
var mNewOrderErrorCh chan HOrderNewError

var mOpenOrders = make(map[string]hbcrossswap.NewOrderParam)
var mOrderCancelCounts = make(map[string]int)

var mFundingRates = make(map[string]hbcrossswap.FundingRate)
var mFundingRatesCh = make(chan map[string]hbcrossswap.FundingRate, 10)
var tPremiumIndexes = make(map[string]bnswap.PremiumIndex)
var tPremiumIndexesCh = make(chan map[string]bnswap.PremiumIndex, 10)
var mtFundingRates = make(map[string]float64)

var mtTradeDirections = make(map[string]int)

var mtRankSymbolMap map[int]string

var mBarsMapCh = make(chan common.KLinesMap)
var mBarsMap = make(common.KLinesMap)
var tBarsMapCh = make(chan common.KLinesMap)
var tBarsMap = make(common.KLinesMap)

var mtMapUpdated = make(map[string]bool)
var mtBarsMapCh = make(chan [2]common.KLinesMap, 10)

var hbQuantilesCh = make(chan map[string]HBDeltaQuantile)
var mtQuantiles = make(map[string]HBDeltaQuantile)
var mLastFilledBuyPrices = make(map[string]float64)
var mLastFilledSellPrices = make(map[string]float64)
var hbRealisedSpread = make(map[string]float64)
var mtSpreads = make(map[string]Spread)
var mtUnHedgeValue float64

var mtConfig *Config

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
	mtConfig = &config

	hostname, err := os.Hostname()
	if err != nil {
		logger.Fatal(err)
	}

	for hSymbol, bSymbol := range mtConfig.SymbolsMap {
		mSymbols = append(mSymbols, hSymbol)
		tSymbols = append(tSymbols, bSymbol)
		tmSymbolsMap[bSymbol] = hSymbol
		mtSymbolsMap[hSymbol] = bSymbol

		mOrderSilentTimes[hSymbol] = time.Now()
		mOrderCancelCounts[hSymbol] = 0
		mLogSilentTimes[hSymbol] = time.Now()
		mSilentTimes[hSymbol] = time.Now().Add(time.Minute)
		hHttpPositionUpdateSilentTimes[hSymbol] = time.Now()
		mPositionsUpdateTimes[hSymbol] = time.Unix(0, 0)
		mLastOrderTimes[hSymbol] = time.Unix(0, 0)

		tOrderSilentTimes[bSymbol] = time.Now()
		mPositionsUpdateTimes[bSymbol] = time.Unix(0, 0)
		tLastOrderTimes[bSymbol] = time.Unix(0, 0)
	}

	mtMapUpdated["huobi"] = false
	mtMapUpdated["binance"] = false

	err = raven.SetDSN("https://5c318e0f10a349308d2ff86f51de31d8:fa0a8f90a8244c6ea762130cdd6d1bb9@sentry.jilinchen.com/12")

	raven.SetTagsContext(map[string]string{
		"influxAddress":     *mtConfig.InternalInflux.Address,
		"influxDatabase":    *mtConfig.InternalInflux.Address,
		"influxMeasurement": *mtConfig.InternalInflux.Address,
		"BnApiKey":          *mtConfig.InternalInflux.Address,
		"symbols":           fmt.Sprintf("%s", mSymbols),
		"hostname":          hostname,
		"name":              *mtConfig.Name,
	})

	if err != nil {
		logger.Fatal(err)
	}
}
