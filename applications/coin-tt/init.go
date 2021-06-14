package main

import (
	"context"
	"flag"
	"fmt"
	bncf "github.com/geometrybase/hft-micro/binance-coinfuture"
	"github.com/geometrybase/hft-micro/common"
	kccf "github.com/geometrybase/hft-micro/kucoin-coinfuture"
	"github.com/geometrybase/hft-micro/logger"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"time"
)

var xSymbols = make([]string, 0)
var ySymbols = make([]string, 0)
var yxSymbolsMap = make(map[string]string, 0)
var xySymbolsMap = make(map[string]string, 0)

var xyGlobalCtx context.Context
var xyGlobalCancel context.CancelFunc
var xyInfluxWriter *common.InfluxWriter
var xyExternalInfluxWriter *common.InfluxWriter

var xStepSizes = make(map[string]float64)
var xMinNotionals = make(map[string]float64)
var xMultipliers = make(map[string]float64)
var yStepSizes = make(map[string]float64)
var yMinNotionals = make(map[string]float64)
var yMultipliers = make(map[string]float64)
var xyUsdStepSizes = make(map[string]float64)

var xBalances = make(map[string]common.Balance)
var xAccountCh = make(chan common.Balance, 200)
var xPositionCh = make(chan common.Position, 200)
var xOrderCh = make(chan common.Order, 200)
var xPositions = make(map[string]common.Position)
var xPositionsUpdateTimes = make(map[string]time.Time)
var xOrderRequestChMap = make(map[string]chan common.OrderRequest)
var xNewOrderErrorCh = make(chan common.OrderError, 200)
var xOrderSilentTimes = make(map[string]time.Time)

var yPositionCh = make(chan common.Position, 200)
var yOrderCh = make(chan common.Order, 200)
var yPositions = make(map[string]common.Position)
var yPositionsUpdateTimes = make(map[string]time.Time)
var yBalances = make(map[string]common.Balance)
var yAccountCh = make(chan common.Balance, 200)
var yNewOrderErrorCh = make(chan common.OrderError, 200)
var yOrderRequestChMap = make(map[string]chan common.OrderRequest)
var yOrderSilentTimes = make(map[string]time.Time)

var xFundingRates = make(map[string]common.FundingRate)
var xFundingRateCh = make(chan common.FundingRate, 200)
var yFundingRates = make(map[string]common.FundingRate)
var yFundingRateCh = make(chan common.FundingRate, 200)
var xyFundingRates = make(map[string]float64)
var xyRankSymbolMap map[int]string

var xySpreads = make(map[string]*XYSpread)

var xLastFilledBuyPrices = make(map[string]float64)
var xLastFilledSellPrices = make(map[string]float64)
var yLastFilledBuyPrices = make(map[string]float64)
var yLastFilledSellPrices = make(map[string]float64)
var xyRealisedSpread = make(map[string]float64)

var xyUnHedgeValue float64
var xyLogSilentTimes = make(map[string]time.Time)
var xyLoopTimer *time.Timer
var xyDualEnds []int
var xSystemStatus = common.SystemStatusNotReady
var ySystemStatus = common.SystemStatusNotReady
var xSystemStatusCh = make(chan common.SystemStatus, 100)
var ySystemStatusCh = make(chan common.SystemStatus, 100)

var xTargetContractValues = make(map[string]float64)
var yTargetContractValues = make(map[string]float64)
var xyTargetPositionUpdateSilentTimes = make(map[string]time.Time)

var xyConfig *Config

var xExchange common.Exchange
var yExchange common.Exchange

var xTimedPositionChange *common.TimedSum
var yTimedPositionChange *common.TimedSum

func init() {

	logger.Debug("####  BUILD @ 20210614 01:43:56  ####")

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

	configStr, err := yaml.Marshal(xyConfig)
	if err != nil {
		logger.Fatal(err)
	}
	fmt.Printf("CONFIG:\n\n%s\n\n", configStr)

	switch xyConfig.XExchange.Name {
	case "binanceCoinFutureWithDepth5":
		xExchange = &bncf.ExchangeWidthDepth5{}
		break
	case "binanceCoinFutureWithDepth20":
		xExchange = &bncf.ExchangeWidthDepth20{}
		break
	case "kucoinCoinFutureWithDepth5":
		xExchange = &kccf.ExchangeWithDepth5{}
	default:
		logger.Fatalf("unsupported exchange %s", xyConfig.XExchange.Name)
	}

	switch xyConfig.YExchange.Name {
	case "binanceCoinFutureWithDepth5":
		yExchange = &bncf.ExchangeWidthDepth5{}
		break
	case "binanceCoinFutureWithDepth20":
		yExchange = &bncf.ExchangeWidthDepth20{}
		break
	case "kucoinCoinFutureWithDepth5":
		yExchange = &kccf.ExchangeWithDepth5{}
		break
	default:
		logger.Fatalf("unsupported exchange %s", xyConfig.YExchange.Name)
	}
	xTimedPositionChange = common.NewTimedSum(xyConfig.TurnoverLookback)
	yTimedPositionChange = common.NewTimedSum(xyConfig.TurnoverLookback)
	for xSymbol, ySymbol := range xyConfig.XYPairs {
		if _, ok := xyConfig.XSymbolAssetMap[xSymbol]; !ok {
			logger.Fatalf("missing asset for x symbol %s", xSymbol)
		}
		if _, ok := xyConfig.YSymbolAssetMap[ySymbol]; !ok {
			logger.Fatalf("missing asset for y symbol %s", ySymbol)
		}
		xSymbols = append(xSymbols, xSymbol)
		ySymbols = append(ySymbols, ySymbol)
		yxSymbolsMap[ySymbol] = xSymbol
		xySymbolsMap[xSymbol] = ySymbol

		xyTargetPositionUpdateSilentTimes[xSymbol] = time.Now()

		xOrderSilentTimes[xSymbol] = time.Now().Add(xyConfig.RestartSilent)
		xyLogSilentTimes[xSymbol] = time.Now()
		xPositionsUpdateTimes[xSymbol] = time.Unix(0, 0)

		yOrderSilentTimes[ySymbol] = time.Now().Add(xyConfig.RestartSilent)
		yPositionsUpdateTimes[ySymbol] = time.Unix(0, 0)
	}
	xyConfig.XExchange.Symbols = xSymbols
	xyConfig.YExchange.Symbols = ySymbols
	xyDualEnds = make([]int, 0)
	for i := 0; i < len(xSymbols)/2; i++ {
		xyDualEnds = append(xyDualEnds, i)
		xyDualEnds = append(xyDualEnds, len(xSymbols)-1-i)
	}
	if len(xSymbols)%2 == 1 {
		xyDualEnds = append(xyDualEnds, len(xSymbols)/2)
	}
	logger.Debugf("dual end ranks %d", xyDualEnds)
}
