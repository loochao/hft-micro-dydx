package ftx_usdspot

import (
	"context"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"testing"
)

func TestNewUserWS(t *testing.T) {

	var ctx = context.Background()
	logger.Debugf("%s %s %s %s",
		os.Getenv("FTX_TEST_KEY"),
		os.Getenv("FTX_TEST_SECRET"),
		os.Getenv("FTX_TEST_SUBACCOUNT"),
		os.Getenv("FTX_TEST_PROXY"),
	)
	ws := NewUserWS(
		ctx,
		os.Getenv("FTX_TEST_KEY"),
		os.Getenv("FTX_TEST_SECRET"),
		os.Getenv("FTX_TEST_SUBACCOUNT"),
		os.Getenv("FTX_TEST_PROXY"),
	)
	for {
		select {
		case <-ws.Done():
			return
		}
	}
}
