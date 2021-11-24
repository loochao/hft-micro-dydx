package binance_busdfuture

import (
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"strconv"
	"time"
	"unsafe"
)

func ParseTrade(bytes []byte) (*Trade, error) {
	// {"stream":"btcusdt@aggTrade","data":{"e":"aggTrade","E":1616945754086,"a":405295371,"s":"BTCUSDT","p":"56183.31","q":"0.003","f":649066620,"l":649066620,"T":1616945753931,"m":false}}
	var err error
	trade := Trade{}
	offset := 53
	collectStart := offset
	bytesLen := len(bytes)
	currentKey := common.JsonKeyUnknown
	for offset < bytesLen-2 {
		switch currentKey {
		case common.JsonKeyPrice:
			if bytes[offset] == '"' {
				trade.Price, err = common.ParseDecimal(bytes[collectStart:offset])
				if err != nil {
					return nil, fmt.Errorf("JsonKeyPrice error %v start %d end %d %s", err, collectStart, offset, bytes[collectStart:offset])
				}
				currentKey = common.JsonKeyUnknown
				offset += 3
				continue
			}
			break
		case common.JsonKeyQuantity:
			if bytes[offset] == '"' {
				trade.Quantity, err = common.ParseDecimal(bytes[collectStart:offset])
				if err != nil {
					return nil, fmt.Errorf("JsonKeyQuantity error %v start %d end %d %s", err, collectStart, offset, bytes[collectStart:offset])
				}
				currentKey = common.JsonKeyUnknown
				offset = bytesLen
				continue
			}
			break
		case common.JsonKeySymbol:
			if bytes[offset] == '"' {
				symbol := bytes[collectStart:offset]
				trade.Symbol = *(*string)(unsafe.Pointer(&symbol))
				currentKey = common.JsonKeyUnknown
				offset += 3
				continue
			}
			break
		case common.JsonKeyUnknown:
			if bytes[offset] == 'E' && bytes[offset-1] == '"' && bytes[offset+1] == '"' && offset+13 < bytesLen {
				eventTime, err := common.ParseInt(bytes[offset+3 : offset+16])
				if err != nil {
					return nil, fmt.Errorf("TimePoint error %v start %d end %d %s", err, collectStart, offset, bytes[collectStart:offset])
				}
				trade.EventTime = time.Unix(0, eventTime*1000000)
				offset += 21
				continue
			} else if bytes[offset] == 's' && bytes[offset-1] == '"' && bytes[offset+1] == '"' {
				currentKey = common.JsonKeySymbol
				offset += 4
				collectStart = offset
				offset += 6
				continue
			} else if bytes[offset] == 'p' && bytes[offset-1] == '"' && bytes[offset+1] == '"' {
				currentKey = common.JsonKeyPrice
				offset += 4
				collectStart = offset
				continue
			} else if bytes[offset] == 'q' && bytes[offset-1] == '"' && bytes[offset+1] == '"' {
				currentKey = common.JsonKeyQuantity
				offset += 4
				collectStart = offset
				continue
			}

		}
		offset += 1
	}
	if bytes[bytesLen-4] == 'u' {
		trade.IsTheBuyerTheMarketMaker = true
	}
	return &trade, nil
}

