package binance_usdtfuture

import (
	"encoding/json"
	"fmt"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)


func TestParseBookTicker(t *testing.T) {
	str := []byte(`{"stream":"scusdt@bookTicker","data":{"e":"bookTicker","u":552297398961,"s":"SCUSDT","b":"0.012805","B":"46556","a":"0.012816","A":"90351","T":1624971386657,"E":1624971386662}}`)
	logger.Debugf("%s", str)
	wsCap := &WSCap{}
	err := json.Unmarshal(str, wsCap)
	if err != nil {
		t.Fatal(err)
	}
	bookTicker1 := &BookTicker{}
	bookTicker2 := &BookTicker{}
	err = json.Unmarshal(wsCap.Data, bookTicker1)
	if err != nil {
		t.Fatal(err)
	}
	err = ParseBookTicker(str, bookTicker2)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, bookTicker1.Symbol, bookTicker2.Symbol)
	assert.Equal(t, bookTicker1.BestBidQty, bookTicker2.BestBidQty)
	assert.Equal(t, bookTicker1.BestBidPrice, bookTicker2.BestBidPrice)
	assert.Equal(t, bookTicker1.BestAskQty, bookTicker2.BestAskQty)
	assert.Equal(t, bookTicker1.BestAskPrice, bookTicker2.BestAskPrice)
	assert.Equal(t, bookTicker1.EventTime.UnixNano(), bookTicker2.EventTime.UnixNano())
	logger.Debugf("%v", bookTicker2)
}

func TestParseDepth20(t *testing.T) {
	depth20Str := []byte(`{"stream":"btcusdt@depth20@100ms","data":{"e":"depthUpdate","E":1616509191577,"T":1616509191571,"s":"CDEFH1INCHUSDT","U":276060537661,"u":276060540084,"pu":276060537525,"b":[["55302.93","1.203"],["55302.33","1.052"],["55302.32","0.036"],["55301.31","0.048"],["55301.30","1.936"],["55299.12","0.036"],["55299.11","0.240"],["55299.06","2.851"],["55299.01","0.124"],["55299.00","1.337"],["55298.52","0.100"],["55298.51","0.008"],["55298.41","0.110"],["55297.71","0.278"],["55297.31","0.292"],["55297.28","0.542"],["55297.18","0.362"],["55295.75","0.136"],["55295.68","0.160"],["55294.81","0.278"]],"a":[["55302.94","0.116"],["55305.98","0.202"],["55306.33","0.001"],["55306.58","0.054"],["55309.34","0.074"],["55309.36","0.090"],["55309.37","0.098"],["55309.52","0.116"],["55309.99","0.033"],["55310.62","0.181"],["55310.72","0.020"],["55311.04","0.217"],["55311.21","0.090"],["55311.41","0.181"],["55311.58","0.180"],["55311.59","0.519"],["55311.76","0.100"],["55311.86","0.243"],["55312.02","0.247"],["55312.42","0.090"]]}}`)
	logger.Debugf("%d", len(`{"stream":"btcusdt@depth20@100ms","data":{"e":"depthUpdate","E":1`))
	logger.Debugf("%s", depth20Str[:77])
	//depth, err := ParseDepth20(depth20Str)
	//if err != nil {
	//	t.Fatal(err)
	//}
	//stream := Depth20Stream{}
	//err = json.Unmarshal(depth20Str, &stream)
	//if err != nil {
	//	t.Fatal(err)
	//}
	//assert.Equal(t, stream.Data.Market, depth.Market)
	//assert.Equal(t, stream.Data.LastUpdateId, depth.LastUpdateId)
	//for i := 0; i < 20; i++ {
	//	assert.Equal(t, stream.Data.Asks[i][0], depth.Asks[i][0])
	//	assert.Equal(t, stream.Data.Asks[i][1], depth.Asks[i][1])
	//	assert.Equal(t, stream.Data.Bids[i][0], depth.Bids[i][0])
	//	assert.Equal(t, stream.Data.Bids[i][1], depth.Bids[i][1])
	//}
}

