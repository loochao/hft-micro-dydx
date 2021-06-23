package binance_usdtspot

import (
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"strings"
	"time"
	"unsafe"
)

func ParseDepth20(bytes []byte, depth20 *Depth20) (err error) {
	//{"stream":"flmusdt@depth20@100ms","data":{"lastUpdateId":165284515,"bids":[["0.48560000","2036.02000000"],["0.48550000","480.00000000"],["0.48520000","14257.67000000"],["0.48510000","1056.25000000"],["0.48500000","1894.32000000"],["0.48480000","2145.67000000"],["0.48460000","2196.59000000"],["0.48330000","3000.00000000"],["0.48320000","2531.26000000"],["0.48310000","21.18000000"],["0.48300000","4292.54000000"],["0.48270000","5042.00000000"],["0.48240000","5051.00000000"],["0.48230000","24.83000000"],["0.48220000","457.11000000"],["0.48200000","4142.12000000"],["0.48160000","31.15000000"],["0.48150000","71.96000000"],["0.48130000","1284.94000000"],["0.48110000","1098.85000000"]],"asks":[["0.48630000","5601.00000000"],["0.48650000","990.00000000"],["0.48670000","7816.00000000"],["0.48680000","7914.96000000"],["0.48690000","963.00000000"],["0.48720000","3640.00000000"],["0.48730000","814.24000000"],["0.48780000","3560.00000000"],["0.48800000","1029.00000000"],["0.48880000","13221.24000000"],["0.48940000","3000.00000000"],["0.48980000","62.75000000"],["0.49000000","1482.94000000"],["0.49040000","516.34000000"],["0.49110000","46.50000000"],["0.49120000","27.10000000"],["0.49130000","31.03000000"],["0.49150000","66.27000000"],["0.49160000","1291.65000000"],["0.49190000","159.76000000"]]}}
	depth20.ParseTime = time.Now()
	offset := 2
	collectStart := offset
	bytesLen := len(bytes)
	currentKey := common.JsonKeyUnknown
	counter := 0
	for offset < bytesLen-4 {
		switch currentKey {
		case common.JsonKeyBids:
			if bytes[offset] == '"' {
				depth20.Bids[counter/2][counter%2], err = common.ParseDecimal(bytes[collectStart:offset])
				if err != nil {
					return  fmt.Errorf("JsonKeyBids error %v mainLoop %d end %d %s", err, collectStart, offset, bytes[collectStart:offset])
				}
				counter += 1
				if counter >= 40 {
					currentKey = common.JsonKeyUnknown
					offset += 4
				} else if counter%2 == 0 {
					offset += 5
					collectStart = offset
				} else {
					offset += 3
					collectStart = offset
				}
				continue
			}
			break
		case common.JsonKeyAsks:
			if bytes[offset] == '"' {
				depth20.Asks[counter/2][counter%2], err = common.ParseDecimal(bytes[collectStart:offset])
				if err != nil {
					return  fmt.Errorf("JsonKeyAsks error %v mainLoop %d end %d %s", err, collectStart, offset, bytes[collectStart:offset])
				}
				counter += 1
				if counter >= 40 {
					currentKey = common.JsonKeyUnknown
					offset += 4
				} else if counter%2 == 0 {
					offset += 5
					collectStart = offset
				} else {
					offset += 3
					collectStart = offset
				}
				continue
			}
			break
		case common.JsonKeyLastUpdateId:
			if bytes[offset] == ',' {
				depth20.LastUpdateId, err = common.ParseInt(bytes[collectStart:offset])
				if err != nil {
					return  fmt.Errorf("JsonKeyLastUpdateId error %v mainLoop %d end %d %s", err, collectStart, offset, bytes[collectStart:offset])
				}
				currentKey = common.JsonKeyUnknown
				offset += 2
				continue
			}
			break
		case common.JsonKeyStream:
			if bytes[offset] == '"' {
				s := strings.Split(string(bytes[collectStart:offset]), "@")
				depth20.Symbol = strings.ToUpper(s[0])
				currentKey = common.JsonKeyUnknown
				offset += 3
				continue
			}
			break
		case common.JsonKeyUnknown:
			if bytes[offset] == 'l' && offset+11 < bytesLen && bytes[offset+11] == 'd' {
				currentKey = common.JsonKeyLastUpdateId
				offset += 14
				collectStart = offset
				continue
			} else if bytes[offset] == 'b' &&
				bytes[offset-1] == '"' &&
				bytes[offset+1] == 'i' &&
				bytes[offset+2] == 'd' &&
				bytes[offset+3] == 's' &&
				bytes[offset+4] == '"' {
				currentKey = common.JsonKeyBids
				offset += 9
				collectStart = offset
				counter = 0
				continue
			} else if bytes[offset] == 'a' &&
				bytes[offset-1] == '"' &&
				bytes[offset+1] == 's' &&
				bytes[offset+2] == 'k' &&
				bytes[offset+3] == 's' &&
				bytes[offset+4] == '"' {
				currentKey = common.JsonKeyAsks
				offset += 9
				collectStart = offset
				counter = 0
				continue
			} else if bytes[offset] == 's' && offset < bytesLen-6 &&
				bytes[offset-1] == '"' &&
				bytes[offset+1] == 't' &&
				bytes[offset+2] == 'r' &&
				bytes[offset+3] == 'e' &&
				bytes[offset+4] == 'a' &&
				bytes[offset+5] == 'm' &&
				bytes[offset+6] == '"' {
				currentKey = common.JsonKeyStream
				offset += 9
				collectStart = offset
				offset += 21
				continue
			}
		}
		offset += 1
	}
	return nil
}

