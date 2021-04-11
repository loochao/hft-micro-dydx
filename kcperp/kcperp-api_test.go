package kcperp

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"log"
	"testing"
)

var api *API
var ctx = context.Background()
func init() {
	var err error
	api, err = NewAPI(&common.Credentials{}, "socks5://127.0.0.1:1081")
	if err != nil {
		log.Fatal(err)
	}
}

func TestAPI_GetContracts(t *testing.T) {
	symbols, err := api.GetContracts(ctx)
	if err != nil {
		logger.Debugf("%v", err)
		t.Fatal(err)
	}
	usdSymbols := make([]string, 0)
	for _, s := range symbols {
		if s.QuoteCurrency == "USDT" && s.Status == "Open" && s.FairMethod == "FundingRate"{
			logger.Debugf("%s %s %s %s %s", s.FairMethod, s.Symbol, s.BaseCurrency, s.QuoteCurrency, s.RootSymbol)
			usdSymbols = append(usdSymbols, s.Symbol)
		}
	}
	logger.Debugf("%d",  len(usdSymbols))
}


