package kcspot

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


func ParseDepth5(bytes []byte) (*Depth5, error) {
	var err error
	orderBook := Depth5{
		Bids:      [5][2]float64{},
		Asks:      [5][2]float64{},
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
				orderBook.Bids[counter/2][counter%2], err = common.ParseBinanceFloat(bytes[collectStart:offset])
				if err != nil {
					return nil, err
				}
				counter += 1
				if counter >= 10 {
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
				if counter >= 10 {
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
			offset += 55
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

func WatchAccountFromHttp(
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
				logger.Debugf("WatchAccountFromHttp GetAccounts error %v", err)
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
