package main

import (
	"github.com/geometrybase/hft-micro/common"
	"time"
)

type Config struct {
	Name   *string `yaml:"name"`
	DryRun *bool   `yaml:"dryRun"`

	InternalInfluxSettings common.InfluxSettings `yaml:"internalInfluxSettings"`
	ExternalInfluxSettings common.InfluxSettings `yaml:"externalInfluxSettings"`

	ExchangeSettings common.ExchangeSettings `yaml:"exchangeSettings"`

	PositionMaxAge time.Duration `yaml:"positionMaxAge,omitempty"`
	LoopInterval   time.Duration `yaml:"loopInterval,omitempty"`
	OrderTimeout   time.Duration `yaml:"orderTimeout,omitempty"`
	OrderSilent    time.Duration `yaml:"orderSilent,omitempty"`
	CancelSilent   time.Duration `yaml:"cancelSilent,omitempty"`
	GlobalSilent   time.Duration `yaml:"globalSilent,omitempty"`

	UpdateInterval time.Duration `yaml:"updateInterval,omitempty"`
	TradeLookback  time.Duration `yaml:"tradeLookback,omitempty"`
	DepthLevel     int           `yaml:"depthLevel,omitempty"`
	BatchSize      int           `yaml:"batchSize,omitempty"`

	StartValue  float64            `yaml:"startValue,omitempty"`
	StartValues map[string]float64 `yaml:"startValues,omitempty"`
}
