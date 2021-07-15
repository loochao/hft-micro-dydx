package kucoin_usdtspot

import (
	"encoding/json"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestParseDepth50(t *testing.T) {
	bytes := []byte(`{"data":{"asks":[["59354.1","0.28639406"],["59359.3","0.0005"],["59364.9","0.60053974"],["59370.9","0.00015993"],["59371.6","0.20092"],["59374.3","0.22759333"],["59374.4","0.08"],["59380.3","0.01758361"],["59382","0.03516722"],["59384.6","0.23586209"],["59385.7","0.07912624"],["59386.4","0.0005"],["59389.8","0.08802514"],["59393.2","0.07906055"],["59393.9","0.13154242"],["59396.2","0.26308485"],["59400.2","0.05"],["59401.3","1.01055517"],["59406.3","0.01757488"],["59412.1","0.19271256"],["59412.5","0.00138617"],["59412.6","0.00918042"],["59419.4","0.5"],["59426","0.3666213"],["59426.1","1.03958439"],["59429.5","0.02575597"],["59431.2","0.00007293"],["59431.7","0.00694695"],["59432.1","0.01543023"],["59432.9","0.00183334"],["59433","1"],["59440.2","0.0303214"],["59441.3","0.0002"],["59444","0.00006034"],["59444.6","0.8"],["59444.7","0.00008252"],["59447.1","0.00000757"],["59450.5","1"],["59456.5","1.5"],["59457.6","0.01601724"],["59458","0.03006194"],["59458.4","0.00549434"],["59459.3","0.01316853"],["59461.5","2"],["59463.1","0.01091932"],["59467","0.05226236"],["59473.6","0.00099066"],["59479.2","0.06151213"],["59484.6","0.07236123"],["59486.9","0.01529323"]],"bids":[["59354","0.1052651"],["59353.8","0.001"],["59353.7","0.0005"],["59353.4","0.01"],["59352.9","0.60053974"],["59352.8","0.01"],["59352.3","0.0105"],["59351.8","0.006"],["59351.4","0.0005"],["59351.2","0.0843295"],["59350.6","0.01"],["59350.1","0.017"],["59349.5","0.01"],["59348.6","0.14994407"],["59347.2","0.01"],["59347.1","0.01799999"],["59345.9","0.08"],["59344.2","0.02"],["59342.3","0.00005975"],["59341.8","0.005"],["59339.3","0.01282978"],["59338.1","0.01"],["59337.3","0.02"],["59326","0.0001191"],["59325.2","0.00015539"],["59324.2","0.01016856"],["59323.7","0.01757997"],["59323.4","0.00033712"],["59323.3","0.00016856"],["59322.3","0.00016857"],["59322.2","0.26308485"],["59321.2","0.03515917"],["59319.8","0.00288964"],["59318.8","0.3949938"],["59318.7","0.13546822"],["59317.5","0.05"],["59316.9","0.07910793"],["59314.3","0.07906219"],["59314.1","0.019"],["59307.8","0.5"],["59307.4","0.00014017"],["59306.3","1.50227325"],["59305.3","0.00031156"],["59304.6","0.00016862"],["59297.4","0.48221985"],["59293.9","1.00074241"],["59288.2","0.01097875"],["59287.8","0.01757648"],["59285.7","0.03211842"],["59280.6","0.0002"]],"timestamp":1618148348758},"subject":"level2","topic":"/spotMarket/level2Depth50:BTC-USDT","type":"message"}`)
	parseD, err := ParseDepth50(bytes)
	if err != nil {
		t.Fatal(err)
	}
	jsonD := Depth50{}
	err = json.Unmarshal(bytes, &jsonD)
	if err != nil {
		t.Fatal(err)
	}
	for i, b := range parseD.Bids {
		assert.Equal(t, jsonD.Bids[i][0], b[0])
		assert.Equal(t, jsonD.Bids[i][1], b[1])
	}
	for i, a := range parseD.Asks {
		assert.Equal(t, jsonD.Asks[i][0], a[0])
		assert.Equal(t, jsonD.Asks[i][1], a[1])
	}
	assert.Equal(t, jsonD.Symbol, parseD.Symbol)
	assert.Equal(t, jsonD.EventTime, parseD.EventTime)
}

func BenchmarkParseDepth50(t *testing.B) {
	bytes := []byte(`{"data":{"asks":[["59354.1","0.28639406"],["59359.3","0.0005"],["59364.9","0.60053974"],["59370.9","0.00015993"],["59371.6","0.20092"],["59374.3","0.22759333"],["59374.4","0.08"],["59380.3","0.01758361"],["59382","0.03516722"],["59384.6","0.23586209"],["59385.7","0.07912624"],["59386.4","0.0005"],["59389.8","0.08802514"],["59393.2","0.07906055"],["59393.9","0.13154242"],["59396.2","0.26308485"],["59400.2","0.05"],["59401.3","1.01055517"],["59406.3","0.01757488"],["59412.1","0.19271256"],["59412.5","0.00138617"],["59412.6","0.00918042"],["59419.4","0.5"],["59426","0.3666213"],["59426.1","1.03958439"],["59429.5","0.02575597"],["59431.2","0.00007293"],["59431.7","0.00694695"],["59432.1","0.01543023"],["59432.9","0.00183334"],["59433","1"],["59440.2","0.0303214"],["59441.3","0.0002"],["59444","0.00006034"],["59444.6","0.8"],["59444.7","0.00008252"],["59447.1","0.00000757"],["59450.5","1"],["59456.5","1.5"],["59457.6","0.01601724"],["59458","0.03006194"],["59458.4","0.00549434"],["59459.3","0.01316853"],["59461.5","2"],["59463.1","0.01091932"],["59467","0.05226236"],["59473.6","0.00099066"],["59479.2","0.06151213"],["59484.6","0.07236123"],["59486.9","0.01529323"]],"bids":[["59354","0.1052651"],["59353.8","0.001"],["59353.7","0.0005"],["59353.4","0.01"],["59352.9","0.60053974"],["59352.8","0.01"],["59352.3","0.0105"],["59351.8","0.006"],["59351.4","0.0005"],["59351.2","0.0843295"],["59350.6","0.01"],["59350.1","0.017"],["59349.5","0.01"],["59348.6","0.14994407"],["59347.2","0.01"],["59347.1","0.01799999"],["59345.9","0.08"],["59344.2","0.02"],["59342.3","0.00005975"],["59341.8","0.005"],["59339.3","0.01282978"],["59338.1","0.01"],["59337.3","0.02"],["59326","0.0001191"],["59325.2","0.00015539"],["59324.2","0.01016856"],["59323.7","0.01757997"],["59323.4","0.00033712"],["59323.3","0.00016856"],["59322.3","0.00016857"],["59322.2","0.26308485"],["59321.2","0.03515917"],["59319.8","0.00288964"],["59318.8","0.3949938"],["59318.7","0.13546822"],["59317.5","0.05"],["59316.9","0.07910793"],["59314.3","0.07906219"],["59314.1","0.019"],["59307.8","0.5"],["59307.4","0.00014017"],["59306.3","1.50227325"],["59305.3","0.00031156"],["59304.6","0.00016862"],["59297.4","0.48221985"],["59293.9","1.00074241"],["59288.2","0.01097875"],["59287.8","0.01757648"],["59285.7","0.03211842"],["59280.6","0.0002"]],"timestamp":1618148348758},"subject":"level2","topic":"/spotMarket/level2Depth50:BTC-USDT","type":"message"}`)
	t.ReportAllocs()
	for i := 0; i < t.N; i++ {
		_, _ = ParseDepth50(bytes)
	}
}

func TestParseDepth5(t *testing.T) {
	bytes := []byte(`{"data":{"asks":[["55447.5","0.00128653"],["55447.6","0.0040067"],["55447.7","5.26962769"],["55449","0.00016278"],["55451.5","0.00013396"]],"bids":[["55403.1","0.01254575"],["55402.5","0.00005319"],["55279.9","0.201"],["55268.3","0.02406837"],["55233.5","0.0004668"]],"timestamp":1618724853172},"subject":"level2","topic":"/spotMarket/level2Depth5:BTC-USDT","type":"message"}`)
	jsonD := Depth5{}
	err := json.Unmarshal(bytes, &jsonD)
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", jsonD)
	depth5 := Depth5{}
	err = ParseDepth5(bytes, &depth5)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, jsonD.Symbol, depth5.Symbol)
	assert.Equal(t, jsonD.EventTime, depth5.EventTime)
	for i := 0; i < 5; i++ {
		assert.Equal(t, jsonD.Bids[i][0], depth5.Bids[i][0])
		assert.Equal(t, jsonD.Bids[i][1], depth5.Bids[i][1])
		assert.Equal(t, jsonD.Asks[i][0], depth5.Asks[i][0])
		assert.Equal(t, jsonD.Asks[i][1], depth5.Asks[i][1])
	}
}

