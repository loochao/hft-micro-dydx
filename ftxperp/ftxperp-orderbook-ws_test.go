package ftxperp

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"os"
	"testing"
)

func TestNewOrderBookWS(t *testing.T) {
	var ctx = context.Background()
	symbols := []string{"LTC-PERP", "ETH-PERP", "DOGE-PERP", "WAVES-PERP"}
	channels := make(map[string]chan common.Depth)
	for _, symbol := range symbols {
		channels[symbol] = make(chan common.Depth, 100)
	}
	_ = NewOrderBookWS(ctx, os.Getenv("FTX_TEST_PROXY"), channels)
	for {
		select {
		case _ = <-channels[symbols[0]]:
		case _ = <-channels[symbols[1]]:
		case _ = <-channels[symbols[2]]:
		case _ = <-channels[symbols[3]]:
		}
	}
}
