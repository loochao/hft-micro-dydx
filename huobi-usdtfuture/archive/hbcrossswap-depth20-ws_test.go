package archive

import (
	"context"
	"github.com/geometrybase/hft-micro/logger"
	"testing"
)

func TestNewDepth20Websocket(t *testing.T) {
	var ctx = context.Background()
	ws := NewDepth20Websocket(ctx, []string{"FIL-USDT"}, "socks5://127.0.0.1:1081")
	for {
		select {
		case d := <-ws.DataCh:
			logger.Debugf("%v", d)
		}
	}
	//msg := `{"ch":"market.FIL-USDT.depth.step6","ts":1618772017322,"tick":{"mrid":18276766369,"id":1618772017,"bids":[[154.881,549],[154.88,55],[154.874,64],[154.86,37],[154.854,63],[154.841,74],[154.84,387],[154.833,300],[154.832,200],[154.831,7],[154.827,97],[154.816,100],[154.815,138],[154.813,13],[154.812,20],[154.805,383],[154.797,287],[154.794,82],[154.792,48],[154.787,200]],"asks":[[154.882,680],[154.883,375],[154.888,148],[154.905,46],[154.928,60],[154.947,93],[154.948,63],[154.956,2],[154.958,97],[154.959,162],[154.962,97],[154.968,42],[154.971,3],[154.982,20],[154.985,454],[155.01,17],[155.016,4],[155.02,97],[155.024,346],[155.028,145]],"ts":1618772017317,"version":1618772017,"ch":"market.FIL-USDT.depth.step6"}}}`
	//logger.Debugf("%d", len(`{"ch":"market.FIL-USDT.depth.step6","ts":`))
	//logger.Debugf("%s", msg[41:54])
}
