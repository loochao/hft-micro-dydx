package main

import (
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
)

var m = make(map[string]int)

func init() {
	for i, symbol := range SYMBOLS {
		m[symbol] = i
	}
}

func TestGetSymbolIndex(t *testing.T) {
	for i, symbol := range SYMBOLS {
		assert.Equal(t, i, GetSymbolIndex(symbol))
	}
}

func BenchmarkGetValueBySymbolIndex(b *testing.B) {
	symbol := SYMBOLS[rand.Intn(SYMBOLS_LEN)]
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		i := GetSymbolIndex(symbol)
		_ = SYMBOLS[i]
	}
}

func BenchmarkGetValueByMapKey(b *testing.B) {
	symbol := SYMBOLS[rand.Intn(SYMBOLS_LEN)]
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		i := m[symbol]
		_ = SYMBOLS[i]
	}
}

