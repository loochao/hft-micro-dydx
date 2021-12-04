package starkex_test

import (
	"encoding/json"
	"fmt"
	"github.com/geometrybase/hft-micro/starkex"
	"math"
	"math/big"
	"testing"
	"time"
)

const (
	MOCK_PUBLIC_KEY        = "3b865a18323b8d147a12c556bfb1d502516c325b1477a23ba6c77af31f020fd"
	MOCK_PRIVATE_KEY       = "58c7d5a90b1776bde86ebac077e053ed85b0f7164f53b080304a531947f46e3"
	MOCK_SIGNATURE         = "00cecbe513ecdbf782cd02b2a5efb03e58d5f63d15f2b840e9bc0029af04e8dd0090b822b16f50b2120e4ea9852b340f7936ff6069d02acca02f2ed03029ace5"
	MOCK_PUBLIC_KEY_EVEN_Y = "5c749cd4c44bdc730bc90af9bfbdede9deb2c1c96c05806ce1bc1cb4fed64f7"
	MOCK_SIGNATURE_EVEN_Y  = "00fc0756522d78bef51f70e3981dc4d1e82273f59cdac6bc31c5776baabae6ec0158963bfd45d88a99fb2d6d72c9bbcf90b24c3c0ef2394ad8d05f9d3983443a"
)

type OrderParams struct {
	NetworkId              int     `json:"network_id"`
	Market                 string  `json:"market"`
	Side                   string  `json:"side"`
	PositionID             int64   `json:"position_id"`
	HumanSize              float64 `json:"human_size"`
	HumanPrice             float64 `json:"human_price"`
	LimitFee               float64 `json:"limit_fee"`
	ClientID               string  `json:"client_id"`
	ExpirationEpochSeconds int64   `json:"expiration_epoch_seconds"`
}

var ORDER_PARAMS = OrderParams{
	NetworkId:  starkex.NETWORK_ID_ROPSTEN,
	Market:     starkex.MARKET_ETH_USD,
	Side:       starkex.ORDER_SIDE_BUY,
	PositionID: 12345,
	HumanSize:  145.0005,
	HumanPrice: 350.00067,
	LimitFee:   0.125,
	ClientID:   "This is an ID that the client came up with to describe this order",
}

func TestSignOrder(t *testing.T) {
	tt, err := time.Parse("2006-01-02T15:04:05.999Z", "2020-09-17T04:15:55.028Z")
	if err != nil {
		t.Fatal(err)
	}
	ORDER_PARAMS.ExpirationEpochSeconds = tt.Unix()
	data, err := json.Marshal(ORDER_PARAMS)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("%s\n", data)

	syntheticAsset := starkex.SYNTHETIC_ASSET_MAP[ORDER_PARAMS.Market]
	order := starkex.StarkwareOrder{}
	order.OrderType = "LIMIT_ORDER_WITH_FEES"
	order.AssetIdSynthetic = starkex.SYNTHETIC_ASSET_ID_MAP[syntheticAsset]
	order.AssetIdCollateral = starkex.COLLATERAL_ASSET_ID_BY_NETWORK_ID[ORDER_PARAMS.NetworkId]
	order.AssetIdFee = starkex.SYNTHETIC_ASSET_ID_MAP[syntheticAsset]
	order.PositionId = ORDER_PARAMS.PositionID
	order.IsBuyingSynthetic = ORDER_PARAMS.Side == starkex.ORDER_SIDE_BUY
	order.QuantumsAmountSynthetic, err = starkex.ToQuantumsExact(ORDER_PARAMS.HumanPrice, syntheticAsset)
	if err != nil {
		t.Fatal(err)
	}
	if order.IsBuyingSynthetic {
		humanCost := math.Ceil(ORDER_PARAMS.HumanPrice * ORDER_PARAMS.HumanPrice)
		order.QuantumsAmountCollateral = starkex.ToQuantumsRoundUp(humanCost, syntheticAsset)
	} else {
		humanCost := math.Floor(ORDER_PARAMS.HumanPrice * ORDER_PARAMS.HumanPrice)
		order.QuantumsAmountCollateral = starkex.ToQuantumsRoundDown(humanCost, syntheticAsset)
	}

	// The limitFee is a fraction, e.g. 0.01 is a 1 % fee.
	// It is always paid in the collateral asset.
	// Constrain the limit fee to six decimals of precision.
	// The final fee amount must be rounded up.
	limitFeeRounded := math.Floor(ORDER_PARAMS.LimitFee/1000000) * 1000000
	order.QuantumsAmountFee = int64(math.Ceil(limitFeeRounded * float64(order.QuantumsAmountCollateral)))

	// Orders may have a short time-to-live on the orderbook, but we need
	// to ensure their signatures are valid by the time they reach the
	// blockchain. Therefore, we enforce that the signed expiration includes
	// a buffer relative to the expiration timestamp sent to the dYdX API.
	order.ExpirationEpochHours = int(math.Ceil(
		float64(ORDER_PARAMS.ExpirationEpochSeconds) / float64(starkex.ONE_HOUR_IN_SECONDS),
	)) + starkex.ORDER_SIGNATURE_EXPIRATION_BUFFER_HOURS


	fmt.Printf("%v\n", order)

}

func TestGCD(t *testing.T) {
	x := big.NewInt(0)
	y := big.NewInt(0)
	c := big.NewInt(0)
	a := big.NewInt(125)
	b := big.NewInt(10000)
	c.GCD(x, y, a, b)

	fmt.Printf("%s*%s + %s*%s = %s\n", x, a, y, b, c)
}
