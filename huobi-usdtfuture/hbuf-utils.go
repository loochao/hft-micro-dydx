package huobi_usdtfuture

import (
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"time"
	"unsafe"
)

//{"ch":"market.BTC-USDT.depth.step6","ts":1618410970115,"tick":{"mrid":28158325357,"id":1618410970,"bids":[[63402.5,88],[63402.2,6],[63402.1,42],[63401.4,50],[63400.7,24],[63400,238],[63398.9,1],[63398.6,31],[63398.5,300],[63397.3,39],[63397.1,200],[63397,115],[63396.3,51],[63394.6,200],[63393.2,1000],[63392.5,1],[63392,177],[63391.6,115],[63391.5,115],[63391.4,115]],"asks":[[63402.6,20318],[63402.8,46],[63405,1583],[63405.2,300],[63406.7,108],[63406.8,484],[63406.9,325],[63407,58],[63407.1,1120],[63407.2,16590],[63407.3,1016],[63407.4,797],[63407.5,270],[63407.6,753],[63407.7,1178],[63407.8,521],[63407.9,330],[63408,170],[63408.1,1064],[63408.2,606]],"ts":1618410970112,"version":1618410970,"ch":"market.BTC-USDT.depth.step6"}}
func ParseDepth20(bytes []byte, orderBook *Depth20) (err error) {
	orderBook.Bids = [20][2]float64{}
	orderBook.Asks = [20][2]float64{}
	orderBook.ParseTime = time.Now()
	if bytes[12] != 't' && bytes[13] != '.' {
		return fmt.Errorf("bad bytes %s", bytes)
	}
	offset := 14
	collectStart := offset
	bytesLen := len(bytes)
	counter := 0
	currentKey := common.JsonKeySymbol
	for offset < bytesLen-6 {
		switch currentKey {
		case common.JsonKeyBids:
			if bytes[offset] == ',' || bytes[offset] == ']' {
				orderBook.Bids[counter/2][counter%2], err = common.ParseFloat(bytes[collectStart:offset])
				if err != nil {
					return fmt.Errorf("JsonKeyBids error %v mainLoop %d end %d %s", err, collectStart, offset, bytes[collectStart:offset])
				}
				counter += 1
				if counter >= 40 || bytes[offset+1] == ']' {
					currentKey = common.JsonKeyAsks
					offset += 12
					collectStart = offset
					counter = 0
				} else if counter%2 == 0 {
					offset += 3
					collectStart = offset
				} else {
					offset += 1
					collectStart = offset
				}
				continue
			}
			break
		case common.JsonKeyAsks:
			if bytes[offset] == ',' || bytes[offset] == ']' {
				orderBook.Asks[counter/2][counter%2], err = common.ParseFloat(bytes[collectStart:offset])
				if err != nil {
					return fmt.Errorf("JsonKeyAsks error %v mainLoop %d end %d %s", err, collectStart, offset, bytes[collectStart:offset])
				}
				counter += 1
				if counter >= 40 || bytes[offset+1] == ']' {
					currentKey = common.JsonKeyEventTime
					offset += 8
					collectStart = offset
					counter = 0
				} else if counter%2 == 0 {
					offset += 3
					collectStart = offset
				} else {
					offset += 1
					collectStart = offset
				}
				continue
			}
			break
		case common.JsonKeyVersion:
			if bytes[offset] == ',' {
				orderBook.Version, err = common.ParseInt(bytes[collectStart:offset])
				if err != nil {
					return fmt.Errorf("JsonKeyVersion error %v mainLoop %d end %d %s", err, collectStart, offset, bytes[collectStart:offset])
				}
				offset = bytesLen
				continue
			}
			break
		case common.JsonKeyEventTime:
			offset += 13
			timestamp, err := common.ParseInt(bytes[collectStart:offset])
			if err != nil {
				return fmt.Errorf("JsonKeyEventTime error %v mainLoop %d end %d %s", err, collectStart, offset, bytes[collectStart:offset])
			}
			orderBook.EventTime = time.Unix(0, timestamp*1000000)
			offset += 11
			collectStart = offset
			currentKey = common.JsonKeyVersion
			continue
		case common.JsonKeyID:
			if bytes[offset] == ',' {
				orderBook.ID, err = common.ParseInt(bytes[collectStart:offset])
				if err != nil {
					return fmt.Errorf("JsonKeyID error %v mainLoop %d end %d %s", err, collectStart, offset, bytes[collectStart:offset])
				}
				offset += 10
				collectStart = offset
				currentKey = common.JsonKeyBids
				counter = 0
			}
			break
		case common.JsonKeyMRID:
			if bytes[offset] == ',' {
				orderBook.MRID, err = common.ParseInt(bytes[collectStart:offset])
				if err != nil {
					return fmt.Errorf("JsonKeyMRID error %v mainLoop %d end %d %s", err, collectStart, offset, bytes[collectStart:offset])
				}
				offset += 6
				collectStart = offset
				currentKey = common.JsonKeyID
			}
			break
		case common.JsonKeySymbol:
			if bytes[offset] == '.' {
				symbol := bytes[collectStart:offset]
				orderBook.Symbol = *(*string)(unsafe.Pointer(&symbol))
				offset += 48
				collectStart = offset
				currentKey = common.JsonKeyMRID
			}
			break
		}
		offset += 1
	}
	return nil
}

