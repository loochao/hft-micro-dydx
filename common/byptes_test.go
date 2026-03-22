package common

import (
	"bytes"
	"testing"
)

func BenchmarkBytesEqualInline(b *testing.B) {
	x := bytes.Repeat([]byte{'a'}, 1<<20)
	y := bytes.Repeat([]byte{'a'}, 1<<20)
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		if string(x) != string(y) {
			b.Fatal("x != y")
		}
	}
}

func BenchmarkBytesEqualExplicit(b *testing.B) {
	x := bytes.Repeat([]byte{'a'}, 1<<20)
	y := bytes.Repeat([]byte{'a'}, 1<<20)
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		q := string(x)
		r := string(y)
		if q != r {
			b.Fatal("x != y")
		}
	}
}


func BenchmarkBytesMapBytesKey(b *testing.B) {
	m := make(map[string]string)
	m["BTCUSDT"] = "BTCUSDT"
	bs := []byte("BTCUSDT")
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_ = m[string(bs)]
	}
}

func BenchmarkBytesMapStringKey(b *testing.B) {
	m := make(map[string]string)
	m["BTCUSDT"] = "BTCUSDT"
	bs := []byte("BTCUSDT")
	key := string(bs)
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_ = m[key]
	}
}
