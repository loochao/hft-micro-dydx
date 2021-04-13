package kcperp

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"time"
	"unsafe"
)

//{"data":{"sequence":1617684642656,"asks":[[60818,160000],[60823.0,10325],[60824.0,3000],[60828.0,5325],[60829.0,5000],[60831.0,112438],[60833.0,3000],[60834.0,3750],[60836.0,80000],[60837.0,71844],[60838.0,54426],[60841.0,74100],[60842.0,60564],[60843.0,3000],[60848.0,73816],[60850.0,49770],[60851,80000],[60852.0,63455],[60855.0,54438],[60857.0,51719],[60859.0,123907],[60869.0,5697],[60888,1397],[60894.0,6089],[60900,6168],[60904,4134],[60905.0,6494],[60906.0,84187],[60938.0,119895],[60952.0,7333],[60990,13840],[61030.0,7764],[61040,12262],[61041.0,7764],[61060,12262],[61080,12262],[61100,12262],[61120,12262],[61138,142396],[61140,12262],[61180.0,8644],[61200,954],[61238,15],[61355,71706],[61400,10],[61416.0,541265],[61425.0,10000],[61427.0,606647],[61448.0,10000],[61500,1436]],"bids":[[60811,160000],[60810.0,124958],[60806.0,70501],[60804,145306],[60803.0,3000],[60801.0,52777],[60800.0,70833],[60798,80000],[60796.0,3000],[60794.0,54705],[60793.0,67823],[60789.0,58175],[60788.0,52072],[60784.0,59274],[60782.0,3000],[60780.0,72154],[60763.0,53314],[60755.0,5325],[60741.0,5697],[60736.0,1397],[60725.0,6494],[60711,100100],[60707.0,6494],[60693.0,62797],[60691.0,74153],[60662.0,6909],[60640,12262],[60627,25],[60622,13840],[60620,12262],[60613.0,7333],[60600.0,12362],[60589.0,7764],[60580,12262],[60560,12262],[60540,12262],[60500,1442],[60490,2029],[60488.0,8201],[60467.0,8644],[60458,141278],[60444,25],[60417,15],[60367.0,8201],[60363,70473],[60347.0,9092],[60327,50],[60300,341727],[60206.0,504206],[60201,12000]],"ts":1618219600281,"timestamp":1618219600281},"subject":"level2","topic":"/contractMarket/level2Depth50:XBTUSDM","type":"message"} {"data":{"sequence":1617684642656,"asks":[[60818,160000],[60823.0,10325],[60824.0,3000],[60828.0,5325],[60829.0,5000],[60831.0,112438],[60833.0,3000],[60834.0,3750],[60836.0,80000],[60837.0,71844],[60838.0,54426],[60841.0,74100],[60842.0,60564],[60843.0,3000],[60848.0,73816],[60850.0,49770],[60851,80000],[60852.0,63455],[60855.0,54438],[60857.0,51719],[60859.0,123907],[60869.0,5697],[60888,1397],[60894.0,6089],[60900,6168],[60904,4134],[60905.0,6494],[60906.0,84187],[60938.0,119895],[60952.0,7333],[60990,13840],[61030.0,7764],[61040,12262],[61041.0,7764],[61060,12262],[61080,12262],[61100,12262],[61120,12262],[61138,142396],[61140,12262],[61180.0,8644],[61200,954],[61238,15],[61355,71706],[61400,10],[61416.0,541265],[61425.0,10000],[61427.0,606647],[61448.0,10000],[61500,1436]],"bids":[[60811,160000],[60810.0,124958],[60806.0,70501],[60804,145306],[60803.0,3000],[60801.0,52777],[60800.0,70833],[60798,80000],[60796.0,3000],[60794.0,54705],[60793.0,67823],[60789.0,58175],[60788.0,52072],[60784.0,59274],[60782.0,3000],[60780.0,72154],[60763.0,53314],[60755.0,5325],[60741.0,5697],[60736.0,1397],[60725.0,6494],[60711,100100],[60707.0,6494],[60693.0,62797],[60691.0,74153],[60662.0,6909],[60640,12262],[60627,25],[60622,13840],[60620,12262],[60613.0,7333],[60600.0,12362],[60589.0,7764],[60580,12262],[60560,12262],[60540,12262],[60500,1442],[60490,2029],[60488.0,8201],[60467.0,8644],[60458,141278],[60444,25],[60417,15],[60367.0,8201],[60363,70473],[60347.0,9092],[60327,50],[60300,341727],[60206.0,504206],[60201,12000]],"ts":1618219600281,"timestamp":1618219600281},"subject":"level2","topic":"/contractMarket/level2Depth50:XBTUSDM","type":"message"}
func ParseDepth50(bytes []byte) (*Depth50, error) {
	var err error
	orderBook := Depth50{
		Bids:      [50][2]float64{},
		Asks:      [50][2]float64{},
		ParseTime: time.Now(),
	}
	offset := 16
	if bytes[offset] != 'c' && bytes[offset+1] != 'e' && bytes[offset+2] != '"' {
		return nil, fmt.Errorf("bad bytes %s", bytes)
	}
	offset = 20
	collectStart := offset
	bytesLen := len(bytes)
	counter := 0
	currentKey := common.JsonKeyLastUpdateId
	for offset < bytesLen-6 {
		switch currentKey {
		case common.JsonKeyLastUpdateId:
			if bytes[offset] == ',' {
				orderBook.Sequence, err = common.ParseBinanceInt(bytes[collectStart:offset])
				if err != nil {
					return nil, err
				}
				if bytes[offset+4] != 'k' && bytes[offset+5] != 's' && bytes[offset+6] != '"' {
					return nil, fmt.Errorf("bad bytes %s", bytes)
				}
				currentKey = common.JsonKeyAsks
				offset += 10
				collectStart = offset
			}
		case common.JsonKeyBids:
			if bytes[offset] == ',' || bytes[offset] == ']' {
				orderBook.Bids[counter/2][counter%2], err = common.ParseBinanceFloat(bytes[collectStart:offset])
				if err != nil {
					return nil, err
				}
				counter += 1
				if counter >= 100 {
					currentKey = common.JsonKeyEventTime
					offset += 8
					collectStart = offset
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
				orderBook.Asks[counter/2][counter%2], err = common.ParseBinanceFloat(bytes[collectStart:offset])
				if err != nil {
					return nil, err
				}
				counter += 1
				if counter >= 100 {
					currentKey = common.JsonKeyBids
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
		case common.JsonKeyEventTime:
			offset += 13
			timestamp, err := common.ParseBinanceInt(bytes[collectStart:offset])
			if err != nil {
				return nil, err
			}
			orderBook.EventTime = time.Unix(0, timestamp*1000000)
			offset += 86
			collectStart = offset
			offset += 6
			currentKey = common.JsonKeySymbol
			continue
		case common.JsonKeySymbol:
			if bytes[offset] == '"' {
				symbol := bytes[collectStart:offset]
				orderBook.Symbol = *(*string)(unsafe.Pointer(&symbol))
				offset = bytesLen
				//在此退出
				continue
			}
			break
		}
		offset += 1
	}
	return &orderBook, nil
}

func passPhraseEncrypt(key, plain []byte) string {
	hm := hmac.New(sha256.New, key)
	hm.Write(plain)
	return base64.StdEncoding.EncodeToString(hm.Sum(nil))
}

func WatchPositionsFromHttp(
	ctx context.Context, api *API,
	symbols []string, interval time.Duration,
	output chan []Position,
) {
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			subCtx, _ := context.WithTimeout(ctx, time.Minute)
			positions, err := api.GetPositions(subCtx)
			if err != nil {
				logger.Debugf("WatchPositionsFromHttp GetPositions error %v", err)
			} else {
				//有一种情况是有的合约的仓位是拉不到的, 拉不到的都是空仓
				positionBySymbols := make(map[string]Position)
				for _, symbol := range symbols {
					positionBySymbols[symbol] = Position{
						Symbol: symbol,
						ParseTime: time.Now(),
						EventTime: time.Now(),
					}
				}
				for _, position := range positions {
					position := position
					positionBySymbols[position.Symbol] = position
				}
				outPositions := make([]Position, len(symbols))
				for i, symbol := range symbols {
					outPositions[i] = positionBySymbols[symbol]
				}
				output <- outPositions
			}
			timer.Reset(time.Now().Truncate(interval).Add(interval).Sub(time.Now()))
		}
	}
}

func WatchAccountFromHttp(
	ctx context.Context, api *API, param AccountParam, interval time.Duration,
	output chan Account,
) {
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			subCtx, _ := context.WithTimeout(ctx, time.Minute)
			account, err := api.GetAccountOverView(subCtx, param)
			if err != nil {
				logger.Debugf("WatchAccountFromHttp GetAccountOverView error %v", err)
			} else {
				output <- *account
			}
			timer.Reset(time.Now().Truncate(interval).Add(interval).Sub(time.Now()))
		}
	}
}

