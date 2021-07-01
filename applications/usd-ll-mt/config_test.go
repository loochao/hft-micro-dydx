package main

import (
	"github.com/geometrybase/hft-micro/logger"
	"github.com/geometrybase/hft-micro/tdigest"
	"testing"
)

func TestConfig_SetDefaultIfNotSet(t *testing.T) {
	//config := Config{}
	//config.SetDefaultIfNotSet()
	//msg, _ := yaml.Marshal(config)
	//for _, l := range strings.Split(string(msg), "\n") {
	//	logger.Debugf("%s", l)
	//}

	td, _ := tdigest.New()
	msg, _  := td.AsBytes()
	_ = td.Add(100.0)
	_ = td.Add(120.0)
	_ = td.Add(110.0)
	_ = td.Add(109.0)
	_ = td.Add(102.0)
	_ = td.Add(110.0)
	logger.Debugf("%f", td.Quantile(0.5))
	logger.Debugf("%s", msg)
	err := td.FromBytes(msg)
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%f", td.Quantile(0.5))

}
