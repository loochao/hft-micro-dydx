package main

import (
	"context"
	"flag"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/ftxperp"
	"github.com/geometrybase/hft-micro/logger"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"time"
)

var trGlobalContext context.Context
var trGlobalCancel context.CancelFunc
var trInternalInfluxWriter *common.InfluxWriter
var trExternalInfluxWriter *common.InfluxWriter
var trConfig *Config

var trExchange common.Exchange

var trOrderCh = make(chan common.Order, 100)
var trPositionCh = make(chan common.Position, 100)
var trOrders = make(map[string]common.Order)
var trPositions = make(map[string]common.Position)
var trAccountCh = make(chan common.Account, 100)
var trStatusCh = make(chan common.SystemStatus, 100)
var trAccount common.Account
var trGlobalSilent = time.Now()
var trSystemStatus common.SystemStatus
var trSignalCh = make(chan Signal, 1000)
var trSignals = make(map[string]Signal)

func init() {

	logger.Debug("####  BUILD @ 20210427 09:53:18  ####")
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
	configStr, err := yaml.Marshal(config)
	if err != nil {
		logger.Fatal(err)
	}
	logger.Debugf("\n\nYAML CONFIG:\n\n%s", configStr)
	trConfig = &config

	if trConfig.ExchangeSettings.Proxy == "" {
		trConfig.ExchangeSettings.Proxy = os.Getenv("FTX_TEST_PROXY")
	}
	if trConfig.ExchangeSettings.ApiKey == "" {
		trConfig.ExchangeSettings.ApiKey = os.Getenv("FTX_TEST_KEY")
	}
	if trConfig.ExchangeSettings.ApiSecret == "" {
		trConfig.ExchangeSettings.ApiSecret = os.Getenv("FTX_TEST_SECRET")
	}
	if trConfig.UpdateInterval == 0 {
		trConfig.UpdateInterval = time.Minute
	}
	if trConfig.TradeLookback == 0 {
		trConfig.TradeLookback = time.Minute
	}
	if trConfig.DepthLevel == 0 {
		trConfig.DepthLevel = 10
	}
	trGlobalContext, trGlobalCancel = context.WithCancel(context.Background())
	switch trConfig.ExchangeSettings.Name {
	case "ftxperp":
		trExchange = new(ftxperp.Ftxperp)
		err = trExchange.Setup(trGlobalContext, trConfig.ExchangeSettings)
		if err != nil {
			logger.Fatal(err)
		}
		break
	case "bnswap":
		trExchange = new(bnswap.Bnswap)
		err = trExchange.Setup(trGlobalContext, trConfig.ExchangeSettings)
		if err != nil {
			logger.Fatal(err)
		}
		break
	default:
		logger.Fatalf("unsupported exchange %s", trConfig.ExchangeSettings.Name)
	}
}
