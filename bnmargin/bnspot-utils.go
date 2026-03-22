package bnmargin

import (
	"context"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"strings"
	"time"
	"unsafe"
)

func ParseDepth20(bytes []byte) (*Depth20, error) {
	//{"stream":"flmusdt@depth20@100ms","data":{"lastUpdateId":165284515,"bids":[["0.48560000","2036.02000000"],["0.48550000","480.00000000"],["0.48520000","14257.67000000"],["0.48510000","1056.25000000"],["0.48500000","1894.32000000"],["0.48480000","2145.67000000"],["0.48460000","2196.59000000"],["0.48330000","3000.00000000"],["0.48320000","2531.26000000"],["0.48310000","21.18000000"],["0.48300000","4292.54000000"],["0.48270000","5042.00000000"],["0.48240000","5051.00000000"],["0.48230000","24.83000000"],["0.48220000","457.11000000"],["0.48200000","4142.12000000"],["0.48160000","31.15000000"],["0.48150000","71.96000000"],["0.48130000","1284.94000000"],["0.48110000","1098.85000000"]],"asks":[["0.48630000","5601.00000000"],["0.48650000","990.00000000"],["0.48670000","7816.00000000"],["0.48680000","7914.96000000"],["0.48690000","963.00000000"],["0.48720000","3640.00000000"],["0.48730000","814.24000000"],["0.48780000","3560.00000000"],["0.48800000","1029.00000000"],["0.48880000","13221.24000000"],["0.48940000","3000.00000000"],["0.48980000","62.75000000"],["0.49000000","1482.94000000"],["0.49040000","516.34000000"],["0.49110000","46.50000000"],["0.49120000","27.10000000"],["0.49130000","31.03000000"],["0.49150000","66.27000000"],["0.49160000","1291.65000000"],["0.49190000","159.76000000"]]}}
	var err error
	orderBook := Depth20{
		Bids:      [20][2]float64{},
		Asks:      [20][2]float64{},
		ParseTime: time.Now(),
	}
	offset := 2
	collectStart := offset
	bytesLen := len(bytes)
	currentKey := common.JsonKeyUnknown
	counter := 0
	for offset < bytesLen-4 {
		switch currentKey {
		case common.JsonKeyBids:
			if bytes[offset] == '"' {
				orderBook.Bids[counter/2][counter%2], err = common.ParseFloat(bytes[collectStart:offset])
				if err != nil {
					return nil, fmt.Errorf("JsonKeyBids error %v mainLoop %d end %d %s", err, collectStart, offset, bytes[collectStart:offset])
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
				orderBook.Asks[counter/2][counter%2], err = common.ParseFloat(bytes[collectStart:offset])
				if err != nil {
					return nil, fmt.Errorf("JsonKeyAsks error %v mainLoop %d end %d %s", err, collectStart, offset, bytes[collectStart:offset])
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
				orderBook.LastUpdateId, err = common.ParseInt(bytes[collectStart:offset])
				if err != nil {
					return nil, fmt.Errorf("JsonKeyLastUpdateId error %v mainLoop %d end %d %s", err, collectStart, offset, bytes[collectStart:offset])
				}
				currentKey = common.JsonKeyUnknown
				offset += 2
				continue
			}
			break
		case common.JsonKeyStream:
			if bytes[offset] == '"' {
				s := strings.Split(string(bytes[collectStart:offset]), "@")
				orderBook.Symbol = strings.ToUpper(s[0])
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
	return &orderBook, nil
}

func WatchAccountFromHttp(
	ctx context.Context, api *API, interval time.Duration,
	output chan Account,
) {
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			account, _, err := api.GetAccount(ctx)
			if err != nil {
				logger.Debugf("api.GetAccount(ctx) error %v", err)
			} else {
				output <- *account
			}
			timer.Reset(time.Now().Truncate(interval).Add(interval).Sub(time.Now()))
		}
	}
}

func GetOrderLimits(ctx context.Context, api *API, symbols []string) (
	tickSizes, stepSizes, minSizes, minNotional map[string]float64, error error,
) {
	exchangeInfo, err := api.GetExchangeInfo(ctx)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	tickSizes = make(map[string]float64)
	stepSizes = make(map[string]float64)
	minSizes = make(map[string]float64)
	minNotional = make(map[string]float64)
	for _, symbol := range exchangeInfo.Symbols {
		if !common.StringDataContains(symbols, symbol.Symbol) {
			continue
		}
		for _, filter := range symbol.Filters {
			switch filter.FilterType {
			case "PRICE_FILTER":
				tickSizes[symbol.Symbol] = filter.TickSize
			case "LOT_SIZE":
				stepSizes[symbol.Symbol] = filter.StepSize
				minSizes[symbol.Symbol] = filter.MinQty
			case "MIN_NOTIONAL":
				minNotional[symbol.Symbol] = filter.MinNotional
			}
		}
	}
	for _, symbol := range symbols {
		if _, ok := tickSizes[symbol]; !ok {
			return nil, nil, nil, nil, fmt.Errorf("NO SPOT TICKSIZE FOR %s", symbol)
		}
		if _, ok := stepSizes[symbol]; !ok {
			return nil, nil, nil, nil, fmt.Errorf("NO SPOT STEPSIZE FOR %s", symbol)
		}
		if _, ok := minSizes[symbol]; !ok {
			return nil, nil, nil, nil, fmt.Errorf("NO SPOT  MINSIZE FOR %s", symbol)
		}
		if _, ok := minNotional[symbol]; !ok {
			return nil, nil, nil, nil, fmt.Errorf("NO SPOT  MIN NOTIONAL FOR %s", symbol)
		}
	}
	logger.Debugf("SPOT TICK SIZES %v", tickSizes)
	logger.Debugf("SPOT STEP SIZES %v", stepSizes)
	logger.Debugf("SPOT  MIN SIZES %v", minSizes)
	logger.Debugf("SPOT MIN NOTIONAL %v", minNotional)
	return tickSizes, stepSizes, minSizes, minNotional, nil
}


func ParseDepth5(bytes []byte) (*Depth5, error) {
	//{"stream":"flmusdt@depth5@100ms","data":{"lastUpdateId":165284515,"bids":[["0.48560000","536.0500000"],["0.48550000","480.00000000"],["0.4855000","14257.67000000"],["0.48510000","1056.25000000"],["0.48500000","1894.3500000"],["0.48480000","2145.67000000"],["0.48460000","2196.59000000"],["0.48330000","3000.00000000"],["0.4835000","2531.26000000"],["0.48310000","21.18000000"],["0.48300000","4292.54000000"],["0.48270000","5042.00000000"],["0.48240000","5051.00000000"],["0.48230000","24.83000000"],["0.4825000","457.11000000"],["0.4850000","4142.1500000"],["0.48160000","31.15000000"],["0.48150000","71.96000000"],["0.48130000","1284.94000000"],["0.48110000","1098.85000000"]],"asks":[["0.48630000","5601.00000000"],["0.48650000","990.00000000"],["0.48670000","7816.00000000"],["0.48680000","7914.96000000"],["0.48690000","963.00000000"],["0.4875000","3640.00000000"],["0.48730000","814.24000000"],["0.48780000","3560.00000000"],["0.48800000","1029.00000000"],["0.48880000","13221.24000000"],["0.48940000","3000.00000000"],["0.48980000","62.75000000"],["0.49000000","1482.94000000"],["0.49040000","516.34000000"],["0.49110000","46.50000000"],["0.4915000","27.10000000"],["0.49130000","31.03000000"],["0.49150000","66.27000000"],["0.49160000","1291.65000000"],["0.49190000","159.76000000"]]}}
	var err error
	orderBook := Depth5{
		Bids:      [5][2]float64{},
		Asks:      [5][2]float64{},
		ParseTime: time.Now(),
	}
	offset := 2
	collectStart := offset
	bytesLen := len(bytes)
	currentKey := common.JsonKeyUnknown
	counter := 0
	for offset < bytesLen-4 {
		switch currentKey {
		case common.JsonKeyBids:
			if bytes[offset] == '"' {
				orderBook.Bids[counter/2][counter%2], err = common.ParseFloat(bytes[collectStart:offset])
				if err != nil {
					return nil, fmt.Errorf("JsonKeyBids error %v mainLoop %d end %d %s", err, collectStart, offset, bytes[collectStart:offset])
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
				orderBook.Asks[counter/2][counter%2], err = common.ParseFloat(bytes[collectStart:offset])
				if err != nil {
					return nil, fmt.Errorf("JsonKeyAsks error %v mainLoop %d end %d %s", err, collectStart, offset, bytes[collectStart:offset])
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
				orderBook.LastUpdateId, err = common.ParseInt(bytes[collectStart:offset])
				if err != nil {
					return nil, fmt.Errorf("JsonKeyLastUpdateId error %v mainLoop %d end %d %s", err, collectStart, offset, bytes[collectStart:offset])
				}
				currentKey = common.JsonKeyUnknown
				offset += 2
				continue
			}
			break
		case common.JsonKeyStream:
			if bytes[offset] == '"' {
				s := strings.Split(string(bytes[collectStart:offset]), "@")
				orderBook.Symbol = strings.ToUpper(s[0])
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
	return &orderBook, nil
}

func HttpPingLoop(
	ctx context.Context, api *API, interval time.Duration,
	output chan bool,
) {
	logger.Debugf("START HttpPingLoop")
	defer logger.Debugf("EXIT HttpPingLoop")
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			subCtx, _ := context.WithTimeout(ctx, time.Minute)
			_, err := api.PingServer(subCtx)
			if err != nil {
				logger.Debugf("api.PingServer error %v", err)
				select {
				case output <- false:
				default:
					logger.Debugf("output <- false failed")
				}
			} else {
				select {
				case output <- true:
				default:
					logger.Debugf("output <- true failed")
				}
			}
			timer.Reset(time.Now().Truncate(interval).Add(interval).Sub(time.Now()))
		}
	}
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
				if offset < bytesLen - 20 {
					offset = bytesLen - 20
				}else{
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
				}else{
					trade.IsTheBuyerTheMarketMaker = true
				}
			}

		}
		offset += 1
	}
	return &trade, nil
}
