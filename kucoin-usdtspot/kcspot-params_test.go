package kucoin_usdtspot

import (
	"encoding/json"
	"github.com/geometrybase/hft-micro/logger"
	"testing"
)

func TestNewOrderParams(t *testing.T) {
	o := NewOrderParam{
		Price: Float64(62.181000000000004),
	}
	d, err := json.Marshal(o)
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%s", d)
	o = NewOrderParam{
		Price: Float64(62),
	}
	d, err = json.Marshal(o)
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%s", d)
}
