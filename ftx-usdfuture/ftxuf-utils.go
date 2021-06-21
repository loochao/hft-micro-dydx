package ftx_usdfuture

import (
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)


func ParseTicker(msg []byte, trade *Ticker) (err error){
	//{"channel": "ticker", "market": "DOGE-PERP", "type": "update", "data": {"bid": 0.278362, "ask": 0.2784135, "bidSize": 107.0, "askSize": 5600.0, "last": 0.2783695, "time": 1624183024.08771}} 189
	collectEnd := 33
	collectStart := collectEnd
	bytesLen := len(msg)
	var t float64
	t, err = common.ParseDecimal(msg[bytesLen-18:bytesLen-2])
	if err != nil {
		return
	}
	trade.Time = time.Unix(0, int64(t*1e9))
	currentKey := common.JsonKeySymbol
	for collectEnd < bytesLen-2 {
		switch currentKey {
		case common.JsonKeySymbol:
			if msg[collectEnd] == '"' {
				logger.Debugf("%s", msg[collectStart:collectEnd])
				trade.Symbol = common.UnsafeBytesToString(msg[collectStart:collectEnd])
				currentKey = common.JsonKeyBidPrice
				collectEnd += 37
				collectStart = collectEnd
				continue
			}
			break
		case common.JsonKeyBidPrice:
			if msg[collectEnd] == ',' {
				logger.Debugf("%s", msg[collectStart:collectEnd])
				trade.Bid, err = common.ParseDecimal(msg[collectStart:collectEnd])
				if err != nil {
					return fmt.Errorf("JsonKeyBidPrice error %v start %d end %d %s", err, collectStart, collectEnd, msg[collectStart:collectEnd])
				}
				currentKey = common.JsonKeyAskPrice
				collectEnd += 9
				collectStart = collectEnd
				continue
			}
			break
		case common.JsonKeyAskPrice:
			if msg[collectEnd] == ',' {
				logger.Debugf("%s", msg[collectStart:collectEnd])
				trade.Ask, err = common.ParseDecimal(msg[collectStart:collectEnd])
				if err != nil {
					return fmt.Errorf("JsonKeyAskPrice error %v start %d end %d %s", err, collectStart, collectEnd, msg[collectStart:collectEnd])
				}
				currentKey = common.JsonKeyBidSize
				collectEnd += 13
				collectStart = collectEnd
				continue
			}
			break
		case common.JsonKeyBidSize:
			if msg[collectEnd] == ',' {
				logger.Debugf("%s", msg[collectStart:collectEnd])
				trade.BidSize, err = common.ParseDecimal(msg[collectStart:collectEnd])
				if err != nil {
					return fmt.Errorf("JsonKeyAskPrice error %v start %d end %d %s", err, collectStart, collectEnd, msg[collectStart:collectEnd])
				}
				currentKey = common.JsonKeyBidSize
				collectEnd += 13
				collectStart = collectEnd
				continue
			}
			break
		case common.JsonKeyAskSize:
			if msg[collectEnd] == ',' {
				logger.Debugf("%s", msg[collectStart:collectEnd])
				trade.AskSize, err = common.ParseDecimal(msg[collectStart:collectEnd])
				if err != nil {
					return fmt.Errorf("JsonKeyAskPrice error %v start %d end %d %s", err, collectStart, collectEnd, msg[collectStart:collectEnd])
				}
				currentKey = common.JsonKeyBidSize
				collectEnd += 13
				collectStart = collectEnd
				return
			}
			break
		}
		collectEnd += 1
	}
	return
}