func ParseDepth5(bytes []byte, depth5 *Depth5) (err error) {
	//{"stream":"flmusdt@depth5@100ms","data":{"lastUpdateId":165284515,"bids":[["0.48560000","536.0500000"],["0.48550000","480.00000000"],["0.4855000","14257.67000000"],["0.48510000","1056.25000000"],["0.48500000","1894.3500000"],["0.48480000","2145.67000000"],["0.48460000","2196.59000000"],["0.48330000","3000.00000000"],["0.4835000","2531.26000000"],["0.48310000","21.18000000"],["0.48300000","4292.54000000"],["0.48270000","5042.00000000"],["0.48240000","5051.00000000"],["0.48230000","24.83000000"],["0.4825000","457.11000000"],["0.4850000","4142.1500000"],["0.48160000","31.15000000"],["0.48150000","71.96000000"],["0.48130000","1284.94000000"],["0.48110000","1098.85000000"]],"asks":[["0.48630000","5601.00000000"],["0.48650000","990.00000000"],["0.48670000","7816.00000000"],["0.48680000","7914.96000000"],["0.48690000","963.00000000"],["0.4875000","3640.00000000"],["0.48730000","814.24000000"],["0.48780000","3560.00000000"],["0.48800000","1029.00000000"],["0.48880000","13221.24000000"],["0.48940000","3000.00000000"],["0.48980000","62.75000000"],["0.49000000","1482.94000000"],["0.49040000","516.34000000"],["0.49110000","46.50000000"],["0.4915000","27.10000000"],["0.49130000","31.03000000"],["0.49150000","66.27000000"],["0.49160000","1291.65000000"],["0.49190000","159.76000000"]]}}
	depth5.ParseTime = time.Now()
	offset := 2
	collectStart := offset
	bytesLen := len(bytes)
	currentKey := common.JsonKeyUnknown
	counter := 0
	for offset < bytesLen-4 {
		switch currentKey {
		case common.JsonKeyBids:
			if bytes[offset] == '"' {
				depth5.Bids[counter/2][counter%2], err = common.ParseDecimal(bytes[collectStart:offset])
				if err != nil {
					return fmt.Errorf("JsonKeyBids error %v mainLoop %d end %d %s", err, collectStart, offset, bytes[collectStart:offset])
				}
				counter += 1
				if counter >= 10 {
					currentKey = common.JsonKeyUnknown
					offset += 4
				} else if counter%2 == 0 {
					offset += 5
					collectStart = offset
				} else {
					offset += 3
					collectStart = offset
				}
				continue
			}
			break
		case common.JsonKeyAsks:
			if bytes[offset] == '"' {
				depth5.Asks[counter/2][counter%2], err = common.ParseDecimal(bytes[collectStart:offset])
				if err != nil {
					return fmt.Errorf("JsonKeyAsks error %v mainLoop %d end %d %s", err, collectStart, offset, bytes[collectStart:offset])
				}
				counter += 1
				if counter >= 10 {
					currentKey = common.JsonKeyUnknown
					offset += 4
				} else if counter%2 == 0 {
					offset += 5
					collectStart = offset
				} else {
					offset += 3
					collectStart = offset
				}
				continue
			}
			break
		case common.JsonKeyLastUpdateId:
			if bytes[offset] == ',' {
				depth5.LastUpdateId, err = common.ParseInt(bytes[collectStart:offset])
				if err != nil {
					return fmt.Errorf("JsonKeyLastUpdateId error %v mainLoop %d end %d %s", err, collectStart, offset, bytes[collectStart:offset])
				}
				currentKey = common.JsonKeyUnknown
				offset += 2
				continue
			}
			break
		case common.JsonKeyStream:
			if bytes[offset] == '"' {
				s := strings.Split(string(bytes[collectStart:offset]), "@")
				depth5.Symbol = strings.ToUpper(s[0])
				currentKey = common.JsonKeyUnknown
				offset += 3
				continue
			}
			break
		case common.JsonKeyUnknown:
			if bytes[offset] == 'l' && offset+11 < bytesLen && bytes[offset+11] == 'd' {
				currentKey = common.JsonKeyLastUpdateId
				offset += 14
				collectStart = offset
				continue
			} else if bytes[offset] == 'b' &&
				bytes[offset-1] == '"' &&
				bytes[offset+1] == 'i' &&
				bytes[offset+2] == 'd' &&
				bytes[offset+3] == 's' &&
				bytes[offset+4] == '"' {
				currentKey = common.JsonKeyBids
				offset += 9
				collectStart = offset
				counter = 0
				continue
			} else if bytes[offset] == 'a' &&
				bytes[offset-1] == '"' &&
				bytes[offset+1] == 's' &&
				bytes[offset+2] == 'k' &&
				bytes[offset+3] == 's' &&
				bytes[offset+4] == '"' {
				currentKey = common.JsonKeyAsks
				offset += 9
				collectStart = offset
				counter = 0
				continue
			} else if bytes[offset] == 's' && offset < bytesLen-6 &&
				bytes[offset-1] == '"' &&
				bytes[offset+1] == 't' &&
				bytes[offset+2] == 'r' &&
				bytes[offset+3] == 'e' &&
				bytes[offset+4] == 'a' &&
				bytes[offset+5] == 'm' &&
				bytes[offset+6] == '"' {
				currentKey = common.JsonKeyStream
				offset += 9
				collectStart = offset
				offset += 21
				continue
			}
		}
		offset += 1
	}
	return nil
}

