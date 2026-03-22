package binance_coinfuture

import (
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"time"
)

//{"stream":"bnbusd_perp@depth20@100ms","data":{"e":"depthUpdate","E":1623294719683,"T":1623294719651,"s":"BNBUSD_PERP","ps":"BNBUSD","U":137362346007,"u":137362346725,"pu":137362345003,"b":[["373.743","691"],["373.700","11209"],["373.643","300"],["373.642","500"],["373.629","180"],["373.628","200"],["373.621","4"],["373.611","7"],["373.610","3"],["373.608","750"],["373.603","1000"],["373.601","300"],["373.598","1"],["373.587","400"],["373.580","375"],["373.563","207"],["373.561","2176"],["373.556","300"],["373.536","1125"],["373.535","4"]],"a":[["373.744","843"],["373.763","210"],["373.764","200"],["373.765","200"],["373.781","1"],["373.797","115"],["373.812","300"],["373.823","34"],["373.825","310"],["373.826","410"],["373.844","180"],["373.845","455"],["373.846","750"],["373.857","21"],["373.858","500"],["373.866","50"],["373.878","619"],["373.879","1125"],["373.886","293"],["373.907","400"]]}}
func ParseDepth20(msg []byte, depth20 *Depth20) error {
	var err error
	msgLen := len(msg)
	msgEnd := msgLen - 2
	collectEnd := 60
	collectStart := collectEnd
	currentKey := common.JsonKeyUnknown
	counter := 0
	for collectEnd < msgEnd {
		switch currentKey {
		case common.JsonKeyBids:
			if msg[collectEnd] == '"' {
				depth20.Bids[counter/2][counter%2], err = common.ParseDecimal(msg[collectStart:collectEnd])
				if err != nil {
					return fmt.Errorf("common.ParseDecimal(msg[collectStart:collectEnd]) error %v start %d end %d. msg: %s", err, collectStart, collectEnd, msg[collectStart:collectEnd])
				}
				counter += 1
				if counter >= 40 {
					currentKey = common.JsonKeyUnknown
					collectEnd += 4
				} else if counter%2 == 0 {
					collectEnd += 5
					collectStart = collectEnd
				} else {
					collectEnd += 3
					collectStart = collectEnd
				}
				continue
			}
			break
		case common.JsonKeyAsks:
			if msg[collectEnd] == '"' {
				depth20.Asks[counter/2][counter%2], err = common.ParseDecimal(msg[collectStart:collectEnd])
				if err != nil {
					return fmt.Errorf("common.ParseDecimal(msg[collectStart:collectEnd]) error %v start %d end %d. msg: %s", err, collectStart, collectEnd, msg[collectStart:collectEnd])
				}
				counter += 1
				if counter >= 40 {
					currentKey = common.JsonKeyUnknown
					collectEnd += 4
				} else if counter%2 == 0 {
					collectEnd += 5
					collectStart = collectEnd
				} else {
					collectEnd += 3
					collectStart = collectEnd
				}
				continue
			}
			break
		case common.JsonKeyLastUpdateId:
			if msg[collectEnd] == ',' {
				depth20.LastUpdateId, err = common.ParseInt(msg[collectStart:collectEnd])
				if err != nil {
					return fmt.Errorf("common.ParseInt(msg[collectStart:collectEnd]) error %v start %d end %d. msg: %s", err, collectStart, collectEnd, msg[collectStart:collectEnd])
				}
				currentKey = common.JsonKeyUnknown
				collectEnd += 2
				continue
			}
			break
		case common.JsonKeySymbol:
			if msg[collectEnd] == '"' {
				depth20.Symbol = common.UnsafeBytesToString(msg[collectStart:collectEnd])
				currentKey = common.JsonKeyUnknown
				collectEnd += 3
				continue
			}
			break
		case common.JsonKeyUnknown:
			if msg[collectEnd] == 'E' && msg[collectEnd-1] == '"' && msg[collectEnd+1] == '"' && collectEnd+13 < msgLen {
				if collectEnd+3 >= msgLen || collectEnd+16 > msgLen {
					return fmt.Errorf("get event time index out of bound end %d start %d len %d. msg %s", collectEnd+3, collectEnd+16, msgLen, msg)
				}
				eventTime, err := common.ParseInt(msg[collectEnd+3 : collectEnd+16])
				if err != nil {
					return fmt.Errorf("common.ParseInt(msg[collectEnd+3 : collectEnd+16]) error %v %s", err, msg[collectEnd+3:collectEnd+16])
				}
				depth20.EventTime = time.Unix(0, eventTime*1000000)
				collectEnd += 17
				continue
			} else if msg[collectEnd] == 'u' && msg[collectEnd-1] == '"' && msg[collectEnd+1] == '"' {
				currentKey = common.JsonKeyLastUpdateId
				collectEnd += 3
				collectStart = collectEnd
				continue
			} else if msg[collectEnd] == 's' && msg[collectEnd-1] == '"' && msg[collectEnd+1] == '"' {
				currentKey = common.JsonKeySymbol
				collectEnd += 4
				collectStart = collectEnd
				collectEnd += 5 //symbol最短为SCUSD
				continue
			} else if msg[collectEnd] == 'b' && msg[collectEnd-1] == '"' && msg[collectEnd+1] == '"' {
				currentKey = common.JsonKeyBids
				collectEnd += 6
				collectStart = collectEnd
				counter = 0
				continue
			} else if msg[collectEnd] == 'a' && msg[collectEnd-1] == '"' && msg[collectEnd+1] == '"' {
				currentKey = common.JsonKeyAsks
				collectEnd += 6
				collectStart = collectEnd
				counter = 0
				continue
			}
		}
		collectEnd += 1
	}
	return nil
}

