package okexv5_usdtspot

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"testing"
	"time"
)

func TestAPI_GetAccounts(t *testing.T) {
	api, err := NewAPI(&Credentials{
		Key: os.Getenv("OK_KEY"),
		Secret: os.Getenv("OK_SECRET"),
		Passphrase: os.Getenv("OK_PASSPHRASE"),
	}, os.Getenv("OK_PROXY"))
	if err != nil {
		t.Fatal(err)
	}
	balances, err := api.GetAccount(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", balances)
}

func TestAPI_GetBalances(t *testing.T) {
	api, err := NewAPI(&Credentials{
		Key: os.Getenv("OK_KEY"),
		Secret: os.Getenv("OK_SECRET"),
		Passphrase: os.Getenv("OK_PASSPHRASE"),
	}, os.Getenv("OK_PROXY"))
	if err != nil {
		t.Fatal(err)
	}
	balances, err := api.GetBalances(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", balances)
}

func TestAPI_SubmitOrder(t *testing.T) {
	api, err := NewAPI(&Credentials{
		Key: os.Getenv("OK_KEY"),
		Secret: os.Getenv("OK_SECRET"),
		Passphrase: os.Getenv("OK_PASSPHRASE"),
	}, os.Getenv("OK_PROXY"))
	if err != nil {
		t.Fatal(err)
	}
	price := new(string)
	*price = common.FormatFloat(32, TickPrecisions["ATOM-USDT"])
	orderResp, err := api.SubmitOrder(context.Background(), NewOrderParam{
		InstId: "ATOM-USDT",
		OrderType: OrderTypePostOnly,
		Side: OrderSideBuy,
		Size: common.FormatFloat(0.5, StepPrecisions["ATOM-USDT"]),
		Price: price,
	})
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", orderResp)
	time.Sleep(time.Second*15)
	cancelResp, err := api.CancelOrders(context.Background(), CancelOrderParam{
		InstId: "ATOM-USDT",
		OrdId: orderResp.OrdId,
	})
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("cancel resp %v", cancelResp)
}

