package main

import (
	"github.com/geometrybase/hft-micro/common"
	"time"
)

type Config struct {
	Name *string `yaml:"name"`

	CpuProfile   string `yaml:"CpuProfile"`
	DryRun       bool   `yaml:"dryRun"`
	ProxyAddress string `yaml:"proxyAddress"`

	InternalInflux common.InfluxSettings `yaml:"internalInflux"`
	ExternalInflux common.InfluxSettings `yaml:"externalInflux"`

	MakerExchange common.ExchangeSettings `yaml:"makerExchange"`
	TakerExchange common.ExchangeSettings `yaml:"takerExchange"`

	LoopInterval          time.Duration `yaml:"loopInterval"`
	LogInterval           time.Duration `yaml:"logInterval"`
	PullInterval          time.Duration `yaml:"pullInterval"`
	RequestInterval       time.Duration `yaml:"requestInterval"`
	BalancePositionMaxAge time.Duration `yaml:"balancePositionMaxAge"`

	MakerTakerPairs map[string]string `yaml:"makerTakerPairs"`

	LongEnterDelta  float64 `yaml:"longEnterDelta"`
	LongExitDelta   float64 `yaml:"longExitDelta"`
	ShortEnterDelta float64 `yaml:"shortEnterDelta"`
	ShortExitDelta  float64 `yaml:"shortExitDelta"`
	OffsetDelta     float64 `yaml:"offsetDelta"`

	MinimalEnterFundingRate float64 `yaml:"minimalEnterFundingRate"`
	MinimalKeepFundingRate  float64 `yaml:"minimalKeepFundingRate"`

	DepthTimeDeltaMax      time.Duration     `yaml:"depthTimeDeltaMax"`
	DepthTimeDeltaMin      time.Duration     `yaml:"depthTimeDeltaMin"`
	DepthTakerDecay        float64           `yaml:"depthTakerDecay"`
	DepthMakerDecay     float64           `yaml:"depthMakerDecay"`
	DepthTakerBias      time.Duration     `yaml:"depthTakerBias"`
	DepthMakerBias      time.Duration     `yaml:"depthMakerBias"`
	BatchSize           int               `yaml:"depthBatchSize"`
	DepthMakerImpact    float64           `yaml:"depthMakerImpact"`
	DepthTakerImpact    float64           `yaml:"depthTakerImpact"`
	DepthMaxAgeDiffBias time.Duration     `yaml:"depthMaxAgeDiffBias"`
	DepthDirLookback    time.Duration     `yaml:"depthDirLookback"`
	ReportCount         int               `yaml:"reportCount"`
	SpreadTimeToLive    time.Duration     `yaml:"spreadTimeToLive"`
	SpreadLookback      time.Duration     `yaml:"spreadLookback"`
	MakerOrderOffsets   map[string]string `yaml:"makerOrderOffsets"`

	HedgeInstantly     bool          `yaml:"hedgeInstantly"`
	HedgeCheckInterval time.Duration `yaml:"hedgeCheckInterval"`
	HedgeTrackOffset   float64 `yaml:"hedgeTrackOffset"`

	StartValue        float64            `yaml:"startValue"`
	EnterFreePct      float64            `yaml:"enterFreePct"`
	EnterMinimalStep  float64            `yaml:"enterMinimalStep"`
	EnterTargetFactor float64            `yaml:"enterTargetFactor"`
	MaxUnHedgeValue   float64            `yaml:"maxUnHedgeValue"`
	StartValues       map[string]float64 `yaml:"startValues"`

	OrderTimeout  time.Duration `yaml:"orderTimeout"`
	OrderSilent   time.Duration `yaml:"orderSilent"`
	CancelSilent  time.Duration `yaml:"cancelSilent"`
	EnterSilent   time.Duration `yaml:"enterSilent"`
	RestartSilent time.Duration `yaml:"restartSilent"`
	HttpSilent    time.Duration `yaml:"httpSilent"`
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
	if config.DepthMaxAgeDiffBias == 0 {
		config.DepthMaxAgeDiffBias = time.Millisecond * 100
	}
	if config.ReportCount == 0 {
		config.RestartSilent = 1000
	}
	if config.SpreadTimeToLive == 0 {
		config.SpreadTimeToLive = time.Second * 3
	}
	if config.SpreadLookback == 0 {
		config.SpreadLookback = time.Second
	}
}