func ParseTrade(msg []byte) (*Trade, error) {
	//{"stream":"wavesusdt@trade","data":{"e":"trade","E":1620120764377,"s":"WAVESUSDT","t":23385964,"p":"37.84900000","q":"1.85300000","b":419242349,"a":419242185,"T":1620120764376,"m":false,"M":true}}
	var err error
	trade := Trade{}
	offset := 46
	collectStart := offset
	bytesLen := len(msg)
	currentKey := common.JsonKeyUnknown
	for offset < bytesLen-2 {
		switch currentKey {
		case common.JsonKeyPrice:
			if msg[offset] == '"' {
				trade.Price, err = common.ParseFloat(msg[collectStart:offset])
				if err != nil {
					return nil, fmt.Errorf("JsonKeyPrice error %v start %d end %d %s", err, collectStart, offset, msg[collectStart:offset])
				}
				currentKey = common.JsonKeyUnknown
				offset += 3
				continue
			}
			break
		case common.JsonKeyQuantity:
			if msg[offset] == '"' {
				trade.Quantity, err = common.ParseFloat(msg[collectStart:offset])
				if err != nil {
					return nil, fmt.Errorf("JsonKeyQuantity error %v start %d end %d %s", err, collectStart, offset, msg[collectStart:offset])
				}
				currentKey = common.JsonKeyUnknown
				if offset < bytesLen-20 {
					offset = bytesLen - 20
				} else {
					return nil, fmt.Errorf("JsonKeyQuantity bad msg, can't locate IsTheBuyerTheMarketMaker, %s", msg)
				}
				continue
			}
			break
		case common.JsonKeySymbol:
			if msg[offset] == '"' {
				symbol := msg[collectStart:offset]
				trade.Symbol = *(*string)(unsafe.Pointer(&symbol))
				currentKey = common.JsonKeyUnknown
				offset += 3
				continue
			}
			break
		case common.JsonKeyUnknown:
			if msg[offset] == 'E' && msg[offset-1] == '"' && msg[offset+1] == '"' && offset+13 < bytesLen {
				eventTime, err := common.ParseInt(msg[offset+3 : offset+16])
				if err != nil {
					return nil, fmt.Errorf("TimePoint error %v start %d end %d %s", err, collectStart, offset, msg[collectStart:offset])
				}
				trade.EventTime = time.Unix(0, eventTime*1000000)
				offset += 18
				continue
			} else if msg[offset] == 's' && msg[offset-1] == '"' && msg[offset+1] == '"' {
				currentKey = common.JsonKeySymbol
				offset += 4
				collectStart = offset
				offset += 6
				continue
			} else if msg[offset] == 'p' && msg[offset-1] == '"' && msg[offset+1] == '"' {
				currentKey = common.JsonKeyPrice
				offset += 4
				collectStart = offset
				continue
			} else if msg[offset] == 'q' && msg[offset-1] == '"' && msg[offset+1] == '"' {
				currentKey = common.JsonKeyQuantity
				offset += 4
				collectStart = offset
				continue
			} else if msg[offset] == 'm' && msg[offset+1] == '"' {
				if msg[offset+3] == 'f' {
					trade.IsTheBuyerTheMarketMaker = false
				} else {
					trade.IsTheBuyerTheMarketMaker = true
				}
			}

		}
		offset += 1
	}
	return &trade, nil
}

