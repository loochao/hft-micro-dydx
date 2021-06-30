package okex_usdtspot

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDepth5(t *testing.T) {
	str := []byte(`{"table":"spot/depth5","data":[{"asks":[["2252.95","0.625033","2"],["2252.96","3.508481","3"],["2252.97","3.105584","1"],["2253.49","4","1"],["2253.96","0.001","1"]],"bids":[["2252.94","4.281","4"],["2252.35","10","1"],["2252.19","0.444012","1"],["2252.06","5.25573","1"],["2251.99","0.689994","1"]],"instrument_id":"ETH-USDT","timestamp":"2021-04-25T09:54:52.237Z"}]}`)
	d5 := Depth5{}
	err := json.Unmarshal(str, &d5)
	if err != nil {
		t.Fatal(err)
	}
	d6 := &Depth5{}
	err = ParseDepth5(str, d6)
	if err != nil {
		t.Fatal(err)
	}
	for i := range d6.Bids {
		assert.Equal(t, d5.Bids[i][0], d6.Bids[i][0])
		assert.Equal(t, d5.Bids[i][1], d6.Bids[i][1])
	}
	for i := range d6.Asks {
		assert.Equal(t, d5.Asks[i][0], d6.Asks[i][0])
		assert.Equal(t, d5.Asks[i][1], d6.Asks[i][1])
	}
}


