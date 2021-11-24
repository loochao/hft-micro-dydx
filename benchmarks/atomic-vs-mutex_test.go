package benchmarks

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
)

func TestDataRaceRange(t *testing.T) {
	i := 0.0
	go func() {
		for {
			i = rand.Float64()
		}
	}()
	var wg sync.WaitGroup
	for n := 0; n < 40; n++ {
		wg.Add(1)
		go func() {
			for n := 0; n < 10000; n++ {
				if i < 0 || i > 1 {
					t.Fatalf("bad i value %f", i)
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func TestDataRace(t *testing.T) {
	type config struct {
		a []int
	}
	cfg := &config{}
	go func() {
		i := 0
		for {
			i++
			cfg.a = []int{i, i + 1, i + 2, i + 3, i + 4, i + 5, i + 6}
		}
	}()
	var wg sync.WaitGroup
	for n := 0; n < 4; n++ {
		wg.Add(1)
		go func() {
			for n := 0; n < 100; n++ {
				fmt.Printf("%v\n", cfg)
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func TestDataRaceMutex(t *testing.T) {
	type config struct {
		a []int
	}
	lock := sync.RWMutex{}
	cfg := &config{}
	go func() {
		i := 0
		for {
			i++
			lock.Lock()
			cfg.a = []int{i, i + 1, i + 2, i + 3, i + 4, i + 5, i + 6}
			lock.Unlock()
		}
	}()
	var wg sync.WaitGroup
	for n := 0; n < 4; n++ {
		wg.Add(1)
		go func() {
			for n := 0; n < 100; n++ {
				lock.RLock()
				fmt.Printf("%v\n", cfg)
				lock.RUnlock()
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func TestDataRaceAtomic(t *testing.T) {
	type config struct {
		a []int
	}
	v := atomic.Value{}
	go func() {
		i := 0
		for {
			i++
			cfg := &config{
				a: []int{i, i + 1, i + 2, i + 3, i + 4, i + 5, i + 6},
			}
			v.Store(cfg)
		}
	}()
	var wg sync.WaitGroup
	for n := 0; n < 4; n++ {
		wg.Add(1)
		go func() {
			for n := 0; n < 100; n++ {
				cfg := v.Load()
				fmt.Printf("%v\n", cfg)
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func BenchmarkOneWriterMultipleReader(b *testing.B) {
	type config struct {
		a []int
	}
	cfg := &config{
		a: []int{0, 0, 0, 0, 0, 0, 0},
	}
	go func() {
		i := 0
		for {
			i++
			cfg.a = []int{i, i + 1, i + 2, i + 3, i + 4, i + 5, i + 6}
		}
	}()
	var wg sync.WaitGroup
	for n := 0; n < 4; n++ {
		wg.Add(1)
		go func() {
			for n := 0; n < b.N; n++ {
				_ = cfg.a[0]
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func BenchmarkMutexOneWriterMultipleReader(b *testing.B) {
	type config struct {
		a []int
	}
	cfg := &config{
		a: []int{0, 0, 0, 0, 0, 0, 0},
	}
	lock := sync.RWMutex{}
	go func() {
		i := 0
		for {
			i++
			lock.Lock()
			cfg.a = []int{i, i + 1, i + 2, i + 3, i + 4, i + 5, i + 6}
			lock.Unlock()
		}
	}()
	var wg sync.WaitGroup
	for n := 0; n < 4; n++ {
		wg.Add(1)
		go func() {
			for n := 0; n < b.N; n++ {
				lock.RLock()
				_ = cfg.a[0]
				lock.RUnlock()
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func BenchmarkAtomicOneWriterMultipleReader(b *testing.B) {
	var v atomic.Value
	type config struct {
		a []int
	}
	cfg := &config{
		a: []int{0, 0, 0, 0, 0, 0, 0},
	}
	v.Store(cfg)
	go func() {
		i := 0
		for {
			i++
			cfg.a = []int{i, i + 1, i + 2, i + 3, i + 4, i + 5, i + 6}
			v.Store(cfg)
		}
	}()
	var wg sync.WaitGroup
	for n := 0; n < 4; n++ {
		wg.Add(1)
		go func() {
			for n := 0; n < b.N; n++ {
				cfg := v.Load().(*config)
				_ = cfg.a[0]
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func BenchmarkMutexNotification(b *testing.B) {

	lock := sync.Mutex{}
	flag := 0
	lastFlag := 0
	go func() {
		for {
			lock.Lock()
			flag++
			lock.Unlock()
		}
	}()

	for n := 0; n < b.N; n++ {
		for {
			lock.Lock()
			if flag != lastFlag {
				lastFlag = flag
				lock.Unlock()
				break
			}
			lock.Unlock()
		}
	}

}

func BenchmarkAtomicNotification(b *testing.B) {

	flag := uint32(0)
	go func() {
		for {
			atomic.CompareAndSwapUint32(&flag, 0, 1)
		}
	}()

	for n := 0; n < b.N; n++ {
		for {
			if atomic.CompareAndSwapUint32(&flag, 1, 0) {
				break
			}
		}
	}

}
