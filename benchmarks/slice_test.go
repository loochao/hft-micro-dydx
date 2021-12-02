package benchmarks

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSliceShadowCopy(t *testing.T) {
	a := []int{0, 1, 2, 3, 4, 5, 6, 7}
	b := a[:5]
	assert.Equal(t, a[0], b[0])
	a[0] = 100
	assert.Equal(t, a[0], b[0])
}

func BenchmarkReSlicing(b *testing.B) {
	var c []int
	a := [8]int{0, 1, 2, 3, 4, 5, 6, 7}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		c = a[:]
	}
	_ = c
}

func BenchmarkSliceClear(b *testing.B) {
	var c []int
	a := [8]int{0, 1, 2, 3, 4, 5, 6, 7}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		c = a[:0]
	}
	_ = c
}

func BenchmarkSliceClearWithNewEmpty(b *testing.B) {
	var c []int
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		c = make([]int, 0)
	}
	_ = c
}