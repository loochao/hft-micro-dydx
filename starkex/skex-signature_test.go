package starkex

import (
	"github.com/geometrybase/hft-micro/logger"
	"github.com/stretchr/testify/assert"
	"math/big"
	"os"
	"testing"
	"time"
)

func TestSignOrderA(t *testing.T) {
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
	assert.Equal(
		t,
		"069f9b9c9e2796c0fcb211ad3553dcb1a6f31f3f814f448257a8f9daeaadb0b902fea7fb4b0bd1f9516cada08d7eb12258fc1a544534c0551c6192b6407faee1",
		sg,
	)
	logger.Debugf("\nGO_RESULT %s", sg)
	logger.Debugf("\nST_RESULT %s", "04d2e19755fb95dfa35055f1352e49c521cb0804be464e7f1b3b4015c66f9722032ccd2ca7f8b51a8057bb300589c5f5fb15b40c14665a852fc03109c7993c3c")
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
	assert.Equal(
		t,
		"053ee3f3d19e14a2e4174e24df3bb3fd7e790f7204675f7b542c1b6f1344b2da0767cc9cde531ab8112be914e6d5c505071b4497d379d4ae7eb3d3e696dc1648",
		sg,
	)
	logger.Debugf("\nST_RESULT %s", "05f8eee0d4152e9c52cee4d36f0afbba40314f45a0a1bcd4893bd0c1faabea39053b8763bbd974fd1c7d79aee773263e667b3084f72fb4e957accc38333d070d")
	logger.Debugf("\nPY_RESULT %s", "053ee3f3d19e14a2e4174e24df3bb3fd7e790f7204675f7b542c1b6f1344b2da0767cc9cde531ab8112be914e6d5c505071b4497d379d4ae7eb3d3e696dc1648")
	logger.Debugf("\nGO_RESULT %s", sg)
}


//{"market":"YFI-USD","side":"BUY","type":"LIMIT","timeInForce":"IOC","size":"0.0040","price":"22010","limitFee":"0.0015","expiration":"2021-12-12T17:41:35.714Z","postOnly":false,"clientId":"16392444950608","signature":"019d958190576e29f4298de36d369f08f6a57ed62a8c50cac894b6ef9af641fc04c28ab70c1029996a1a79fe9478f87164356a4bcbc29c3393ed14c69603fc57"}
func TestSignOrderC(t *testing.T) {
	tt, err := time.Parse("2006-01-02T15:04:05.999Z", "2021-12-12T17:41:35.714Z")
	if err != nil {
		t.Fatal(err)
	}
	so, err := NewStarkwareOrder(
		NETWORK_ID_MAINNET,
		"YFI-USD",
		"BUY",
		132352,
		0.004,
		22010,
		0.0015,
		"16392444950608",
		tt.Unix(),
	)
	if err != nil {
		t.Fatal(err)
	}
	pk, _ := new(big.Int).SetString(os.Getenv("DYDX_VC_STARK_PRIVATE_KEY"), 16)
	sg, err := so.Sign(pk)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(
		t,
		"019d958190576e29f4298de36d369f08f6a57ed62a8c50cac894b6ef9af641fc04c28ab70c1029996a1a79fe9478f87164356a4bcbc29c3393ed14c69603fc57",
		sg,
	)
}


//FUND12 D 2021/12/11 17:47:55.214013 dduf-api.go:81: 	{"market":"DOGE-USD","side":"SELL","type":"LIMIT","timeInForce":"IOC","size":"4280","price":"0.1682","limitFee":"0.0015","expiration":"2021-12-12T17:47:55.204Z","postOnly":false,"clientId":"16392448753551","signature":"028b7b6c6a84cc3f161788e8e35694aa638bdf8d0f8ca77dd8fa650ef573b7530149e591d2db8af2a46004ef46616f7a6c5b980ae9b4a933e638969699ce2204"}
func TestSignOrderD(t *testing.T) {
	tt, err := time.Parse("2006-01-02T15:04:05.999Z", "2021-12-12T17:47:55.204Z")
	if err != nil {
		t.Fatal(err)
	}
	so, err := NewStarkwareOrder(
		NETWORK_ID_MAINNET,
		"DOGE-USD",
		"SELL",
		132352,
		4280,
		0.1682,
		0.0015,
		"16392448753551",
		tt.Unix(),
	)
	if err != nil {
		t.Fatal(err)
	}
	pk, _ := new(big.Int).SetString(os.Getenv("DYDX_VC_STARK_PRIVATE_KEY"), 16)
	sg, err := so.Sign(pk)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(
		t,
		"028b7b6c6a84cc3f161788e8e35694aa638bdf8d0f8ca77dd8fa650ef573b7530149e591d2db8af2a46004ef46616f7a6c5b980ae9b4a933e638969699ce2204",
		sg,
	)
}
