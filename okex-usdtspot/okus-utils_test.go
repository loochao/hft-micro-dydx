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

func TestParseTicker(t *testing.T) {
	msg := []byte(`{"table":"spot/ticker","data":[{"last":"16.486","open_24h":"16.206","best_bid":"16.475","high_24h":"16.619","low_24h":"15.945","open_utc0":"16.173","open_utc8":"16.288","base_volume_24h":"453392.90975651","quote_volume_24h":"7406915.38374752","best_ask":"16.496","instrument_id":"WAVES-USDT","timestamp":"2021-07-07T14:20:36.555Z","best_bid_size":"203.32785582","best_ask_size":"3.08008475","last_qty":"2.31531004"}]}`)
	ticker := Ticker{}
	tickerData := TickerData{}
	err := json.Unmarshal(msg, &tickerData)
	if err != nil {
		t.Fatal(err)
	}
	err = ParseTicker(msg, &ticker)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 1, len(tickerData.Data))
	assert.Equal(t, tickerData.Data[0].InstrumentID, ticker.InstrumentID)
	assert.Equal(t, tickerData.Data[0].BestBid, ticker.BestBid)
	assert.Equal(t, tickerData.Data[0].BestBidSize, ticker.BestBidSize)
	assert.Equal(t, tickerData.Data[0].BestAsk, ticker.BestAsk)
	assert.Equal(t, tickerData.Data[0].BestAskSize, ticker.BestAskSize)
	assert.Equal(t, tickerData.Data[0].Timestamp.UnixNano(), ticker.Timestamp.UnixNano())
}


func BenchmarkParseTicker(b *testing.B) {
	msg := []byte(`{"table":"spot/ticker","data":[{"last":"16.486","open_24h":"16.206","best_bid":"16.475","high_24h":"16.619","low_24h":"15.945","open_utc0":"16.173","open_utc8":"16.288","base_volume_24h":"453392.90975651","quote_volume_24h":"7406915.38374752","best_ask":"16.496","instrument_id":"WAVES-USDT","timestamp":"2021-07-07T14:20:36.555Z","best_bid_size":"203.32785582","best_ask_size":"3.08008475","last_qty":"2.31531004"}]}`)
	ticker := Ticker{}
	b.ReportAllocs()
	for i := 0; i < b.N; i ++ {
		_ = ParseTicker(msg, &ticker)
	}
}