package main

import (
	"context"
	"flag"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/geometrybase/hft-micro/okspot"
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
var mtInternalInfluxWriter *common.InfluxWriter
var mtExternalInfluxWriter *common.InfluxWriter

var mAPI *okspot.API
var tAPI *bnswap.API

var mUserWebsocket *okspot.UserWebsocket
var tUserWebsocket *bnswap.UserWebsocket

var mHttpPositionUpdateSilentTimes = make(map[string]time.Time)
var tHttpPositionUpdateSilentTimes = make(map[string]time.Time)

var mTickSizes = make(map[string]float64)
var mStepSizes = make(map[string]float64)
var mMinSizes = make(map[string]float64)
var tStepSizes = make(map[string]float64)
var tMinNotional = make(map[string]float64)
var mtStepSizes = make(map[string]float64)

var mAccount *okspot.Balance
var mBalancesCh = make(chan []okspot.Balance, 10)
var mBalances = make(map[string]okspot.Balance)
var mBalancesUpdateTimes = make(map[string]time.Time)
var mOrderRequestChs = make(map[string]chan MakerOrderRequest)
var mNewOrderErrorCh chan MakerOrderNewError
var mOrderSilentTimes = make(map[string]time.Time)
var mSilentTimes = make(map[string]time.Time)
var mOpenOrders = make(map[string]MakerOpenOrder)
var mOrderCancelCounts = make(map[string]int)
var mCancelSilentTimes = make(map[string]time.Time)
var mOpenOrderCh = make(chan MakerOpenOrder, 10000)

var tPositionsCh = make(chan []bnswap.Position, 10)
var tPositions = make(map[string]bnswap.Position)
var tPositionsUpdateTimes = make(map[string]time.Time)
var tAccount *bnswap.Asset
var tAccountCh = make(chan bnswap.Account, 10)
var tNewOrderErrorCh = make(chan TakerOrderNewError, 10)
var tOrderRequestChs = make(map[string]chan bnswap.NewOrderParams)
var tOrderSilentTimes = make(map[string]time.Time)

var tPremiumIndexes map[string]bnswap.PremiumIndex
var tPremiumIndexesCh = make(chan map[string]bnswap.PremiumIndex, 10)
var mtFundingRates = make(map[string]float64)
var mtRankSymbolMap map[int]string

var mtSpreads = make(map[string]*common.ShortSpread)

var mLastFilledBuyPrices = make(map[string]float64)
var mLastFilledSellPrices = make(map[string]float64)
var mtRealisedSpread = make(map[string]float64)

var mtUnHedgeValue float64
var mtUnHedgeLogSilentTimes = time.Unix(0, 0)
var mtLogSilentTimes = make(map[string]time.Time)
var mtLoopTimer *time.Timer
var mtDualEnds []int
var mSystemReady = false
var tSystemReady = false
var mSystemStatusCh = make(chan bool, 10)
var tSystemStatusCh = make(chan bool, 10)
var mtGlobalSilent = time.Now()
var mOrderOffsets = make(map[string]Offset)

var mtConfig *Config

func init() {

	logger.Debug("####  BUILD @ 20210507 06:31:45  ####")

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

	for makerSymbol, takerSymbol := range mtConfig.MakerTakerSymbolsMap {
		mSymbols = append(mSymbols, makerSymbol)
		tSymbols = append(tSymbols, takerSymbol)
		tmSymbolsMap[takerSymbol] = makerSymbol
		mtSymbolsMap[makerSymbol] = takerSymbol
		if offset, ok := mtConfig.MakerOrderOffsets[makerSymbol]; ok {
			mOrderOffsets[makerSymbol], err = NewOffset(offset)
			if err != nil {
				logger.Fatalf("NewOffset error %s %v", makerSymbol, err)
			}
		} else {
			logger.Fatalf("Miss offset for %s", makerSymbol)
		}

		mOrderSilentTimes[makerSymbol] = time.Now()
		mtLogSilentTimes[makerSymbol] = time.Now()
		mSilentTimes[makerSymbol] = time.Now().Add(*mtConfig.RestartSilent)
		mBalancesUpdateTimes[makerSymbol] = time.Unix(0, 0)
		mOrderCancelCounts[makerSymbol] = 0
		mCancelSilentTimes[makerSymbol] = time.Now()

		tOrderSilentTimes[takerSymbol] = time.Now()
		tPositionsUpdateTimes[takerSymbol] = time.Unix(0, 0)

		mHttpPositionUpdateSilentTimes[makerSymbol] = time.Now()
		tHttpPositionUpdateSilentTimes[makerSymbol] = time.Now()
	}
	mtDualEnds = make([]int, 0)
	for i := 0; i < len(mSymbols)/2; i++ {
		mtDualEnds = append(mtDualEnds, i)
		mtDualEnds = append(mtDualEnds, len(mSymbols)-1-i)
	}
	if len(mSymbols)%2 == 1 {
		mtDualEnds = append(mtDualEnds, len(mSymbols)/2)
	}
	logger.Debugf("DUAL ENDS RANK %d", mtDualEnds)

	mtGlobalSilent = time.Now().Add(*mtConfig.RestartSilent)

	//hostname, err := os.Hostname()
	//if err != nil {
	//	logger.Fatal(err)
	//}

	//err = raven.SetDSN("https://5c318e0f10a349308d2ff86f51de31d8:fa0a8f90a8244c6ea762130cdd6d1bb9@sentry.jilinchen.com/12")
	//if err != nil {
	//	logger.Fatal(err)
	//}
	//raven.SetTagsContext(map[string]string{
	//	"influxAddress":     *mtConfig.InternalInflux.Address,
	//	"influxDatabase":    *mtConfig.InternalInflux.Address,
	//	"influxMeasurement": *mtConfig.InternalInflux.Address,
	//	"BnApiKey":          *mtConfig.InternalInflux.Address,
	//	"symbols":           fmt.Sprintf("%s", mSymbols),
	//	"hostname":          hostname,
	//	"name":              *mtConfig.Name,
	//})
}
