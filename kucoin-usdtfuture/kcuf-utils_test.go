package kucoin_usdtfuture

import (
	"encoding/json"
	"github.com/minio/simdjson-go"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestSimdJson(t *testing.T) {
	bytes := []byte(`{"data":{"sequence":1617684642656,"asks":[[60818,160000],[60823.0,10325],[60824.0,3000],[60828.0,5325],[60829.0,5000],[60831.0,112438],[60833.0,3000],[60834.0,3750],[60836.0,80000],[60837.0,71844],[60838.0,54426],[60841.0,74100],[60842.0,60564],[60843.0,3000],[60848.0,73816],[60850.0,49770],[60851,80000],[60852.0,63455],[60855.0,54438],[60857.0,51719],[60859.0,123907],[60869.0,5697],[60888,1397],[60894.0,6089],[60900,6168],[60904,4134],[60905.0,6494],[60906.0,84187],[60938.0,119895],[60952.0,7333],[60990,13840],[61030.0,7764],[61040,12262],[61041.0,7764],[61060,12262],[61080,12262],[61100,12262],[61120,12262],[61138,142396],[61140,12262],[61180.0,8644],[61200,954],[61238,15],[61355,71706],[61400,10],[61416.0,541265],[61425.0,10000],[61427.0,606647],[61448.0,10000],[61500,1436]],"bids":[[60811,160000],[60810.0,124958],[60806.0,70501],[60804,145306],[60803.0,3000],[60801.0,52777],[60800.0,70833],[60798,80000],[60796.0,3000],[60794.0,54705],[60793.0,67823],[60789.0,58175],[60788.0,52072],[60784.0,59274],[60782.0,3000],[60780.0,72154],[60763.0,53314],[60755.0,5325],[60741.0,5697],[60736.0,1397],[60725.0,6494],[60711,100100],[60707.0,6494],[60693.0,62797],[60691.0,74153],[60662.0,6909],[60640,12262],[60627,25],[60622,13840],[60620,12262],[60613.0,7333],[60600.0,12362],[60589.0,7764],[60580,12262],[60560,12262],[60540,12262],[60500,1442],[60490,2029],[60488.0,8201],[60467.0,8644],[60458,141278],[60444,25],[60417,15],[60367.0,8201],[60363,70473],[60347.0,9092],[60327,50],[60300,341727],[60206.0,504206],[60201,12000]],"ts":1618219600281,"timestamp":1618219600281},"subject":"level2","topic":"/contractMarket/level2Depth50:XBTUSDM","type":"message"}`)
	reuse := &simdjson.ParsedJson{}
	var err error
	reuse, err = simdjson.Parse(bytes, reuse)
	if err != nil {
		t.Fatal(err)
	}
	//depth := Depth5{}
	var tmp *simdjson.Iter
	var obj *simdjson.Object
	var elem simdjson.Element
	iter := reuse.Iter()
	typ := iter.Advance()
	if typ != simdjson.TypeRoot {
		return
	}
	typ, tmp, err = iter.Root(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if obj, err = tmp.Object(obj); err != nil {
		t.Fatal(err)
		return
	}
	e := obj.FindKey("data", &elem)
	if e == nil && elem.Type != simdjson.TypeObject {
		return
	}

	elem.Iter.Advance()

	//for {
	//	typ := iter.Advance()
	//	switch typ {
	//	case simdjson.TypeRoot:
	//		if typ, tmp, err = iter.Root(tmp); err != nil {
	//			t.Fatal(err)
	//			return
	//		}
	//		logger.Debugf("%v", typ)
	//		if typ == simdjson.TypeObject {
	//			if obj, err = tmp.Object(obj); err != nil {
	//				t.Fatal(err)
	//				return
	//			}
	//			e := obj.FindKey("data", &elem)
	//			logger.Debugf("%v", elem)
	//			if e != nil && elem.Type == simdjson.TypeString {
	//				v, _ := elem.Iter.StringBytes()
	//				fmt.Println(string(v))
	//			}
	//		}
	//	default:
	//		return
	//	}
	//}
}

func BenchmarkJsonParseDepth50(t *testing.B) {
	bytes := []byte(`{"data":{"sequence":1617684642656,"asks":[[60818,160000],[60823.0,10325],[60824.0,3000],[60828.0,5325],[60829.0,5000],[60831.0,112438],[60833.0,3000],[60834.0,3750],[60836.0,80000],[60837.0,71844],[60838.0,54426],[60841.0,74100],[60842.0,60564],[60843.0,3000],[60848.0,73816],[60850.0,49770],[60851,80000],[60852.0,63455],[60855.0,54438],[60857.0,51719],[60859.0,123907],[60869.0,5697],[60888,1397],[60894.0,6089],[60900,6168],[60904,4134],[60905.0,6494],[60906.0,84187],[60938.0,119895],[60952.0,7333],[60990,13840],[61030.0,7764],[61040,12262],[61041.0,7764],[61060,12262],[61080,12262],[61100,12262],[61120,12262],[61138,142396],[61140,12262],[61180.0,8644],[61200,954],[61238,15],[61355,71706],[61400,10],[61416.0,541265],[61425.0,10000],[61427.0,606647],[61448.0,10000],[61500,1436]],"bids":[[60811,160000],[60810.0,124958],[60806.0,70501],[60804,145306],[60803.0,3000],[60801.0,52777],[60800.0,70833],[60798,80000],[60796.0,3000],[60794.0,54705],[60793.0,67823],[60789.0,58175],[60788.0,52072],[60784.0,59274],[60782.0,3000],[60780.0,72154],[60763.0,53314],[60755.0,5325],[60741.0,5697],[60736.0,1397],[60725.0,6494],[60711,100100],[60707.0,6494],[60693.0,62797],[60691.0,74153],[60662.0,6909],[60640,12262],[60627,25],[60622,13840],[60620,12262],[60613.0,7333],[60600.0,12362],[60589.0,7764],[60580,12262],[60560,12262],[60540,12262],[60500,1442],[60490,2029],[60488.0,8201],[60467.0,8644],[60458,141278],[60444,25],[60417,15],[60367.0,8201],[60363,70473],[60347.0,9092],[60327,50],[60300,341727],[60206.0,504206],[60201,12000]],"ts":1618219600281,"timestamp":1618219600281},"subject":"level2","topic":"/contractMarket/level2Depth50:XBTUSDM","type":"message"}`)
	t.ReportAllocs()
	for i := 0; i < t.N; i++ {
		wsCap := WsCap{}
		err := json.Unmarshal(bytes, &wsCap)
		if err != nil {
			t.Fatal(err)
		}
		jsonD := Depth50{}
		err = json.Unmarshal(wsCap.Data, &jsonD)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestParseDepth5(t *testing.T) {
	for _, line := range strings.Split(Depth5SampleLines, "\n") {
		//logger.Debugf("%s", line)
		wsCap := WsCap{}
		err := json.Unmarshal([]byte(line), &wsCap)
		if err != nil {
			t.Fatal(err)
		}
		jsonD := Depth5{}
		err = json.Unmarshal(wsCap.Data, &jsonD)
		if err != nil {
			t.Fatal(err)
		}
		jsonD.Symbol = strings.Split(wsCap.Topic, ":")[1]
		depth5 := &Depth5{}
		err = ParseDepth5([]byte(line), depth5)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, jsonD.Symbol, depth5.Symbol)
		assert.Equal(t, jsonD.EventTime, depth5.EventTime)
		assert.Equal(t, 5, len(jsonD.Bids))
		assert.Equal(t, 5, len(jsonD.Asks))
		assert.Equal(t, 5, len(depth5.Bids))
		assert.Equal(t, 5, len(depth5.Asks))
		for i := 0; i < 5; i++ {
			assert.Equal(t, jsonD.Bids[i][0], depth5.Bids[i][0])
			assert.Equal(t, jsonD.Bids[i][1], depth5.Bids[i][1])
			assert.Equal(t, jsonD.Asks[i][0], depth5.Asks[i][0])
			assert.Equal(t, jsonD.Asks[i][1], depth5.Asks[i][1])
		}
	}
}

var GlobalD *Depth5

func BenchmarkParseDepth5(t *testing.B) {
	b := []byte(`{"data":{"sequence":1616576945844,"asks":[[17.834,10],[18.019,10154],[18.082,11060],[18.082,11060],[17.779,407]],"bids":[[17.797,701],[17.793,1061],[17.784,199],[17.781,881],[17.779,407]],"ts":1618717277315,"timestamp":1618717277315},"subject":"level2","topic":"/contractMarket/level2Depth5:ATOMUSDTM","type":"message"}`)
	//b := []byte(`{"type":"message","topic":"/contractMarket/level2Depth5:CHZUSDTM","subject":"level2","data":{"sequence":1627365884233,"asks":[[0.2621,7501],[0.2622,3599],[0.2623,52851],[0.2624,38379],[0.2625,39980]],"bids":[[0.2619,2298],[0.2618,19222],[0.2617,17837],[0.2616,21857],[0.2615,31419]],"ts":1627723139251,"timestamp":1627723139251}}`)
	x := &Depth5{}
	t.ReportAllocs()
	for i := 0; i < t.N; i++ {
		_ = ParseDepth5(b, x)
	}
	GlobalD = x
}

func BenchmarkParseDepth5StdJson(t *testing.B) {
	b := []byte(`{"data":{"sequence":1616576945844,"asks":[[17.834,10],[18.019,10154],[18.082,11060],[18.082,11060],[17.779,407]],"bids":[[17.797,701],[17.793,1061],[17.784,199],[17.781,881],[17.779,407]],"ts":1618717277315,"timestamp":1618717277315},"subject":"level2","topic":"/contractMarket/level2Depth5:ATOMUSDTM","type":"message"}`)
	wsCap := WsCap{}
	jsonD := &Depth5{}
	t.ReportAllocs()
	for i := 0; i < t.N; i++ {
		_ = json.Unmarshal(b, &wsCap)
		_ = json.Unmarshal(wsCap.Data, jsonD)
	}
	GlobalD = jsonD
}

func TestParseTicker2(t *testing.T) {
	msg := []byte(`{"type":"message","topic":"/contractMarket/ticker:1INCHUSDTM","subject":"ticker","data":{"symbol":"1INCHUSDTM","sequence":1627371661456,"side":"buy","size":21,"price":2.379,"bestBidSize":203,"bestBidPrice":"2.377","bestAskPrice":"2.38","tradeId":"6105178a991e1303211759d8","ts":1627723658671236584,"bestAskSize":251}}`)
	ticker := Ticker{}
	err := ParseTicker(msg, &ticker)
	if err != nil {
		t.Fatal(err)
	}
}

func TestParseTicker(t *testing.T) {
	ticker := Ticker{}
	jTicker := TickerData{}
	for _, msg := range strings.Split(TickerSampleLines, "\n") {
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

var GlobalT *Ticker

func BenchmarkParseTicker(t *testing.B) {
	b := []byte(`{"data":{"symbol":"XBTUSDTM","sequence":1624824091680,"side":"buy","size":63,"price":33679,"bestBidSize":16,"bestBidPrice":"33678.0","bestAskPrice":"33679.0","tradeId":"60e93a803c7feb289d2be531","ts":1625897600449461736,"bestAskSize":1119},"subject":"ticker","topic":"/contractMarket/ticker:XBTUSDTM","type":"message"}`)
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
