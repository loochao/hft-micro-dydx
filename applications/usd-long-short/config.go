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

	LogInterval           time.Duration `yaml:"logInterval"`
	BalancePositionMaxAge time.Duration `yaml:"balancePositionMaxAge"`

	BatchSize int `yaml:"batchSize"`

	StartValue  float64            `yaml:"startValue"`
	EnterTarget float64            `yaml:"enterTarget"`
	StartValues map[string]float64 `yaml:"startValues"`

	OrderTimeout       time.Duration `yaml:"orderTimeout"`
	XOrderSilent       time.Duration `yaml:"xOrderSilent"`
	YOrderSilent       time.Duration `yaml:"yOrderSilent"`
	UpdateTargetSilent time.Duration `yaml:"updateTargetSilent"`
	RestartSilent      time.Duration `yaml:"restartSilent"`
	RestartInterval    time.Duration `yaml:"restartInterval"`

	XYPairs map[string]string `yaml:"xyPairs"`
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
	if config.UpdateTargetSilent == 0 {
		config.UpdateTargetSilent = time.Minute * 30
	}
	if config.RestartSilent == 0 {
		config.RestartSilent = time.Minute * 3
	}
	if config.BatchSize <= 0 {
		config.BatchSize = 20
	}
	if config.XOrderSilent == 0 {
		config.XOrderSilent = time.Second
	}
	if config.YOrderSilent == 0 {
		config.YOrderSilent = time.Second * 5
	}
	config.XExchange.DryRun = config.DryRun
	config.YExchange.DryRun = config.DryRun
}
