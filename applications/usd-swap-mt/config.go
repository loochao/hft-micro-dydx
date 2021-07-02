package main

import (
	"github.com/geometrybase/hft-micro/common"
	"time"
)

type Config struct {
	Name *string `yaml:"name"`

	CpuProfile string `yaml:"cpuProfile"`
	DryRun     bool   `yaml:"dryRun"`

	InternalInflux common.InfluxSettings `yaml:"internalInflux"`
	ExternalInflux common.InfluxSettings `yaml:"externalInflux"`

	XExchange common.ExchangeSettings `yaml:"xExchange"`
	YExchange common.ExchangeSettings `yaml:"yExchange"`

	DepthWalkDelay        time.Duration `yaml:"depthWalkDelay"`
	SpreadWalkDelay       time.Duration `yaml:"spreadWalkDelay"`
	LogInterval           time.Duration `yaml:"logInterval"`
	TurnoverLookback      time.Duration `yaml:"turnoverLookback"`
	BalancePositionMaxAge time.Duration `yaml:"balancePositionMaxAge"`

	EnterDelta         float64 `yaml:"enterDelta"`
	CancelOffsetFactor float64 `yaml:"cancelOffsetFactor"`

	DepthMaxTimeDelta    time.Duration `yaml:"depthTimeDeltaMax"`
	DepthMinTimeDelta    time.Duration `yaml:"depthTimeDeltaMin"`
	DepthYDecay          float64       `yaml:"depthYDecay"`
	DepthXDecay          float64       `yaml:"depthXDecay"`
	DepthYBias           time.Duration `yaml:"depthYBias"`
	DepthXBias           time.Duration `yaml:"depthXBias"`
	DepthTakerImpact     float64       `yaml:"depthTakerImpact"`
	DepthMaxAgeDiffBias  time.Duration `yaml:"depthMaxAgeDiffBias"`
	DepthReportCount     int           `yaml:"depthReportCount"`
	SpreadTimeToCancel   time.Duration `yaml:"spreadTimeToCancel"`
	SpreadTimeToEnter    time.Duration `yaml:"spreadTimeToEnter"`
	SpreadLookback       time.Duration `yaml:"spreadLookback"`
	SpreadMinDepthCount  int           `yaml:"spreadMinDepthCount"`
	EnterDepthMatchRatio float64       `yaml:"enterDepthMatchRatio"`
	BatchSize            int           `yaml:"batchSize"`

	StartValue       float64            `yaml:"startValue"`
	EnterPct         float64            `yaml:"enterPct"`
	EnterMinimalStep float64            `yaml:"enterMinimalStep"`
	StartValues      map[string]float64 `yaml:"startValues"`

	OrderTimeout       time.Duration `yaml:"orderTimeout"`
	OrderSilent        time.Duration `yaml:"orderSilent"`
	CancelSilent       time.Duration `yaml:"cancelSilent"`
	ErrorSilent        time.Duration `yaml:"errorSilent"`
	OrderCheckInterval time.Duration `yaml:"orderCheckInterval"`
	EnterSilent        time.Duration `yaml:"enterSilent"`
	RestartSilent      time.Duration `yaml:"restartSilent"`
	RestartInterval    time.Duration `yaml:"restartInterval"`

	XYPairs        map[string]string  `yaml:"xyPairs"`
	XOrderOffsets  map[string]string  `yaml:"xOrderOffsets,omitempty"`
	YOrderOffsets  map[string]string  `yaml:"yOrderOffsets,omitempty"`
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
	if config.DepthTakerImpact <= 0 {
		config.DepthTakerImpact = 1000
	}
	if config.DepthMaxAgeDiffBias == 0 {
		config.DepthMaxAgeDiffBias = time.Millisecond * 100
	}
	if config.DepthReportCount == 0 {
		config.RestartSilent = 1000
	}
	if config.SpreadTimeToCancel == 0 {
		config.SpreadTimeToCancel = time.Second * 3
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
	if config.OrderSilent == 0 {
		config.OrderSilent = time.Second
	}
	if config.CancelSilent == 0 {
		config.CancelSilent = config.OrderSilent
	}
	config.XExchange.DryRun = config.DryRun
	config.YExchange.DryRun = config.DryRun
}