func ParseDepth20(bytes []byte, depth20 *Depth20) error {
	var err error
	offset := 60
	collectStart := offset
	bytesLen := len(bytes)
	currentKey := common.JsonKeyUnknown
	counter := 0
	for offset < bytesLen-2 {
		switch currentKey {
		case common.JsonKeyBids:
			if bytes[offset] == '"' {
				depth20.Bids[counter/2][counter%2], err = common.ParseDecimal(bytes[collectStart:offset])
				if err != nil {
					return fmt.Errorf("JsonKeyBids error %v start %d end %d %s", err, collectStart, offset, bytes[collectStart:offset])
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
					return fmt.Errorf("JsonKeyAsks error %v start %d end %d %s", err, collectStart, offset, bytes[collectStart:offset])
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
					return fmt.Errorf("JsonKeyLastUpdateId error %v start %d end %d %s", err, collectStart, offset, bytes[collectStart:offset])
				}
				currentKey = common.JsonKeyUnknown
				offset += 2
				continue
			}
			break
		case common.JsonKeySymbol:
			if bytes[offset] == '"' {
				symbol := bytes[collectStart:offset]
				depth20.Symbol = *(*string)(unsafe.Pointer(&symbol))
				currentKey = common.JsonKeyUnknown
				offset += 3
				continue
			}
			break
		case common.JsonKeyUnknown:
			if bytes[offset] == 'E' && bytes[offset-1] == '"' && bytes[offset+1] == '"' && offset+13 < bytesLen {
				eventTime, err := common.ParseInt(bytes[offset+3 : offset+16])
				if err != nil {
					return fmt.Errorf("TimePoint error %v start %d end %d %s", err, collectStart, offset, bytes[collectStart:offset])
				}
				depth20.EventTime = time.Unix(0, eventTime*1000000)
				offset += 17
				continue
			} else if bytes[offset] == 'u' && bytes[offset-1] == '"' && bytes[offset+1] == '"' {
				currentKey = common.JsonKeyLastUpdateId
				offset += 3
				collectStart = offset
				continue
			} else if bytes[offset] == 's' && bytes[offset-1] == '"' && bytes[offset+1] == '"' {
				currentKey = common.JsonKeySymbol
				offset += 4
				collectStart = offset
				offset += 6
				continue
			} else if bytes[offset] == 'b' && bytes[offset-1] == '"' && bytes[offset+1] == '"' {
				currentKey = common.JsonKeyBids
				offset += 6
				collectStart = offset
				counter = 0
				continue
			} else if bytes[offset] == 'a' && bytes[offset-1] == '"' && bytes[offset+1] == '"' {
				currentKey = common.JsonKeyAsks
				offset += 6
				collectStart = offset
				counter = 0
				continue
			}
		}
		offset += 1
	}
	return nil
}

func ParseMarkPrice(bytes []byte) (*MarkPrice, error) {
	//{"stream":"eosusdt@markPrice@1s","data":{"e":"markPriceUpdate","E":1616555105001,"s":"EOSUSDT","p":"4.11998561","P":"4.11278428","i":"4.11519211","r":"0.00030438","T":1616572800000}}
	var err error
	markPrice := MarkPrice{
		ArrivalTime: time.Now(),
	}
	offset := 60
	collectStart := offset
	bytesLen := len(bytes)
	currentKey := common.JsonKeyUnknown
	for offset < bytesLen-2 {
		switch currentKey {
		case common.JsonKeySymbol:
			if bytes[offset] == '"' {
				symbol := bytes[collectStart:offset]
				markPrice.Symbol = *(*string)(unsafe.Pointer(&symbol))
				currentKey = common.JsonKeyUnknown
				offset += 3
				continue
			}
			break
		case common.JsonKeyMarkPrice:
			if bytes[offset] == '"' {
				markPrice.MarkPrice, err = common.ParseDecimal(bytes[collectStart:offset])
				if err != nil {
					return nil, fmt.Errorf("JsonKeyMarkPrice error %v start %d end %d %s", err, collectStart, offset, bytes[collectStart:offset])
				}
				currentKey = common.JsonKeyUnknown
				offset += 3
				continue
			}
			break
		case common.JsonKeyIndexPrice:
			if bytes[offset] == '"' {
				markPrice.IndexPrice, err = common.ParseDecimal(bytes[collectStart:offset])
				if err != nil {
					return nil, fmt.Errorf("JsonKeyIndexPrice error %v start %d end %d %s", err, collectStart, offset, bytes[collectStart:offset])
				}
				currentKey = common.JsonKeyUnknown
				offset += 3
				continue
			}
			break
		case common.JsonKeyEstimatedSettlePrice:
			if bytes[offset] == '"' {
				markPrice.EstimatedSettlePrice, err = common.ParseDecimal(bytes[collectStart:offset])
				if err != nil {
					return nil, fmt.Errorf("JsonKeyEstimatedSettlePrice error %v start %d end %d %s", err, collectStart, offset, bytes[collectStart:offset])
				}
				currentKey = common.JsonKeyUnknown
				offset += 3
				continue
			}
			break
		case common.JsonKeyFundingRate:
			if bytes[offset] == '"' {
				markPrice.FundingRate, err = common.ParseDecimal(bytes[collectStart:offset])
				if err != nil {
					return nil, fmt.Errorf("JsonKeyFundingRate error %v start %d end %d %s", err, collectStart, offset, bytes[collectStart:offset])
				}
				currentKey = common.JsonKeyUnknown
				offset += 3
				continue
			}
			break
		case common.JsonKeyUnknown:
			if bytes[offset] == 'E' && bytes[offset-1] == '"' && bytes[offset+1] == '"' && offset+16 < bytesLen {
				timeStr := bytes[offset+3 : offset+16]
				eventTime, err := strconv.ParseInt(*(*string)(unsafe.Pointer(&timeStr)), 10, 64)
				if err != nil {
					return nil, fmt.Errorf("TimePoint error %v start %d end %d %s", err, collectStart, offset, bytes[collectStart:offset])
				}
				markPrice.EventTime = time.Unix(0, eventTime*1000000)
				offset += 16
				collectStart = offset
				continue
			} else if bytes[offset] == 'T' && bytes[offset-1] == '"' && bytes[offset+1] == '"' && offset+16 < bytesLen {
				timeStr := bytes[offset+3 : offset+16]
				nextFundingTime, err := strconv.ParseInt(*(*string)(unsafe.Pointer(&timeStr)), 10, 64)
				if err != nil {
					return nil, fmt.Errorf("NextFundingTime error %v start %d end %d %s", err, collectStart, offset, bytes[collectStart:offset])
				}
				markPrice.NextFundingTime = time.Unix(0, nextFundingTime*1000000)
				offset += 16
				collectStart = offset
				continue
			} else if bytes[offset] == 's' && bytes[offset-1] == '"' && bytes[offset+1] == '"' {
				currentKey = common.JsonKeySymbol
				offset += 4
				collectStart = offset
				continue
			} else if bytes[offset] == 'p' && bytes[offset-1] == '"' && bytes[offset+1] == '"' {
				currentKey = common.JsonKeyMarkPrice
				offset += 4
				collectStart = offset
				continue
			} else if bytes[offset] == 'i' && bytes[offset-1] == '"' && bytes[offset+1] == '"' {
				currentKey = common.JsonKeyIndexPrice
				offset += 4
				collectStart = offset
				continue
			} else if bytes[offset] == 'P' && bytes[offset-1] == '"' && bytes[offset+1] == '"' {
				currentKey = common.JsonKeyEstimatedSettlePrice
				offset += 4
				collectStart = offset
				continue
			} else if bytes[offset] == 'r' && bytes[offset-1] == '"' && bytes[offset+1] == '"' {
				currentKey = common.JsonKeyFundingRate
				offset += 4
				collectStart = offset
				continue
			}
		}
		offset += 1
	}
	return &markPrice, nil
}

