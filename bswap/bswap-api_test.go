package bswap

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"testing"
	"time"
)

func TestAPI_GetPools(t *testing.T) {
	api, err := NewAPI(&common.Credentials{
		Key:    os.Getenv("BN_TEST_KEY"),
		Secret: os.Getenv("BN_TEST_SECRET"),
	}, "socks5://127.0.0.1:1080")
	if err != nil {
		t.Fatal(err)
	}
	pools, err := api.GetPools(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", pools)
}

func TestAPI_GetLiquidity(t *testing.T) {
	api, err := NewAPI(&common.Credentials{
		Key:    os.Getenv("BN_TEST_KEY"),
		Secret: os.Getenv("BN_TEST_SECRET"),
	}, "socks5://127.0.0.1:1080")
	if err != nil {
		t.Fatal(err)
	}
	pools, err := api.GetLiquidity(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", pools)
}

func TestAPI_GetQuote(t *testing.T) {
	api, err := NewAPI(&common.Credentials{
		Key:    os.Getenv("BN_TEST_KEY"),
		Secret: os.Getenv("BN_TEST_SECRET"),
	}, "socks5://127.0.0.1:1080")
	if err != nil {
		t.Fatal(err)
	}
	for {
		select {
		case <-time.After(time.Second * 5):
			bid, err := api.GetQuote(context.Background(), QuoteParam{
				QuoteAsset: "FIL",
				BaseAsset:  "USDT",
				QuoteQty:   1,
			})
			if err != nil {
				t.Fatal(err)
			}
			logger.Debugf("1 %v", bid)
			ask, err := api.GetQuote(context.Background(), QuoteParam{
				QuoteAsset: "USDT",
				BaseAsset:  "FIL",
				QuoteQty:   1,
			})
			if err != nil {
				t.Fatal(err)
			}
			logger.Debugf("2 %v", ask)
			logger.Debugf("spread %f cost %f", (ask.Price-1.0/bid.Price)/ask.Price, ask.Slippage+ask.Fee+bid.Fee+bid.Slippage)
		}
	}
}

func TestDepthLoop(t *testing.T) {
	api, err := NewAPI(&common.Credentials{
		Key:    os.Getenv("BN_TEST_KEY"),
		Secret: os.Getenv("BN_TEST_SECRET"),
	}, "socks5://127.0.0.1:1080")
	if err != nil {
		t.Fatal(err)
	}
	depthCh := make(chan Depth, 100)
	go DepthLoop(context.Background(), "ADAUSDT", 1000, 0.1, api, time.Second*5, depthCh)
	for {
		select {
		case depth := <-depthCh:
			logger.Debugf("buy %f sell %f spread %f fee %f",
				depth.BuyPrice,
				depth.SellPrice,
				(depth.BuyPrice-depth.SellPrice)/depth.SellPrice,
				depth.BuySlippage+depth.SellSlippage+depth.BuyFee+depth.SellFee,
			)
		}
	}
}
