package binance_usdtfuture

import (
	"context"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"sort"
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

func TestAPI_GetPositions(t *testing.T) {
	proxy := "socks5://127.0.0.1:1083"

	api, err := NewAPI(&common.Credentials{
		Key: "DG8E0kxWVSlgSEZOS5VMiOPGg81xv4LhQ54YWTMN4LtDX7C8OEFlE7m6Dy2UBf2G",
		Secret: "xYM12yQqiMBGdXTAihzOaNFBf7yGKO1zXev9TSRXWBWHdV3on14UiT91GQH1T05g",
	}, proxy)
	if err != nil {
		t.Fatal(err)
	}
	positions, err := api.GetPositions(context.Background())
	for _, pos := range positions {
		if pos.Symbol == "BNBUSDT" {
			logger.Debugf("%v %f", pos, pos.PositionAmt)
			if pos.PositionAmt < 0 {
				order, err := api.SubmitOrder(context.Background(), NewOrderParams{
					Symbol: "BNBUSDT",
					ReduceOnly: true,
					Side: OrderSideBuy,
					Type: OrderTypeMarket,
					Quantity: -pos.PositionAmt,
				})
				if err != nil {
					t.Fatal(err)
				}
				logger.Debugf("%v", order)
			}
		}
	}
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

	indexes, err := api.GetPremiumIndex(ctx)
	if err != nil {
		t.Fatal(err)
	}

	indexPrices := make(map[string]float64)
	for _, index := range indexes {
		indexPrices[index.Symbol] = index.IndexPrice
	}

	tickSizes := make(map[string]float64)
	stepSizes := make(map[string]float64)
	minSizes := make(map[string]float64)
	multiplierUps := make(map[string]float64)
	multiplierDowns := make(map[string]float64)
	minNotional := make(map[string]float64)
	tickPrecisions := make(map[string]int)
	stepPrecisions := make(map[string]int)
	maxPosSizes := make(map[string]float64)
	maxPosValues := make(map[string]float64)

	symbols := make([]string, 0)
	for _, symbol := range exchangeInfo.Symbols {
		//logger.Debugf("%s", symbol.ContractType)
		if symbol.ContractType != "PERPETUAL" || symbol.Status != "TRADING" || symbol.QuoteAsset != "USDT"{
			continue
		}
		symbols = append(symbols, symbol.Symbol)
		for _, filter := range symbol.Filters {
			switch filter.FilterType {
			case "PRICE_FILTER":
				tickSizes[symbol.Symbol] = filter.TickSize
				tickPrecisions[symbol.Symbol] = common.GetFloatPrecision(filter.TickSize)
			case "MARKET_LOT_SIZE":
				stepSizes[symbol.Symbol] = filter.StepSize
				minSizes[symbol.Symbol] = filter.MinQty
				maxPosSizes[symbol.Symbol] = filter.MaxQty
				maxPosValues[symbol.Symbol] = math.Floor(filter.MaxQty*indexPrices[symbol.Symbol]/10000)*10000
				stepPrecisions[symbol.Symbol] = common.GetFloatPrecision(filter.StepSize)
			case "PERCENT_PRICE":
				multiplierUps[symbol.Symbol] = filter.MultiplierUp
				multiplierDowns[symbol.Symbol] = filter.MultiplierDown
			case "MIN_NOTIONAL":
				minNotional[symbol.Symbol] = filter.Notional
			}
		}
	}
	sort.Strings(symbols)
	str := "var TickSizes = map[string]float64{\n"
	for _, symbol := range symbols {
		value := tickSizes[symbol]
		str += fmt.Sprintf("  \"%s\": %s,\n", symbol, strconv.FormatFloat(value, 'f', -1, 64))
	}
	str += "}\n\n"
	str += "var StepSizes = map[string]float64{\n"
	for _, symbol := range symbols {
		value := stepSizes[symbol]
		str += fmt.Sprintf("  \"%s\": %s,\n", symbol, strconv.FormatFloat(value, 'f', -1, 64))
	}
	str += "}\n\n"
	str += "var MinSizes = map[string]float64{\n"
	for _, symbol := range symbols {
		value := minSizes[symbol]
		str += fmt.Sprintf("  \"%s\": %s,\n", symbol, strconv.FormatFloat(value, 'f', -1, 64))
	}
	str += "}\n\n"
	str += "var MinNotional = map[string]float64{\n"
	for _, symbol := range symbols {
		value := minNotional[symbol]
		str += fmt.Sprintf("  \"%s\": %s,\n", symbol, strconv.FormatFloat(value, 'f', -1, 64))
	}
	str += "}\n\n"
	str += "var MultiplierUps = map[string]float64{\n"
	for _, symbol := range symbols {
		value := multiplierUps[symbol]
		str += fmt.Sprintf("  \"%s\": %s,\n", symbol, strconv.FormatFloat(value, 'f', -1, 64))
	}
	str += "}\n\n"
	str += "var MultiplierDowns = map[string]float64{\n"
	for _, symbol := range symbols {
		value := multiplierDowns[symbol]
		str += fmt.Sprintf("  \"%s\": %s,\n", symbol, strconv.FormatFloat(value, 'f', -1, 64))
	}
	str += "}\n\n"
	str += "var TickPrecisions = map[string]int{\n"
	for _, symbol := range symbols {
		value := tickPrecisions[symbol]
		str += fmt.Sprintf("  \"%s\": %d,\n", symbol, value)
	}
	str += "}\n\n"
	str += "var StepPrecisions = map[string]int{\n"
	for _, symbol := range symbols {
		value := stepPrecisions[symbol]
		str += fmt.Sprintf("  \"%s\": %d,\n", symbol, value)
	}
	str += "}\n\n"
	str += "var MaxPosSizes = map[string]float64{\n"
	for _, symbol := range symbols {
		value := maxPosSizes[symbol]
		str += fmt.Sprintf("  \"%s\": %.0f,\n", symbol, value)
	}
	str += "}\n\n"
	str += "var MaxPosValues = map[string]float64{\n"
	for _, symbol := range symbols {
		value := maxPosValues[symbol]
		str += fmt.Sprintf("  \"%s\": %.0f,\n", symbol, value)
	}
	str += "}\n\n"
	fmt.Printf("%s", str)
}