var Depth5Lines = `{"stream":"btcusdt@depth5@100ms","data":{"lastUpdateId":11719214114,"bids":[["40069.98000000","0.00002300"],["40067.04000000","0.06000000"],["40067.03000000","0.03743200"],["40066.08000000","0.13019700"],["40066.07000000","0.24799400"]],"asks":[["40069.99000000","1.32492400"],["40072.90000000","0.00099900"],["40073.10000000","0.20501400"],["40073.11000000","0.31000000"],["40073.29000000","0.06158400"]]}}
{"stream":"ethusdt@depth5@100ms","data":{"lastUpdateId":8641847242,"bids":[["2548.07000000","3.57932000"],["2548.03000000","4.00000000"],["2548.02000000","0.08351000"],["2548.00000000","0.00589000"],["2547.88000000","1.00000000"]],"asks":[["2548.09000000","0.00469000"],["2548.18000000","2.77814000"],["2548.20000000","4.00000000"],["2548.21000000","2.14000000"],["2548.22000000","1.43557000"]]}}
{"stream":"flmusdt@depth5@100ms","data":{"lastUpdateId":306654321,"bids":[["0.54250000","645.16000000"],["0.54200000","92.25000000"],["0.54190000","2131.66000000"],["0.54120000","615.75000000"],["0.54110000","477.34000000"]],"asks":[["0.54290000","5590.00000000"],["0.54300000","755.83000000"],["0.54310000","2227.18000000"],["0.54330000","6500.00000000"],["0.54340000","1286.20000000"]]}}
{"stream":"trxusdt@depth5@100ms","data":{"lastUpdateId":1946379544,"bids":[["0.07143000","1527955.90000000"],["0.07142000","87193.90000000"],["0.07141000","21299.00000000"],["0.07140000","53344.00000000"],["0.07139000","145392.10000000"]],"asks":[["0.07144000","4294.70000000"],["0.07145000","126571.00000000"],["0.07146000","91291.70000000"],["0.07147000","268610.50000000"],["0.07148000","390926.80000000"]]}}
{"stream":"btcusdt@depth5@100ms","data":{"lastUpdateId":11719214137,"bids":[["40069.98000000","0.00002300"],["40067.04000000","0.06000000"],["40067.03000000","0.03743200"],["40066.08000000","0.13019700"],["40066.07000000","0.24799400"]],"asks":[["40069.99000000","1.30648300"],["40072.90000000","0.00099900"],["40073.10000000","0.20501400"],["40073.11000000","0.31000000"],["40073.29000000","0.06158400"]]}}
{"stream":"eosusdt@depth5@100ms","data":{"lastUpdateId":4110393400,"bids":[["5.15730000","89.82000000"],["5.15720000","51.65000000"],["5.15690000","127.88000000"],["5.15670000","55.30000000"],["5.15660000","118.14000000"]],"asks":[["5.15780000","1071.02000000"],["5.15820000","21.50000000"],["5.15840000","232.19000000"],["5.15890000","2615.45000000"],["5.15910000","1771.80000000"]]}}
{"stream":"ethusdt@depth5@100ms","data":{"lastUpdateId":8641847257,"bids":[["2548.07000000","3.57932000"],["2548.02000000","0.08351000"],["2548.00000000","0.00589000"],["2547.84000000","1.00000000"],["2547.82000000","0.38768000"]],"asks":[["2548.08000000","4.59661000"],["2548.09000000","0.00469000"],["2548.20000000","4.00000000"],["2548.21000000","2.14000000"],["2548.22000000","1.44758000"]]}}
{"stream":"btcusdt@depth5@100ms","data":{"lastUpdateId":11719214152,"bids":[["40069.98000000","0.00002300"],["40067.04000000","0.06000000"],["40067.03000000","0.03743200"],["40066.08000000","0.13019700"],["40066.07000000","0.24799400"]],"asks":[["40069.99000000","1.27648300"],["40072.90000000","0.00099900"],["40073.10000000","0.20501400"],["40073.11000000","0.31000000"],["40073.29000000","0.06158400"]]}}
{"stream":"trxusdt@depth5@100ms","data":{"lastUpdateId":1946379545,"bids":[["0.07143000","1527955.90000000"],["0.07142000","87193.90000000"],["0.07141000","21299.00000000"],["0.07140000","53344.00000000"],["0.07139000","145392.10000000"]],"asks":[["0.07144000","4294.70000000"],["0.07145000","126571.00000000"],["0.07146000","91291.70000000"],["0.07147000","268610.50000000"],["0.07148000","390926.80000000"]]}}
{"stream":"eosusdt@depth5@100ms","data":{"lastUpdateId":4110393406,"bids":[["5.15730000","89.82000000"],["5.15720000","51.65000000"],["5.15700000","55.30000000"],["5.15690000","127.88000000"],["5.15660000","118.14000000"]],"asks":[["5.15780000","36.84000000"],["5.15820000","21.50000000"],["5.15840000","232.19000000"],["5.15890000","2615.45000000"],["5.15910000","1771.80000000"]]}}
{"stream":"flmusdt@depth5@100ms","data":{"lastUpdateId":306654325,"bids":[["0.54250000","645.16000000"],["0.54200000","92.25000000"],["0.54190000","2131.66000000"],["0.54120000","615.75000000"],["0.54110000","477.34000000"]],"asks":[["0.54290000","5590.00000000"],["0.54300000","755.83000000"],["0.54310000","3759.35000000"],["0.54330000","6500.00000000"],["0.54340000","1286.20000000"]]}}
{"stream":"flmusdt@depth5@100ms","data":{"lastUpdateId":306654327,"bids":[["0.54250000","645.16000000"],["0.54200000","92.25000000"],["0.54190000","2131.66000000"],["0.54120000","615.75000000"],["0.54110000","477.34000000"]],"asks":[["0.54290000","5590.00000000"],["0.54300000","755.83000000"],["0.54310000","9630.56000000"],["0.54330000","6500.00000000"],["0.54340000","1286.20000000"]]}}
{"stream":"ethusdt@depth5@100ms","data":{"lastUpdateId":8641847291,"bids":[["2548.07000000","3.57059000"],["2548.02000000","0.08351000"],["2548.00000000","0.00589000"],["2547.82000000","0.38768000"],["2547.80000000","0.00569000"]],"asks":[["2548.08000000","8.59661000"],["2548.09000000","0.01680000"],["2548.20000000","4.00000000"],["2548.21000000","2.14000000"],["2548.22000000","1.44758000"]]}}
{"stream":"eosusdt@depth5@100ms","data":{"lastUpdateId":4110393414,"bids":[["5.15730000","12.60000000"],["5.15720000","51.65000000"],["5.15700000","55.30000000"],["5.15690000","127.88000000"],["5.15660000","118.14000000"]],"asks":[["5.15780000","36.84000000"],["5.15800000","77.62000000"],["5.15820000","21.50000000"],["5.15840000","232.19000000"],["5.15890000","2495.90000000"]]}}
{"stream":"flmusdt@depth5@100ms","data":{"lastUpdateId":306654330,"bids":[["0.54250000","645.16000000"],["0.54200000","1119.14000000"],["0.54120000","615.75000000"],["0.54110000","477.34000000"],["0.54060000","2163.00000000"]],"asks":[["0.54290000","5539.24000000"],["0.54300000","755.83000000"],["0.54310000","9630.56000000"],["0.54330000","6500.00000000"],["0.54340000","1286.20000000"]]}}
{"stream":"ethusdt@depth5@100ms","data":{"lastUpdateId":8641847326,"bids":[["2548.02000000","0.08351000"],["2548.00000000","0.00589000"],["2547.82000000","0.38768000"],["2547.80000000","0.00569000"],["2547.75000000","2.91066000"]],"asks":[["2548.03000000","3.11257000"],["2548.06000000","0.73789000"],["2548.08000000","3.09661000"],["2548.09000000","0.03764000"],["2548.20000000","4.00000000"]]}}
{"stream":"trxusdt@depth5@100ms","data":{"lastUpdateId":1946379546,"bids":[["0.07143000","1527955.90000000"],["0.07142000","87193.90000000"],["0.07141000","21299.00000000"],["0.07140000","53344.00000000"],["0.07139000","145392.10000000"]],"asks":[["0.07144000","4294.70000000"],["0.07145000","126571.00000000"],["0.07146000","91291.70000000"],["0.07147000","268610.50000000"],["0.07148000","390926.80000000"]]}}`