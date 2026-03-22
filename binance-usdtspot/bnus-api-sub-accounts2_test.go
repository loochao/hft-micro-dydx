package binance_usdtspot

import (
	"context"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"testing"
)

func TestAPI_GetAccount2(t *testing.T) {
	api, err := NewAPI(&common.Credentials{
		Key:    os.Getenv("BN_TEST_MASTER_KEY"),
		Secret: os.Getenv("BN_TEST_MASTER_SECRET"),
	}, os.Getenv("BN_TEST_MASTER_PROXY"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	account, _, err := api.GetAccount(ctx)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("\n\nMAIN ACCOUNT ASSETS:\n")
	for _, b := range account.Balances {
		if b.Free != 0 {
			fmt.Printf("%6s: %f\n", b.Asset, b.Free)
		}
	}
	fmt.Printf("\n\n")
}

func TestAPI_QuerySubAccountAssets2(t *testing.T) {
	//api, err := NewAPI(&common.Credentials{
	//	Key:    os.Getenv("BN_TEST_MASTER_KEY"),
	//	Secret: os.Getenv("BN_TEST_MASTER_SECRET"),
	//}, os.Getenv("BN_TEST_MASTER_PROXY"))
	api, err := NewAPI(&common.Credentials{
		Key:    os.Getenv("BN_TEST_MASTER_KEY"),
		Secret: os.Getenv("BN_TEST_MASTER_SECRET"),
	}, os.Getenv("BN_TEST_MASTER_PROXY"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	email := "fund27@vf2021.com"
	balances, _, err := api.QuerySubAccountAssets(ctx, SubAccountParams{
		Email: email,
	})
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("%s\n", email)
	for _, b := range balances {
		if b.Free+b.Locked != 0 {
			fmt.Printf("%6s: %.10f\n", b.Asset, b.Free+b.Locked)
		}
	}
}

func TestAPI_SubAccountUniversalTransfer2(t *testing.T) {
	api, err := NewAPI(&common.Credentials{
		Key:    os.Getenv("BN_TEST_MASTER_KEY"),
		Secret: os.Getenv("BN_TEST_MASTER_SECRET"),
	}, os.Getenv("BN_TEST_MASTER_PROXY"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	email := "fund15@vf2021.com"
	balances, _, err := api.QuerySubAccountAssets(ctx, SubAccountParams{
		email,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range balances {
		if b.Free != 0 {
			logger.Debugf("%s %f", b.Asset, b.Free)
			resp, _, err := api.SubAccountUniversalTransfer(ctx, SubAccountUniversalTransferParams{
				FromEmail:       email,
				ToEmail:         "visioncapital2021@protonmail.com",
				FromAccountType: SubAccountTypeSpot,
				ToAccountType:   SubAccountTypeSpot,
				Asset:           b.Asset,
				Amount:          b.Free,
			})
			if err != nil {
				t.Fatal(err)
			}
			logger.Debugf("%s %f transId: %d", b.Asset, b.Free, resp.TranId)
		}
	}
}

func TestAPI_QuerySubAccountAssets3(t *testing.T) {
	api, err := NewAPI(&common.Credentials{
		Key:    os.Getenv("BN_TEST_MASTER_KEY"),
		Secret: os.Getenv("BN_TEST_MASTER_SECRET"),
	}, os.Getenv("BN_TEST_MASTER_PROXY"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	accounts, _, err := api.QuerySubAccountList(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range accounts {
		email := a.Email
		balances, _, err := api.QuerySubAccountAssets(ctx, SubAccountParams{
			Email: email,
		})
		if err != nil {
			t.Fatal(err)
		}
		fmt.Printf("\n\n%s\n", email)
		for _, b := range balances {
			if b.Free+b.Locked != 0 {
				fmt.Printf("%6s: %f\n", b.Asset, b.Free+b.Locked)
			}
		}
	}
}

func TestAPI_SpotMainToSubSpot2(t *testing.T) {
	api, err := NewAPI(&common.Credentials{
		Key:    os.Getenv("BN_TEST_MASTER_KEY"),
		Secret: os.Getenv("BN_TEST_MASTER_SECRET"),
	}, os.Getenv("BN_TEST_MASTER_PROXY"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	resp, _, err := api.SubAccountUniversalTransfer(ctx, SubAccountUniversalTransferParams{
		//FromEmail:       "bd@visioncap.io",
		FromEmail:       "fund32@vf2021.com",
		ToEmail:         "bd@visioncap.io",
		FromAccountType: SubAccountTypeSpot,
		ToAccountType:   SubAccountTypeSpot,
		Asset:           "USDC",
		Amount:          29981,
	})
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("transId: %d", resp.TranId)
}
