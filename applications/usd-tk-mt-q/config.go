package main

import (
	"github.com/geometrybase/hft-micro/common"
	"time"
)

type Config struct {
	Name *string `yaml:"name"`

	CpuProfile string `yaml:"cpuProfile"`
	DryRun     bool   `yaml:"dryRun"`
	ReduceOnly bool   `yaml:"reduceOnly"`

	InternalInflux common.InfluxSettings `yaml:"internalInflux"`
	ExternalInflux common.InfluxSettings `yaml:"externalInflux"`

	XExchange common.ExchangeSettings `yaml:"xExchange"`
	YExchange common.ExchangeSettings `yaml:"yExchange"`

	QuantileSampleInterval time.Duration `yaml:"quantileSampleInterval"`
	QuantileLookback       time.Duration `yaml:"quantileLookback"`
	QuantileSubInterval    time.Duration `yaml:"quantileSubInterval"`
	QuantilePath           string        `yaml:"quantilePath"`
	QuantileSaveInterval   time.Duration `yaml:"quantileSaveInterval"`

	SpreadWalkDelay       time.Duration `yaml:"spreadWalkDelay"`
	LogInterval           time.Duration `yaml:"logInterval"`
	TurnoverLookback      time.Duration `yaml:"turnoverLookback"`
	BalancePositionMaxAge time.Duration `yaml:"balancePositionMaxAge"`

	EnterOffsetDelta        float64       `yaml:"enterOffsetDelta"`
	ExitOffsetDelta         float64       `yaml:"exitOffsetDelta"`
	LongEnterDelta          float64       `yaml:"longEnterDelta"`
	ShortEnterDelta         float64       `yaml:"shortEnterDelta"`
	LongExitDelta           float64       `yaml:"longExitDelta"`
	ShortExitDelta          float64       `yaml:"shortExitDelta"`
	MinimalEnterFundingRate float64       `yaml:"minimalEnterFundingRate"`
	MinimalKeepFundingRate  float64       `yaml:"minimalKeepFundingRate"`
	FrOffsetFactor          float64       `yaml:"frOffsetFactor"`
	FundingRateSilentTime   time.Duration `yaml:"fundingRateSilentTime"`
	FundingInterval         time.Duration `yaml:"fundingInterval"`

	TickerMaxTimeDelta   time.Duration `yaml:"tickerTimeDeltaMax"`
	TickerMinTimeDelta   time.Duration `yaml:"tickerTimeDeltaMin"`
	TickerYDecay         float64       `yaml:"tickerYDecay"`
	TickerXDecay         float64       `yaml:"tickerXDecay"`
	TickerYBias          time.Duration `yaml:"tickerYBias"`
	TickerXBias          time.Duration `yaml:"tickerXBias"`
	TickerMaxAgeDiffBias time.Duration `yaml:"tickerMaxAgeDiffBias"`
	TickerReportCount    int           `yaml:"tickerReportCount"`
	SpreadTimeToEnter    time.Duration `yaml:"spreadTimeToEnter"`
	YTickerTimeToCancel  time.Duration `yaml:"yTickerTimeToCancel"`
	SpreadLookback       time.Duration `yaml:"spreadLookback"`
	SpreadMinTickerCount int           `yaml:"spreadMinTickerCount"`
	BatchSize            int           `yaml:"batchSize"`

	StartValue        float64            `yaml:"startValue"`
	EnterFreePct      float64            `yaml:"enterFreePct"`
	EnterMinimalStep  float64            `yaml:"enterMinimalStep"`
	EnterTargetFactor float64            `yaml:"enterTargetFactor"`
	StartValues       map[string]float64 `yaml:"startValues"`

	OrderTimeout     time.Duration `yaml:"orderTimeout"`
	XOrderSilent     time.Duration `yaml:"xOrderSilent"`
	XErrorSilent     time.Duration `yaml:"xErrorSilent"`
	YOrderSilent     time.Duration `yaml:"yOrderSilent"`
	XCancelSilent    time.Duration `yaml:"xCancelSilent"`
	RealisedLogDelay time.Duration `yaml:"realisedLogDelay"`
	HedgeXTimeout    time.Duration `yaml:"hedgeXTimeout"`

	XOrderCheckInterval time.Duration `yaml:"xOrderCheckInterval"`
	EnterSilent         time.Duration `yaml:"enterSilent"`
	RestartSilent       time.Duration `yaml:"restartSilent"`
	RestartInterval     time.Duration `yaml:"restartInterval"`

	XYPairs        map[string]string  `yaml:"xyPairs"`
	TargetWeights  map[string]float64 `yaml:"targetWeights,omitempty"`
	MaxOrderValues map[string]float64 `yaml:"maxOrderValues,omitempty"`
	OrderOffsets   map[string]string  `yaml:"orderOffsets,omitempty"`
}

func (config *Config) SetDefaultIfNotSet() {
	if config.LogInterval == 0 {
		config.LogInterval = time.Minute
	}
	if config.HedgeXTimeout == 0 {
		config.HedgeXTimeout = time.Minute * 3
	}
	if config.BalancePositionMaxAge == 0 {
		config.BalancePositionMaxAge = time.Minute * 3
	}
	if config.RealisedLogDelay == 0 {
		config.RealisedLogDelay = time.Millisecond * 10
	}
	if config.OrderTimeout == 0 {
		config.OrderTimeout = time.Second * 5
	}
	if config.EnterSilent == 0 {
		config.EnterSilent = time.Minute * 30
	}
	if config.RestartSilent == 0 {
		config.RestartSilent = time.Minute * 3
	}
	if config.BatchSize <= 0 {
		config.BatchSize = 20
	}
	if config.TickerMaxAgeDiffBias == 0 {
		config.TickerMaxAgeDiffBias = time.Millisecond * 100
	}
	if config.TickerReportCount == 0 {
		config.RestartSilent = 1000
	}
	if config.YTickerTimeToCancel == 0 {
		config.YTickerTimeToCancel = time.Second * 3
	}
	if config.SpreadLookback == 0 {
		config.SpreadLookback = time.Second
	}
	if config.RestartInterval == 0 {
		config.RestartInterval = time.Hour * 9999
	}
	if config.TurnoverLookback == 0 {
		config.TurnoverLookback = time.Hour * 24
	}
	if config.XOrderSilent == 0 {
		config.XOrderSilent = time.Second
	}
	if config.XCancelSilent == 0 {
		config.XCancelSilent = config.XOrderSilent
	}
	if config.YOrderSilent == 0 {
		config.YOrderSilent = time.Second * 5
	}
	if config.XOrderCheckInterval == 0 {
		config.XOrderCheckInterval = time.Millisecond * 100
	}
	if config.FundingRateSilentTime == 0 {
		config.FundingRateSilentTime = time.Minute
	}
	if config.FundingInterval == 0 {
		config.FundingInterval = time.Hour * 4
	}
	if config.XErrorSilent == 0 {
		config.XErrorSilent = config.EnterSilent
	}
	config.XExchange.DryRun = config.DryRun
	config.YExchange.DryRun = config.DryRun
}
