package common

import (
	"fmt"
	"github.com/geometrybase/hft-micro/logger"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"unsafe"
)

//func TestFindTailZeroStart(t *testing.T) {
//	data := []byte("0.53100000")
//	hasZero, start := FindTailZeroStart(&data)
//	assert.Equal(t, true, hasZero)
//	assert.Equal(t, 5, start)
//	cut := data[:start]
//	assert.Equal(t, "0.531", *(*string)(unsafe.Pointer(&cut)))
//}
//
//func TestRemoveTailZero(t *testing.T) {
//	data := []byte("0.53100000")
//	b := RemoveTailZero(&data)
//	assert.Equal(t, "0.531", *(*string)(unsafe.Pointer(b)))
//}
//
//func BenchmarkFastBytesToStringRemoveTailZero(t *testing.B) {
//	b := []byte("0.53100000")
//	for n := 0; n < t.N; n++ {
//		a := RemoveTailZero(&b)
//		_, err := strconv.ParseBinanceFloat(*(*string)(unsafe.Pointer(a)), 64)
//		if err != nil {
//			t.Fatal(err)
//		}
//	}
//}
//
//func BenchmarkFastBytesToString(t *testing.B) {
//	b := []byte("0.53100000")
//	for n := 0; n < t.N; n++ {
//		_, err := strconv.ParseBinanceFloat(*(*string)(unsafe.Pointer(&b)), 64)
//		if err != nil {
//			t.Fatal(err)
//		}
//	}
//}
//
//func BenchmarkBytesToString(t *testing.B) {
//	b := []byte("0.53100000")
//	for n := 0; n < t.N; n++ {
//		_, err := strconv.ParseBinanceFloat(string(b), 64)
//		if err != nil {
//			t.Fatal(err)
//		}
//	}
//}

//{"stream":"eosusdt@markPrice@1s","data":{"e":"markPriceUpdate","E":1616555105001,"s":"EOSUSDT","p":"4.11998561","P":"4.11278428","i":"4.11519211","r":"0.00030438","T":1616572800000}}

