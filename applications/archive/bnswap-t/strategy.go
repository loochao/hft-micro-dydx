package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"time"
)

type Strategy struct {
	OrderSilentTimes     [SYMBOLS_LEN]time.Time
	PositionsUpdateTimes [SYMBOLS_LEN]time.Time
	LastOrderTimes       [SYMBOLS_LEN]time.Time
	LogSilentTimes       [SYMBOLS_LEN]time.Time
	TickSizes            [SYMBOLS_LEN]float64
	StepSizes            [SYMBOLS_LEN]float64
	MinNotional          [SYMBOLS_LEN]float64
	Positions            [SYMBOLS_LEN]bnswap.Position
	RealisedProfits      [SYMBOLS_LEN]float64
	Signals              [SYMBOLS_LEN]Signal
	MarkPrices           [SYMBOLS_LEN]bnswap.MarkPrice

	OrderNewErrorCh      chan SwapOrderNewError
	OrderFinishCh        chan bnswap.Order

	OrderSubmittingChs [SYMBOLS_LEN]chan bnswap.NewOrderParams

	Config               *Config
	InternalInfluxWriter *common.InfluxWriter
	ExternalInfluxWriter *common.InfluxWriter
	API                  *bnswap.API
	UserWebsocket        *bnswap.UserWebsocket
	USDTAsset            bnswap.Asset
	BNBAsset             bnswap.Asset
	GlobalCtx            context.Context
	GlobalCancel         context.CancelFunc
}

func NewStrategy() (*Strategy, error) {
	st := Strategy{}
	configPath := flag.String("config", "", "config path")
	flag.Parse()
	if *configPath == "" {
		return nil, errors.New("config is empty")
	}
	configFile, err := ioutil.ReadFile(*configPath)
	if err != nil {
		return nil, err
	}
	st.Config = &Config{}
	err = yaml.Unmarshal(configFile, st.Config)
	if err != nil {
		return nil, err
	}
	logger.Debugf("\n\nYAML CONFIG:\n\n%s", st.Config.ToString())
	valid, reason := st.Config.IsValid()
	if !valid {
		return nil, fmt.Errorf("CONFIG IS NOT VALID:\n%s\n", reason)
	}
	st.API, err = bnswap.NewAPI(&common.Credentials{
		Key:    *st.Config.ApiKey,
		Secret: *st.Config.ApiSecret,
	}, *st.Config.ProxyAddress)
	if err != nil {
		return nil, err
	}
	st.GlobalCtx, st.GlobalCancel = context.WithCancel(context.Background())

	st.InternalInfluxWriter, err = common.NewInfluxWriter(
		*st.Config.InternalInflux.Address,
		*st.Config.InternalInflux.Username,
		*st.Config.InternalInflux.Password,
		*st.Config.InternalInflux.Database,
		*st.Config.InternalInflux.BatchSize,
	)
	if err != nil {
		return nil, err
	}

	st.ExternalInfluxWriter, err = common.NewInfluxWriter(
		*st.Config.ExternalInflux.Address,
		*st.Config.ExternalInflux.Username,
		*st.Config.ExternalInflux.Password,
		*st.Config.ExternalInflux.Database,
		*st.Config.ExternalInflux.BatchSize,
	)
	if err != nil {
		return nil, err
	}

	tickSizes, stepSizes, _, minNotional, _, _, err := bnswap.GetOrderLimits(st.GlobalCtx, st.API, SYMBOLS[:])
	if err != nil {
		return nil, err
	}

	st.OrderFinishCh = make(chan bnswap.Order, SYMBOLS_LEN)
	st.OrderNewErrorCh = make(chan SwapOrderNewError, SYMBOLS_LEN)

	st.TickSizes = [SYMBOLS_LEN]float64{}
	st.StepSizes = [SYMBOLS_LEN]float64{}
	st.MinNotional = [SYMBOLS_LEN]float64{}
	st.OrderSilentTimes = [SYMBOLS_LEN]time.Time{}
	st.OrderSubmittingChs = [SYMBOLS_LEN]chan bnswap.NewOrderParams{}
	st.PositionsUpdateTimes = [SYMBOLS_LEN]time.Time{}
	st.Positions = [SYMBOLS_LEN]bnswap.Position{}
	st.LastOrderTimes = [SYMBOLS_LEN]time.Time{}
	st.LogSilentTimes = [SYMBOLS_LEN]time.Time{}
	st.PositionsUpdateTimes = [SYMBOLS_LEN]time.Time{}
	st.PositionsUpdateTimes = [SYMBOLS_LEN]time.Time{}
	st.RealisedProfits = [SYMBOLS_LEN]float64{}
	st.Signals = [SYMBOLS_LEN]Signal{}
	st.MarkPrices = [SYMBOLS_LEN]bnswap.MarkPrice{}
	for i, symbol := range SYMBOLS {
		st.TickSizes[i] = tickSizes[symbol]
		st.StepSizes[i] = stepSizes[symbol]
		st.MinNotional[i] = minNotional[symbol]
		st.OrderSilentTimes[i] = time.Now()
		st.OrderSubmittingChs[i] = make(chan bnswap.NewOrderParams, 2)
		st.PositionsUpdateTimes[i] = time.Unix(0, 0)
		st.Positions[i] = bnswap.Position{}
		st.RealisedProfits[i] = 0
		st.Signals[i] = Signal{}
		st.MarkPrices[i] = bnswap.MarkPrice{}
	}
	if *st.Config.ChangeLeverage {
		for _, symbol := range SYMBOLS {
			res, err := st.API.UpdateLeverage(st.GlobalCtx, bnswap.UpdateLeverageParams{
				Symbol:   symbol,
				Leverage: int64(*st.Config.Leverage),
			})
			if err != nil {
				logger.Debugf("UPDATE LEVERAGE FOR %s ERROR %v", symbol, err)
			} else {
				logger.Debugf("UPDATE LEVERAGE FOR %s RESPONSE %v", symbol, res)
			}
			res, err = st.API.UpdateMarginType(st.GlobalCtx, bnswap.UpdateMarginTypeParams{
				Symbol:     symbol,
				MarginType: *st.Config.MarginType,
			})
			if err != nil {
				logger.Debugf("UPDATE MARGIN TYPE FOR %s ERROR %v", symbol, err)
			} else {
				logger.Debugf("UPDATE MARGIN TYPE FOR %s RESPONSE %v", symbol, res)
			}
		}
	}
	return &st, nil
}

func (st Strategy) Stop() {
	err := st.InternalInfluxWriter.Stop()
	if err != nil {
		logger.Debugf("Stop InternalInfluxWriter error %v", err)
	}
	err = st.ExternalInfluxWriter.Stop()
	if err != nil {
		logger.Debugf("Stop InternalInfluxWriter error %v", err)
	}
}
