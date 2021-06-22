package benchmarks

import (
	"sync/atomic"
	"testing"
)

func BenchmarkPool(b *testing.B){
	i := int32(0)
	for !atomic.CompareAndSwapInt32(&i, 1, 0){
		//if atomic.LoadInt32(&i) == 1 {
		//
		//}
	}
}
