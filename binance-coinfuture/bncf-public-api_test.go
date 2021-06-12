package binance_coinfuture

import (
	"context"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/stretchr/testify/assert"
	"os"
	"strconv"
	"testing"
	"time"
)

func TestAPI_GetServerTime(t *testing.T) {
	api, err := NewAPI(&common.Credentials{}, os.Getenv("BN_TEST_PROXY"))
	if err != nil {
		t.Fatal(err)
	}
	tt, err := api.GetServerTime(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	diff := time.Now().UnixNano()/1000000 - tt.ServerTime
	logger.Debugf("DIFF %v", diff)
}

func TestAPI_GetExchangeInfo(t *testing.T) {
	api, err := NewAPI(&common.Credentials{}, os.Getenv("BN_TEST_PROXY"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	exchangeInfo, err := api.GetExchangeInfo(ctx)
	if err != nil {
		t.Fatal(err)
	}

	tickSizes := make(map[string]float64)
	stepSizes := make(map[string]float64)
	minSizes := make(map[string]float64)
	multiplierUps := make(map[string]float64)
	multiplierDowns := make(map[string]float64)
	contractSizes := make(map[string]float64)
	for _, symbol := range exchangeInfo.Symbols {
		logger.Debugf("%v", symbol)
		if symbol.ContractStatus != "TRADING" {
			continue
		}
		contractSizes[symbol.Symbol] = float64(symbol.ContractSize)
		for _, filter := range symbol.Filters {
			switch filter.FilterType {
			case "PRICE_FILTER":
				tickSizes[symbol.Symbol] = filter.TickSize
			case "MARKET_LOT_SIZE":
				stepSizes[symbol.Symbol] = filter.StepSize
				minSizes[symbol.Symbol] = filter.MinQty
			case "PERCENT_PRICE":
				multiplierUps[symbol.Symbol] = filter.MultiplierUp
				multiplierDowns[symbol.Symbol] = filter.MultiplierDown
			}
		}
	}
	str := "var TickSizes = map[string]float64{\n"
	for symbol, value := range tickSizes {
		str += fmt.Sprintf("  \"%s\": %s,\n", symbol, strconv.FormatFloat(value, 'f', -1, 64))
	}
	str += "}\n\n"
	str += "var StepSizes = map[string]float64{\n"
	for symbol, value := range stepSizes {
		str += fmt.Sprintf("  \"%s\": %s,\n", symbol, strconv.FormatFloat(value, 'f', -1, 64))
	}
	str += "}\n\n"
	str += "var MinSizes = map[string]float64{\n"
	for symbol, value := range minSizes {
		str += fmt.Sprintf("  \"%s\": %s,\n", symbol, strconv.FormatFloat(value, 'f', -1, 64))
	}
	str += "}\n\n"
	str += "var ContractSizes = map[string]float64{\n"
	for symbol, value := range contractSizes {
		str += fmt.Sprintf("  \"%s\": %s,\n", symbol, strconv.FormatFloat(value, 'f', -1, 64))
	}
	str += "}\n\n"
	str += "var MultiplierUps = map[string]float64{\n"
	for symbol, value := range multiplierUps {
		str += fmt.Sprintf("  \"%s\": %s,\n", symbol, strconv.FormatFloat(value, 'f', -1, 64))
	}
	str += "}\n\n"
	str += "var MultiplierDowns = map[string]float64{\n"
	for symbol, value := range multiplierDowns {
		str += fmt.Sprintf("  \"%s\": %s,\n", symbol, strconv.FormatFloat(value, 'f', -1, 64))
	}
	str += "}\n\n"
	fmt.Printf("%s", str)
}

func TestAPI_GetPremiumIndex(t *testing.T) {
	api, err := NewAPI(&common.Credentials{}, os.Getenv("BN_TEST_PROXY"))
	if err != nil {
		t.Fatal(err)
	}
	indexes, err := api.GetPremiumIndex(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", indexes)
}

func TestAPI_GetKLines(t *testing.T) {
	api, err := NewAPI(&common.Credentials{}, os.Getenv("BN_TEST_PROXY"))
	if err != nil {
		t.Fatal(err)
	}
	klines, err := api.GetKLines(context.Background(), KlineParams{
		Symbol: "LTCUSD_PERP",
		Limit: 10,
		Interval: KlineInterval1d,
	})
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 10, len(klines))
	assert.Equal(t, time.Duration(0),time.Now().Truncate(time.Hour*24).Add(time.Hour*24).Sub(klines[len(klines)-1].Timestamp))
}
