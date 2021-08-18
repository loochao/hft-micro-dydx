package ftx_usdspot

import (
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"math"
	"time"
)

func ParseTicker(msg []byte, ticker *Ticker) (err error) {
	//{"channel": "ticker", "market": "DOGE-PERP", "type": "update", "data": {"bid": 0.278362, "ask": 0.2784135, "bidSize": 107.0, "askSize": 5600.0, "last": 0.2783695, "time": 1624183024.08771}} 189
	collectEnd := 33
	collectStart := collectEnd
	bytesLen := len(msg)

	currentKey := common.JsonKeySymbol
	for collectEnd < bytesLen-2 {
		switch currentKey {
		case common.JsonKeySymbol:
			if msg[collectEnd] == '"' {
				ticker.Symbol = common.UnsafeBytesToString(msg[collectStart:collectEnd])
				currentKey = common.JsonKeyBidPrice
				collectEnd += 37
				collectStart = collectEnd
				continue
			}
			break
		case common.JsonKeyBidPrice:
			if msg[collectEnd] == ',' {
				ticker.Bid, err = common.ParseDecimal(msg[collectStart:collectEnd])
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
				ticker.Ask, err = common.ParseDecimal(msg[collectStart:collectEnd])
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
				ticker.BidSize, err = common.ParseDecimal(msg[collectStart:collectEnd])
				if err != nil {
					return fmt.Errorf("JsonKeyAskPrice error %v start %d end %d %s", err, collectStart, collectEnd, msg[collectStart:collectEnd])
				}
				currentKey = common.JsonKeyAskSize
				collectEnd += 13
				collectStart = collectEnd
				continue
			}
			break
		case common.JsonKeyAskSize:
			if msg[collectEnd] == ',' {
				ticker.AskSize, err = common.ParseDecimal(msg[collectStart:collectEnd])
				if err != nil {
					return fmt.Errorf("JsonKeyAskPrice error %v start %d end %d %s", err, collectStart, collectEnd, msg[collectStart:collectEnd])
				}
				collectEnd = bytesLen - 24
				currentKey = common.JsonKeyEventTime
			}
			break
		case common.JsonKeyEventTime:
			if msg[collectEnd] == ':' {
				var t float64
				t, err = common.ParseDecimal(msg[collectEnd+2 : bytesLen-2])
				if err != nil {
					return
				}
				ticker.Time = time.Unix(int64(t), int64((t - math.Floor(t))*1e9))
				return
			}
		}
		collectEnd += 1
	}
	return
}
