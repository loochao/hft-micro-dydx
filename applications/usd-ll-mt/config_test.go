package main

import (
	"github.com/geometrybase/hft-micro/logger"
	"gopkg.in/yaml.v2"
	"strings"
	"testing"
)

func TestConfig_SetDefaultIfNotSet(t *testing.T) {
	config := Config{}
	config.SetDefaultIfNotSet()
	msg, _ := yaml.Marshal(config)
	for _, l := range strings.Split(string(msg), "\n") {
		logger.Debugf("%s", l)
	}
}
