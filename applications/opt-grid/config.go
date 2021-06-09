package main

import (
	"github.com/geometrybase/hft-micro/common"
	"time"
)

type Config struct {
	Name *string `yaml:"name"`

	//CpuProfile   string `yaml:"CpuProfile"`
	DryRun       bool   `yaml:"dryRun"`
	ProxyAddress string `yaml:"proxyAddress"`

	InternalInflux common.InfluxSettings `yaml:"internalInflux"`
	ExternalInflux common.InfluxSettings `yaml:"externalInflux"`

	MakerExchange common.ExchangeSettings `yaml:"makerExchange"`

	LoopInterval          time.Duration `yaml:"loopInterval"`
	LogInterval           time.Duration `yaml:"logInterval"`
	PullInterval          time.Duration `yaml:"pullInterval"`
	RequestInterval       time.Duration `yaml:"requestInterval"`
	BalancePositionMaxAge time.Duration `yaml:"balancePositionMaxAge"`

	DepthTimeDeltaMax time.Duration `yaml:"depthTimeDeltaMax"`
	DepthTimeDeltaMin time.Duration `yaml:"depthTimeDeltaMin"`
	DepthMakerDecay   float64       `yaml:"depthMakerDecay"`
	DepthMakerBias    time.Duration `yaml:"depthMakerBias"`
	BatchSize         int           `yaml:"depthBatchSize"`
	DepthMakerImpact  float64       `yaml:"depthMakerImpact"`
	DepthTakerImpact  float64       `yaml:"depthTakerImpact"`
	DepthTimeToLive   time.Duration `yaml:"depthTimeToLive"`

	TradeDir int `yaml:"tradeDir"`

	StartValue        float64            `yaml:"startValue"`
	EnterFreePct      float64            `yaml:"enterFreePct"`
	EnterMinimalStep  float64            `yaml:"enterMinimalStep"`
	EnterTargetFactor float64            `yaml:"enterTargetFactor"`
	StartValues       map[string]float64 `yaml:"startValues"`

	OrderTimeout            time.Duration `yaml:"orderTimeout"`
	OrderSilent             time.Duration `yaml:"orderSilent"`
	CancelSilent            time.Duration `yaml:"cancelSilent"`
	EnterSilent             time.Duration `yaml:"enterSilent"`
	RestartSilent           time.Duration `yaml:"restartSilent"`
	HttpSilent              time.Duration `yaml:"httpSilent"`
	TurnoverLookback        time.Duration `yaml:"turnoverLookback"`
	EnterTriggerFilterRatio float64       `yaml:"enterTriggerFilterRatio"`
	EnterTriggerDelay       time.Duration `yaml:"enterTriggerDelay"`
	EnterDuration           time.Duration `yaml:"enterDuration"`
	ReportCount             int           `yaml:"reportCount"`

	MakerOrderOffsets map[string]string `yaml:"makerOrderOffsets"`
}

func (config *Config) SetDefaultIfNotSet() {
	if config.LoopInterval == 0 {
		config.LoopInterval = time.Second
	}
	if config.LogInterval == 0 {
		config.LoopInterval = time.Minute
	}
	if config.PullInterval == 0 {
		config.PullInterval = time.Minute
	}
	if config.RequestInterval == 0 {
		config.RequestInterval = time.Second
	}
	if config.BalancePositionMaxAge == 0 {
		config.BalancePositionMaxAge = time.Minute * 3
	}

	if config.OrderTimeout == 0 {
		config.OrderTimeout = time.Second * 5
	}
	if config.OrderSilent == 0 {
		config.OrderSilent = time.Second * 5
	}
	if config.CancelSilent == 0 {
		config.CancelSilent = time.Second * 5
	}
	if config.EnterSilent == 0 {
		config.EnterSilent = time.Minute * 30
	}
	if config.RestartSilent == 0 {
		config.RestartSilent = time.Minute * 3
	}
	if config.HttpSilent == 0 {
		config.HttpSilent = time.Minute * 3
	}
	if config.BatchSize <= 0 {
		config.BatchSize = 20
	}
	if config.DepthMakerImpact <= 0 {
		config.DepthMakerImpact = 10
	}
	if config.DepthTakerImpact <= 0 {
		config.DepthTakerImpact = 1000
	}
	if config.DepthTimeToLive == 0 {
		config.DepthTimeToLive = time.Second * 3
	}
	if config.TurnoverLookback == 0 {
		config.TurnoverLookback = time.Hour * 24
	}
}