//{"ch":"market.1INCH-USDT.bbo","ts":1626480000472,"tick":{"mrid":13218587529,"id":1626480000,"bid":[1.9631,10],"ask":[1.9641,38],"ts":1626480000472,"version":13218587529,"ch":"market.1INCH-USDT.bbo"}}
func ParseTicker(msg []byte, ticker *Ticker) (err error) {
	ticker.Bid = [2]float64{}
	ticker.Ask = [2]float64{}
	bytesLen := len(msg)
	if msg[2] == 'c' && bytesLen > 57 {
		if msg[32] == ':' {
			ticker.Symbol = common.UnsafeBytesToString(msg[14:22])
		} else if msg[33] == ':' {
			ticker.Symbol = common.UnsafeBytesToString(msg[14:23])
		} else if msg[34] == ':' {
			ticker.Symbol = common.UnsafeBytesToString(msg[14:24])
		} else if msg[31] == ':' {
			ticker.Symbol = common.UnsafeBytesToString(msg[14:21])
		} else if msg[35] == ':' {
			ticker.Symbol = common.UnsafeBytesToString(msg[14:25])
		} else {
			return fmt.Errorf("bad msg, can't find timestamp: %s", msg)
		}
	} else {
		return fmt.Errorf("bad msg, too short, %s", msg)
	}

	offset := 80
	collectStart := offset
	currentKey := common.JsonKeyUnknown
	var ts int64
	for offset < bytesLen-20 {
		switch currentKey {
		case common.JsonKeyBids:
			if msg[offset] == ',' {
				ticker.Bid[0], err = common.ParseDecimal(msg[collectStart:offset])
				if err != nil {
					return fmt.Errorf("JsonKeyBids error %v mainLoop %d end %d %s", err, collectStart, offset, msg[collectStart:offset])
				}
				offset += 1
				collectStart = offset
			} else if msg[offset] == ']' {
				ticker.Bid[1], err = common.ParseDecimal(msg[collectStart:offset])
				if err != nil {
					return fmt.Errorf("JsonKeyBids error %v mainLoop %d end %d %s", err, collectStart, offset, msg[collectStart:offset])
				}
				offset += 9
				collectStart = offset
				currentKey = common.JsonKeyAsks
			}
			break
		case common.JsonKeyAsks:
			if msg[offset] == ',' {
				ticker.Ask[0], err = common.ParseDecimal(msg[collectStart:offset])
				if err != nil {
					return fmt.Errorf("JsonKeyAsks error %v mainLoop %d end %d %s", err, collectStart, offset, msg[collectStart:offset])
				}
				offset += 1
				collectStart = offset
			} else if msg[offset] == ']' {
				ticker.Ask[1], err = common.ParseDecimal(msg[collectStart:offset])
				if err != nil {
					return fmt.Errorf("JsonKeyAsks error %v mainLoop %d end %d %s", err, collectStart, offset, msg[collectStart:offset])
				}
				collectStart = offset + 7
				offset += 20
				ts, err = common.ParseInt(msg[collectStart:offset])
				if err != nil {
					return fmt.Errorf("timestamp error %v mainLoop %d end %d %s", err, collectStart, offset, msg[collectStart:offset])
				}
				ticker.EventTime = time.Unix(0, ts*1000000)
				return
			}
			break
		case common.JsonKeyUnknown:
			if msg[offset] == 'b' && msg[offset+2] == 'd' {
				currentKey = common.JsonKeyBids
				offset += 6
				collectStart = offset
			}
			break
		}
		offset += 1
	}
	return fmt.Errorf("bad msg %s", msg)
}
