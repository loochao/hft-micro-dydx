package starkex_test

import (
	"fmt"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/geometrybase/hft-micro/starkex"
	"math/big"
	"math/rand"
	"testing"
)


// randInt returns a pseudo-random Int in the range [1<<(size-1), (1<<size) - 1]
func randInt(r *rand.Rand, size uint) *big.Int {
	n := new(big.Int).Lsh(starkex.IntOne, size-1)
	x := new(big.Int).Rand(r, n)
	return x.Add(x, n) // make sure result > 1<<(size-1)
}

func benchmarkDiv(b *testing.B, aSize, bSize int) {
	var r = rand.New(rand.NewSource(1234))
	aa := randInt(r, uint(aSize))
	bb := randInt(r, uint(bSize))
	if aa.Cmp(bb) < 0 {
		aa, bb = bb, aa
	}
	x := new(big.Int)
	y := new(big.Int)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		x.DivMod(aa, bb, y)
	}
}

func BenchmarkDiv(b *testing.B) {
	sizes := []int{
		10, 20, 50, 100, 200, 500, 1000,
		1e4, 1e5, 1e6, 1e7,
	}
	for _, i := range sizes {
		j := 2 * i
		b.Run(fmt.Sprintf("%d/%d", j, i), func(b *testing.B) {
			benchmarkDiv(b, j, i)
		})
	}
}

func benchmarkDivMod(b *testing.B, aSize, bSize int) {
	var r = rand.New(rand.NewSource(1234))
	aa := randInt(r, uint(aSize))
	bb := randInt(r, uint(bSize))
	if aa.Cmp(bb) < 0 {
		aa, bb = bb, aa
	}
	//x := new(big.Int)
	y := new(big.Int)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = starkex.DivMod(aa, bb, y)
	}
}


func BenchmarkDivMod(b *testing.B) {
	sizes := []int{
		10, 20, 50, 100, 200, 500, 1000,
		1e4, 1e5, 1e6, 1e7,
	}
	for _, i := range sizes {
		j := 2 * i
		b.Run(fmt.Sprintf("%d/%d", j, i), func(b *testing.B) {
			benchmarkDivMod(b, j, i)
		})
	}
}

func TestDivMod2(t *testing.T) {
	a := big.NewInt(100)
	b := big.NewInt(7)
	p := big.NewInt(3)
	p2 := big.NewInt(3)
	logger.Debugf("a %s b %s p %s", a, b, p)
	//p.ModInverse()
	d1, d2 := new(big.Int).DivMod(a, b, p)
	logger.Debugf("%s %s", d1, d2)
	logger.Debugf("a %s b %s p %s", a, b, p2)
	d3, err := starkex.DivMod(a, b, p2)
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%s", d3)
}
