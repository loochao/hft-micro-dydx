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

	Exchange common.ExchangeSettings `yaml:"exchange"`

	LoopInterval          time.Duration `yaml:"loopInterval"`
	LogInterval           time.Duration `yaml:"logInterval"`
	TurnoverLookback      time.Duration `yaml:"turnoverLookback"`
	BalancePositionMaxAge time.Duration `yaml:"balancePositionMaxAge"`

	EnterOffsetDelta        float64 `yaml:"enterOffsetDelta"`
	ExitOffsetDelta         float64 `yaml:"exitOffsetDelta"`
	LongEnterDelta          float64 `yaml:"longEnterDelta"`
	ShortEnterDelta         float64 `yaml:"shortEnterDelta"`
	LongExitDelta           float64 `yaml:"longExitDelta"`
	ShortExitDelta          float64 `yaml:"shortExitDelta"`
	MinimalEnterFundingRate float64 `yaml:"minimalEnterFundingRate"`
	MinimalKeepFundingRate  float64 `yaml:"minimalKeepFundingRate"`

	DepthTimeDeltaMax   time.Duration `yaml:"depthTimeDeltaMax"`
	DepthTimeDeltaMin   time.Duration `yaml:"depthTimeDeltaMin"`
	DepthYDecay         float64       `yaml:"depthYDecay"`
	DepthXDecay         float64       `yaml:"depthXDecay"`
	DepthYBias          time.Duration `yaml:"depthYBias"`
	DepthXBias          time.Duration `yaml:"depthXBias"`
	BatchSize           int           `yaml:"depthBatchSize"`
	DepthTakerImpact    float64       `yaml:"depthTakerImpact"`
	DepthMaxAgeDiffBias time.Duration `yaml:"depthMaxAgeDiffBias"`
	ReportCount         int           `yaml:"reportCount"`
	SpreadTimeToLive    time.Duration `yaml:"spreadTimeToLive"`
	SpreadLookback      time.Duration `yaml:"spreadLookback"`

	StartValue  float64            `yaml:"startValue"`
	EnterStep   float64            `yaml:"enterStep"`
	EnterTarget float64            `yaml:"enterTarget"`
	StartValues map[string]float64 `yaml:"startValues"`

	OrderTimeout    time.Duration `yaml:"orderTimeout"`
	OrderSilent     time.Duration `yaml:"orderSilent"`
	CancelSilent    time.Duration `yaml:"cancelSilent"`
	EnterSilent     time.Duration `yaml:"enterSilent"`
	RestartSilent   time.Duration `yaml:"restartSilent"`
	HttpSilent      time.Duration `yaml:"httpSilent"`
	RestartInterval time.Duration `yaml:"restartInterval"`
	DeliDuration    time.Duration `yaml:"deliDuration"`

	XYPairs        map[string]string    `yaml:"xyPairs"`
	SymbolAssetMap map[string]string    `yaml:"symbolAssetMap"`
	ExpireDates    map[string]time.Time `yaml:"expireDates"`
}

func (config *Config) SetDefaultIfNotSet() {
	if config.LoopInterval == 0 {
		config.LoopInterval = time.Second
	}
	if config.LogInterval == 0 {
		config.LoopInterval = time.Minute
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
	if config.RestartInterval == 0 {
		config.RestartInterval = time.Hour * 9999
	}
	if config.TurnoverLookback == 0 {
		config.TurnoverLookback = time.Hour * 24
	}

	if config.EnterTarget == 0 {
		config.EnterTarget = 1.0
	}
	if config.DeliDuration == 0 {
		config.DeliDuration = time.Hour*24*90
	}
}
