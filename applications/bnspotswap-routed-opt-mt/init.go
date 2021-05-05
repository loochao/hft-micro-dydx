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

var bnInternalInfluxWriter *common.InfluxWriter
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
var bnswapOrderResponseCh = make(chan bnswap.Order, 10)

var bnspotHttpBalanceUpdateSilentTimes = make(map[string]time.Time)
var bnswapHttpPositionUpdateSilentTimes = make(map[string]time.Time)

var bnSymbols = make([]string, 0)

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
var bnswapAvgFundingRate *float64
var bnswapAvgFundingRateCh = make(chan float64, 100)
var bnRankSymbolMap map[int]string

var bnswapBarsMapCh = make(chan common.KLinesMap, 10)
var bnswapBarsMap common.KLinesMap
var bnspotBarsMapCh = make(chan common.KLinesMap, 10)
var bnspotBarsMap common.KLinesMap
var bnBarsMapUpdated = make(map[string]bool)
var bnBarsMapCh = make(chan [2]common.KLinesMap, 100)
var bnQuantilesCh = make(chan map[string]Quantile, 100)
var bnQuantiles = make(map[string]Quantile)
var bnspotLastLimitBuyPrices = make(map[string]float64)
var bnspotLastLimitSellPrices = make(map[string]float64)
var bnRealisedSpread = make(map[string]float64)
var bnSpreads = make(map[string]*common.MakerTakerSpread)
var bnGlobalSilent time.Time
var bnspotSystemStatusCh = make(chan bool, 100)
var bnswapSystemStatusCh = make(chan bool, 100)
var bnspotSystemReady = false
var bnswapSystemReady = false
var bnUnHedgeValue float64
var bnGlobalLogSilentTime = time.Now()
var bnspotOffsets = make(map[string]Offset)
var bnExpectedInsuranceFund float64

var bnConfig *Config

const bnBNBSymbol = "BNBUSDT"

func init() {

	logger.Debug("####  BUILD @ 20210505 01:49:03  ####")

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

	for symbol, offsets := range bnConfig.OrderOffsets {
		bnSymbols = append(bnSymbols, symbol)
		bnspotOffsets[symbol], err = NewOffset(offsets)
		if err != nil {
			logger.Fatal(err)
		}
		logger.Debugf("%s %s", symbol, bnspotOffsets[symbol].ToString())
		bnswapOrderSilentTimes[symbol] = time.Now()
		bnswapPositionsUpdateTimes[symbol] = time.Unix(0, 0)
		bnspotOrderSilentTimes[symbol] = time.Now()
		bnspotBalancesUpdateTimes[symbol] = time.Unix(0, 0)

		bnspotOrderCancelCounts[symbol] = 0

		bnOpenLogSilentTimes[symbol] = time.Now()
		bnspotSilentTimes[symbol] = time.Now().Add(time.Minute)
		bnspotHttpBalanceUpdateSilentTimes[symbol] = time.Now()
		bnswapHttpPositionUpdateSilentTimes[symbol] = time.Now()
		bnGlobalSilent = time.Now().Add(*bnConfig.RestartSilent)
	}
	if !common.StringDataContains(bnSymbols, bnBNBSymbol) {
		bnSymbols = append(bnSymbols, bnBNBSymbol)
		bnspotOffsets[bnBNBSymbol] = Offset{}
	}

	bnExpectedInsuranceFund = *bnConfig.StartValue * (1 - *bnConfig.InsuranceFundingRatio) * *bnConfig.Leverage / (*bnConfig.Leverage + 1) * *bnConfig.InsuranceFundingRatio
	logger.Debugf("bnExpectedInsuranceFund %f", bnExpectedInsuranceFund)

	bnBarsMapUpdated["swap"] = false
	bnBarsMapUpdated["spot"] = false
}
