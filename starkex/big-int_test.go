package starkex_test

import (
	"fmt"
	"github.com/geometrybase/hft-micro/starkex"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

func TestNewBigIntFromString(t *testing.T) {
	v1, _ := starkex.ParseBig256("0x02893294412a4c8f915f75892b395ebbf6859ec246ec365c3b1f56f47c3a0a5d")
	v2, _ := new(big.Int).SetString("02893294412a4c8f915f75892b395ebbf6859ec246ec365c3b1f56f47c3a0a5d", 16)
	v3, _ := new(big.Int).SetString("1147032829293317481173155891309375254605214077236177772270270553197624560221", 10)
	assert.Equal(t, 0, v1.Cmp(v2))
	assert.Equal(t, 0, v2.Cmp(v3))
	fmt.Printf("%s\n%s\n%s\n", v1, v2, v3)
}
