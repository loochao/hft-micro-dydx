package common

import (
	"github.com/stretchr/testify/assert"
	"math/rand"
	"sort"
	"testing"
)

var m = make(map[string]int)
func init() {
	for i, symbol := range BnSymbols {
		m[symbol] = i
	}
}

func TestGetSymbolIndex(t *testing.T) {
	for i, symbol := range BnSymbols {
		assert.Equal(t, i, GetSymbolIndex(symbol))
	}
}

func BenchmarkGetValueBySymbolIndex(b *testing.B) {
	b.ReportAllocs()
	symbol := BnSymbols[rand.Intn(len(BnSymbols))]
	for n := 0; n < b.N; n++ {
		i := GetSymbolIndex(symbol)
		_ = BnSymbols[i]
	}
}

func BenchmarkGetValueByMapKey(b *testing.B) {
	b.ReportAllocs()
	symbol := BnSymbols[rand.Intn(len(BnSymbols))]
	for n := 0; n < b.N; n++ {
		i := m[symbol]
		_ = BnSymbols[i]
	}
}

func BenchmarkGetValueByBinarySearch(b *testing.B) {
	b.ReportAllocs()
	symbol := BnSymbols[rand.Intn(len(BnSymbols))]
	for n := 0; n < b.N; n++ {
		i := sort.SearchStrings(BnSymbols, symbol)
		_ = BnSymbols[i]
	}
}
