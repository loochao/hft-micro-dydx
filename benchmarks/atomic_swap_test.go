package benchmarks

import (
	"sync"
	"sync/atomic"
	"testing"
)

func BenchmarkAtomicLoad(b *testing.B) {
	a := new(int32)
	*a = 0
	go func() {
		for{
			atomic.LoadInt32(a)
		}
	}()
	b.ReportAllocs()
	b.ResetTimer()
	for t := int32(0); t < int32(b.N); t++ {
		atomic.StoreInt32(a, t)
	}
}


func BenchmarkAtomic(b *testing.B) {
	a := new(int32)
	*a = 0
	b.ReportAllocs()
	b.ResetTimer()
	for t := 0; t < b.N; t++ {
		atomic.LoadInt32(a)
	}
}

func BenchmarkMutex(b *testing.B) {
	mu := sync.Mutex{}
	b.ReportAllocs()
	b.ResetTimer()
	for t := 0; t < b.N; t++ {
		mu.Lock()
		mu.Unlock()
	}
}