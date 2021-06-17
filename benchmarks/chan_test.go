package benchmarks

import (
	"github.com/geometrybase/hft-micro/logger"
	"testing"
	"time"
)

func TestChanDefaultLen(t *testing.T) {
	ch := make(chan time.Time)
	done := make(chan int)
	go func() {
		delta := time.Duration(0)
		for {
			select {
			case <- done:
				logger.Debugf("%d", delta.Nanoseconds()/100000)
				return
			case t := <-ch:
				delta += time.Now().Sub(t)
			}
		}
	}()
	for i := 0; i < 100000; i++ {
		ch <- time.Now()
	}
	close(done)
	time.Sleep(time.Second)
}

func TestChanDefaultLen1(t *testing.T) {
	ch := make(chan time.Time, 1)
	done := make(chan int)
	go func() {
		delta := time.Duration(0)
		for {
			select {
			case <- done:
				logger.Debugf("%d", delta.Nanoseconds()/100000)
				return
			case t := <-ch:
				delta += time.Now().Sub(t)
			}
		}
	}()
	for i := 0; i < 100000; i++ {
		ch <- time.Now()
	}
	close(done)
	time.Sleep(time.Second)
}

func TestChanDefaultLen128(t *testing.T) {
	ch := make(chan time.Time, 128)
	done := make(chan int)
	go func() {
		delta := time.Duration(0)
		for {
			select {
			case <- done:
				logger.Debugf("%d", delta.Nanoseconds()/100000)
				return
			case t := <-ch:
				delta += time.Now().Sub(t)
			}
		}
	}()
	for i := 0; i < 100000; i++ {
		ch <- time.Now()
	}
	close(done)
	time.Sleep(time.Second)
}

func BenchmarkChanDefaultLen(b *testing.B) {
	ch := make(chan time.Time)
	done := make(chan int)
	go func() {
		delta := time.Duration(0)
		defer logger.Debugf("%v", delta)
		for {
			select {
			case <- done:
				return
			case t := <-ch:
				delta += time.Now().Sub(t)
			}
		}
	}()
	for i := 0; i < 10000; i++ {
		ch <- time.Now()
	}
}

func BenchmarkChanLen1(b *testing.B) {
	ch := make(chan int, 1)
	go func() {
		for {
			select {
			case <-ch:
			}
		}
	}()
	for i := 0; i < b.N; i++ {
		ch <- 0
	}
}

func BenchmarkChanLen128(b *testing.B) {
	ch := make(chan int, 128)
	go func() {
		for {
			select {
			case <-ch:
			}
		}
	}()
	for i := 0; i < b.N; i++ {
		ch <- 0
	}
}