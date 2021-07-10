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

	SpreadWalkDelay       time.Duration `yaml:"spreadWalkDelay"`
	LogInterval           time.Duration `yaml:"logInterval"`
	TurnoverLookback      time.Duration `yaml:"turnoverLookback"`
	BalancePositionMaxAge time.Duration `yaml:"balancePositionMaxAge"`

	EnterOffsetDelta float64 `yaml:"enterOffsetDelta"`
	ExitOffsetDelta  float64 `yaml:"exitOffsetDelta"`
	LongEnterDelta   float64 `yaml:"longEnterDelta"`
	ShortEnterDelta  float64 `yaml:"shortEnterDelta"`
	LongExitDelta    float64 `yaml:"longExitDelta"`
	ShortExitDelta   float64 `yaml:"shortExitDelta"`

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
	SpreadLookback       time.Duration `yaml:"spreadLookback"`
	SpreadMinTickerCount int           `yaml:"spreadMinTickerCount"`
	BatchSize            int           `yaml:"batchSize"`

	StartValue        float64            `yaml:"startValue"`
	EnterFreePct      float64            `yaml:"enterFreePct"`
	EnterMinimalStep  float64            `yaml:"enterMinimalStep"`
	EnterTargetFactor float64            `yaml:"enterTargetFactor"`
	StartValues       map[string]float64 `yaml:"startValues"`

	OrderTimeout       time.Duration `yaml:"orderTimeout"`
	XOrderSilent       time.Duration `yaml:"xOrderSilent"`
	YOrderSilent       time.Duration `yaml:"yOrderSilent"`
	XErrorSilent       time.Duration `yaml:"xErrorSilent"`
	HedgeDelay         time.Duration `yaml:"hedgeDelay"`
	HedgeCheckDuration time.Duration `yaml:"hedgeCheckDuration"`
	HedgeCheckInterval time.Duration `yaml:"hedgeCheckInterval"`
	EnterSilent        time.Duration `yaml:"enterSilent"`
	RestartSilent      time.Duration `yaml:"restartSilent"`
	RestartInterval    time.Duration `yaml:"restartInterval"`

	XYPairs        map[string]string  `yaml:"xyPairs"`
	TargetWeights  map[string]float64 `yaml:"targetWeights,omitempty"`
	MaxOrderValues map[string]float64 `yaml:"maxOrderValues,omitempty"`
}

func (config *Config) SetDefaultIfNotSet() {
	if config.LogInterval == 0 {
		config.LogInterval = time.Minute
	}
	if config.BalancePositionMaxAge == 0 {
		config.BalancePositionMaxAge = time.Minute * 3
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
	if config.YOrderSilent == 0 {
		config.YOrderSilent = time.Second * 5
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
