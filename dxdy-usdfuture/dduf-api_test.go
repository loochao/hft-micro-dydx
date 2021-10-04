package dxdy_usdtfuture

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
		Key:    "DG8E0kxWVSlgSEZOS5VMiOPGg81xv4LhQ54YWTMN4LtDX7C8OEFlE7m6Dy2UBf2G",
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
					Symbol:     "BNBUSDT",
					ReduceOnly: true,
					Side:       OrderSideBuy,
					Type:       OrderTypeMarket,
					Quantity:   -pos.PositionAmt,
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
	markets, err := api.GetMarkets(ctx)
	if err != nil {
		t.Fatal(err)
	}

	tickSizes := make(map[string]float64)
	stepSizes := make(map[string]float64)
	minSizes := make(map[string]float64)
	tickPrecisions := make(map[string]int)
	stepPrecisions := make(map[string]int)
	for _, market := range markets {
		if market.Type != "PERPETUAL" || market.Status != "ONLINE" {
			continue
		}
		tickSizes[market.Market] = market.TickSize
		tickPrecisions[market.Market] = common.GetFloatPrecision(market.TickSize)
		stepSizes[market.Market] = market.StepSize
		minSizes[market.Market] = market.MinOrderSize
		stepPrecisions[market.Market] = common.GetFloatPrecision(market.StepSize)
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