func ParseDepth5(bytes []byte, depth5 *Depth5) error {
	depth5.ParseTime = time.Now()
	var err error
	offset := 60
	collectStart := offset
	bytesLen := len(bytes)
	currentKey := common.JsonKeyUnknown
	counter := 0
	for offset < bytesLen-2 {
		switch currentKey {
		case common.JsonKeyBids:
			if bytes[offset] == '"' {
				depth5.Bids[counter/2][counter%2], err = common.ParseDecimal(bytes[collectStart:offset])
				if err != nil {
					return fmt.Errorf("JsonKeyBids error %v start %d end %d %s", err, collectStart, offset, bytes[collectStart:offset])
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
					return fmt.Errorf("JsonKeyAsks error %v start %d end %d %s", err, collectStart, offset, bytes[collectStart:offset])
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
					return fmt.Errorf("JsonKeyLastUpdateId error %v start %d end %d %s", err, collectStart, offset, bytes[collectStart:offset])
				}
				currentKey = common.JsonKeyUnknown
				offset += 2
				continue
			}
			break
		case common.JsonKeySymbol:
			if bytes[offset] == '"' {
				symbol := bytes[collectStart:offset]
				depth5.Symbol = *(*string)(unsafe.Pointer(&symbol))
				currentKey = common.JsonKeyUnknown
				offset += 3
				continue
			}
			break
		case common.JsonKeyUnknown:
			if bytes[offset] == 'E' && bytes[offset-1] == '"' && bytes[offset+1] == '"' && offset+13 < bytesLen {
				eventTime, err := common.ParseInt(bytes[offset+3 : offset+16])
				if err != nil {
					return fmt.Errorf("TimePoint error %v start %d end %d %s", err, collectStart, offset, bytes[collectStart:offset])
				}
				depth5.EventTime = time.Unix(0, eventTime*1000000)
				offset += 17
				continue
			} else if bytes[offset] == 'u' && bytes[offset-1] == '"' && bytes[offset+1] == '"' {
				currentKey = common.JsonKeyLastUpdateId
				offset += 3
				collectStart = offset
				continue
			} else if bytes[offset] == 's' && bytes[offset-1] == '"' && bytes[offset+1] == '"' {
				currentKey = common.JsonKeySymbol
				offset += 4
				collectStart = offset
				offset += 6
				continue
			} else if bytes[offset] == 'b' && bytes[offset-1] == '"' && bytes[offset+1] == '"' {
				currentKey = common.JsonKeyBids
				offset += 6
				collectStart = offset
				counter = 0
				continue
			} else if bytes[offset] == 'a' && bytes[offset-1] == '"' && bytes[offset+1] == '"' {
				currentKey = common.JsonKeyAsks
				offset += 6
				collectStart = offset
				counter = 0
				continue
			}
		}
		offset += 1
	}
	return nil
}

