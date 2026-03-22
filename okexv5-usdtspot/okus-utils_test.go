package okexv5_usdtspot

import (
	"encoding/json"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestDepth5(t *testing.T) {
	str := []byte(`{"arg":{"channel":"books5","instId":"DOGE-USDT"},"data":[{"asks":[["0.257659","0.000011","0","1"],["0.257744","1380","0","1"],["0.257749","300","0","1"],["0.257769","1942.701948","0","1"],["0.25777","1000","0","1"]],"bids":[["0.257634","949.769316","0","1"],["0.257633","1380","0","1"],["0.257627","20250","0","1"],["0.25762","2929.149403","0","1"],["0.257614","5350","0","1"]],"instId":"DOGE-USDT","ts":"1636741161692"}]}`)

	djson := Depth5{}
	err := json.Unmarshal(str, &djson)
	if err != nil {
		t.Fatal(err)
	}
	dparse := &Depth5{}
	err = ParseDepth5(str, dparse)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, djson.InstId, dparse.InstId)
	logger.Debugf("%v", dparse)
	for i := range dparse.Bids {
		assert.Equal(t, djson.Bids[i][0], dparse.Bids[i][0])
		assert.Equal(t, djson.Bids[i][1], dparse.Bids[i][1])
	}
	for i := range dparse.Asks {
		assert.Equal(t, djson.Asks[i][0], dparse.Asks[i][0])
		assert.Equal(t, djson.Asks[i][1], dparse.Asks[i][1])
	}
}

func TestParseTicker(t *testing.T) {
	msg := []byte(`{"arg":{"channel":"tickers","instId":"DOGE-USDT"},"data":[{"instType":"SPOT","instId":"DOGE-USDT","last":"0.254381","lastSz":"600","askPx":"0.254381","askSz":"1400","bidPx":"0.25438","bidSz":"400","open24h":"0.263668","high24h":"0.268614","low24h":"0.248601","sodUtc0":"0.260658","sodUtc8":"0.253989","volCcy24h":"125310776.54685","vol24h":"486148293.462458","ts":"1636737706397"}]}`)
	ticker := Ticker{}
	comCap := CommonCapture{}
	err := json.Unmarshal(msg, &comCap)
	if err != nil {
		t.Fatal(err)
	}
	tickers := make([]Ticker, 0)
	err = json.Unmarshal(comCap.Data, &tickers)
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, tickers, 1)
	err = ParseTicker(msg, &ticker)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, tickers[0].InstId, ticker.InstId)
	assert.Equal(t, tickers[0].BidSz, ticker.BidSz)
	assert.Equal(t, tickers[0].AskSz, ticker.AskSz)
	assert.Equal(t, tickers[0].BidPx, ticker.BidPx)
	assert.Equal(t, tickers[0].AskPx, ticker.AskPx)
	assert.Equal(t, tickers[0].EventTime.Sub(ticker.EventTime), time.Duration(0))
}

func TestParseTrade(t *testing.T) {
	msg := []byte(`{"arg":{"channel":"trades","instId":"DOGE-USDT"},"data":[{"instId":"DOGE-USDT","tradeId":"106645495","px":"0.256222","sz":"14.19554","side":"buy","ts":"1636778780284"}]}`)

	trade := Trade{}
	comCap := CommonCapture{}
	err := json.Unmarshal(msg, &comCap)
	if err != nil {
		t.Fatal(err)
	}
	trades := make([]Trade, 0)
	err = json.Unmarshal(comCap.Data, &trades)
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, trades, 1)
	err = ParseTrade(msg, &trade)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, trades[0].InstId, trade.InstId)
	assert.Equal(t, trades[0].Px, trade.Px)
	assert.Equal(t, trades[0].Sz, trade.Sz)
	assert.Equal(t, trades[0].Side, trade.Side)
	assert.Equal(t, trades[0].TS.Sub(trade.TS), time.Duration(0))
}


func BenchmarkParseTicker(b *testing.B) {
	msg := []byte(`{"arg":{"channel":"tickers","instId":"DOGE-USDT"},"data":[{"instType":"SPOT","instId":"DOGE-USDT","last":"0.254381","lastSz":"600","askPx":"0.254381","askSz":"1400","bidPx":"0.25438","bidSz":"400","open24h":"0.263668","high24h":"0.268614","low24h":"0.248601","sodUtc0":"0.260658","sodUtc8":"0.253989","volCcy24h":"125310776.54685","vol24h":"486148293.462458","ts":"1636737706397"}]}`)
	ticker := Ticker{}
	b.ReportAllocs()
	for i := 0; i < b.N; i ++ {
		_ = ParseTicker(msg, &ticker)
	}
}

func BenchmarkParseDepth5(b *testing.B) {
	msg := []byte(`{"arg":{"channel":"books5","instId":"DOGE-USDT"},"data":[{"asks":[["0.257659","0.000011","0","1"],["0.257744","1380","0","1"],["0.257749","300","0","1"],["0.257769","1942.701948","0","1"],["0.25777","1000","0","1"]],"bids":[["0.257634","949.769316","0","1"],["0.257633","1380","0","1"],["0.257627","20250","0","1"],["0.25762","2929.149403","0","1"],["0.257614","5350","0","1"]],"instId":"DOGE-USDT","ts":"1636741161692"}]}`)
	ticker := Depth5{}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i ++ {
		_ = ParseDepth5(msg, &ticker)
	}
}

func BenchmarkParseTrade(b *testing.B) {
	msg := []byte(`{"arg":{"channel":"trades","instId":"DOGE-USDT"},"data":[{"instId":"DOGE-USDT","tradeId":"106645495","px":"0.256222","sz":"14.19554","side":"buy","ts":"1636778780284"}]}`)
	ticker := &Trade{}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i ++ {
		_ = ParseTrade(msg, ticker)
	}
}
