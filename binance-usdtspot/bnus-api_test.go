package binance_usdtspot

import (
	"context"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
)

func TestFormatByPrecision(t *testing.T) {
	logger.Debugf("%s", strconv.FormatFloat(0.1111111111, 'f', 0, 64))
	logger.Debugf("%s", strconv.FormatFloat(0.1111111111, 'f', 1, 64))
	logger.Debugf("%s", strconv.FormatFloat(0.1111111111, 'f', 2, 64))
	logger.Debugf("%s", strconv.FormatFloat(0.1111111111, 'f', 3, 64))
	logger.Debugf("%s", strconv.FormatFloat(0.1111111111, 'f', 4, 64))
	logger.Debugf("%s", strconv.FormatFloat(0.1111111111, 'f', 5, 64))
	logger.Debugf("%s", strconv.FormatFloat(0.1111111111, 'f', 6, 64))
	logger.Debugf("%s", strconv.FormatFloat(0.1111111111, 'f', 7, 64))
	logger.Debugf("%s", strconv.FormatFloat(0.1111111111, 'f', 8, 64))
	logger.Debugf("%s", strconv.FormatFloat(0.1111111111, 'f', 9, 64))
	logger.Debugf("%s", strconv.FormatFloat(0.1111111111, 'f', 10, 64))
	logger.Debugf("%s", strconv.FormatFloat(0.1111111111, 'f', 11, 64))
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
	tickPrecisions := make(map[string]int)
	stepPrecisions := make(map[string]int)
	stepSizes := make(map[string]float64)
	minSizes := make(map[string]float64)
	multiplierUps := make(map[string]float64)
	multiplierDowns := make(map[string]float64)
	minNotional := make(map[string]float64)
	for _, symbol := range exchangeInfo.Symbols {
		//logger.Debugf(symbol.Status)
		if symbol.Status !=  "TRADING" || symbol.QuoteAsset != "USDT"{
			continue
		}
		for _, filter := range symbol.Filters {
			//logger.Debugf("%s", filter.FilterType)
			switch filter.FilterType {
			case "PRICE_FILTER":
				tickSizes[symbol.Symbol] = filter.TickSize
				tickPrecisions[symbol.Symbol] = common.GetFloatPrecision(filter.TickSize)
				//logger.Debugf("TICK %f %d", filter.TickSize, common.GetFloatPrecision(filter.TickSize))
			case "LOT_SIZE":
				stepSizes[symbol.Symbol] = filter.StepSize
				stepPrecisions[symbol.Symbol] = common.GetFloatPrecision(filter.StepSize)
				minSizes[symbol.Symbol] = filter.MinQty
				//logger.Debugf("STEP %f %d", filter.StepSize, common.GetFloatPrecision(filter.StepSize))
			case "PERCENT_PRICE":
				multiplierUps[symbol.Symbol] = filter.MultiplierUp
				multiplierDowns[symbol.Symbol] = filter.MultiplierDown
			case "MIN_NOTIONAL":
				minNotional[symbol.Symbol] = filter.MinNotional
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
	str += "var MinNotionals = map[string]float64{\n"
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
	str += "var TickPrecisions = map[string]int{\n"
	for symbol, value := range tickPrecisions {
		str += fmt.Sprintf("  \"%s\": %d,\n", symbol, value)
	}
	str += "}\n\n"
	str += "var StepPrecisions = map[string]int{\n"
	for symbol, value := range stepPrecisions {
		str += fmt.Sprintf("  \"%s\": %d,\n", symbol, value)
	}
	str += "}\n\n"
	fmt.Printf("%s", str)
}


func TestAPI_GetTicker(t *testing.T) {
	proxy := "socks5://127.0.0.1:1080"

	api, err := NewAPI(&common.Credentials{}, proxy)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	ticker, err := api.GetTicker(ctx, TickerParam{Symbol: "BNBUSDT"})
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "BNBUSDT", ticker.Symbol)
	logger.Debugf("%f", ticker.Price)
}