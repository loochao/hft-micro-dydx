package binance_busdfuture

import (
	"context"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"strconv"
	"testing"
	"time"
)

func TestAPI_GetServerTime(t *testing.T) {
	proxy := "socks5://127.0.0.1:1081"

	api, err := NewAPI(&common.Credentials{}, proxy)
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
	proxy := "socks5://127.0.0.1:1083"

	api, err := NewAPI(&common.Credentials{}, proxy)
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
	minNotional := make(map[string]float64)
	for _, symbol := range exchangeInfo.Symbols {
		if symbol.ContractType != "PERPETUAL" || symbol.Status != "TRADING" || symbol.QuoteAsset != "BUSD"{
			continue
		}
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
			case "MIN_NOTIONAL":
				minNotional[symbol.Symbol] = filter.Notional
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
	str += "var MinNotional = map[string]float64{\n"
	for symbol, value := range minNotional {
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
