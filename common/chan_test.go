package common_test

import (
	"github.com/geometrybase/hft-micro/common"
	"testing"
	"time"
)

func BenchmarkChanSendNil(b *testing.B) {
	ch := make(chan interface{})
	go func() {
		for {
			select {
			case <-ch:
			}
		}
	}()
	b.ReportAllocs()
	b.ResetTimer()
	for t := 0; t < b.N; t++ {
		ch <- nil
	}
}

func BenchmarkChanSendNil4Buffered(b *testing.B) {
	ch := make(chan interface{}, 4)
	go func() {
		for {
			select {
			case <-ch:
			}
		}
	}()
	b.ReportAllocs()
	b.ResetTimer()
	for t := 0; t < b.N; t++ {
		ch <- nil
	}
}

func BenchmarkChanSendNil16Buffered(b *testing.B) {
	ch := make(chan interface{}, 16)
	go func() {
		for {
			select {
			case <-ch:
			}
		}
	}()
	b.ReportAllocs()
	b.ResetTimer()
	for t := 0; t < b.N; t++ {
		ch <- nil
	}
}

func BenchmarkChanSendNil64Buffered(b *testing.B) {
	ch := make(chan interface{}, 64)
	go func() {
		for {
			select {
			case <-ch:
			}
		}
	}()
	b.ReportAllocs()
	b.ResetTimer()
	for t := 0; t < b.N; t++ {
		ch <- nil
	}
}

func BenchmarkChanSendNil128Buffered(b *testing.B) {
	ch := make(chan interface{}, 128)
	go func() {
		for {
			select {
			case <-ch:
			}
		}
	}()
	b.ReportAllocs()
	b.ResetTimer()
	for t := 0; t < b.N; t++ {
		ch <- nil
	}
}

func BenchmarkChanIntSendNil128Buffered(b *testing.B) {
	ch := make(chan uint8, 128)
	go func() {
		for {
			select {
			case <-ch:
			}
		}
	}()
	b.ReportAllocs()
	b.ResetTimer()
	for t := 0; t < b.N; t++ {
		ch <- 0
	}
}

func BenchmarkChanStructSendNil128Buffered(b *testing.B) {
	ch := make(chan struct{}, 128)
	go func() {
		for {
			select {
			case <-ch:
			}
		}
	}()
	d := struct{}{}
	b.ReportAllocs()
	b.ResetTimer()
	for t := 0; t < b.N; t++ {
		ch <- d
	}
}

type Depth20 struct {
	EventTime    time.Time      `json:"-"`
	Symbol       string         `json:"s,omitempty"`
	LastUpdateId int64          `json:"u,omitempty"`
	Bids         [20][2]float64 `json:"b,omitempty"`
	Asks         [20][2]float64 `json:"a,omitempty"`
	ParseTime    time.Time      `json:"-"`
}

func (depth *Depth20) GetParseTime() time.Time {
	return depth.ParseTime
}

func (depth *Depth20) GetExchange() common.ExchangeID {
	return common.BinanceUsdtFuture
}

func (depth *Depth20) GetBids() common.Bids {
	return depth.Bids[:]
}
func (depth *Depth20) GetAsks() common.Asks {
	return depth.Asks[:]
}
func (depth *Depth20) GetSymbol() string {
	return depth.Symbol
}
func (depth *Depth20) GetEventTime() time.Time {
	return depth.EventTime
}


func BenchmarkChanDepthSendNil128Buffered(b *testing.B) {
	ch := make(chan common.Depth, 128)
	go func() {
		for {
			select {
			case <-ch:
			}
		}
	}()

	d := &Depth20{}
	b.ReportAllocs()
	b.ResetTimer()
	for t := 0; t < b.N; t++ {
		ch <- d
	}
}

func BenchmarkChanDepthMultipleSendNil128Buffered(b *testing.B) {
	ch1 := make(chan common.Depth, 128)
	ch2 := make(chan common.Depth, 128)
	ch3 := make(chan common.Depth, 128)
	ch4 := make(chan common.Depth, 128)
	ch5 := make(chan common.Depth, 128)
	go func() {
		for {
			select {
			case <-ch1:
			case <-ch2:
			case <-ch3:
			case <-ch4:
			case <-ch5:
			}
		}
	}()

	d := &Depth20{}
	b.ReportAllocs()
	b.ResetTimer()
	for t := 0; t < b.N; t++ {
		ch1 <- d
	}
}

func BenchmarkChanDepthMultipleRawSendNil128Buffered(b *testing.B) {
	ch1 := make(chan Depth20, 128)
	ch2 := make(chan Depth20, 128)
	ch3 := make(chan Depth20, 128)
	ch4 := make(chan Depth20, 128)
	ch5 := make(chan Depth20, 128)
	go func() {
		for {
			select {
			case <-ch1:
			case <-ch2:
			case <-ch3:
			case <-ch4:
			case <-ch5:
			}
		}
	}()

	d := Depth20{}
	b.ReportAllocs()
	b.ResetTimer()
	for t := 0; t < b.N; t++ {
		ch1 <- d
	}
}
