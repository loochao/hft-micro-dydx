package kcspot

import (
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"time"
	"unsafe"
)

func ParseDepth50(bytes []byte) (*Depth50, error) {
	var err error
	orderBook := Depth50{
		Bids:        [50][2]float64{},
		Asks:        [50][2]float64{},
		ArrivalTime: time.Now(),
	}
	offset := 12
	collectStart := offset
	bytesLen := len(bytes)
	currentKey := common.JsonKeyAsks
	counter := 0
	if bytes[offset] != 'k' && bytes[offset+1] != 's' && bytes[offset+2] != '"' {
		return nil, fmt.Errorf("bad bytes %s", bytes)
	}
	offset = 19
	collectStart = offset
	for offset < bytesLen-18 {
		switch currentKey {
		case common.JsonKeyBids:
			if bytes[offset] == '"' {
				orderBook.Bids[counter/2][counter%2], err = common.ParseBinanceFloat(bytes[collectStart:offset])
				if err != nil {
					return nil, err
				}
				counter += 1
				if counter >= 100 {
					currentKey = common.JsonKeyEventTime
					offset += 16
					collectStart = offset
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
				orderBook.Asks[counter/2][counter%2], err = common.ParseBinanceFloat(bytes[collectStart:offset])
				if err != nil {
					return nil, err
				}
				counter += 1
				if counter >= 100 {
					currentKey = common.JsonKeyBids
					offset += 14
					collectStart = offset
					counter = 0
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
		case common.JsonKeyEventTime:
			offset += 13
			timestamp, err := common.ParseBinanceInt(bytes[collectStart:offset])
			if err != nil {
				return nil, err
			}
			orderBook.EventTime = time.Unix(0, timestamp*1000000)
			currentKey = common.JsonKeySymbol
			offset += 56
			collectStart = offset
			continue
		case common.JsonKeySymbol:
			if bytes[offset] == '"' {
				symbol := bytes[collectStart:offset]
				orderBook.Symbol = *(*string)(unsafe.Pointer(&symbol))
				offset = bytesLen
				//此下退出
				continue
			}
			break
		}
		offset += 1
	}
	return &orderBook, nil
}