//{"stream":"bnbusd_perp@depth5@100ms","data":{"e":"depthUpdate","E":1623297648173,"T":1623297648166,"s":"BNBUSD_PERP","ps":"BNBUSD","U":137388060548,"u":137388063414,"pu":137388059926,"b":[["369.073","1564"],["369.034","6"],["369.033","115"],["369.031","34"],["369.017","400"]],"a":[["369.074","375"],["369.137","79"],["369.138","115"],["369.141","34"],["369.145","246"]]}}
func ParseDepth5(msg []byte, reuse *Depth5) error {
	var err error
	msgLen := len(msg)
	msgEnd := msgLen - 2
	collectEnd := 60
	collectStart := collectEnd
	currentKey := common.JsonKeyUnknown
	counter := 0
	for collectEnd < msgEnd {
		switch currentKey {
		case common.JsonKeyBids:
			if msg[collectEnd] == '"' {
				reuse.Bids[counter/2][counter%2], err = common.ParseDecimal(msg[collectStart:collectEnd])
				if err != nil {
					return fmt.Errorf("common.ParseDecimal(msg[collectStart:collectEnd]) error %v start %d end %d. msg: %s", err, collectStart, collectEnd, msg[collectStart:collectEnd])
				}
				counter += 1
				if counter >= 10 {
					currentKey = common.JsonKeyUnknown
					collectEnd += 4
				} else if counter%2 == 0 {
					collectEnd += 5
					collectStart = collectEnd
				} else {
					collectEnd += 3
					collectStart = collectEnd
				}
				continue
			}
			break
		case common.JsonKeyAsks:
			if msg[collectEnd] == '"' {
				reuse.Asks[counter/2][counter%2], err = common.ParseDecimal(msg[collectStart:collectEnd])
				if err != nil {
					return fmt.Errorf("common.ParseDecimal(msg[collectStart:collectEnd]) error %v start %d end %d. msg: %s", err, collectStart, collectEnd, msg[collectStart:collectEnd])
				}
				counter += 1
				if counter >= 10 {
					currentKey = common.JsonKeyUnknown
					collectEnd += 4
				} else if counter%2 == 0 {
					collectEnd += 5
					collectStart = collectEnd
				} else {
					collectEnd += 3
					collectStart = collectEnd
				}
				continue
			}
			break
		case common.JsonKeyLastUpdateId:
			if msg[collectEnd] == ',' {
				reuse.LastUpdateId, err = common.ParseInt(msg[collectStart:collectEnd])
				if err != nil {
					return fmt.Errorf("common.ParseInt(msg[collectStart:collectEnd]) error %v start %d end %d. msg: %s", err, collectStart, collectEnd, msg[collectStart:collectEnd])
				}
				currentKey = common.JsonKeyUnknown
				collectEnd += 2
				continue
			}
			break
		case common.JsonKeySymbol:
			if msg[collectEnd] == '"' {
				reuse.Symbol = common.UnsafeBytesToString(msg[collectStart:collectEnd])
				currentKey = common.JsonKeyUnknown
				collectEnd += 3
				continue
			}
			break
		case common.JsonKeyUnknown:
			if msg[collectEnd] == 'E' && msg[collectEnd-1] == '"' && msg[collectEnd+1] == '"' && collectEnd+13 < msgLen {
				if collectEnd+3 >= msgLen || collectEnd+16 > msgLen {
					return fmt.Errorf("get event time index out of bound end %d start %d len %d. msg %s", collectEnd+3, collectEnd+16, msgLen, msg)
				}
				eventTime, err := common.ParseInt(msg[collectEnd+3 : collectEnd+16])
				if err != nil {
					return fmt.Errorf("common.ParseInt(msg[collectEnd+3 : collectEnd+16]) error %v %s", err, msg[collectEnd+3:collectEnd+16])
				}
				reuse.EventTime = time.Unix(0, eventTime*1000000)
				collectEnd += 17
				continue
			} else if msg[collectEnd] == 'u' && msg[collectEnd-1] == '"' && msg[collectEnd+1] == '"' {
				currentKey = common.JsonKeyLastUpdateId
				collectEnd += 3
				collectStart = collectEnd
				continue
			} else if msg[collectEnd] == 's' && msg[collectEnd-1] == '"' && msg[collectEnd+1] == '"' {
				currentKey = common.JsonKeySymbol
				collectEnd += 4
				collectStart = collectEnd
				collectEnd += 5 //symbol最短为SCUSD
				continue
			} else if msg[collectEnd] == 'b' && msg[collectEnd-1] == '"' && msg[collectEnd+1] == '"' {
				currentKey = common.JsonKeyBids
				collectEnd += 6
				collectStart = collectEnd
				counter = 0
				continue
			} else if msg[collectEnd] == 'a' && msg[collectEnd-1] == '"' && msg[collectEnd+1] == '"' {
				currentKey = common.JsonKeyAsks
				collectEnd += 6
				collectStart = collectEnd
				counter = 0
				continue
			}
		}
		collectEnd += 1
	}
	return nil
}