func TestParseDepth52(t *testing.T) {
	bytes := []byte(`{"data":{"asks":[["55447.6","0.0040067"],["55447.7","5.26962769"],["55449","0.00016278"],["55451.5","0.00013396"]],"bids":[["55403.1","0.01254575"],["55402.5","0.00005319"],["55279.9","0.201"],["55268.3","0.02406837"],["55233.5","0.0004668"]],"timestamp":1618724853172},"subject":"level2","topic":"/spotMarket/level2Depth5:BTC-USDT","type":"message"}`)
	jsonD := Depth5{}
	err := json.Unmarshal(bytes, &jsonD)
	if err != nil {
		t.Fatal(err)
	}
	depth5 := Depth5{}
	err = ParseDepth5(bytes, &depth5)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, jsonD.Symbol, depth5.Symbol)
	assert.Equal(t, jsonD.EventTime, depth5.EventTime)
	for i := 0; i < 5; i++ {
		assert.Equal(t, jsonD.Bids[i][0], depth5.Bids[i][0])
		assert.Equal(t, jsonD.Bids[i][1], depth5.Bids[i][1])
		assert.Equal(t, jsonD.Asks[i][0], depth5.Asks[i][0])
		assert.Equal(t, jsonD.Asks[i][1], depth5.Asks[i][1])
	}
}

