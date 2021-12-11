package starkex

import (
	"github.com/geometrybase/hft-micro/logger"
	"math/big"
	"os"
	"testing"
	"time"
)

func TestSignOrderA(t *testing.T) {
	//{"market":"LINK-USD","side":"SELL","type":"LIMIT","timeInForce":"IOC","size":"7.0","price":"19.139","limitFee":"0.0015","expiration":"2021-12-11T17:06:54.23Z","postOnly":false,"clientId":"16391560147954","sig      nature":"04d2e19755fb95dfa35055f1352e49c521cb0804be464e7f1b3b4015c66f9722032ccd2ca7f8b51a8057bb300589c5f5fb15b40c14665a852fc03109c7993c3c"}
	tt, err := time.Parse("2006-01-02T15:04:05.999Z", "2021-12-11T17:06:54.23Z")
	if err != nil {
		t.Fatal(err)
	}
	so, err := NewStarkwareOrder(
		NETWORK_ID_MAINNET,
		"LINK-USD",
		"SELL",
		119684,
		7.0,
		19.139,
		0.0015,
		"16391560147954",
		tt.Unix(),
	)
	if err != nil {
		t.Fatal(err)
	}
	pk, _ := new(big.Int).SetString(os.Getenv("DYDX_TEST_STARK_PRIVATE_KEY"), 16)
	sg, err := so.Sign(pk)
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("\nGO_RESULT %s", sg)
	logger.Debugf("\nGO_RESULT %s", "04d2e19755fb95dfa35055f1352e49c521cb0804be464e7f1b3b4015c66f9722032ccd2ca7f8b51a8057bb300589c5f5fb15b40c14665a852fc03109c7993c3c")
}


func TestSignOrderB(t *testing.T) {
	tt, err := time.Parse("2006-01-02T15:04:05.999Z", "2021-12-11T17:07:03.499Z")
	if err != nil {
		t.Fatal(err)
	}
	so, err := NewStarkwareOrder(
		NETWORK_ID_MAINNET,
		"MATIC-USD",
		"SELL",
		119684,
		578,
		2.167,
		0.0015,
		"16391560232443",
		tt.Unix(),
	)
	if err != nil {
		t.Fatal(err)
	}
	pk, _ := new(big.Int).SetString(os.Getenv("DYDX_TEST_STARK_PRIVATE_KEY"), 16)
	sg, err := so.Sign(pk)
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("\nST_RESULT %s", "05f8eee0d4152e9c52cee4d36f0afbba40314f45a0a1bcd4893bd0c1faabea39053b8763bbd974fd1c7d79aee773263e667b3084f72fb4e957accc38333d070d")
	logger.Debugf("\nPY_RESULT %s", "053ee3f3d19e14a2e4174e24df3bb3fd7e790f7204675f7b542c1b6f1344b2da0767cc9cde531ab8112be914e6d5c505071b4497d379d4ae7eb3d3e696dc1648")
	logger.Debugf("\nGO_RESULT %s", sg)
}
