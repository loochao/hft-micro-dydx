package binance_usdtspot

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"testing"
)

func TestAPI_GetAccount3(t *testing.T) {
	api, err := NewAPI(&common.Credentials{
		Key:    os.Getenv("BN_LC_KEY"),
		Secret: os.Getenv("BN_LC_SECRET"),
	}, os.Getenv("BN_LC_PROXY"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	account, _, err := api.GetAccount(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range account.Balances {
		if b.Free != 0 {
			logger.Debugf("%s %f", b.Asset, b.Free+b.Locked)
			if b.Asset == "USDT" {
				logger.Debugf("%s %f", b.Asset, b.Free)
				resp, _, err :=api.NewFutureAccountTransfer(ctx, FutureAccountTransferParams{
					Asset:  "USDT",
					Type:   TransferSpotToUSDTFuture,
					Amount: b.Free,
				})
				if err != nil {
					t.Fatal(err)
				}
				logger.Debugf("%v", resp)
			}
		}
	}
}