func TestParseDepth5(t *testing.T) {

	strs := []string {
		`{"stream":"scusdt@depth5@100ms","data":{"e":"depthUpdate","E":1623494540877,"T":1623494540870,"s":"SCUSDT","U":510743908847,"u":510743911822,"pu":510743908726,"b":[["35701.24","2.079"],["35701.23","0.276"],["35701.22","0.001"],["35700.35","0.400"],["35699.59","0.147"]],"a":[["35701.25","0.134"],["35704.02","0.248"],["35704.03","0.272"],["35704.55","0.001"],["35704.56","0.003"]]}}`,
		`{"stream":"btcusdt@depth5@100ms","data":{"e":"depthUpdate","E":1623494540877,"T":1623494540870,"s":"BTCUSDT","U":510743908847,"u":510743911822,"pu":510743908726,"b":[["35701.24","2.079"],["35701.23","0.276"],["35701.22","0.001"],["35700.35","0.400"],["35699.59","0.147"]],"a":[["35701.25","0.134"],["35704.02","0.248"],["35704.03","0.272"],["35704.55","0.001"],["35704.56","0.003"]]}}`,
		`{"stream":"linkusdt@depth5@100ms","data":{"e":"depthUpdate","E":1623494540955,"T":1623494540947,"s":"LINKUSDT","U":510743911258,"u":510743914224,"pu":510743910356,"b":[["21.030","12.37"],["21.029","448.68"],["21.027","2.12"],["21.024","240.12"],["21.022","47.62"]],"a":[["21.031","4.66"],["21.034","20.68"],["21.036","7.17"],["21.038","20.53"],["21.039","251.82"]]}}`,
		`{"stream":"wavesusdt@depth5@100ms","data":{"e":"depthUpdate","E":1623494540937,"T":1623494540873,"s":"WAVESUSDT","U":510743910668,"u":510743911915,"pu":510743903045,"b":[["14.2300","0.4"],["14.2270","59.0"],["14.2260","112.0"],["14.2250","78.5"],["14.2240","195.9"]],"a":[["14.2310","11.0"],["14.2340","38.4"],["14.2350","105.0"],["14.2360","3.5"],["14.2370","193.0"]]}}`,

	}
	for _, str := range strs {
		depth := &Depth5{}
		err := ParseDepth5([]byte(str), depth)
		if err != nil {
			t.Fatal(err)
		}
		stream := Depth5Stream{}
		err = json.Unmarshal([]byte(str), &stream)
		if err != nil {
			t.Fatal(err)
		}
		logger.Debugf("%v", depth)
		assert.Equal(t, stream.Data.Symbol, depth.Symbol)
		assert.Equal(t, stream.Data.LastUpdateId, depth.LastUpdateId)
		for i := 0; i < 5; i++ {
			assert.Equal(t, stream.Data.Asks[i][0], depth.Asks[i][0], fmt.Sprintf("level %d ask price", i))
			assert.Equal(t, stream.Data.Asks[i][1], depth.Asks[i][1], fmt.Sprintf("level %d ask size", i))
			assert.Equal(t, stream.Data.Bids[i][0], depth.Bids[i][0], fmt.Sprintf("level %d bid price", i))
			assert.Equal(t, stream.Data.Bids[i][1], depth.Bids[i][1], fmt.Sprintf("level %d ask size", i))
		}
	}
}

