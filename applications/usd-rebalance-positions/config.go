package main

import (
	"github.com/geometrybase/hft-micro/common"
	"time"
)

type Config struct {
	Name *string `yaml:"name"`

	CpuProfile string `yaml:"cpuProfile"`
	DryRun     bool   `yaml:"dryRun"`

	XExchange             common.ExchangeSettings `yaml:"xExchange"`
	YExchange             common.ExchangeSettings `yaml:"yExchange"`
	BalancePositionMaxAge time.Duration           `yaml:"balancePositionMaxAge"`
	OrderValue            float64                 `yaml:"orderValue"`
	OrderTimeout          time.Duration           `yaml:"orderTimeout"`
	OrderSilent           time.Duration           `yaml:"orderSilent"`
	LogInterval           time.Duration           `yaml:"logInterval"`
	BatchSize             int                     `yaml:"batchSize"`

	RestartSilent   time.Duration `yaml:"restartSilent"`
	RestartInterval time.Duration `yaml:"restartInterval"`

	XYPairs map[string]string `yaml:"xyPairs"`
}

func (config *Config) SetDefaultIfNotSet() {
	if config.BalancePositionMaxAge == 0 {
		config.BalancePositionMaxAge = time.Minute * 3
	}
	if config.OrderTimeout == 0 {
		config.OrderTimeout = time.Second * 5
	}
	if config.RestartSilent == 0 {
		config.RestartSilent = time.Minute * 3
	}
	if config.RestartInterval == 0 {
		config.RestartInterval = time.Hour * 9999
	}
	if config.OrderSilent == 0 {
		config.OrderSilent = time.Minute
	}
	if config.BatchSize == 0 {
		config.BatchSize = 30
	}
	config.XExchange.DryRun = config.DryRun
	config.YExchange.DryRun = config.DryRun
}
