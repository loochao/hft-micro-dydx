package main

import (
	"github.com/geometrybase/hft-micro/common"
	"time"
)

type Config struct {
	DryRun         bool                  `yaml:"dryRun"`
	Influx         common.InfluxSettings `yaml:"influx"`
	UpdateInterval time.Duration         `yaml:"updateInterval"`
	StartValues    map[string]float64    `yaml:"startValues"`
	ValueField     string                `yaml:"valueField"`
}

func (config *Config) SetDefaultIfNotSet() {
	if config.UpdateInterval == 0 {
		config.UpdateInterval = time.Minute
	}
}
