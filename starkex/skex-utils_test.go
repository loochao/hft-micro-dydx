package starkex

import (
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/geometrybase/hft-micro/logger"
	"testing"
)

func TestPiAsString(t *testing.T) {
	h := math.Decimal256{}
	err := h.UnmarshalText([]byte("0x02893294412a4c8f915f75892b395ebbf6859ec246ec365c3b1f56f47c3a0a5d"))
	if err != nil {
		t.Fatal(err)
	}
	s, err := h.MarshalText()
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%s", s)
}
