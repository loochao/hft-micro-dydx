package common

import (
	"context"
	"fmt"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/stretchr/testify/assert"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
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
//		_, err := strconv.ParseFloat(*(*string)(unsafe.Pointer(a)), 64)
//		if err != nil {
//			t.Fatal(err)
//		}
//	}
//}
//
//func BenchmarkFastBytesToString(t *testing.B) {
//	b := []byte("0.53100000")
//	for n := 0; n < t.N; n++ {
//		_, err := strconv.ParseFloat(*(*string)(unsafe.Pointer(&b)), 64)
//		if err != nil {
//			t.Fatal(err)
//		}
//	}
//}
//
//func BenchmarkBytesToString(t *testing.B) {
//	b := []byte("0.53100000")
//	for n := 0; n < t.N; n++ {
//		_, err := strconv.ParseFloat(string(b), 64)
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
	b := []byte("376.720000")
	t.ReportAllocs()
	for n := 0; n < t.N; n++ {
		_, _ = ParseFloat(b)
	}
}

func TestParseFloat(t *testing.T) {
	b := []byte("376.7200000000010000000000001")
	f, err := ParseFloat(b)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 376.7200000000000000000000001, f)
}

func BenchmarkParseInt(t *testing.B) {
	b := []byte("314159265")
	for n := 0; n < t.N; n++ {
		_, _ = strconv.ParseInt(*(*string)(unsafe.Pointer(&b)), 10, 64)
	}
}

func BenchmarkParseBinanceInt(t *testing.B) {
	b := []byte("314159265")
	for n := 0; n < t.N; n++ {
		_, _ = ParseInt(b)
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

func TestFormatByPrecision(t *testing.T) {
	f := FormatByPrecision(0.0123123123, 0)
	assert.Equal(t, "0", f)
	f = FormatByPrecision(0.0123123123, 1)
	assert.Equal(t, "0.0", f)
	f = FormatByPrecision(0.0123123123, 2)
	assert.Equal(t, "0.01", f)
	f = FormatByPrecision(0.0123123123, 3)
	assert.Equal(t, "0.012", f)
	f = FormatByPrecision(0.0123123123, 4)
	assert.Equal(t, "0.0123", f)
	f = FormatByPrecision(0.0123123123, 5)
	assert.Equal(t, "0.01231", f)
}

func TestMergedStepSize(t *testing.T) {
	a := 0.1
	b := 0.03
	logger.Debugf("%f %f %f", a, b, MergedStepSize(a, b))
	a = 0.1
	b = 0.1
	logger.Debugf("%f %f %f", a, b, MergedStepSize(a, b))
	a = 0.1
	b = 0.033333
	logger.Debugf("%f %f %f", a, b, MergedStepSize(a, b))
}

func BenchmarkSelectWithContext(t *testing.B) {
	ch := make(chan interface{})
	ctx := context.Background()
	go func() {
		for {
			select {
			case <-ch:
			}
		}
	}()
	t.ReportAllocs()
	t.ResetTimer()
	for n := 0; n < t.N; n++ {
		select {
		case <-ctx.Done():
		case <-ctx.Done():
		case <-ctx.Done():
		case <-ctx.Done():
		case <-ctx.Done():
		case <-ctx.Done():
		case <-ctx.Done():
		case ch <- nil:
		}
	}
}

func BenchmarkWithOutSelect(t *testing.B) {
	ch := make(chan interface{})
	go func() {
		for {
			<-ch
		}
	}()
	t.ReportAllocs()
	t.ResetTimer()
	for n := 0; n < t.N; n++ {
		ch <- nil
	}
}

func BenchmarkWithOutSelectMoreConsumer(t *testing.B) {
	ch := make(chan interface{})
	for i := 0; i < 4; i++ {
		go func() {
			for {
				<-ch
			}
		}()
	}
	t.ReportAllocs()
	t.ResetTimer()
	for n := 0; n < t.N; n++ {
		ch <- nil
	}
}

func BenchmarkSelectWithoutContext(t *testing.B) {
	ch := make(chan interface{})
	go func() {
		for {
			select {
			case <-ch:
			}
		}
	}()
	t.ReportAllocs()
	t.ResetTimer()
	for n := 0; n < t.N; n++ {
		select {
		case ch <- nil:
		}
	}
}

func BenchmarkSelectWithBufferWithoutContext(t *testing.B) {
	ch := make(chan interface{}, 100)
	go func() {
		timer := time.NewTimer(time.Microsecond)
		for {
			select {
			case <- timer.C:
				select {
				case <-ch:
				}
				timer.Reset(time.Microsecond)
			}
		}
	}()
	t.ReportAllocs()
	t.ResetTimer()
	for n := 0; n < t.N; n++ {
		select {
		case ch <- nil:
		}
	}
}

func BenchmarkSelectWithoutContextMoreConsumer(t *testing.B) {
	ch := make(chan interface{})
	for i := 0; i < 4; i++ {
		go func() {
			timer := time.NewTimer(time.Microsecond)
			for {
				select {
				case <- timer.C:
					select {
					case <-ch:
					}
					timer.Reset(time.Microsecond)
				}
			}
		}()
	}
	t.ReportAllocs()
	t.ResetTimer()
	for n := 0; n < t.N; n++ {
		select {
		case ch <- nil:
		}
	}
}

func BenchmarkSelectWithoutContextAndBufferAndMoreConsumer(t *testing.B) {
	ch := make(chan interface{}, 10000)
	for i := 0; i < 4; i++ {
		go func() {
			timer := time.NewTimer(time.Microsecond)
			for {
				select {
				case <- timer.C:
					select {
					case <-ch:
					}
					timer.Reset(time.Microsecond)
				}
			}
		}()
	}
	t.ReportAllocs()
	t.ResetTimer()
	for n := 0; n < t.N; n++ {
		select {
		case ch <- nil:
		}
	}
}




func BenchmarkSelectWithContexts(t *testing.B) {
	ch := make(chan interface{})
	ctx1 := context.Background()
	ctx2 := context.Background()
	ctx3 := context.Background()
	ctx4 := context.Background()
	ctx5 := context.Background()
	ctx6 := context.Background()
	ctx7 := context.Background()
	ctx8 := context.Background()
	ctx9 := context.Background()
	ctx0 := context.Background()
	go func() {
		for {
			select {
			case <-ch:
			}
		}
	}()
	t.ReportAllocs()
	t.ResetTimer()
	for n := 0; n < t.N; n++ {
		select {
		case <-ctx0.Done():
		case <-ctx1.Done():
		case <-ctx2.Done():
		case <-ctx3.Done():
		case <-ctx4.Done():
		case <-ctx5.Done():
		case <-ctx6.Done():
		case <-ctx7.Done():
		case <-ctx8.Done():
		case <-ctx9.Done():
		case ch <- nil:
		}
	}
}


func TestSelect(t *testing.T) {
	ch1 := make(chan interface{}, 100)
	ch2 := make(chan interface{}, 100)
	ch3 := make(chan interface{}, 100)
	for i := 0; i < 10; i ++ {
		select {
		case ch1 <- nil:
			logger.Debugf("LOOP %d ch1", i)
		case ch2 <- nil:
			logger.Debugf("LOOP %d ch2", i)
		case ch3 <- nil:
			logger.Debugf("LOOP %d ch3", i)
		default:
			logger.Debugf("LOOP %d default", i)
		}
	}
}

