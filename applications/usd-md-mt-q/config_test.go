package main

import (
	"context"
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
	msg, _ := td.AsBytes()
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

func TestDepthInputCh(t *testing.T) {
	xExchangeCh := make(chan int, 16)
	yExchangeCh := make(chan int, 16)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		for i := 0; i < 100; i++ {
			if i  % 2 == 0 {
				xExchangeCh <- i
			}else {
				yExchangeCh <- i
			}
		}
		cancel()
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case x := <-xExchangeCh:
			logger.Debugf("x %d", x)
		case y := <-yExchangeCh:
			logger.Debugf("y %d", y)
		}
	}
}
