package okexv5_usdtswap

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
		Key:        os.Getenv("OK_KEY"),
		Secret:     os.Getenv("OK_SECRET"),
		Passphrase: os.Getenv("OK_PASSPHRASE"),
	}, os.Getenv("OK_PROXY"))
	if err != nil {
		t.Fatal(err)
	}
	account, err := api.GetAccount(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", account)
}

func TestAPI_GetPositions(t *testing.T) {
	api, err := NewAPI(&Credentials{
		Key:        os.Getenv("OK_KEY"),
		Secret:     os.Getenv("OK_SECRET"),
		Passphrase: os.Getenv("OK_PASSPHRASE"),
	}, os.Getenv("OK_PROXY"))
	if err != nil {
		t.Fatal(err)
	}
	positions, err := api.GetPositions(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", positions)
}

func TestAPI_UpdatePositionMode(t *testing.T) {
	api, err := NewAPI(&Credentials{
		Key:        os.Getenv("OK_KEY"),
		Secret:     os.Getenv("OK_SECRET"),
		Passphrase: os.Getenv("OK_PASSPHRASE"),
	}, os.Getenv("OK_PROXY"))
	if err != nil {
		t.Fatal(err)
	}
	err = api.UpdatePositionMode(context.Background())
	if err != nil {
		t.Fatal(err)
	}
}

func TestAPI_UpdateLeverage(t *testing.T) {
	api, err := NewAPI(&Credentials{
		Key:        os.Getenv("OK_KEY"),
		Secret:     os.Getenv("OK_SECRET"),
		Passphrase: os.Getenv("OK_PASSPHRASE"),
	}, os.Getenv("OK_PROXY"))
	if err != nil {
		t.Fatal(err)
	}
	err = api.UpdateLeverage(context.Background(), Leverage{
		InstId: "ETH-USDT-SWAP",
		Lever: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestAPI_SubmitOrder(t *testing.T) {
	api, err := NewAPI(&Credentials{
		Key:        os.Getenv("OK_KEY"),
		Secret:     os.Getenv("OK_SECRET"),
		Passphrase: os.Getenv("OK_PASSPHRASE"),
	}, os.Getenv("OK_PROXY"))
	if err != nil {
		t.Fatal(err)
	}
	price := new(string)
	*price = common.FormatFloat(30, TickPrecisions["ATOM-USDT-SWAP"])
	orderResp, err := api.SubmitOrder(context.Background(), NewOrderParam{
		InstId:    "ATOM-USDT-SWAP",
		OrderType: OrderTypePostOnly,
		Side:      OrderSideBuy,
		Size:      common.FormatFloat(1, StepPrecisions["ATOM-USDT-SWAP"]),
		Price:     price,
	})
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", orderResp)
	time.Sleep(time.Second * 30)
	cancelResp, err := api.CancelOrders(context.Background(), CancelOrderParam{
		InstId: "ATOM-USDT-SWAP",
		OrdId:  orderResp.OrdId,
	})
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("cancel resp %v", cancelResp)
}
