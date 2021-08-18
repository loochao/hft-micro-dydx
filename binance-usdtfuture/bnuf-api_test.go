package binance_usdtfuture

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

	tickSizes := make(map[string]float64)
	stepSizes := make(map[string]float64)
	minSizes := make(map[string]float64)
	multiplierUps := make(map[string]float64)
	multiplierDowns := make(map[string]float64)
	minNotional := make(map[string]float64)
	for _, symbol := range exchangeInfo.Symbols {
		//logger.Debugf("%s", symbol.ContractType)
		if symbol.ContractType != "PERPETUAL" || symbol.Status != "TRADING" || symbol.QuoteAsset != "USDT"{
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