//{"stream":"scusdt@bookTicker","data":{"e":"bookTicker","u":552297398961,"s":"SCUSDT","b":"0.012805","B":"46556","a":"0.012816","A":"90351","T":1624971386657,"E":1624971386662}}
func ParseBookTicker(msg []byte, bookTicker *BookTicker) (err error) {
	bookTicker.ParseTime = time.Now()
	msgLen := len(msg)
	if msgLen < 15 {
		return fmt.Errorf("bad msg %s", msg)
	}
	var t int64
	t, err = common.ParseInt(msg[msgLen-15 : msgLen-2])
	if err != nil {
		return
	}
	bookTicker.EventTime = time.Unix(0, t*1000000)
	collectEnd := 59
	collectStart := collectEnd
	currentKey := common.JsonKeyUnknown
	for collectEnd < msgLen {
		switch currentKey {
		case common.JsonKeySymbol:
			if msg[collectEnd] == '"' {
				bookTicker.Symbol = common.UnsafeBytesToString(msg[collectStart:collectEnd])
				currentKey = common.JsonKeyBidPrice
				collectEnd += 7
				collectStart = collectEnd
				continue
			}
			break
		case common.JsonKeyBidPrice:
			if msg[collectEnd] == '"' {
				bookTicker.BestBidPrice, err = common.ParseDecimal(msg[collectStart:collectEnd])
				if err != nil {
					return fmt.Errorf("JsonKeyBidPrice error %v start %d end %d %s", err, collectStart, collectEnd, msg[collectStart:collectEnd])
				}
				currentKey = common.JsonKeyBidSize
				collectEnd += 7
				collectStart = collectEnd
				continue
			}
			break
		case common.JsonKeyBidSize:
			if msg[collectEnd] == '"' {
				bookTicker.BestBidQty, err = common.ParseDecimal(msg[collectStart:collectEnd])
				if err != nil {
					return fmt.Errorf("JsonKeyBidSize error %v start %d end %d %s", err, collectStart, collectEnd, msg[collectStart:collectEnd])
				}
				currentKey = common.JsonKeyAskPrice
				collectEnd += 7
				collectStart = collectEnd
				continue
			}
			break
		case common.JsonKeyAskPrice:
			if msg[collectEnd] == '"' {
				bookTicker.BestAskPrice, err = common.ParseDecimal(msg[collectStart:collectEnd])
				if err != nil {
					return fmt.Errorf("JsonKeyAskSize error %v start %d end %d %s", err, collectStart, collectEnd, msg[collectStart:collectEnd])
				}
				currentKey = common.JsonKeyAskSize
				collectEnd += 7
				collectStart = collectEnd
				continue
			}
			break
		case common.JsonKeyAskSize:
			if msg[collectEnd] == '"' {
				bookTicker.BestAskQty, err = common.ParseDecimal(msg[collectStart:collectEnd])
				if err != nil {
					return fmt.Errorf("JsonKeyAskSize error %v start %d end %d %s", err, collectStart, collectEnd, msg[collectStart:collectEnd])
				}
				return
			}
		case common.JsonKeyUnknown:
			if msg[collectEnd] == 's' {
				currentKey = common.JsonKeySymbol
				collectEnd += 4
				collectStart = collectEnd
				collectEnd += 6
				continue
			}
		}
		collectEnd += 1
	}
	return
}