//func TestParseDepth53(t *testing.T) {
//	bytes := []byte(`{"data":{"asks":[],"bids":[["55403.1","0.01254575"],["55402.5","0.00005319"],["55279.9","0.201"],["55268.3","0.02406837"],["55233.5","0.0004668"]],"timestamp":1618724853172},"subject":"level2","topic":"/spotMarket/level2Depth5:BTC-USDT","type":"message"}`)
//	jsonD := Depth5{}
//	err := json.Unmarshal(bytes, &jsonD)
//	if err != nil {
//		t.Fatal(err)
//	}
//	depth5, err := ParseDepth5(bytes)
//	if err != nil {
//		t.Fatal(err)
//	}
//	assert.Equal(t, jsonD.Symbol, depth5.Symbol)
//	assert.Equal(t, jsonD.EventTime, depth5.EventTime)
//	for i := 0; i < 5; i++ {
//		assert.Equal(t, jsonD.Bids[i][0], depth5.Bids[i][0])
//		assert.Equal(t, jsonD.Bids[i][1], depth5.Bids[i][1])
//		assert.Equal(t, jsonD.Asks[i][0], depth5.Asks[i][0])
//		assert.Equal(t, jsonD.Asks[i][1], depth5.Asks[i][1])
//	}
//}

var GlobalT *Ticker


func TestParseTicker(t *testing.T) {
	ticker := Ticker{}
	jTicker := TickerData{}
	for _, msg := range strings.Split(TickerTestLines, "\n") {
		err := json.Unmarshal([]byte(msg), &jTicker)
		if err != nil {
			t.Fatal(err)
		}
		err = ParseTicker([]byte(msg), &ticker)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, jTicker.Data.Symbol, ticker.Symbol)
		assert.Equal(t, jTicker.Data.BestBidSize, ticker.BestBidSize)
		assert.Equal(t, jTicker.Data.BestBidPrice, ticker.BestBidPrice)
		assert.Equal(t, jTicker.Data.BestAskSize, ticker.BestAskSize)
		assert.Equal(t, jTicker.Data.BestAskPrice, ticker.BestAskPrice)
	}
}

func BenchmarkParseTicker(t *testing.B) {
	b := []byte(`{"data":{"sequence":"1612914399696","bestAsk":"13.3792","size":"8","bestBidSize":"167.5597","price":"13.3365","time":1626290933890,"bestAskSize":"2.4461","bestBid":"13.3339"},"subject":"trade.ticker","topic":"/market/ticker:WAVES-USDT","type":"message"}`)
	x := &Ticker{}
	t.ReportAllocs()
	for i := 0; i < t.N; i++ {
		_ = ParseTicker(b, x)
	}
	GlobalT = x
}

func BenchmarkParseTickerStdJson(t *testing.B) {
	b := []byte(`{"data":{"symbol":"XBTUSDTM","sequence":1624824091680,"side":"buy","size":63,"price":33679,"bestBidSize":16,"bestBidPrice":"33678.0","bestAskPrice":"33679.0","tradeId":"60e93a803c7feb289d2be531","ts":1625897600449461736,"bestAskSize":1119},"subject":"ticker","topic":"/contractMarket/ticker:XBTUSDTM","type":"message"}`)
	x := &TickerData{}
	t.ReportAllocs()
	for i := 0; i < t.N; i++ {
		_ = json.Unmarshal(b, x)
	}
}
