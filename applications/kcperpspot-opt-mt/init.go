package main

import (
	"context"
	"flag"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/kcperp"
	"github.com/geometrybase/hft-micro/kcspot"
	"github.com/geometrybase/hft-micro/logger"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"time"
)

var kcInternalInfluxWriter *common.InfluxWriter
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
var kcperpHttpPositionUpdateSilentTimes = make(map[string]time.Time)

var kcspotSymbols = make([]string, 0)
var kcperpSymbols = make([]string, 0)
var kcspSymbolsMap = make(map[string]string, 0)
var kcpsSymbolsMap = make(map[string]string, 0)

var kcOpenLogSilentTimes = make(map[string]time.Time)
var kcUnHedgeLogSilentTime = time.Now()

var kcperpAccountCh = make(chan kcperp.Account, 10)

var kcperpUSDTAccount *kcperp.Account

var kcperpMultipliers = make(map[string]float64)

var kcspotTickSizes = make(map[string]float64)
var kcspotStepSizes = make(map[string]float64)
var kcspotMinNotional = make(map[string]float64)
var kcMergedStepSizes = make(map[string]float64)

var kcGlobalCtx context.Context
var kcGlobalCancel context.CancelFunc
var kcperpPositionCh = make(chan []kcperp.Position, 10)
var kcperpPositions = make(map[string]kcperp.Position)

var kcspotBalances = make(map[string]kcspot.Account)
var kcspotUSDTBalance *kcspot.Account

var kcspotAccountCh = make(chan []kcspot.Account, 10)

var kcspotOrderRequestChs = make(map[string]chan SpotOrderRequest)
var kcspotNewOrderErrorCh chan SpotOrderNewError

var kcspotOpenOrders = make(map[string]kcspot.NewOrderParam)
var kcspotOrderCancelCounts = make(map[string]int)

var kcperpFundingRates = make(map[string]kcperp.CurrentFundingRate)
var kcperpFundingRatesCh = make(chan kcperp.CurrentFundingRate, 1000)
var kcRankSymbolMap map[int]string

var kcspotLastFilledBuyPrices = make(map[string]float64)
var kcspotLastFilledSellPrices = make(map[string]float64)
var kcRealisedSpread = make(map[string]float64)
var kcSpreads = make(map[string]*common.ShortSpread)
var kcUnHedgeValue float64
var kcLoopTimer = time.NewTimer(time.Hour * 24)
var kcspotSystemReady = false
var kcperpSystemReady = false
var kcspotSystemStatusCh = make(chan bool, 10)
var kcperpSystemStatusCh = make(chan bool, 10)
var kcGlobalSilent time.Time
var kcspotOffsets = make(map[string]Offset)

var kcConfig *Config

func init() {

	logger.Debug("####  BUILD @ 20210506 13:18:47  ####")

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
	for spotSymbol, perpSymbol := range kcConfig.SpotPerpPairs {

		if offsets, ok := kcConfig.OrderOffsets[spotSymbol]; ok {
			kcspotOffsets[spotSymbol], err = NewOffset(offsets)
			if err != nil {
				logger.Fatal("NewOffset for %s error %v", spotSymbol, err)
			}
		} else {
			logger.Fatal("MISS OFFSET FOR %s", spotSymbol)
		}

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
		kcspotSilentTimes[spotSymbol] = time.Now().Add(time.Minute * 5)

		kcspotHttpBalanceUpdateSilentTimes[spotSymbol] = time.Now()
		kcperpHttpPositionUpdateSilentTimes[perpSymbol] = time.Now()
	}

	kcGlobalSilent = time.Now().Add(*kcConfig.EnterSilent)
}
