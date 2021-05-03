package main

import (
	"context"
	"flag"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/hbcrossswap"
	"github.com/geometrybase/hft-micro/hbspot"
	"github.com/geometrybase/hft-micro/logger"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"time"
)

var hbInternalInfluxWriter *common.InfluxWriter
var hbExternalInfluxWriter *common.InfluxWriter

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
var hbUnHedgeLogSilentTime = time.Now()
var hbLoopTimer *time.Timer

var hbspotBalancesUpdateTimes = make(map[string]time.Time)
var hbcrossswapNewOrderErrorCh = make(chan SwapOrderNewError, 10)
var hbcrossswapOrderRequestChs = make(map[string]chan hbcrossswap.NewOrderParam)

var hbspotHttpBalanceUpdateSilentTimes = make(map[string]time.Time)
var hbcrossswapHttpPositionUpdateSilentTimes = make(map[string]time.Time)

var hbspotSymbols = make([]string, 0)
var hbcrossswapSymbols = make([]string, 0)
var hbSpotSwapSymbolsMap = make(map[string]string, 0)
var hbSwapSpotSymbolsMap = make(map[string]string, 0)

var hbLogSilentTimes = make(map[string]time.Time)

var hbcrossswapAccountCh = make(chan hbcrossswap.Account, 10)

var hbcrossswapAccount *hbcrossswap.Account

var hbcrossswapContractSizes = make(map[string]float64)
var hbspotPricePrecisions = make(map[string]int)
var hbspotAmountPrecisions = make(map[string]int)

var hbspotTickSizes = make(map[string]float64)
var hbspotStepSizes = make(map[string]float64)
var hbspotMinNotional = make(map[string]float64)

var hbMergedStepSizes = make(map[string]float64)

var hbGlobalCtx context.Context
var hbGlobalCancel context.CancelFunc
var hbcrossswapPositionCh = make(chan []hbcrossswap.Position, 10)
var hbcrossswapPositions = make(map[string]hbcrossswap.Position)

var hbspotBalances = make(map[string]*hbspot.Balance)
var hbspotUSDTBalance *hbspot.Balance
var hbcrossswapAssetUpdatedForInflux = false
var hbspotBalanceUpdatedForInflux = false
var hbcrossswapAssetUpdatedForExternalInflux = false
var hbspotBalanceUpdatedForExternalInflux = false

var hbspotAccountCh = make(chan hbspot.Account, 10)

var hbspotOrderRequestChs = make(map[string]chan SpotOrderRequest)
var hbspotNewOrderErrorCh chan SpotOrderNewError

var hbspotOpenOrders = make(map[string]hbspot.NewOrderParam)
var hbspotOrderCancelCounts = make(map[string]int)

var hbcrossswapFundingRates = make(map[string]hbcrossswap.FundingRate)
var hbcrossswapFundingRatesCh = make(chan map[string]hbcrossswap.FundingRate, 10)
var hbRankSymbolMap map[int]string

var hbcrossswapBarsMapCh = make(chan common.KLinesMap)
var hbcrossswapBarsMap = make(common.KLinesMap)
var hbspotBarsMapCh = make(chan common.KLinesMap)
var hbspotBarsMap = make(common.KLinesMap)
var hbBarsMapUpdated = make(map[string]bool)
var hbBarsMapCh = make(chan [2]common.KLinesMap, 10)
var hbQuantilesCh = make(chan map[string]Quantile)
var hbQuantiles = make(map[string]Quantile)
var hbspotLastFilledBuyPrices = make(map[string]float64)
var hbspotLastFilledSellPrices = make(map[string]float64)
var hbRealisedSpread = make(map[string]float64)
var hbSpreads = make(map[string]*common.MakerTakerSpread)

var hbUnHedgeValue float64

var hbspotSystemReady = false
var hbcrossswapSystemReady = false
var hbspotSystemStatusCh = make(chan bool, 10)
var hbcrossswapSystemStatusCh = make(chan bool, 10)
var hbGlobalSilent time.Time

var hbConfig *Config

func init() {

	logger.Debug("####  BUILD @ 20210503 16:48:29  ####")

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
	for spotSymbol, swapSymbol := range hbConfig.SpotSwapPairs {
		hbspotSymbols = append(hbspotSymbols, spotSymbol)
		hbcrossswapSymbols = append(hbcrossswapSymbols, swapSymbol)
		hbSpotSwapSymbolsMap[spotSymbol] = swapSymbol
		hbSwapSpotSymbolsMap[swapSymbol] = spotSymbol

		hbcrossswapOrderSilentTimes[swapSymbol] = time.Now()
		hbcrossswapPositionsUpdateTimes[swapSymbol] = time.Unix(0, 0)

		hbspotOrderSilentTimes[spotSymbol] = time.Now()
		hbspotBalancesUpdateTimes[spotSymbol] = time.Unix(0, 0)

		hbspotOrderCancelCounts[spotSymbol] = 0

		hbLogSilentTimes[spotSymbol] = time.Now()
		hbspotSilentTimes[spotSymbol] = time.Now().Add(time.Minute)
		hbspotHttpBalanceUpdateSilentTimes[spotSymbol] = time.Now()
		hbcrossswapHttpPositionUpdateSilentTimes[swapSymbol] = time.Now()

	}

	hbGlobalSilent = time.Now().Add(*hbConfig.RestartSilent)

	hbBarsMapUpdated["swap"] = false
	hbBarsMapUpdated["spot"] = false
}
