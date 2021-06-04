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

var mGlobalCtx context.Context
var mGlobalCancel context.CancelFunc
var mInfluxWriter *common.InfluxWriter
var mExternalInfluxWriter *common.InfluxWriter

var mTickSizes = make(map[string]float64)
var mStepSizes = make(map[string]float64)
var mMinNotional = make(map[string]float64)

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

var mLogSilentTimes = make(map[string]time.Time)
var mLoopTimer *time.Timer
var mSystemStatus = common.SystemStatusNotReady
var mSystemStatusCh = make(chan common.SystemStatus, 100)
var mOrderOffsets = make(map[string]Offset)

var mWalkedDepths = make(map[string]*common.WalkedMakerTakerDepth)

var mConfig *Config
var mExchange common.Exchange
var mTimedPositionChange *common.TimedSum


func init() {

	logger.Debug("####  BUILD @ 20210604 15:55:11  ####")

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
	mConfig = &config

	configStr, err := yaml.Marshal(mConfig)
	if err != nil {
		logger.Fatal(err)
	}
	fmt.Printf("CONFIG:\n\n%s\n\n", configStr)

	switch mConfig.MakerExchange.Name {
	case "ftxperp":
		mExchange = &ftxperp.Ftxperp{}
	case "bnswap":
		mExchange = &bnswap.Bnswap{}
	default:
		logger.Fatal("unsupported exchange %s", mConfig.MakerExchange.Name)
	}

	for makerSymbol, offset := range mConfig.MakerOrderOffsets {
		mOrderOffsets[makerSymbol], err = NewOffset(offset)
		if err != nil {
			logger.Fatalf("NewOffset for %s error %v", makerSymbol, err)
		}
		mSymbols = append(mSymbols, makerSymbol)

		mOrderSilentTimes[makerSymbol] = time.Now()
		mLogSilentTimes[makerSymbol] = time.Now()
		mEnterSilentTimes[makerSymbol] = time.Now()
		mPositionsUpdateTimes[makerSymbol] = time.Unix(0, 0)
		mCancelSilentTimes[makerSymbol] = time.Now()
	}
	mConfig.MakerExchange.Symbols = mSymbols
	mTimedPositionChange = common.NewTimedSum(mConfig.TurnoverLookback)
}