func GetOrderLimits(
	ctx context.Context,
	api *API,
	symbols []string,
) (lotSizes, multipliers, tickSizes, maxPrices map[string]float64, err error) {
	var contracts []Contract
	contracts, err = api.GetContracts(ctx)
	if err != nil {
		return
	}
	lotSizes = make(map[string]float64)
	multipliers = make(map[string]float64)
	tickSizes = make(map[string]float64)
	maxPrices = make(map[string]float64)
	symbolsMap := make(map[string]string)
	for _, symbol := range symbols {
		symbolsMap[symbol] = symbol
	}
	for _, contract := range contracts {
		if _, ok := symbolsMap[contract.Symbol]; ok {
			delete(symbolsMap, contract.Symbol)
			multipliers[contract.Symbol] = contract.Multiplier
			lotSizes[contract.Symbol] = contract.LotSize
			tickSizes[contract.Symbol] = contract.TickSize
			maxPrices[contract.Symbol] = contract.MaxPrice
		}
	}
	if len(symbolsMap) != 0 {
		err = fmt.Errorf("NO ORDER LIMITS FOR %v", symbolsMap)
	} else {
		logger.Debugf("LOT SIZES %v", lotSizes)
		logger.Debugf("MULTIPLIERS %v", multipliers)
		logger.Debugf("TICK SIZES %v", tickSizes)
		logger.Debugf("MAX PRICES %v", maxPrices)
	}
	return
}