func BenchmarkStdLibParseFloat64(t *testing.B) {
	b := []byte("3.14159265")
	for n := 0; n < t.N; n++ {
		_, err := strconv.ParseFloat(*(*string)(unsafe.Pointer(&b)), 64)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func BenchmarkParseBinanceFloat(t *testing.B) {
	b := []byte("3.14159265")
	for n := 0; n < t.N; n++ {
		_, err := ParseBinanceFloat(b)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func BenchmarkParseInt(t *testing.B) {
	b := []byte("314159265")
	for n := 0; n < t.N; n++ {
		_, err := strconv.ParseInt(*(*string)(unsafe.Pointer(&b)), 10, 64)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func BenchmarkParseBinanceInt(t *testing.B) {
	b := []byte("314159265")
	for n := 0; n < t.N; n++ {
		_, err := ParseBinanceInt(b)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func BenchmarkMapIntInt(t *testing.B) {
	m := make(map[int]int)
	for i := 0; i < 100; i++ {
		m[i] = i
	}
	for n := 0; n < t.N; n++ {
		for i := 0; i < 100; i++ {
			_ = m[i]
		}
	}
}

func BenchmarkMapStringInt(t *testing.B) {
	m := make(map[string]int)
	for i := 0; i < 100; i++ {
		m[fmt.Sprintf("BTCUSDT%d", i)] = i
	}
	for n := 0; n < t.N; n++ {
		for i := 0; i < 100; i++ {
			_ = m[fmt.Sprintf("BTCUSDT%d", i)]
		}
	}
}

func BenchmarkSliceMap(t *testing.B) {
	symbols := []string{"BTCUSDT", "LTCUSDT", "ETHUSDT", "NEOUSDT", "QTUMUSDT", "EOSUSDT", "ZRXUSDT", "OMGUSDT", "LRCUSDT", "TRXUSDT", "KNCUSDT", "IOTAUSDT", "LINKUSDT", "CVCUSDT", "ETCUSDT", "ZECUSDT", "BATUSDT", "DASHUSDT", "XMRUSDT", "ENJUSDT", "XRPUSDT", "STORJUSDT", "BTSUSDT", "ADAUSDT", "XLMUSDT", "WAVESUSDT", "ICXUSDT", "RLCUSDT", "IOSTUSDT", "BLZUSDT", "ONTUSDT", "ZILUSDT", "ZENUSDT", "THETAUSDT", "VETUSDT", "RENUSDT", "MATICUSDT", "ATOMUSDT", "FTMUSDT", "CHZUSDT", "ALGOUSDT", "DOGEUSDT", "ANKRUSDT", "TOMOUSDT", "BANDUSDT", "XTZUSDT", "KAVAUSDT", "BCHUSDT", "SOLUSDT", "HNTUSDT", "COMPUSDT", "MKRUSDT", "SXPUSDT", "SNXUSDT", "DOTUSDT", "RUNEUSDT", "BALUSDT", "YFIUSDT", "SRMUSDT", "CRVUSDT", "SANDUSDT", "OCEANUSDT", "LUNAUSDT", "RSRUSDT", "TRBUSDT", "EGLDUSDT", "BZRXUSDT", "KSMUSDT", "SUSHIUSDT", "YFIIUSDT", "BELUSDT", "UNIUSDT", "AVAXUSDT", "FLMUSDT", "ALPHAUSDT", "NEARUSDT", "AAVEUSDT", "FILUSDT", "CTKUSDT", "AXSUSDT", "AKROUSDT", "SKLUSDT", "GRTUSDT", "1INCHUSDT", "LITUSDT", "RVNUSDT", "SFPUSDT", "REEFUSDT", "DODOUSDT", "COTIUSDT", "CHRUSDT", "ALICEUSDT", "HBARUSDT", "MANAUSDT", "STMXUSDT", "UNFIUSDT", "XEMUSDT"}
	sort.Strings(symbols)
	logger.Debugf("%s", symbols)
	for n := 0; n < t.N; n++ {
		for _, symbol := range symbols {
			i := sort.SearchStrings(symbols, symbol)
			if i != -1 {
				_ = symbols[i]
			}
		}
	}
}

func BenchmarkStringMap(t *testing.B) {
	symbols := []string{"BTCUSDT", "LTCUSDT", "ETHUSDT", "NEOUSDT", "QTUMUSDT", "EOSUSDT", "ZRXUSDT", "OMGUSDT", "LRCUSDT", "TRXUSDT", "KNCUSDT", "IOTAUSDT", "LINKUSDT", "CVCUSDT", "ETCUSDT", "ZECUSDT", "BATUSDT", "DASHUSDT", "XMRUSDT", "ENJUSDT", "XRPUSDT", "STORJUSDT", "BTSUSDT", "ADAUSDT", "XLMUSDT", "WAVESUSDT", "ICXUSDT", "RLCUSDT", "IOSTUSDT", "BLZUSDT", "ONTUSDT", "ZILUSDT", "ZENUSDT", "THETAUSDT", "VETUSDT", "RENUSDT", "MATICUSDT", "ATOMUSDT", "FTMUSDT", "CHZUSDT", "ALGOUSDT", "DOGEUSDT", "ANKRUSDT", "TOMOUSDT", "BANDUSDT", "XTZUSDT", "KAVAUSDT", "BCHUSDT", "SOLUSDT", "HNTUSDT", "COMPUSDT", "MKRUSDT", "SXPUSDT", "SNXUSDT", "DOTUSDT", "RUNEUSDT", "BALUSDT", "YFIUSDT", "SRMUSDT", "CRVUSDT", "SANDUSDT", "OCEANUSDT", "LUNAUSDT", "RSRUSDT", "TRBUSDT", "EGLDUSDT", "BZRXUSDT", "KSMUSDT", "SUSHIUSDT", "YFIIUSDT", "BELUSDT", "UNIUSDT", "AVAXUSDT", "FLMUSDT", "ALPHAUSDT", "NEARUSDT", "AAVEUSDT", "FILUSDT", "CTKUSDT", "AXSUSDT", "AKROUSDT", "SKLUSDT", "GRTUSDT", "1INCHUSDT", "LITUSDT", "RVNUSDT", "SFPUSDT", "REEFUSDT", "DODOUSDT", "COTIUSDT", "CHRUSDT", "ALICEUSDT", "HBARUSDT", "MANAUSDT", "STMXUSDT", "UNFIUSDT", "XEMUSDT"}
	m := make(map[string]string)
	for _, s := range symbols {
		m[s] = s
	}
	sort.Strings(symbols)
	for n := 0; n < t.N; n++ {
		_ = m["BTCUSDT"]
	}
}

func BenchmarkSyncMutex(t *testing.B) {
	counter := int32(0)
	for n := 0; n < t.N; n++ {
		mu := sync.Mutex{}
		wg := sync.WaitGroup{}
		for i := 0; i < 1000; i++ {
			wg.Add(1)
			go func() {
				mu.Lock()
				counter++
				mu.Unlock()
				wg.Done()
			}()
		}
		wg.Wait()
	}
}

func BenchmarkSyncAtomic(t *testing.B) {
	counter := int32(0)
	for n := 0; n < t.N; n++ {
		wg := sync.WaitGroup{}
		for i := 0; i < 1000; i++ {
			wg.Add(1)
			go func() {
				atomic.AddInt32(&counter, 1)
				wg.Done()
			}()
		}
		wg.Wait()
	}
}

func BenchmarkChanAdd(t *testing.B) {
	for n := 0; n < t.N; n++ {
		wg := sync.WaitGroup{}
		counter := int32(0)
		ch := make(chan int32)
		go func() {
			for {
				select {
				case i := <-ch:
					wg.Done()
					counter += i
				}
			}
		}()
		for i := 0; i < 1000; i++ {
			wg.Add(1)
			ch <- 1
		}
		wg.Wait()
	}
}
