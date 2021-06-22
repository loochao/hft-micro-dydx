package benchmarks

import (
	"testing"
)


var mod *int

func BenchmarkMod(b *testing.B){
	v := 0
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		v ++
		v %= 1024
	}
	mod = &v
}

func BenchmarkCompare(b *testing.B){
	v := 0
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		v ++
		if v == 1024 {
			v -= 1024
		}
	}
	mod = &v
}