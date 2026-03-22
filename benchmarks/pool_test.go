package benchmarks

import (
	"github.com/geometrybase/hft-micro/common"
	"testing"
)

var poolRef []byte

func BenchmarkPool(b *testing.B){
	b.ReportAllocs()
	pool := [common.BufferSizeForHighLoadRealTimeData][]byte{}
	for i := range pool {
		pool[i] = make([]byte, 1024)
	}
	poolIndex := -1
	for i := 0; i < b.N; i++ {
		poolIndex ++
		if common.BufferSizeForHighLoadRealTimeData == poolIndex {
			poolIndex = 0
		}
		poolRef = pool[poolIndex]
	}
}

//:noescape
func BenchmarkWithoutPool(b *testing.B){
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		poolRef = make([]byte, 1024)
	}
}
