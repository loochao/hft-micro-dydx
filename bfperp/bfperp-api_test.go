package bfperp

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"testing"
)

func TestAPI_GetTradingParis(t *testing.T) {
	api, err := NewAPI(&common.Credentials{}, "socks5://127.0.0.1:1081")
	if err != nil {
		t.Fatal(err)
	}
	pairs, err := api.GetDerivativeParis(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", pairs)
}
