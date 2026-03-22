package bitfinex_usdtfuture

import (
	"fmt"
	"strconv"
	"testing"
)

func TestParsePairs(t *testing.T) {
	msg := []byte(`[[["ADAF0:USTF0",[null,null,null,"0.001","250000.0",null,null,null,0.01,0.005]],["AMPF0:USTF0",[null,null,null,"2.0","100000.0",null,null,null,0.05,0.025]],["BTCDOMF0:USTF0",[null,null,null,"0.008","5000.0",null,null,null,0.01,0.005]],["BTCF0:USTF0",[null,null,null,"0.0002","100.0",null,null,null,0.01,0.005]],["DOGEF0:USTF0",[null,null,null,"0.001","500000.0",null,null,null,0.01,0.005]],["DOTF0:BTCF0",[null,null,null,"0.001","50000.0",null,null,null,0.01,0.005]],["DOTF0:USTF0",[null,null,null,"0.001","50000.0",null,null,null,0.01,0.005]],["EOSF0:USTF0",[null,null,null,"0.001","250000.0",null,null,null,0.01,0.005]],["ETHF0:BTCF0",[null,null,null,"0.001","100.0",null,null,null,0.01,0.005]],["ETHF0:USTF0",[null,null,null,"0.006","1000.0",null,null,null,0.01,0.005]],["EURF0:USTF0",[null,null,null,"0.8","250000.0",null,null,null,0.01,0.005]],["EUROPE50IXF0:USTF0",[null,null,null,"0.0006","1000.0",null,null,null,0.01,0.005]],["GBPF0:USTF0",[null,null,null,"0.8","250000.0",null,null,null,0.01,0.005]],["GERMANY30IXF0:USTF0",[null,null,null,"0.0002","1000.0",null,null,null,0.01,0.005]],["IOTF0:USTF0",[null,null,null,"0.001","250000.0",null,null,null,0.01,0.005]],["JPYF0:USTF0",[null,null,null,"106.0","10000000.0",null,null,null,0.01,0.005]],["LINKF0:USTF0",[null,null,null,"0.001","250000.0",null,null,null,0.01,0.005]],["LTCF0:BTCF0",[null,null,null,"0.001","7500.0",null,null,null,0.01,0.005]],["LTCF0:USTF0",[null,null,null,"0.001","250000.0",null,null,null,0.01,0.005]],["TESTBTCF0:TESTUSDTF0",[null,null,null,"0.0002","1000.0",null,null,null,0.01,0.005]],["UNIF0:USTF0",[null,null,null,"0.001","250000.0",null,null,null,0.01,0.005]],["XAGF0:USTF0",[null,null,null,"0.001","10000.0",null,null,null,0.01,0.005]],["XAUTF0:BTCF0",[null,null,null,"0.001","500.0",null,null,null,0.01,0.005]],["XAUTF0:USTF0",[null,null,null,"0.002","400.0",null,null,null,0.01,0.005]],["XLMF0:USTF0",[null,null,null,"0.001","250000.0",null,null,null,0.01,0.005]]]]`)
	pairs, err := ParsePairs(msg)
	if err != nil {
		t.Fatal(err)
	}
	minSizes := make(map[string]float64)
	for _, p := range pairs {
		minSizes[p.Symbol] = p.MinOrderSize
	}
	str := ""
	str += "var MinSizes = map[string]float64{\n"
	for symbol, value := range minSizes {
		str += fmt.Sprintf("  \"%s\": %s,\n", symbol, strconv.FormatFloat(value, 'f', -1, 64))
	}
	str += "}\n\n"
	fmt.Printf("%s", str)
}
