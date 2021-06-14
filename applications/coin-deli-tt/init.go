package main

import (
	"context"
	"flag"
	"fmt"
	bncf "github.com/geometrybase/hft-micro/binance-coinfuture"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"time"
)

var xSymbols = make([]string, 0)
var ySymbols = make([]string, 0)
var xySymbols = make([]string, 0)
var yxSymbolsMap = make(map[string]string, 0)
var xySymbolsMap = make(map[string]string, 0)

var xyGlobalCtx context.Context
var xyGlobalCancel context.CancelFunc
var xyInfluxWriter *common.InfluxWriter
var xyExternalInfluxWriter *common.InfluxWriter

var xyMultipliers = make(map[string]float64)
var xyStepSizes = make(map[string]float64)
var xyMinNotionals = make(map[string]float64)

var xyBalanceMap = make(map[string]common.Balance)
var xyBalanceCh = make(chan common.Balance, 200)
var xyPositionCh = make(chan common.Position, 200)
var xyOrderCh = make(chan common.Order, 200)

var xyPositionsUpdateTimes = make(map[string]time.Time)
var xyPositions = make(map[string]common.Position)

var xyOrderRequestChMap = make(map[string]chan common.OrderRequest)
var xyNewOrderErrorCh = make(chan common.OrderError, 200)
var xyOrderSilentTimes = make(map[string]time.Time)

var xFundingRates = make(map[string]common.FundingRate)
var xFundingRateCh = make(chan common.FundingRate, 200)

var xySpreads = make(map[string]*XYSpread)

var xLastFilledBuyPrices = make(map[string]float64)
var xLastFilledSellPrices = make(map[string]float64)
var yLastFilledBuyPrices = make(map[string]float64)
var yLastFilledSellPrices = make(map[string]float64)
var xyRealisedSpread = make(map[string]float64)

var xyLogSilentTimes = make(map[string]time.Time)
var xyLoopTimer *time.Timer
var xySystemStatus = common.SystemStatusNotReady
var xySystemStatusCh = make(chan common.SystemStatus, 100)

var xyTargetValues = make(map[string]float64)
var xyTargetPositionUpdateSilentTimes = make(map[string]time.Time)

var xyConfig *Config

var xyExchange common.Exchange

var xyTimedPositionChange *common.TimedSum

func init() {

	logger.Debug("####  BUILD @ 20210614 13:46:37  ####")

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
	config.SetDefaultIfNotSet()
	xyConfig = &config

	xyTimedPositionChange = common.NewTimedSum(xyConfig.TurnoverLookback)
	for xSymbol, ySymbol := range xyConfig.XYPairs {
		if _, ok := xyConfig.SymbolAssetMap[xSymbol]; !ok {
			logger.Fatalf("missing asset for symbol %s", xSymbol)
		}
		if _, ok := xyConfig.ExpireDates[ySymbol]; !ok {
			logger.Fatalf("missing expire date for symbol %s", ySymbol)
		}
		xSymbols = append(xSymbols, xSymbol)
		ySymbols = append(ySymbols, ySymbol)
		xySymbols = append(xySymbols, xSymbol)
		xySymbols = append(xySymbols, ySymbol)
		yxSymbolsMap[ySymbol] = xSymbol
		xySymbolsMap[xSymbol] = ySymbol
		xyTargetPositionUpdateSilentTimes[xSymbol] = time.Now()
		xyOrderSilentTimes[xSymbol] = time.Now().Add(xyConfig.RestartSilent)
		xyLogSilentTimes[xSymbol] = time.Now()
		xyPositionsUpdateTimes[xSymbol] = time.Unix(0, 0)
		xyPositionsUpdateTimes[ySymbol] = time.Unix(0, 0)
	}
	xyConfig.Exchange.Symbols = xySymbols
	configStr, err := yaml.Marshal(xyConfig)
	if err != nil {
		logger.Fatal(err)
	}

	switch xyConfig.Exchange.Name {
	case "binanceCoinFutureWithDepth5":
		xyExchange = &bncf.ExchangeWidthDepth5{}
		break
	case "binanceCoinFutureWithDepth20":
		xyExchange = &bncf.ExchangeWidthDepth20{}
		break
	default:
		logger.Fatalf("unsupported exchange %s", xyConfig.Exchange.Name)
	}
	fmt.Printf("CONFIG:\n\n%s\n\n", configStr)
}
