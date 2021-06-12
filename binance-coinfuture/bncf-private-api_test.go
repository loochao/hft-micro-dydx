package binance_coinfuture

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestAPI_ChangePositionMode(t *testing.T) {
	api, err := NewAPI(&common.Credentials{
		Key:    os.Getenv("BN_TEST_KEY"),
		Secret: os.Getenv("BN_TEST_SECRET"),
	}, os.Getenv("BN_TEST_PROXY"))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := api.ChangePositionMode(context.Background(), ChangePositionModeParam{DualSidePosition: false})
	if err != nil {
		assert.Equal(t, "No need to change position side.", err.Error())
		assert.Equal(t, (*Response)(nil), resp)
	}
	if resp != nil {
		assert.Equal(t, 200, resp.Code)
		assert.Equal(t, "success", resp.Msg)
	}
	mode, err := api.GetPositionMode(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, false, mode.DualSidePosition)
}

func TestAPI_GetPositions(t *testing.T) {
	api, err := NewAPI(&common.Credentials{
		Key:    os.Getenv("BN_TEST_KEY"),
		Secret: os.Getenv("BN_TEST_SECRET"),
	}, os.Getenv("BN_TEST_PROXY"))
	if err != nil {
		t.Fatal(err)
	}
	positions, err := api.GetPositions(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", positions)
}

func TestAPI_ChangeLeverage(t *testing.T) {
	api, err := NewAPI(&common.Credentials{
		Key:    os.Getenv("BN_TEST_KEY"),
		Secret: os.Getenv("BN_TEST_SECRET"),
	}, os.Getenv("BN_TEST_PROXY"))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := api.ChangeLeverage(context.Background(), LeverageParams{
		Symbol:   "BNBUSD_PERP",
		Leverage: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, int64(10), resp.Leverage)
	assert.Equal(t, "BNBUSD_PERP", resp.Symbol)
}

func TestAPI_ChangeMarginType(t *testing.T) {
	api, err := NewAPI(&common.Credentials{
		Key:    os.Getenv("BN_TEST_KEY"),
		Secret: os.Getenv("BN_TEST_SECRET"),
	}, os.Getenv("BN_TEST_PROXY"))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := api.ChangeMarginType(context.Background(), MarginTypeParams{
		Symbol:     "BNBUSD_PERP",
		MarginType: MarginTypeIsolated,
	})
	if err != nil {
		assert.Equal(t, "No need to change margin type.", err.Error())
		assert.Equal(t, (*Response)(nil), resp)
	}
	if resp != nil {
		assert.Equal(t, 200, resp.Code)
		assert.Equal(t, "success", resp.Msg)
	}
}

func TestAPI_GetAccount(t *testing.T) {
	api, err := NewAPI(&common.Credentials{
		Key:    os.Getenv("BN_TEST_KEY"),
		Secret: os.Getenv("BN_TEST_SECRET"),
	}, os.Getenv("BN_TEST_PROXY"))
	if err != nil {
		t.Fatal(err)
	}
	account, err := api.GetAccount(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, pos := range account.Positions {
		if pos.Symbol == "BNBUSD_PERP" {
			logger.Debugf("%v", pos)
		}
	}
}
