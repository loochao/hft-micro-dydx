package binance_usdtspot

import (
	"context"
	binance_busdspot "github.com/geometrybase/hft-micro/binance-busdspot"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"strings"
	"testing"
)

func TestAPI_QuerySubAccountList(t *testing.T) {
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
	logger.Debugf("%v", accounts)
}

func TestAPI_QuerySubAccountAssets(t *testing.T) {
	api, err := NewAPI(&common.Credentials{
		Key:    os.Getenv("BN_TEST_MASTER_KEY"),
		Secret: os.Getenv("BN_TEST_MASTER_SECRET"),
	}, os.Getenv("BN_TEST_MASTER_PROXY"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	balances, _, err := api.QuerySubAccountAssets(ctx, SubAccountParams{
		"fund5@vf2021.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range balances {
		if b.Free+b.Locked != 0 {
			logger.Debugf("%v", b)
		}
	}
}

func TestAPI_QuerySubAccountFuturesAccount(t *testing.T) {
	api, err := NewAPI(&common.Credentials{
		Key:    os.Getenv("BN_TEST_MASTER_KEY"),
		Secret: os.Getenv("BN_TEST_MASTER_SECRET"),
	}, os.Getenv("BN_TEST_MASTER_PROXY"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	account, _, err := api.QuerySubAccountFuturesAccount(ctx, SubAccountParams{
		"fund5@vf2021.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range account.Assets {
		logger.Debugf("%s %f", b.Asset, b.WalletBalance)
	}
}

func TestAPI_SubAccountUniversalTransfer(t *testing.T) {
	api, err := NewAPI(&common.Credentials{
		Key:    os.Getenv("BN_TEST_MASTER_KEY"),
		Secret: os.Getenv("BN_TEST_MASTER_SECRET"),
	}, os.Getenv("BN_TEST_MASTER_PROXY"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	account, _, err := api.QuerySubAccountFuturesAccount(ctx, SubAccountParams{
		"fund5@vf2021.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range account.Assets {
		//logger.Debugf("%s %f", b.Asset, b.WalletBalance)
		if b.WalletBalance != 0 {
			resp, _, err := api.SubAccountUniversalTransfer(ctx, SubAccountUniversalTransferParams{
				FromEmail:       "fund5@vf2021.com",
				ToEmail:         "fund12@vf2021.com",
				FromAccountType: SubAccountTypeUsdtFuture,
				ToAccountType:   SubAccountTypeSpot,
				Asset:           b.Asset,
				Amount:          b.WalletBalance,
			})
			if err != nil {
				t.Fatal(err)
			}
			logger.Debugf("%s %f transId: %d",  b.Asset, b.WalletBalance, resp.TranId)
		}
	}
}

func TestAPI_SpotFund6ToFund1(t *testing.T) {
	api, err := NewAPI(&common.Credentials{
		Key:    os.Getenv("BN_TEST_MASTER_KEY"),
		Secret: os.Getenv("BN_TEST_MASTER_SECRET"),
	}, os.Getenv("BN_TEST_MASTER_PROXY"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	account, _, err := api.QuerySubAccountAssets(ctx, SubAccountParams{
		"fund6@vf2021.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range account {
		//logger.Debugf("%s %f", b.Asset, b.Free)
		if b.Free != 0 {
			resp, _, err := api.SubAccountUniversalTransfer(ctx, SubAccountUniversalTransferParams{
				FromEmail:       "fund6@vf2021.com",
				ToEmail:         "fund1@vf2021.com",
				FromAccountType: SubAccountTypeSpot,
				ToAccountType:   SubAccountTypeSpot,
				Asset:           b.Asset,
				Amount:          b.Free,
			})
			if err != nil {
				t.Fatal(err)
			}
			logger.Debugf("%s %f transId: %d",  b.Asset, b.Free, resp.TranId)
		}
	}
}

func TestAPI_FutureFund6ToFund1(t *testing.T) {
	api, err := NewAPI(&common.Credentials{
		Key:    os.Getenv("BN_TEST_MASTER_KEY"),
		Secret: os.Getenv("BN_TEST_MASTER_SECRET"),
	}, os.Getenv("BN_TEST_MASTER_PROXY"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	account, _, err := api.QuerySubAccountFuturesAccount(ctx, SubAccountParams{
		"fund6@vf2021.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range account.Assets {
		if b.WalletBalance != 0 {
			resp, _, err := api.SubAccountUniversalTransfer(ctx, SubAccountUniversalTransferParams{
				FromEmail:       "fund6@vf2021.com",
				ToEmail:         "fund1@vf2021.com",
				FromAccountType: SubAccountTypeUsdtFuture,
				ToAccountType:   SubAccountTypeSpot,
				Asset:           b.Asset,
				Amount:          b.WalletBalance,
			})
			if err != nil {
				t.Fatal(err)
			}
			logger.Debugf("%s %f transId: %d",  b.Asset, b.WalletBalance, resp.TranId)
		}
	}
}

func TestAPI_SpotFund4ToFund10(t *testing.T) {
	api, err := NewAPI(&common.Credentials{
		Key:    os.Getenv("BN_TEST_MASTER_KEY"),
		Secret: os.Getenv("BN_TEST_MASTER_SECRET"),
	}, os.Getenv("BN_TEST_MASTER_PROXY"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	account, _, err := api.QuerySubAccountAssets(ctx, SubAccountParams{
		"fund4@vf2021.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range account {
		//logger.Debugf("%s %f", b.Asset, b.Free)
		if b.Free != 0 {
			resp, _, err := api.SubAccountUniversalTransfer(ctx, SubAccountUniversalTransferParams{
				FromEmail:       "fund4@vf2021.com",
				ToEmail:         "fund10@vf2021.com",
				FromAccountType: SubAccountTypeSpot,
				ToAccountType:   SubAccountTypeSpot,
				Asset:           b.Asset,
				Amount:          b.Free,
			})
			if err != nil {
				t.Fatal(err)
			}
			logger.Debugf("%s %f transId: %d",  b.Asset, b.Free, resp.TranId)
		}
	}
}

func TestAPI_SpotFund11ToFund10(t *testing.T) {
	api, err := NewAPI(&common.Credentials{
		Key:    os.Getenv("BN_TEST_MASTER_KEY"),
		Secret: os.Getenv("BN_TEST_MASTER_SECRET"),
	}, os.Getenv("BN_TEST_MASTER_PROXY"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	account, _, err := api.QuerySubAccountAssets(ctx, SubAccountParams{
		"fund11@vf2021.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range account {
		//logger.Debugf("%s %f", b.Asset, b.Free)
		if b.Free != 0 {
			resp, _, err := api.SubAccountUniversalTransfer(ctx, SubAccountUniversalTransferParams{
				FromEmail:       "fund11@vf2021.com",
				ToEmail:         "fund10@vf2021.com",
				FromAccountType: SubAccountTypeSpot,
				ToAccountType:   SubAccountTypeSpot,
				Asset:           b.Asset,
				Amount:          b.Free,
			})
			if err != nil {
				t.Fatal(err)
			}
			logger.Debugf("%s %f transId: %d",  b.Asset, b.Free, resp.TranId)
		}
	}
}



func TestAPI_SpotFund10(t *testing.T) {
	api, err := NewAPI(&common.Credentials{
		Key:    os.Getenv("BN_TEST_MASTER_KEY"),
		Secret: os.Getenv("BN_TEST_MASTER_SECRET"),
	}, os.Getenv("BN_TEST_MASTER_PROXY"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	account, _, err := api.QuerySubAccountAssets(ctx, SubAccountParams{
		"fund10@vf2021.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range account {
		if b.Free != 0 {
			if _, ok := binance_busdspot.TickSizes[b.Asset+"BUSD"]; !ok {
				logger.Debugf("%s", b.Asset)
			}
		}
	}
}

func TestAPI_QuerySubAccountFuturePositionRisk(t *testing.T) {
	api, err := NewAPI(&common.Credentials{
		Key:    os.Getenv("BN_TEST_MASTER_KEY"),
		Secret: os.Getenv("BN_TEST_MASTER_SECRET"),
	}, os.Getenv("BN_TEST_MASTER_PROXY"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	positionRisks, _, err := api.QuerySubAccountFuturePositionRisk(ctx, SubAccountParams{
		"fund10@vf2021.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range positionRisks {
		if b.PositionAmount != 0 {
			if _, ok := binance_busdspot.TickSizes[strings.Replace(b.Symbol, "USDT", "BUSD", -1)]; !ok {
				logger.Debugf("%s", b.Symbol)
			}
		}
	}
}

func TestAPI_QuerySubAccountFuturePositionRisk2(t *testing.T) {
	api, err := NewAPI(&common.Credentials{
		Key:    os.Getenv("BN_TEST_MASTER_KEY"),
		Secret: os.Getenv("BN_TEST_MASTER_SECRET"),
	}, os.Getenv("BN_TEST_MASTER_PROXY"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()


	account, _, err := api.QuerySubAccountAssets(ctx, SubAccountParams{
		"fund10@vf2021.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	spot := make(map[string]float64)
	for _, b := range account {
		if b.Free != 0 {
			spot[b.Asset+"BUSD"] = b.Free
		}
	}


	positionRisks, _, err := api.QuerySubAccountFuturePositionRisk(ctx, SubAccountParams{
		"fund10@vf2021.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range positionRisks {
		if b.PositionAmount != 0 {
			if _, ok := spot[strings.Replace(b.Symbol, "USDT", "BUSD", -1)]; !ok {
				logger.Debugf("%s", b.Symbol)
			//}else if size != -b.PositionAmount {
			//	logger.Debugf("%s %f %f", b.Symbol, size, -b.PositionAmount)
			}
		}
	}
}
