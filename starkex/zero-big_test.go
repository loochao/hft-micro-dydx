package starkex

import (
	"math/big"
	"testing"
)

func BenchmarkCompare(b *testing.B) {
	zero := big.NewInt(0)
	i1 := big.NewInt(1)
	i2 := big.NewInt(0)
	for i := 0; i < b.N; i++ {
		if i1.Cmp(zero) == 0 {
		}
		if i2.Cmp(zero) == 0 {
		}
	}
}

func BenchmarkBits(b *testing.B) {
	i1 := big.NewInt(1)
	i2 := big.NewInt(0)
	for i := 0; i < b.N; i++ {
		if len(i1.Bits()) == 0 {
		}
		if len(i2.Bits()) == 0 {
		}
	}
}

func BenchmarkBitLen(b *testing.B) {
	i1 := big.NewInt(1)
	i2 := big.NewInt(0)
	for i := 0; i < b.N; i++ {
		if i1.BitLen() == 0 {
		}
		if i2.BitLen() == 0 {
		}
	}
}
