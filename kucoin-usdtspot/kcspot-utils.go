package kucoin_usdtspot

import (
	"context"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"time"
	"unsafe"
)

func ParseDepth50(bytes []byte) (*Depth50, error) {
	var err error
	orderBook := Depth50{
		Bids:      [50][2]float64{},
		Asks:      [50][2]float64{},
		ParseTime: time.Now(),
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
				orderBook.Bids[counter/2][counter%2], err = common.ParseFloat(bytes[collectStart:offset])
				if err != nil {
					return nil, fmt.Errorf("JsonKeyBids error %v mainLoop %d end %d %s", err, collectStart, offset, bytes[collectStart:offset])
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
				orderBook.Asks[counter/2][counter%2], err = common.ParseFloat(bytes[collectStart:offset])
				if err != nil {
					return nil, fmt.Errorf("JsonKeyAsks error %v mainLoop %d end %d %s", err, collectStart, offset, bytes[collectStart:offset])
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
			timestamp, err := common.ParseInt(bytes[collectStart:offset])
			if err != nil {
				return nil, fmt.Errorf("JsonKeyEventTime error %v mainLoop %d end %d %s", err, collectStart, offset, bytes[collectStart:offset])
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

func ParseDepth5(msg []byte, depth5 *Depth5) (err error) {

	//{"data":{"asks":[["55447.5","0.00128653"],["55447.6","0.0040067"],["55447.7","5.26962769"],["55449","0.00016278"],["55451.5","0.00013396"]],"bids":[["55403.1","0.01254575"],["55402.5","0.00005319"],["55279.9","0.201"],["55268.3","0.02406837"],["55233.5","0.0004668"]],"timestamp":1618724853172},"subject":"level2","topic":"/spotMarket/level2Depth5:BTC-USDT","type":"message"}`)
	//{"type":"message","topic":"/spotMarket/level2Depth5:ENJ-USDT","subject":"level2","data":{"asks":[["1.421","291.4019"],["1.4211","257.9855"],["1.4214","17.2666"],["1.4215","538.358"],["1.4217","2111.2333"]],"bids":[["1.4195","507.9287"],["1.4193","538.358"],["1.4191","308.6314"],["1.419","2110.5551"],["1.4188","4320.975"]],"timestamp":1627748904217}}

	msgLen := len(msg)
	end := 0
	start := 0
	currentKey := common.JsonKeyAsks
	counter := 0
	if msgLen > 128 {
		//{"data":{"asks":[["55447.5","0.00128653"],["55447.6","0.0040067"],["55447.7","5.26962769"],["55449","0.00016278"],["55451.5","0.00013396"]],"bids":[["55403.1","0.01254575"],["55402.5","0.00005319"],["55279.9","0.201"],["55268.3","0.02406837"],["55233.5","0.0004668"]],"timestamp":1618724853172},"subject":"level2","topic":"/spotMarket/level2Depth5:BTC-USDT","type":"message"}`)
		//{"type":"message","topic":"/spotMarket/level2Depth5:ENJ-USDT","subject":"level2","data":{"asks":[["1.421","291.4019"],["1.4211","257.9855"],["1.4214","17.2666"],["1.4215","538.358"],["1.4217","2111.2333"]],"bids":[["1.4195","507.9287"],["1.4193","538.358"],["1.4191","308.6314"],["1.419","2110.5551"],["1.4188","4320.975"]],"timestamp":1627748904217}}
		if msg[2] == 't' && msg[51] == ':' {
			if msg[60] == '"' {
				depth5.Symbol = common.UnsafeBytesToString(msg[52:60])
				end = 92
			} else if msg[61] == '"' {
				depth5.Symbol = common.UnsafeBytesToString(msg[52:61])
				end = 93
			} else if msg[62] == '"' {
				depth5.Symbol = common.UnsafeBytesToString(msg[52:62])
				end = 94
			} else if msg[63] == '"' {
				depth5.Symbol = common.UnsafeBytesToString(msg[52:63])
				end = 95
			} else if msg[59] == '"' {
				depth5.Symbol = common.UnsafeBytesToString(msg[52:59])
				end = 91
			} else if msg[64] == '"' {
				depth5.Symbol = common.UnsafeBytesToString(msg[52:64])
				end = 96
			} else {
				return fmt.Errorf("symbol not found %s", msg)
			}
			if msg[end] != 'k' && msg[end+1] != 's' && msg[end+2] != '"' {
				return fmt.Errorf("bad msg %s", msg)
			}
			end += 7
			start = end
		} else if msg[2] == 'd' {
			if msg[msgLen-28] == ':' {
				depth5.Symbol = common.UnsafeBytesToString(msg[msgLen-27 : msgLen-19])
			} else if msg[msgLen-29] == ':' {
				depth5.Symbol = common.UnsafeBytesToString(msg[msgLen-28 : msgLen-19])
			} else if msg[msgLen-30] == ':' {
				depth5.Symbol = common.UnsafeBytesToString(msg[msgLen-29 : msgLen-19])
			} else if msg[msgLen-31] == ':' {
				depth5.Symbol = common.UnsafeBytesToString(msg[msgLen-30 : msgLen-19])
			} else {
				return fmt.Errorf("symbol not found %s", msg)
			}
			end = 12
			if msg[end] != 'k' && msg[end+1] != 's' && msg[end+2] != '"' {
				return fmt.Errorf("bad msg %s", msg)
			}
			end += 7
			start = end
		} else {
			return fmt.Errorf("bad msg %s", msg)
		}
	} else {
		return fmt.Errorf("bad msg %s", msg)
	}

	for end < msgLen-2 {
		switch currentKey {
		case common.JsonKeyBids:
			if msg[end] == '"' {
				depth5.Bids[counter/2][counter%2], err = common.ParseFloat(msg[start:end])
				if err != nil {
					return fmt.Errorf("JsonKeyBids error %v mainLoop %d end %d %s", err, start, end, msg[start:end])
				}
				counter += 1
				if counter >= 10 || (msg[end+1] == ']' && msg[end+2] == ']') {
					currentKey = common.JsonKeyEventTime
					end += 16
					start = end
				} else if counter%2 == 0 {
					end += 5
					start = end
				} else {
					end += 3
					start = end
				}
				continue
			}
			break
		case common.JsonKeyAsks:
			if msg[end] == '"' {
				depth5.Asks[counter/2][counter%2], err = common.ParseFloat(msg[start:end])
				if err != nil {
					return fmt.Errorf("JsonKeyAsks error %v mainLoop %d end %d %s", err, start, end, msg[start:end])
				}
				counter += 1
				if counter >= 10 || (msg[end+1] == ']' && msg[end+2] == ']') {
					currentKey = common.JsonKeyBids
					end += 14
					start = end
					counter = 0
				} else if counter%2 == 0 {
					end += 5
					start = end
				} else {
					end += 3
					start = end
				}
				continue
			}
			break
		case common.JsonKeyEventTime:
			end += 13
			if end < msgLen {
				timestamp, err := common.ParseInt(msg[start:end])
				if err != nil {
					return fmt.Errorf("JsonKeyEventTime error %v mainLoop %d end %d %s", err, start, end, msg[start:end])
				}
				depth5.EventTime = time.Unix(0, timestamp*1000000)
				return nil
			} else {
				return fmt.Errorf("bad timestamp, out of range %s", msg)
			}
		}
		end += 1
	}
	return fmt.Errorf("bad msg, fall to end: %s", msg)
}

func AccountHttpLoop(
	ctx context.Context, api *API, param AccountsParam, interval time.Duration,
	output chan []Account,
) {
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			subCtx, _ := context.WithTimeout(ctx, time.Minute)
			account, err := api.GetAccounts(subCtx, param)
			if err != nil {
				logger.Debugf("AccountHttpLoop GetAccount error %v", err)
			} else {
				output <- account
			}
			timer.Reset(time.Now().Truncate(interval).Add(interval).Sub(time.Now()))
		}
	}
}

func GetOrderLimits(
	ctx context.Context,
	api *API,
	symbols []string,
) (minSizes, stepSizes, tickSizes, minNotional map[string]float64, err error) {
	var ss []Symbol
	ss, err = api.GetSymbols(ctx)
	if err != nil {
		return
	}
	stepSizes = make(map[string]float64)
	minSizes = make(map[string]float64)
	tickSizes = make(map[string]float64)
	minNotional = make(map[string]float64)
	symbolsMap := make(map[string]string)
	for _, symbol := range symbols {
		symbolsMap[symbol] = symbol
	}
	for _, s := range ss {
		if _, ok := symbolsMap[s.Symbol]; ok {
			delete(symbolsMap, s.Symbol)
			stepSizes[s.Symbol] = s.BaseIncrement
			minSizes[s.Symbol] = s.BaseMinSize
			tickSizes[s.Symbol] = s.QuoteIncrement
			minNotional[s.Symbol] = s.QuoteMinSize
		}
	}
	if len(symbolsMap) != 0 {
		err = fmt.Errorf("NO ORDER LIMITS FOR %v", symbolsMap)
	} else {
		logger.Debugf("STEP SIZES %v", minSizes)
		logger.Debugf("MIN SIZES %v", minSizes)
		logger.Debugf("TICK SIZES %v", tickSizes)
		logger.Debugf("MIN NOTIONAL %v", minNotional)
	}
	return
}

func WatchSystemStatusHttp(
	ctx context.Context, api *API, interval time.Duration,
	output chan bool,
) {
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			subCtx, _ := context.WithTimeout(ctx, time.Minute)
			systemStatus, err := api.GetSystemStatus(subCtx)
			if err != nil {
				logger.Debugf("api.GetSystemStatus error %v", err)
				select {
				case output <- false:
				default:
					logger.Debugf("WatchSystemStatusHttp send status out failed")
				}
			} else {
				if systemStatus.Status == SystemStatusOpen {
					select {
					case output <- true:
					default:
						logger.Debugf("WatchSystemStatusHttp send status out failed")
					}
				} else {
					select {
					case output <- false:
					default:
						logger.Debugf("WatchSystemStatusHttp send status out failed")
					}
				}
			}
			timer.Reset(time.Now().Truncate(interval).Add(interval).Sub(time.Now()))
		}
	}
}

func ParseTicker(msg []byte, ticker *Ticker) (err error) {

	//{"data":{"sequence":"1618200194453","bestAsk":"32704.5","size":"0.00058862","bestBidSize":"0.06704767","price":"32704.5","time":1626290937603,"bestAskSize":"0.01955972","bestBid":"32704.4"},"subject":"trade.ticker","topic":"/market/ticker:BTC-USDT","type":"message"}
	//{"type":"message","topic":"/market/ticker:BTC-USDT","subject":"trade.ticker","data":{"bestAsk":"41217.7","bestAskSize":"0.21545096","bestBid":"41217.6","bestBidSize":"0.0265","price":"41217.7","sequence":"1618607525224","size":"0.00043659","time":1627752855836}}
	msgLen := len(msg)
	end := 0
	if msgLen > 128 {
		if msg[2] == 't' && msg[9] == 'm' {
			if msg[50] == '"' {
				end = 85
				ticker.Symbol = common.UnsafeBytesToString(msg[42:50])
			} else if msg[51] == '"' {
				end = 86
				ticker.Symbol = common.UnsafeBytesToString(msg[42:51])
			} else if msg[52] == '"' {
				end = 87
				ticker.Symbol = common.UnsafeBytesToString(msg[42:52])
			} else if msg[53] == '"' {
				end = 88
				ticker.Symbol = common.UnsafeBytesToString(msg[42:53])
			} else if msg[49] == '"' {
				end = 84
				ticker.Symbol = common.UnsafeBytesToString(msg[42:49])
			} else {
				return fmt.Errorf("bad msg, symbol not found %s", msg)
			}
		} else if msg[2] == 'd' {
			if msg[msgLen-27] == ':' {
				ticker.Symbol = common.UnsafeBytesToString(msg[msgLen-26 : msgLen-19])
			} else if msg[msgLen-28] == ':' {
				ticker.Symbol = common.UnsafeBytesToString(msg[msgLen-27 : msgLen-19])
			} else if msg[msgLen-29] == ':' {
				ticker.Symbol = common.UnsafeBytesToString(msg[msgLen-28 : msgLen-19])
			} else if msg[msgLen-30] == ':' {
				ticker.Symbol = common.UnsafeBytesToString(msg[msgLen-29 : msgLen-19])
			} else if msg[msgLen-31] == ':' {
				ticker.Symbol = common.UnsafeBytesToString(msg[msgLen-30 : msgLen-19])
			} else {
				return fmt.Errorf("bad msg, symbol not found %s", msg)
			}
			end = 37
		}else{
			return fmt.Errorf("bad msg, symbol not found %s", msg)
		}
	} else {
		return fmt.Errorf("msg len less than 128, %s", msg)
	}

	start := end
	counter := 0
	currentKey := common.JsonKeyUnknown
	var t int64
	for end < msgLen-10 {
		switch currentKey {
		case common.JsonKeyBidSize:
			if msg[end] == '"' {
				ticker.BestBidSize, err = common.ParseDecimal(msg[start:end])
				if err != nil {
					return
				}
				currentKey = common.JsonKeyUnknown
				counter++
			}
			break
		case common.JsonKeyAskSize:
			if msg[end] == '"' {
				ticker.BestAskSize, err = common.ParseDecimal(msg[start:end])
				if err != nil {
					return
				}
				currentKey = common.JsonKeyUnknown
				counter++
			}
			break
		case common.JsonKeyBidPrice:
			if msg[end] == '"' {
				ticker.BestBidPrice, err = common.ParseDecimal(msg[start:end])
				if err != nil {
					return
				}
				currentKey = common.JsonKeyUnknown
				counter++
			}
			break
		case common.JsonKeyAskPrice:
			if msg[end] == '"' {
				ticker.BestAskPrice, err = common.ParseDecimal(msg[start:end])
				if err != nil {
					return
				}
				currentKey = common.JsonKeyUnknown
				counter++
			}
			break
		case common.JsonKeyUnknown:
			if msg[end] == 'b' && msg[end+4] == 'B' && msg[end+10] == 'e' {
				currentKey = common.JsonKeyBidSize
				end += 14
				start = end
				break
			} else if msg[end] == 'b' && msg[end+4] == 'B' && msg[end+7] == '"' {
				currentKey = common.JsonKeyBidPrice
				end += 10
				start = end
				break
			} else if msg[end] == 'b' && msg[end+4] == 'A' && msg[end+10] == 'e' {
				currentKey = common.JsonKeyAskSize
				end += 14
				start = end
				break
			} else if msg[end] == 'b' && msg[end+4] == 'A' && msg[end+7] == '"' {
				currentKey = common.JsonKeyAskPrice
				end += 10
				start = end
				break
			}else if msg[end] == 'm' && msg[end+1] == 'e' && msg[end+2] == '"' {
				end += 4
				start = end
				end += 13
				if end < msgLen {
					t, err = common.ParseInt(msg[start:end])
					if err != nil {
						return
					}
					ticker.EventTime = time.Unix(0, t*1000000)
					counter ++
				}
				break
			}
		}
		end ++
	}
	if counter != 5 {
		err = fmt.Errorf("bad msg, %d miss fileds %s", counter, msg)
	}
	return
}
