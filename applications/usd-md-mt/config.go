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

	DepthWalkDelay        time.Duration `yaml:"depthWalkDelay"`
	SpreadWalkDelay       time.Duration `yaml:"spreadWalkDelay"`
	LogInterval           time.Duration `yaml:"logInterval"`
	TurnoverLookback      time.Duration `yaml:"turnoverLookback"`
	BalancePositionMaxAge time.Duration `yaml:"balancePositionMaxAge"`

	EnterOffsetDelta        float64 `yaml:"enterOffsetDelta"`
	ExitOffsetDelta         float64 `yaml:"exitOffsetDelta"`
	LongEnterDelta          float64 `yaml:"longEnterDelta"`
	ShortEnterDelta         float64 `yaml:"shortEnterDelta"`
	LongExitDelta           float64 `yaml:"longExitDelta"`
	ShortExitDelta          float64 `yaml:"shortExitDelta"`
	CancelOffsetFactor      float64 `yaml:"cancelOffsetFactor"`
	MinimalEnterFundingRate float64 `yaml:"minimalEnterFundingRate"`
	MinimalKeepFundingRate  float64 `yaml:"minimalKeepFundingRate"`

	DepthMaxTimeDelta   time.Duration `yaml:"depthTimeDeltaMax"`
	DepthMinTimeDelta   time.Duration `yaml:"depthTimeDeltaMin"`
	DepthYDecay         float64       `yaml:"depthYDecay"`
	DepthXDecay         float64       `yaml:"depthXDecay"`
	DepthYBias          time.Duration `yaml:"depthYBias"`
	DepthXBias          time.Duration `yaml:"depthXBias"`
	DepthTakerImpact    float64       `yaml:"depthTakerImpact"`
	DepthMaxAgeDiffBias time.Duration `yaml:"depthMaxAgeDiffBias"`
	DepthReportCount    int           `yaml:"depthReportCount"`
	SpreadTimeToLive    time.Duration `yaml:"spreadTimeToLive"`
	SpreadLookback      time.Duration `yaml:"spreadLookback"`
	SpreadMinDepthCount int           `yaml:"spreadMinDepthCount"`
	BatchSize           int           `yaml:"batchSize"`

	StartValue        float64            `yaml:"startValue"`
	EnterFreePct      float64            `yaml:"enterFreePct"`
	EnterMinimalStep  float64            `yaml:"enterMinimalStep"`
	EnterTargetFactor float64            `yaml:"enterTargetFactor"`
	StartValues       map[string]float64 `yaml:"startValues"`

	OrderTimeout        time.Duration `yaml:"orderTimeout"`
	XOrderSilent        time.Duration `yaml:"xOrderSilent"`
	YOrderSilent        time.Duration `yaml:"yOrderSilent"`
	XCancelSilent       time.Duration `yaml:"xCancelSilent"`
	XOrderCheckInterval time.Duration `yaml:"xOrderCheckInterval"`
	EnterSilent         time.Duration `yaml:"enterSilent"`
	RestartSilent       time.Duration `yaml:"restartSilent"`
	RestartInterval     time.Duration `yaml:"restartInterval"`

	XYPairs       map[string]string  `yaml:"xyPairs"`
	EnterScales   map[string]float64 `yaml:"enterScales,omitempty"`
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
	if config.DepthTakerImpact <= 0 {
		config.DepthTakerImpact = 1000
	}
	if config.DepthMaxAgeDiffBias == 0 {
		config.DepthMaxAgeDiffBias = time.Millisecond * 100
	}
	if config.DepthReportCount == 0 {
		config.RestartSilent = 1000
	}
	if config.SpreadTimeToLive == 0 {
		config.SpreadTimeToLive = time.Second * 3
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
}
