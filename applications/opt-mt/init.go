package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/ftxperp"
	"github.com/geometrybase/hft-micro/logger"
	"gopkg.in/yaml.v2"
	"io/ioutil"
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

var mTickSizes = make(map[string]float64)
var mStepSizes = make(map[string]float64)
var mMinNotional = make(map[string]float64)
var tStepSizes = make(map[string]float64)
var tMinNotional = make(map[string]float64)
var mtStepSizes = make(map[string]float64)

var mAccount common.Account
var mAccountCh = make(chan common.Account, 200)
var mPositionCh = make(chan common.Position, 200)
var mOrderCh = make(chan common.Order, 200)
var mPositions = make(map[string]common.Position)
var mPositionsUpdateTimes = make(map[string]time.Time)
var mOrderRequestChMap = make(map[string]chan common.OrderRequest)
var mNewOrderErrorCh = make(chan common.OrderError, 200)
var mOrderSilentTimes = make(map[string]time.Time)
var mEnterSilentTimes = make(map[string]time.Time)
var mOpenOrders = make(map[string]common.NewOrderParam)
var mCancelSilentTimes = make(map[string]time.Time)

var tPositionCh = make(chan common.Position, 200)
var tOrderCh = make(chan common.Order, 200)
var tPositions = make(map[string]common.Position)
var tPositionsUpdateTimes = make(map[string]time.Time)
var tAccount common.Account
var tAccountCh = make(chan common.Account, 200)
var tNewOrderErrorCh = make(chan common.OrderError, 200)
var tOrderRequestChMap = make(map[string]chan common.OrderRequest)
var tOrderSilentTimes = make(map[string]time.Time)

var mFundingRates = make(map[string]common.FundingRate)
var mFundingRateCh = make(chan common.FundingRate, 200)
var tFundingRates = make(map[string]common.FundingRate)
var tFundingRateCh = make(chan common.FundingRate, 200)
var mtFundingRates = make(map[string]float64)
var mtRankSymbolMap map[int]string

var mtSpreads = make(map[string]*common.MakerTakerSpread)

var mLastFilledBuyPrices = make(map[string]float64)
var mLastFilledSellPrices = make(map[string]float64)
var mtRealisedSpread = make(map[string]float64)

var mtUnHedgeValue float64
var mtLogSilentTimes = make(map[string]time.Time)
var mtLoopTimer *time.Timer
var mtDualEnds []int
var mSystemStatus = common.SystemStatusNoteReady
var tSystemStatus = common.SystemStatusNoteReady
var mSystemStatusCh = make(chan common.SystemStatus, 100)
var tSystemStatusCh = make(chan common.SystemStatus, 100)
var mOrderOffsets = make(map[string]Offset)
var mtDeltas = make(map[string]Delta)

var tHedgeMarkPrices = make(map[string]float64)

var mtConfig *Config

var mExchange common.Exchange
var tExchange common.Exchange

func init() {

	logger.Debug("####  BUILD @ 20210517 01:00:17  ####")

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
	mtConfig = &config

	configStr, err := yaml.Marshal(mtConfig)
	if err != nil {
		logger.Fatal(err)
	}
	fmt.Printf("CONFIG:\n\n%s\n\n", configStr)

	switch mtConfig.MakerExchange.Name {
	case "ftxperp":
		mExchange = &ftxperp.Ftxperp{}
	case "bnswap":
		mExchange = &bnswap.Bnswap{}
	default:
		logger.Fatal("unsupported exchange %s", mtConfig.MakerExchange.Name)
	}

	switch mtConfig.TakerExchange.Name {
	case "ftxperp":
		tExchange = &ftxperp.Ftxperp{}
	case "bnswap":
		tExchange = &bnswap.Bnswap{}
	default:
		logger.Fatal("unsupported exchange %s", mtConfig.TakerExchange.Name)
	}
	for makerSymbol, takerSymbol := range mtConfig.MakerTakerPairs {
		if delta, ok := mtConfig.Deltas[makerSymbol]; ok {
			dt, err := NewDelta(delta)
			if err != nil {
				logger.Fatalf("NewOffset for %s error %v", makerSymbol, err)
			}
			//if dt.LongTop - dt.LongBot < mtConfig.MinimalDelta || dt.ShortTop - dt.ShortBot < mtConfig.MinimalDelta{
			//	logger.Debugf("%s delta too small %s", makerSymbol, delta)
			//	continue
			//}
			mtDeltas[makerSymbol] = dt
		}else{
			logger.Fatalf("MISS DELTA FOR %s", makerSymbol)
		}
		if offset, ok := mtConfig.MakerOrderOffsets[makerSymbol]; ok {
			mOrderOffsets[makerSymbol], err = NewOffset(offset)
			if err != nil {
				logger.Fatalf("NewOffset for %s error %v", makerSymbol, err)
			}
		} else {
			logger.Fatalf("MISS OFFSET FOR %s", makerSymbol)
		}
		mSymbols = append(mSymbols, makerSymbol)
		tSymbols = append(tSymbols, takerSymbol)
		tmSymbolsMap[takerSymbol] = makerSymbol
		mtSymbolsMap[makerSymbol] = takerSymbol

		mOrderSilentTimes[makerSymbol] = time.Now()
		mtLogSilentTimes[makerSymbol] = time.Now()
		mEnterSilentTimes[makerSymbol] = time.Now()
		mPositionsUpdateTimes[makerSymbol] = time.Unix(0, 0)
		mCancelSilentTimes[makerSymbol] = time.Now()

		tOrderSilentTimes[takerSymbol] = time.Now()
		tPositionsUpdateTimes[takerSymbol] = time.Unix(0, 0)
	}
	mtConfig.MakerExchange.Symbols = mSymbols
	mtConfig.TakerExchange.Symbols = tSymbols
	mtDualEnds = make([]int, 0)
	for i := 0; i < len(mSymbols)/2; i++ {
		mtDualEnds = append(mtDualEnds, i)
		mtDualEnds = append(mtDualEnds, len(mSymbols)-1-i)
	}
	if len(mSymbols)%2 == 1 {
		mtDualEnds = append(mtDualEnds, len(mSymbols)/2)
	}
	logger.Debugf("dual end ranks %d", mtDualEnds)
}
