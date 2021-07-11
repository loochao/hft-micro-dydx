package main

import (
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"testing"
)

func TestReadFile(t *testing.T) {
	contents, err := os.ReadFile("/Users/chenjilin/Projects/hft-micro/applications/usd-ll-mt-q/configs/quantiles/1INCHUSDT-1INCHUSDT-long-td.json")
	if err != nil {
		t.Fatal(err)
	}else{
		logger.Debugf("%s", contents)
	}
}