func BenchmarkParseTrade(t *testing.B) {
	depth20Str := []byte(`{"stream":"btcusdt@aggTrade","data":{"e":"aggTrade","E":1616945754086,"a":405295371,"s":"BTCUSDT","p":"56183.31","q":"0.003","f":649066620,"l":649066620,"T":1616945753931,"m":false}}`)
	for n := 0; n < t.N; n++ {
		_, err := ParseTrade(depth20Str)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func BenchmarkParseDepth20(t *testing.B) {
	depth20Str := []byte(`{"stream":"btcusdt@depth20@100ms","data":{"e":"depthUpdate","E":1616509191577,"T":1616509191571,"s":"CDEFH1INCHUSDT","U":276060537661,"u":276060540084,"pu":276060537525,"b":[["55302.93","1.203"],["55302.33","1.052"],["55302.32","0.036"],["55301.31","0.048"],["55301.30","1.936"],["55299.12","0.036"],["55299.11","0.240"],["55299.06","2.851"],["55299.01","0.124"],["55299.00","1.337"],["55298.52","0.100"],["55298.51","0.008"],["55298.41","0.110"],["55297.71","0.278"],["55297.31","0.292"],["55297.28","0.542"],["55297.18","0.362"],["55295.75","0.136"],["55295.68","0.160"],["55294.81","0.278"]],"a":[["55302.94","0.116"],["55305.98","0.202"],["55306.33","0.001"],["55306.58","0.054"],["55309.34","0.074"],["55309.36","0.090"],["55309.37","0.098"],["55309.52","0.116"],["55309.99","0.033"],["55310.62","0.181"],["55310.72","0.020"],["55311.04","0.217"],["55311.21","0.090"],["55311.41","0.181"],["55311.58","0.180"],["55311.59","0.519"],["55311.76","0.100"],["55311.86","0.243"],["55312.02","0.247"],["55312.42","0.090"]]}}`)
	depth20 := &Depth20{}
	for n := 0; n < t.N; n++ {
		_ = ParseDepth20(depth20Str, depth20)
	}
}

func BenchmarkParseDepth20ByStdJson(t *testing.B) {
	depth20Str := []byte(`{"stream":"btcusdt@depth20@100ms","data":{"e":"depthUpdate","E":1616509191577,"T":1616509191571,"s":"CDEFH1INCHUSDT","U":276060537661,"u":276060540084,"pu":276060537525,"b":[["55302.93","1.203"],["55302.33","1.052"],["55302.32","0.036"],["55301.31","0.048"],["55301.30","1.936"],["55299.12","0.036"],["55299.11","0.240"],["55299.06","2.851"],["55299.01","0.124"],["55299.00","1.337"],["55298.52","0.100"],["55298.51","0.008"],["55298.41","0.110"],["55297.71","0.278"],["55297.31","0.292"],["55297.28","0.542"],["55297.18","0.362"],["55295.75","0.136"],["55295.68","0.160"],["55294.81","0.278"]],"a":[["55302.94","0.116"],["55305.98","0.202"],["55306.33","0.001"],["55306.58","0.054"],["55309.34","0.074"],["55309.36","0.090"],["55309.37","0.098"],["55309.52","0.116"],["55309.99","0.033"],["55310.62","0.181"],["55310.72","0.020"],["55311.04","0.217"],["55311.21","0.090"],["55311.41","0.181"],["55311.58","0.180"],["55311.59","0.519"],["55311.76","0.100"],["55311.86","0.243"],["55312.02","0.247"],["55312.42","0.090"]]}}`)
	for n := 0; n < t.N; n++ {
		depth20 := Depth20Stream{}
		err := json.Unmarshal(depth20Str, &depth20)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func BenchmarkParseMarkPrice(t *testing.B) {
	markPriceData := []byte(`{"stream":"eosusdt@markPrice@1s","data":{"e":"markPriceUpdate","E":1616555105001,"s":"EOSUSDT","p":"4.11998561","P":"4.11278428","i":"4.11519211","r":"0.00030438","T":1616572800000}}`)
	for n := 0; n < t.N; n++ {
		_, err := ParseMarkPrice(markPriceData)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func BenchmarkParseMarkPriceByStdJson(t *testing.B) {
	markPriceData := []byte(`{"stream":"eosusdt@markPrice@1s","data":{"e":"markPriceUpdate","E":1616555105001,"s":"EOSUSDT","p":"4.11998561","P":"4.11278428","i":"4.11519211","r":"0.00030438","T":1616572800000}}`)
	for n := 0; n < t.N; n++ {
		mp := MarkPriceStream{}
		err := json.Unmarshal(markPriceData, &mp)
		if err != nil {
			t.Fatal(err)
		}
	}
}

//
//func BenchmarkParseDepthFast20ByStdJson(t *testing.B) {
//	depth20Str := []byte(`{"stream":"btcusdt@depth20@100ms","data":{"e":"depthUpdate","E":1616509191577,"T":1616509191571,"s":"CDEFH1INCHUSDT","U":276060537661,"u":276060540084,"pu":276060537525,"b":[["55302.93","1.203"],["55302.33","1.052"],["55302.32","0.036"],["55301.31","0.048"],["55301.30","1.936"],["55299.12","0.036"],["55299.11","0.240"],["55299.06","2.851"],["55299.01","0.124"],["55299.00","1.337"],["55298.52","0.100"],["55298.51","0.008"],["55298.41","0.110"],["55297.71","0.278"],["55297.31","0.292"],["55297.28","0.542"],["55297.18","0.362"],["55295.75","0.136"],["55295.68","0.160"],["55294.81","0.278"]],"a":[["55302.94","0.116"],["55305.98","0.202"],["55306.33","0.001"],["55306.58","0.054"],["55309.34","0.074"],["55309.36","0.090"],["55309.37","0.098"],["55309.52","0.116"],["55309.99","0.033"],["55310.62","0.181"],["55310.72","0.020"],["55311.04","0.217"],["55311.21","0.090"],["55311.41","0.181"],["55311.58","0.180"],["55311.59","0.519"],["55311.76","0.100"],["55311.86","0.243"],["55312.02","0.247"],["55312.42","0.090"]]}}`)
//	for n := 0; n < t.N; n++ {
//		depth20 := DepthFastStream{}
//		err := json.Unmarshal(depth20Str, &depth20)
//		if err != nil {
//			t.Fatal(err)
//		}
//	}
//}

func TestFundingTime(t *testing.T) {
	fundingInterval := time.Hour * 8
	fundingSilent := time.Minute
	tt, err := time.Parse(time.RFC3339, "2006-01-02T16:01:05Z")
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", tt)
	logger.Debugf("%v", tt.Truncate(fundingInterval))
	logger.Debugf("%v", tt.Sub(tt.Truncate(fundingInterval)))
	logger.Debugf("%v", tt.Sub(tt.Truncate(fundingInterval)) > fundingSilent)
	logger.Debugf("%v", tt.Truncate(fundingInterval).Add(fundingInterval).Sub(tt) > fundingSilent)
	//time.RFC3339
	//time.Now().Sub(time.Now().Truncate(fundingInterval)) > fundingSilent &&
	//	time.Now().Truncate(fundingInterval).Add(fundingInterval).Sub(time.Now()) > fundingSilent
}
