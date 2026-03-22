package kucoin_coinfuture

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"strconv"
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
				orderBook.Sequence, err = common.ParseInt(bytes[collectStart:offset])
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
				orderBook.Bids[counter/2][counter%2], err = common.ParseFloat(bytes[collectStart:offset])
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
				orderBook.Asks[counter/2][counter%2], err = common.ParseFloat(bytes[collectStart:offset])
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
			timestamp, err := common.ParseInt(bytes[collectStart:offset])
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

func ParseDepth5(bytes []byte, depth5 *Depth5) error {
	var err error
	offset := 16
	if bytes[offset] != 'c' && bytes[offset+1] != 'e' && bytes[offset+2] != '"' {
		return fmt.Errorf("bad bytes %s", bytes)
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
				depth5.Sequence, err = common.ParseInt(bytes[collectStart:offset])
				if err != nil {
					return fmt.Errorf("JsonKeyLastUpdateId error %v mainLoop %d end %d %s", err, collectStart, offset, bytes[collectStart:offset])
				}
				if bytes[offset+4] != 'k' && bytes[offset+5] != 's' && bytes[offset+6] != '"' {
					return fmt.Errorf("bad bytes %s", bytes)
				}
				currentKey = common.JsonKeyAsks
				offset += 10
				collectStart = offset
			}
		case common.JsonKeyBids:
			if bytes[offset] == ',' || bytes[offset] == ']' {
				depth5.Bids[counter/2][counter%2], err = common.ParseDecimal(bytes[collectStart:offset])
				if err != nil {
					return fmt.Errorf("JsonKeyBids error %v mainLoop %d end %d %s", err, collectStart, offset, bytes[collectStart:offset])
				}
				counter += 1
				if counter >= 10 || bytes[offset+1] == ']' {
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
				depth5.Asks[counter/2][counter%2], err = common.ParseDecimal(bytes[collectStart:offset])
				if err != nil {
					return fmt.Errorf("JsonKeyAsks error %v mainLoop %d end %d %s", err, collectStart, offset, bytes[collectStart:offset])
				}
				counter += 1
				if counter >= 10 || bytes[offset+1] == ']' {
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
			timestamp, err := common.ParseInt(bytes[collectStart:offset])
			if err != nil {
				return fmt.Errorf("JsonKeyEventTime error %v mainLoop %d end %d %s", err, collectStart, offset, bytes[collectStart:offset])
			}
			depth5.EventTime = time.Unix(0, timestamp*1000000)
			offset += 85
			collectStart = offset
			offset += 6
			currentKey = common.JsonKeySymbol
			continue
		case common.JsonKeySymbol:
			if bytes[offset] == '"' {
				symbol := bytes[collectStart:offset]
				depth5.Symbol = *(*string)(unsafe.Pointer(&symbol))
				offset = bytesLen
				//在此退出
				continue
			}
			break
		}
		offset += 1
	}
	return nil
}

func passPhraseEncrypt(key, plain []byte) string {
	hm := hmac.New(sha256.New, key)
	hm.Write(plain)
	return base64.StdEncoding.EncodeToString(hm.Sum(nil))
}

func PositionsHttpLoop(
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
				logger.Debugf("api.GetPositions error %v", err)
			} else {
				//有一种情况是有的合约的仓位是拉不到的, 拉不到的都是空仓
				positionBySymbols := make(map[string]Position)
				for _, symbol := range symbols {
					positionBySymbols[symbol] = Position{
						Symbol:    symbol,
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

func AccountHttpLoop(
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
				logger.Debugf("api.GetAccountOverView error %v", err)
			} else {
				output <- *account
			}
			timer.Reset(time.Now().Truncate(interval).Add(interval).Sub(time.Now()))
		}
	}
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

func FundingRateLoop(
	ctx context.Context, api *API, symbols []string, interval time.Duration,
	output chan CurrentFundingRate,
) {
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			for _, symbol := range symbols {
				subCtx, _ := context.WithTimeout(ctx, time.Minute)
				fr, err := api.GetCurrentFundingRate(subCtx, symbol)
				if err != nil {
					logger.Debugf("api.GetCurrentFundingRate error %v", err)
				} else {
					output <- *fr
				}
			}
			timer.Reset(time.Now().Truncate(interval).Add(interval).Sub(time.Now()))
		}
	}
}

func ParseDepth5JsonWalker(data []byte) (*Depth5, error) {
	var err error
	orderBook := Depth5{
		Bids:      [5][2]float64{},
		Asks:      [5][2]float64{},
	}
	walker := common.NewJsonWalker(data)
	if !walker.Advance(16) {
		return nil, fmt.Errorf("bad bytes %s", data)
	}
	walker.ResetStart()
	if !walker.Advance(3) {
		return nil, fmt.Errorf("bad bytes %s", data)
	}
	if walker.CollectString() != "ce\"" {
		return nil, fmt.Errorf("bad bytes %s", data)
	}
	if !walker.Advance(17) {
		return nil, fmt.Errorf("bad bytes %s", data)
	}
	walker.ResetStart()
	counter := 0
	currentKey := common.JsonKeyLastUpdateId
	for walker.Advance(1) && walker.End() < walker.Len()-6 {
		switch currentKey {
		case common.JsonKeyLastUpdateId:
			if walker.Get(0) == ',' {
				orderBook.Sequence, err = common.ParseInt(walker.Collect())
				if err != nil {
					return nil, fmt.Errorf("JsonKeyLastUpdateId error %v mainLoop %d end %d %s", err, walker.Start(), walker.End(), walker.Collect())
				}
				if walker.Get(4) != 'k' && walker.Get(5) != 's' && walker.Get(6) != '"' {
					return nil, fmt.Errorf("bad bytes %s", data)
				}
				currentKey = common.JsonKeyAsks
				if !walker.Advance(10) {
					return nil, fmt.Errorf("bad bytes %s", data)
				}
				walker.ResetStart()
			}
		case common.JsonKeyBids:
			if walker.Get(0) == ',' || walker.Get(0) == ']' {
				orderBook.Bids[counter/2][counter%2], err = common.ParseDecimal(walker.Collect())
				if err != nil {
					return nil, fmt.Errorf("JsonKeyBids error %v mainLoop %d end %d %s", err, walker.Start(), walker.End(), walker.Collect())
				}
				counter += 1
				if counter >= 10 || walker.Get(1) == ']' {
					currentKey = common.JsonKeyEventTime
					if !walker.Advance(8) {
						return nil, fmt.Errorf("bad bytes %s", data)
					}
					walker.ResetStart()
				} else if counter%2 == 0 {
					if !walker.Advance(3) {
						return nil, fmt.Errorf("bad bytes %s", data)
					}
					walker.ResetStart()
				} else {
					if !walker.Advance(1) {
						return nil, fmt.Errorf("bad bytes %s", data)
					}
					walker.ResetStart()
				}
				continue
			}
			break
		case common.JsonKeyAsks:
			if walker.Get(0) == ',' || walker.Get(0) == ']' {
				orderBook.Asks[counter/2][counter%2], err = common.ParseDecimal(walker.Collect())
				if err != nil {
					return nil, fmt.Errorf("JsonKeyAsks error %v mainLoop %d end %d %s", err, walker.Start(), walker.End(), walker.Collect())
				}
				counter += 1
				if counter >= 10 || walker.Get(1) == ']' {
					currentKey = common.JsonKeyBids
					if !walker.Advance(12) {
						return nil, fmt.Errorf("bad bytes %s", data)
					}
					walker.ResetStart()
					counter = 0
				} else if counter%2 == 0 {
					if !walker.Advance(3) {
						return nil, fmt.Errorf("bad bytes %s", data)
					}
					walker.ResetStart()
				} else {
					if !walker.Advance(1) {
						return nil, fmt.Errorf("bad bytes %s", data)
					}
					walker.ResetStart()
				}
				continue
			}
			break
		case common.JsonKeyEventTime:
			if !walker.Advance(13) {
				return nil, fmt.Errorf("bad bytes %s", data)
			}
			walker.ResetStart()
			timestamp, err := strconv.ParseInt(common.UnsafeBytesToString(walker.Collect()), 10, 64)
			if err != nil {
				return nil, fmt.Errorf("JsonKeyEventTime error %v mainLoop %d end %d %s", err, walker.Start(), walker.End(), walker.Collect())
			}
			orderBook.EventTime = time.Unix(0, timestamp*1000000)
			if !walker.Advance(85) {
				return nil, fmt.Errorf("bad bytes %s", data)
			}
			walker.ResetStart()
			if !walker.Advance(6) {
				return nil, fmt.Errorf("bad bytes %s", data)
			}
			currentKey = common.JsonKeySymbol
			continue
		case common.JsonKeySymbol:
			if walker.Get(0) == '"' {
				orderBook.Symbol = common.UnsafeBytesToString(walker.Collect())
				walker.Advance(walker.Len())
				//在此退出
				continue
			}
			break
		}
	}
	return &orderBook, nil
}


var Depth5SampleLines = `{"data":{"sequence":1617731938253,"asks":[[20.648,103],[20.652,10],[20.653,187],[20.654,255],[20.659,98]],"bids":[[20.628,96],[20.624,88],[20.619,270],[20.617,117],[20.615,98]],"ts":1623545077189,"timestamp":1623545077189},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617731938254,"asks":[[20.648,103],[20.652,10],[20.653,187],[20.654,359],[20.659,98]],"bids":[[20.628,96],[20.624,88],[20.619,270],[20.617,117],[20.615,98]],"ts":1623545077346,"timestamp":1623545077346},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617731938255,"asks":[[20.648,103],[20.652,10],[20.653,187],[20.654,359],[20.659,98]],"bids":[[20.628,96],[20.624,88],[20.621,289],[20.619,10],[20.617,117]],"ts":1623545077463,"timestamp":1623545077463},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617731938256,"asks":[[20.648,103],[20.652,10],[20.653,187],[20.654,359],[20.659,98]],"bids":[[20.628,96],[20.624,88],[20.621,289],[20.619,10],[20.617,117]],"ts":1623545077564,"timestamp":1623545077564},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617731938257,"asks":[[20.648,103],[20.652,10],[20.653,187],[20.654,359],[20.659,98]],"bids":[[20.628,96],[20.624,88],[20.622,257],[20.619,10],[20.617,117]],"ts":1623545077741,"timestamp":1623545077741},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617731938258,"asks":[[20.647,187],[20.648,103],[20.652,10],[20.654,359],[20.659,98]],"bids":[[20.628,96],[20.624,88],[20.622,257],[20.619,10],[20.617,117]],"ts":1623545077920,"timestamp":1623545077920},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617731938259,"asks":[[20.647,187],[20.648,103],[20.652,10],[20.654,359],[20.659,98]],"bids":[[20.628,96],[20.624,88],[20.622,257],[20.619,10],[20.617,117]],"ts":1623545078020,"timestamp":1623545078020},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617731938260,"asks":[[20.647,187],[20.648,103],[20.652,10],[20.654,359],[20.659,98]],"bids":[[20.628,96],[20.624,88],[20.622,257],[20.619,10],[20.617,117]],"ts":1623545078120,"timestamp":1623545078120},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617732876572,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.75,10435],[2367.85,7000]],"bids":[[2366.25,20000],[2366.20,6786],[2366.15,7000],[2365.40,6342],[2365.05,5532]],"ts":1623545078123,"timestamp":1623545078123},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731938261,"asks":[[20.647,187],[20.648,103],[20.652,10],[20.654,359],[20.659,98]],"bids":[[20.628,96],[20.624,88],[20.622,257],[20.619,10],[20.617,117]],"ts":1623545078220,"timestamp":1623545078220},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617732876573,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.75,10435],[2367.85,7000]],"bids":[[2366.25,20000],[2366.20,6786],[2366.15,7000],[2365.40,6342],[2365.05,5532]],"ts":1623545078223,"timestamp":1623545078223},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731938262,"asks":[[20.647,187],[20.648,103],[20.652,10],[20.654,359],[20.659,98]],"bids":[[20.628,96],[20.624,88],[20.622,257],[20.619,10],[20.617,117]],"ts":1623545078321,"timestamp":1623545078321},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617732876574,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.75,10435],[2367.85,7000]],"bids":[[2366.25,20000],[2366.20,6786],[2366.15,7000],[2365.40,6342],[2365.05,5532]],"ts":1623545078323,"timestamp":1623545078323},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731938263,"asks":[[20.647,187],[20.648,103],[20.652,10],[20.654,359],[20.659,98]],"bids":[[20.628,96],[20.624,88],[20.622,257],[20.619,10],[20.617,117]],"ts":1623545078421,"timestamp":1623545078421},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617732876575,"asks":[[2367.20,5622],[2367.35,10103],[2367.75,10435],[2367.85,7000],[2368.00,6324]],"bids":[[2366.25,20000],[2366.20,6786],[2366.15,7000],[2365.40,6342],[2365.05,5532]],"ts":1623545078509,"timestamp":1623545078509},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731938264,"asks":[[20.647,187],[20.648,103],[20.652,10],[20.654,359],[20.659,98]],"bids":[[20.628,96],[20.624,88],[20.622,257],[20.619,10],[20.617,117]],"ts":1623545078521,"timestamp":1623545078521},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617732876576,"asks":[[2367.20,5622],[2367.35,10103],[2367.75,10435],[2367.85,7000],[2368.00,6324]],"bids":[[2366.25,20000],[2366.20,6786],[2366.15,7000],[2365.40,6342],[2365.05,5532]],"ts":1623545078609,"timestamp":1623545078609},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731938265,"asks":[[20.647,187],[20.648,103],[20.652,10],[20.654,359],[20.659,98]],"bids":[[20.628,96],[20.624,88],[20.622,257],[20.619,10],[20.617,117]],"ts":1623545078622,"timestamp":1623545078622},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617732876577,"asks":[[2367.20,5622],[2367.35,10103],[2367.75,10435],[2367.85,7000],[2368.00,6324]],"bids":[[2366.25,20000],[2366.20,6786],[2366.15,7000],[2365.40,6342],[2365.05,5532]],"ts":1623545078709,"timestamp":1623545078709},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731938266,"asks":[[20.647,187],[20.648,103],[20.652,10],[20.654,359],[20.659,98]],"bids":[[20.628,96],[20.624,88],[20.622,257],[20.619,10],[20.617,117]],"ts":1623545078723,"timestamp":1623545078723},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617732876578,"asks":[[2367.20,5622],[2367.35,10103],[2367.75,10435],[2367.85,7000],[2368.00,6324]],"bids":[[2366.25,20000],[2366.20,6786],[2366.15,7000],[2365.40,6342],[2365.05,5532]],"ts":1623545078809,"timestamp":1623545078809},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731938267,"asks":[[20.647,187],[20.648,103],[20.652,10],[20.654,359],[20.659,98]],"bids":[[20.628,96],[20.624,88],[20.622,257],[20.619,10],[20.617,117]],"ts":1623545078823,"timestamp":1623545078823},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617732876579,"asks":[[2367.20,5622],[2367.35,10103],[2367.75,10435],[2367.85,7000],[2368.00,6324]],"bids":[[2366.25,20000],[2366.20,6786],[2366.15,7000],[2365.40,6342],[2365.05,5532]],"ts":1623545078909,"timestamp":1623545078909},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731938268,"asks":[[20.647,187],[20.648,103],[20.652,10],[20.654,104],[20.656,264]],"bids":[[20.628,96],[20.624,88],[20.622,257],[20.619,10],[20.617,117]],"ts":1623545078979,"timestamp":1623545078979},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617732876580,"asks":[[2367.20,5622],[2367.35,10103],[2367.75,10435],[2367.85,7000],[2368.00,6324]],"bids":[[2366.25,20000],[2366.20,6786],[2366.15,7000],[2365.40,6342],[2365.05,5532]],"ts":1623545079009,"timestamp":1623545079009},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731938269,"asks":[[20.647,187],[20.648,103],[20.652,10],[20.654,104],[20.658,301]],"bids":[[20.628,96],[20.624,88],[20.622,257],[20.619,10],[20.617,117]],"ts":1623545079107,"timestamp":1623545079107},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617732876581,"asks":[[2367.20,5622],[2367.35,10103],[2367.75,10435],[2367.85,7000],[2368.00,6324]],"bids":[[2366.25,20000],[2366.20,6786],[2366.15,7000],[2365.40,6342],[2365.05,5532]],"ts":1623545079109,"timestamp":1623545079109},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731473905,"asks":[[35520.0,64205],[35524,31469],[35525,31572],[35534,27899],[35536.00000000,3158]],"bids":[[35504,219],[35503.00000000,3487],[35500.0,631],[35498,12290],[35494,31560]],"ts":1623545079162,"timestamp":1623545079162},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617732876582,"asks":[[2367.20,5622],[2367.35,10103],[2367.75,10435],[2367.85,7000],[2368.00,6324]],"bids":[[2366.25,20000],[2366.20,6786],[2366.15,7000],[2365.40,6342],[2365.05,5532]],"ts":1623545079209,"timestamp":1623545079209},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731938270,"asks":[[20.647,187],[20.648,103],[20.652,10],[20.654,104],[20.658,301]],"bids":[[20.628,96],[20.624,88],[20.622,257],[20.619,10],[20.617,117]],"ts":1623545079239,"timestamp":1623545079239},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617731473906,"asks":[[35520.0,64205],[35525,31572],[35536.00000000,3158],[35539,80000],[35545,31860]],"bids":[[35504,219],[35503.00000000,3487],[35500.0,631],[35498,12290],[35493,80000]],"ts":1623545079264,"timestamp":1623545079264},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617732876583,"asks":[[2367.20,5622],[2367.35,10103],[2367.75,10435],[2367.85,7000],[2368.00,6324]],"bids":[[2366.25,20000],[2366.20,6786],[2366.15,7000],[2365.40,6342],[2365.05,5532]],"ts":1623545079309,"timestamp":1623545079309},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617732876584,"asks":[[2367.20,5622],[2367.35,10103],[2367.75,10435],[2367.85,7000],[2368.00,6324]],"bids":[[2366.25,20000],[2366.20,6786],[2366.15,7000],[2365.40,6342],[2365.05,5532]],"ts":1623545079409,"timestamp":1623545079409},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731473907,"asks":[[35520.0,64205],[35525,31572],[35526,24240],[35536.00000000,3158],[35539,80000]],"bids":[[35504,219],[35503.00000000,3487],[35500.0,631],[35498,12290],[35497,26370]],"ts":1623545079415,"timestamp":1623545079415},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617731938271,"asks":[[20.647,187],[20.648,103],[20.652,10],[20.654,104],[20.658,301]],"bids":[[20.628,96],[20.625,293],[20.624,88],[20.620,246],[20.619,10]],"ts":1623545079415,"timestamp":1623545079415},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617732876585,"asks":[[2367.20,5622],[2367.35,10103],[2367.75,10435],[2367.85,7000],[2368.00,6324]],"bids":[[2366.25,20000],[2366.20,6786],[2366.15,7000],[2365.40,6342],[2365.05,5532]],"ts":1623545079509,"timestamp":1623545079509},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731473908,"asks":[[35520.0,64205],[35525,31572],[35526,24240],[35536.00000000,3158],[35539,80000]],"bids":[[35504,219],[35503.00000000,3487],[35500.0,631],[35498,12290],[35497,26370]],"ts":1623545079515,"timestamp":1623545079515},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617731938272,"asks":[[20.647,187],[20.648,103],[20.652,10],[20.654,104],[20.658,301]],"bids":[[20.628,96],[20.625,293],[20.624,88],[20.620,246],[20.619,10]],"ts":1623545079519,"timestamp":1623545079519},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617732876586,"asks":[[2367.20,5622],[2367.35,10103],[2367.75,10435],[2367.85,7000],[2368.00,6324]],"bids":[[2366.25,20000],[2366.20,6786],[2366.15,7000],[2365.40,6342],[2365.05,5532]],"ts":1623545079610,"timestamp":1623545079610},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731473909,"asks":[[35520.0,64205],[35525,31572],[35526,24240],[35536.00000000,3158],[35539,80000]],"bids":[[35504,219],[35503.00000000,3487],[35500.0,631],[35498,12290],[35497,26370]],"ts":1623545079615,"timestamp":1623545079615},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617731938273,"asks":[[20.647,187],[20.648,103],[20.652,10],[20.654,104],[20.658,301]],"bids":[[20.628,96],[20.625,293],[20.624,88],[20.620,246],[20.619,10]],"ts":1623545079619,"timestamp":1623545079619},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617732876587,"asks":[[2367.20,5622],[2367.35,10103],[2367.75,10435],[2367.85,7000],[2368.00,6324]],"bids":[[2366.25,20000],[2366.20,6786],[2366.15,7000],[2365.40,6342],[2365.05,5532]],"ts":1623545079710,"timestamp":1623545079710},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731473910,"asks":[[35520.0,64205],[35525,31572],[35526,24240],[35536.00000000,3158],[35539,80000]],"bids":[[35504,219],[35503.00000000,3487],[35500.0,631],[35498,12290],[35497,26370]],"ts":1623545079715,"timestamp":1623545079715},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617731938274,"asks":[[20.647,187],[20.648,103],[20.652,10],[20.654,104],[20.658,301]],"bids":[[20.628,96],[20.625,293],[20.624,88],[20.620,246],[20.619,10]],"ts":1623545079719,"timestamp":1623545079719},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617732876588,"asks":[[2367.20,5622],[2367.35,10103],[2367.75,10435],[2367.85,7000],[2368.00,6324]],"bids":[[2366.25,20000],[2366.20,6786],[2366.15,7000],[2365.40,6342],[2365.05,5532]],"ts":1623545079810,"timestamp":1623545079810},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731473911,"asks":[[35520.0,64205],[35525,31572],[35526,24240],[35536.00000000,3158],[35539,80000]],"bids":[[35504,219],[35503.00000000,3487],[35500.0,631],[35498,12290],[35497,26370]],"ts":1623545079815,"timestamp":1623545079815},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617731938275,"asks":[[20.647,187],[20.648,103],[20.652,10],[20.654,104],[20.658,301]],"bids":[[20.628,96],[20.625,293],[20.624,88],[20.620,246],[20.619,10]],"ts":1623545079819,"timestamp":1623545079819},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617732876589,"asks":[[2367.20,5622],[2367.35,10103],[2367.75,10435],[2367.85,7000],[2368.00,6324]],"bids":[[2366.25,20000],[2366.20,6786],[2366.15,7000],[2365.40,6342],[2365.05,5532]],"ts":1623545079911,"timestamp":1623545079911},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731473912,"asks":[[35520.0,64205],[35525,31572],[35526,24240],[35536.00000000,3158],[35539,80000]],"bids":[[35504,219],[35503.00000000,3487],[35500.0,631],[35498,12290],[35497,26370]],"ts":1623545079915,"timestamp":1623545079915},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617731938276,"asks":[[20.647,187],[20.648,103],[20.652,10],[20.654,104],[20.658,301]],"bids":[[20.628,96],[20.625,293],[20.624,88],[20.620,246],[20.619,10]],"ts":1623545079919,"timestamp":1623545079919},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617732876590,"asks":[[2367.20,5622],[2367.35,10103],[2367.75,10435],[2367.85,7000],[2368.00,6324]],"bids":[[2366.25,20000],[2366.20,6786],[2366.15,7000],[2365.40,6342],[2365.05,5532]],"ts":1623545080012,"timestamp":1623545080012},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731473913,"asks":[[35520.0,64205],[35525,31572],[35526,24240],[35536.00000000,3158],[35539,80000]],"bids":[[35504,219],[35503.00000000,3487],[35500.0,631],[35498,12290],[35497,26370]],"ts":1623545080015,"timestamp":1623545080015},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617731938277,"asks":[[20.647,187],[20.648,103],[20.652,10],[20.654,104],[20.658,301]],"bids":[[20.628,96],[20.626,330],[20.624,88],[20.620,246],[20.619,10]],"ts":1623545080031,"timestamp":1623545080031},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617730623009,"asks":[[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750],[0.8322,4455]],"bids":[[0.8316,240],[0.8312,4815],[0.8311,5965],[0.8310,9809],[0.8309,9395]],"ts":1623545080086,"timestamp":1623545080086},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617732876591,"asks":[[2367.20,5622],[2367.35,10103],[2367.75,10435],[2367.85,7000],[2368.00,6324]],"bids":[[2366.25,20000],[2366.20,6786],[2366.15,7000],[2365.40,6342],[2365.05,5532]],"ts":1623545080112,"timestamp":1623545080112},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731473914,"asks":[[35520.0,64205],[35525,31572],[35526,24240],[35536.00000000,3158],[35539,80000]],"bids":[[35504,219],[35503.00000000,3487],[35500.0,631],[35498,12290],[35497,26370]],"ts":1623545080115,"timestamp":1623545080115},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617731938278,"asks":[[20.647,187],[20.648,103],[20.652,10],[20.654,104],[20.658,301]],"bids":[[20.628,96],[20.626,330],[20.624,88],[20.620,246],[20.619,10]],"ts":1623545080131,"timestamp":1623545080131},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617730623010,"asks":[[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750],[0.8322,10055]],"bids":[[0.8316,240],[0.8312,4815],[0.8311,5965],[0.8310,4014],[0.8309,5010]],"ts":1623545080191,"timestamp":1623545080191},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617732876592,"asks":[[2367.20,5622],[2367.35,10103],[2367.75,10435],[2367.85,7000],[2368.00,6324]],"bids":[[2366.25,20000],[2366.20,6786],[2366.15,7000],[2365.40,6342],[2365.05,5532]],"ts":1623545080212,"timestamp":1623545080212},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731473915,"asks":[[35520.0,64205],[35525,31572],[35526,24240],[35536.00000000,3158],[35539,80000]],"bids":[[35504,219],[35503.00000000,3487],[35500.0,631],[35498,12290],[35497,26370]],"ts":1623545080216,"timestamp":1623545080216},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617731938279,"asks":[[20.647,187],[20.648,103],[20.652,10],[20.654,104],[20.658,301]],"bids":[[20.628,96],[20.624,88],[20.622,357],[20.619,10],[20.617,117]],"ts":1623545080267,"timestamp":1623545080267},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617731473916,"asks":[[35520.0,64205],[35525,31572],[35526,24240],[35536.00000000,3158],[35539,80000]],"bids":[[35504,219],[35503.00000000,3487],[35500.0,631],[35498,12290],[35497,26370]],"ts":1623545080316,"timestamp":1623545080316},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617732876593,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.75,10435],[2367.85,7000]],"bids":[[2366.25,20000],[2366.20,6786],[2366.15,7000],[2365.40,6342],[2365.05,5532]],"ts":1623545080326,"timestamp":1623545080326},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617730623011,"asks":[[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750],[0.8322,10055]],"bids":[[0.8316,240],[0.8312,4815],[0.8311,5965],[0.8310,4014],[0.8309,5010]],"ts":1623545080365,"timestamp":1623545080365},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617731938280,"asks":[[20.647,187],[20.648,103],[20.652,10],[20.654,104],[20.658,301]],"bids":[[20.628,96],[20.624,88],[20.622,357],[20.619,10],[20.617,117]],"ts":1623545080411,"timestamp":1623545080411},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617731473917,"asks":[[35520.0,64205],[35525,31572],[35526,24240],[35536.00000000,3158],[35539,80000]],"bids":[[35504,219],[35503.00000000,3487],[35500.0,631],[35498,12290],[35497,26370]],"ts":1623545080416,"timestamp":1623545080416},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617732876594,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.75,10435],[2367.85,7000]],"bids":[[2366.25,20000],[2366.20,6786],[2366.15,7000],[2365.40,6342],[2365.05,5532]],"ts":1623545080426,"timestamp":1623545080426},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617730623012,"asks":[[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750],[0.8322,10055]],"bids":[[0.8316,240],[0.8312,4815],[0.8311,5965],[0.8310,4014],[0.8309,5010]],"ts":1623545080480,"timestamp":1623545080480},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617731473918,"asks":[[35520.0,64205],[35525,31572],[35526,24240],[35536.00000000,3158],[35539,80000]],"bids":[[35504,219],[35503.00000000,3487],[35500.0,631],[35498,12290],[35497,26370]],"ts":1623545080516,"timestamp":1623545080516},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617732876595,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.75,10435],[2367.85,7000]],"bids":[[2366.25,20000],[2366.20,6786],[2366.15,7000],[2365.40,6342],[2365.05,5532]],"ts":1623545080526,"timestamp":1623545080526},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731938281,"asks":[[20.647,187],[20.648,103],[20.652,10],[20.654,104],[20.659,98]],"bids":[[20.628,96],[20.624,88],[20.619,10],[20.617,423],[20.615,98]],"ts":1623545080515,"timestamp":1623545080515},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617730623013,"asks":[[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750],[0.8322,10055]],"bids":[[0.8316,240],[0.8312,4815],[0.8311,5965],[0.8310,4014],[0.8309,5010]],"ts":1623545080580,"timestamp":1623545080580},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617731473919,"asks":[[35520.0,64205],[35525,31572],[35526,24240],[35536.00000000,3158],[35539,80000]],"bids":[[35504,219],[35503.00000000,3487],[35500.0,631],[35498,12290],[35497,26370]],"ts":1623545080616,"timestamp":1623545080616},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617732876596,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.75,10435],[2367.85,7000]],"bids":[[2366.25,20000],[2366.20,6786],[2366.15,7000],[2365.40,6342],[2365.05,5532]],"ts":1623545080626,"timestamp":1623545080626},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731938282,"asks":[[20.647,187],[20.648,103],[20.652,10],[20.654,104],[20.656,322]],"bids":[[20.628,96],[20.624,88],[20.619,10],[20.617,423],[20.615,98]],"ts":1623545080658,"timestamp":1623545080658},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617730623014,"asks":[[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750],[0.8322,10055]],"bids":[[0.8316,240],[0.8312,4815],[0.8311,5965],[0.8310,4014],[0.8309,5010]],"ts":1623545080705,"timestamp":1623545080705},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617731473920,"asks":[[35520.0,64205],[35525,31572],[35526,24240],[35536.00000000,3158],[35539,80000]],"bids":[[35504,219],[35503.00000000,3487],[35500.0,631],[35498,12290],[35497,26370]],"ts":1623545080716,"timestamp":1623545080716},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617732876597,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.75,10435],[2367.85,7000]],"bids":[[2366.25,20000],[2366.20,6786],[2366.15,7000],[2365.40,6342],[2365.05,5532]],"ts":1623545080726,"timestamp":1623545080726},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731938283,"asks":[[20.647,187],[20.648,103],[20.652,10],[20.654,104],[20.656,322]],"bids":[[20.628,96],[20.624,88],[20.619,10],[20.617,423],[20.615,98]],"ts":1623545080758,"timestamp":1623545080758},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617731473921,"asks":[[35520.0,64205],[35525,31572],[35526,24240],[35536.00000000,3158],[35539,80000]],"bids":[[35504,219],[35503.00000000,3487],[35500.0,631],[35498,12290],[35497,26370]],"ts":1623545080816,"timestamp":1623545080816},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617732876598,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.75,10435],[2367.85,7000]],"bids":[[2366.25,20000],[2366.20,6786],[2366.15,7000],[2365.40,6342],[2365.05,5532]],"ts":1623545080826,"timestamp":1623545080826},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617730623015,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8312,4815],[0.8311,5965],[0.8310,4014],[0.8309,5010]],"ts":1623545080850,"timestamp":1623545080850},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617731938284,"asks":[[20.647,187],[20.648,103],[20.652,10],[20.654,104],[20.658,264]],"bids":[[20.628,96],[20.624,88],[20.619,10],[20.617,423],[20.615,98]],"ts":1623545080879,"timestamp":1623545080879},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617732876599,"asks":[[2367.20,5622],[2367.35,10103],[2367.75,10435],[2367.85,7000],[2368.00,6324]],"bids":[[2366.25,20000],[2366.20,6786],[2366.15,7000],[2365.40,6342],[2365.05,5532]],"ts":1623545080981,"timestamp":1623545080981},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731938285,"asks":[[20.647,187],[20.648,103],[20.652,10],[20.654,104],[20.659,98]],"bids":[[20.628,96],[20.624,88],[20.619,10],[20.617,423],[20.615,98]],"ts":1623545080995,"timestamp":1623545080995},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617730623016,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8312,4815],[0.8311,5965],[0.8310,4014],[0.8309,5010]],"ts":1623545080996,"timestamp":1623545080996},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617731473922,"asks":[[35520.0,64205],[35525,31572],[35526,24240],[35536.00000000,3158],[35539,80000]],"bids":[[35504,219],[35503.00000000,3487],[35500.0,631],[35498,12290],[35497,26370]],"ts":1623545081004,"timestamp":1623545081004},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617732876600,"asks":[[2367.20,5622],[2367.35,10103],[2367.75,10435],[2367.85,7000],[2368.00,6324]],"bids":[[2366.25,20000],[2366.20,6786],[2366.15,7000],[2365.40,6342],[2365.05,5532]],"ts":1623545081081,"timestamp":1623545081081},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731938286,"asks":[[20.647,187],[20.648,103],[20.652,10],[20.654,104],[20.659,98]],"bids":[[20.628,96],[20.624,88],[20.619,10],[20.617,423],[20.615,98]],"ts":1623545081095,"timestamp":1623545081095},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617730623017,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8312,4815],[0.8311,5965],[0.8310,4014],[0.8309,5010]],"ts":1623545081096,"timestamp":1623545081096},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617731473923,"asks":[[35520.0,64205],[35525,31572],[35526,24240],[35536.00000000,3158],[35539,80000]],"bids":[[35504,219],[35503.00000000,3487],[35500.0,631],[35498,12290],[35497,26370]],"ts":1623545081104,"timestamp":1623545081104},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617732876601,"asks":[[2367.20,5622],[2367.35,10103],[2367.75,10435],[2367.85,7000],[2368.00,6324]],"bids":[[2366.25,20000],[2366.20,6786],[2366.15,7000],[2365.40,6342],[2365.05,5532]],"ts":1623545081181,"timestamp":1623545081181},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617730623018,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8312,4815],[0.8311,5965],[0.8310,4014],[0.8309,5010]],"ts":1623545081196,"timestamp":1623545081196},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617731938287,"asks":[[20.647,187],[20.648,103],[20.652,10],[20.654,104],[20.659,98]],"bids":[[20.628,96],[20.624,88],[20.619,10],[20.617,423],[20.615,98]],"ts":1623545081200,"timestamp":1623545081200},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617731473924,"asks":[[35520.0,64205],[35525,31572],[35526,24240],[35536.00000000,3158],[35539,80000]],"bids":[[35504,219],[35503.00000000,3487],[35500.0,631],[35498,12290],[35497,26370]],"ts":1623545081235,"timestamp":1623545081235},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617732876602,"asks":[[2367.20,5622],[2367.35,10103],[2367.75,10435],[2367.85,7000],[2368.00,6324]],"bids":[[2366.25,20000],[2366.20,6786],[2366.15,7000],[2365.40,6342],[2365.05,5532]],"ts":1623545081281,"timestamp":1623545081281},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617730623019,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8312,4815],[0.8311,5965],[0.8310,4014],[0.8309,5010]],"ts":1623545081296,"timestamp":1623545081296},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617731938288,"asks":[[20.647,187],[20.648,103],[20.652,10],[20.654,104],[20.659,98]],"bids":[[20.628,96],[20.624,88],[20.619,10],[20.617,423],[20.615,98]],"ts":1623545081300,"timestamp":1623545081300},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617731473925,"asks":[[35520.0,64205],[35525,31572],[35526,24240],[35536.00000000,3158],[35539,80000]],"bids":[[35504,219],[35503.00000000,3487],[35500.0,631],[35498,12290],[35497,26370]],"ts":1623545081335,"timestamp":1623545081335},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617730623020,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8312,4815],[0.8311,5965],[0.8310,4014],[0.8309,5010]],"ts":1623545081396,"timestamp":1623545081396},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617731938289,"asks":[[20.647,187],[20.648,103],[20.652,10],[20.654,104],[20.659,98]],"bids":[[20.628,96],[20.624,88],[20.619,10],[20.617,423],[20.615,98]],"ts":1623545081400,"timestamp":1623545081400},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617732876603,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.75,10435],[2367.85,7000]],"bids":[[2366.25,20000],[2366.20,6786],[2366.15,7000],[2365.40,6342],[2365.05,5532]],"ts":1623545081431,"timestamp":1623545081431},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731473926,"asks":[[35520.0,64205],[35525,31572],[35526,24240],[35536.00000000,3158],[35539,80000]],"bids":[[35504,219],[35503.00000000,3487],[35500.0,631],[35498,12290],[35497,26370]],"ts":1623545081435,"timestamp":1623545081435},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617730623021,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8312,4815],[0.8311,5965],[0.8310,4014],[0.8309,5010]],"ts":1623545081496,"timestamp":1623545081496},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617731938290,"asks":[[20.647,187],[20.648,103],[20.652,10],[20.654,104],[20.659,98]],"bids":[[20.628,96],[20.624,88],[20.619,10],[20.617,423],[20.615,98]],"ts":1623545081500,"timestamp":1623545081500},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617731473927,"asks":[[35520.0,64205],[35525,31572],[35526,24240],[35536.00000000,3158],[35539,80000]],"bids":[[35504,219],[35503.00000000,3487],[35500.0,631],[35498,12290],[35497,26370]],"ts":1623545081535,"timestamp":1623545081535},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617732876604,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.75,10435],[2367.80,5118]],"bids":[[2365.40,6342],[2364.75,20059],[2364.70,6522],[2364.45,5694],[2364.20,9520]],"ts":1623545081556,"timestamp":1623545081556},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617730623022,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8312,4815],[0.8311,5965],[0.8310,4014],[0.8309,5010]],"ts":1623545081597,"timestamp":1623545081597},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617731938291,"asks":[[20.647,187],[20.648,103],[20.652,10],[20.654,104],[20.659,98]],"bids":[[20.628,96],[20.624,88],[20.619,10],[20.617,423],[20.615,98]],"ts":1623545081600,"timestamp":1623545081600},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617731473928,"asks":[[35520.0,64205],[35525,31572],[35526,24240],[35536.00000000,3158],[35539,80000]],"bids":[[35504,219],[35503.00000000,3487],[35500.0,631],[35498,12290],[35497,26370]],"ts":1623545081635,"timestamp":1623545081635},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617732876605,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.75,10435],[2367.80,5118]],"bids":[[2365.90,5550],[2365.40,6342],[2364.75,20059],[2364.70,6522],[2364.45,5694]],"ts":1623545081671,"timestamp":1623545081671},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617730623023,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8312,4815],[0.8311,5965],[0.8310,4014],[0.8309,5010]],"ts":1623545081697,"timestamp":1623545081697},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617731938292,"asks":[[20.647,187],[20.648,103],[20.652,10],[20.654,104],[20.658,267]],"bids":[[20.628,96],[20.624,88],[20.619,10],[20.617,423],[20.615,98]],"ts":1623545081707,"timestamp":1623545081707},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617731473929,"asks":[[35520.0,64205],[35525,31572],[35526,24240],[35536.00000000,3158],[35539,80000]],"bids":[[35504,219],[35503.00000000,3487],[35500.0,631],[35498,12290],[35497,26370]],"ts":1623545081735,"timestamp":1623545081735},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617732876606,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.75,10435],[2367.80,5118]],"bids":[[2365.90,5550],[2365.40,6342],[2364.75,20059],[2364.70,6522],[2364.45,5694]],"ts":1623545081772,"timestamp":1623545081772},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617730623024,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8312,4815],[0.8311,5965],[0.8310,4014],[0.8309,5010]],"ts":1623545081797,"timestamp":1623545081797},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617731473930,"asks":[[35520.0,64205],[35525,31572],[35526,24240],[35536.00000000,3158],[35539,80000]],"bids":[[35504,219],[35503.00000000,3487],[35500.0,631],[35498,12290],[35497,26370]],"ts":1623545081835,"timestamp":1623545081835},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617731938293,"asks":[[20.647,187],[20.648,103],[20.652,10],[20.654,104],[20.658,267]],"bids":[[20.628,96],[20.624,88],[20.619,10],[20.617,423],[20.615,98]],"ts":1623545081850,"timestamp":1623545081850},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617732876607,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.75,10435],[2367.80,5118]],"bids":[[2365.90,5550],[2365.40,6342],[2364.75,20059],[2364.70,6522],[2364.45,5694]],"ts":1623545081872,"timestamp":1623545081872},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731473931,"asks":[[35520.0,64205],[35525,31572],[35526,24240],[35536.00000000,3158],[35539,80000]],"bids":[[35504,219],[35503.00000000,3487],[35500.0,631],[35498,12290],[35497,26370]],"ts":1623545081935,"timestamp":1623545081935},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617731938294,"asks":[[20.647,187],[20.648,103],[20.652,10],[20.654,104],[20.658,267]],"bids":[[20.628,96],[20.624,88],[20.619,10],[20.617,423],[20.615,98]],"ts":1623545081950,"timestamp":1623545081950},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617732876608,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.75,10435],[2367.80,5118]],"bids":[[2365.90,5550],[2365.40,6342],[2364.75,20059],[2364.70,6522],[2364.45,5694]],"ts":1623545081972,"timestamp":1623545081972},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617730623025,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8311,5965],[0.8310,4014],[0.8309,5010],[0.8308,9715]],"ts":1623545081994,"timestamp":1623545081994},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617731473932,"asks":[[35520.0,64205],[35525,31572],[35526,24240],[35536.00000000,3158],[35539,80000]],"bids":[[35504,219],[35503.00000000,3487],[35500.0,631],[35498,12290],[35497,26370]],"ts":1623545082035,"timestamp":1623545082035},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617731938295,"asks":[[20.647,187],[20.648,103],[20.652,10],[20.654,104],[20.658,267]],"bids":[[20.628,96],[20.624,88],[20.619,10],[20.617,423],[20.615,98]],"ts":1623545082051,"timestamp":1623545082051},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617731473933,"asks":[[35520.0,64205],[35525,31572],[35526,24240],[35536.00000000,3158],[35539,80000]],"bids":[[35504,219],[35503.00000000,3487],[35500.0,631],[35498,12290],[35497,26370]],"ts":1623545082135,"timestamp":1623545082135},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617731938296,"asks":[[20.647,187],[20.648,103],[20.652,10],[20.654,104],[20.658,267]],"bids":[[20.628,96],[20.624,88],[20.619,10],[20.617,423],[20.615,98]],"ts":1623545082151,"timestamp":1623545082151},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617730623026,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8312,4845],[0.8311,5965],[0.8310,4014],[0.8309,5010]],"ts":1623545082171,"timestamp":1623545082171},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617732876609,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.55,7000],[2367.75,10435]],"bids":[[2365.90,5550],[2365.85,7000],[2365.40,6342],[2364.75,20059],[2364.70,6522]],"ts":1623545082170,"timestamp":1623545082170},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731473934,"asks":[[35520.0,64205],[35525,31572],[35526,24240],[35536.00000000,3158],[35539,80000]],"bids":[[35504,219],[35503.00000000,3487],[35500.0,631],[35498,12290],[35497,26370]],"ts":1623545082235,"timestamp":1623545082235},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617731938297,"asks":[[20.647,187],[20.648,103],[20.652,10],[20.654,104],[20.658,267]],"bids":[[20.628,96],[20.624,88],[20.619,10],[20.617,423],[20.615,98]],"ts":1623545082251,"timestamp":1623545082251},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617730623027,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8312,4845],[0.8311,5965],[0.8310,4014],[0.8309,5010]],"ts":1623545082271,"timestamp":1623545082271},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617732876610,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.55,7000],[2367.75,10435]],"bids":[[2365.90,5550],[2365.85,7000],[2365.40,6342],[2364.75,20059],[2364.70,6522]],"ts":1623545082270,"timestamp":1623545082270},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731938298,"asks":[[20.647,187],[20.648,103],[20.652,10],[20.654,104],[20.658,267]],"bids":[[20.628,96],[20.624,88],[20.619,10],[20.617,423],[20.615,98]],"ts":1623545082352,"timestamp":1623545082352},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617731473935,"asks":[[35520.0,64205],[35525,31572],[35530,32520],[35536.00000000,3158],[35539,80000]],"bids":[[35505,80000],[35504,219],[35503.00000000,3487],[35501,29910],[35500.0,631]],"ts":1623545082368,"timestamp":1623545082368},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617732876611,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.55,7000],[2367.75,10435]],"bids":[[2365.90,5550],[2365.85,7000],[2365.40,6342],[2364.75,20059],[2364.70,6522]],"ts":1623545082371,"timestamp":1623545082371},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617730623028,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8312,4845],[0.8311,5965],[0.8310,4014],[0.8309,5010]],"ts":1623545082371,"timestamp":1623545082371},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617730623029,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8312,4845],[0.8311,5965],[0.8310,4014],[0.8309,5010]],"ts":1623545082471,"timestamp":1623545082471},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617731473936,"asks":[[35520.0,64205],[35525,19899],[35530,32520],[35536.00000000,3158],[35539,80000]],"bids":[[35506,219],[35505,80000],[35503.00000000,3487],[35501,29910],[35498,12290]],"ts":1623545082482,"timestamp":1623545082482},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617731938299,"asks":[[20.647,187],[20.648,103],[20.652,10],[20.654,104],[20.659,88]],"bids":[[20.628,96],[20.624,88],[20.619,10],[20.617,423],[20.615,98]],"ts":1623545082514,"timestamp":1623545082514},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617732876612,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.55,7000],[2367.75,10435]],"bids":[[2365.95,20000],[2365.90,5550],[2365.85,7000],[2365.40,6342],[2364.75,20059]],"ts":1623545082564,"timestamp":1623545082564},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617730623030,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8311,5965],[0.8310,4014],[0.8309,5010],[0.8308,9715]],"ts":1623545082618,"timestamp":1623545082618},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617731938300,"asks":[[20.661,2000],[20.662,348],[20.666,80],[20.669,98],[20.671,296]],"bids":[[20.629,10],[20.628,96],[20.624,88],[20.617,423],[20.615,98]],"ts":1623545082619,"timestamp":1623545082619},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617732876613,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.55,7000],[2367.75,10435]],"bids":[[2365.95,20000],[2365.90,5550],[2365.85,7000],[2365.40,6342],[2364.75,20059]],"ts":1623545082664,"timestamp":1623545082664},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731473937,"asks":[[35520.0,64205],[35525,19899],[35532,28410],[35539,80000],[35545,31860]],"bids":[[35506,219],[35505,80000],[35503.00000000,3487],[35502.0,631],[35501,29910]],"ts":1623545082678,"timestamp":1623545082678},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617731938301,"asks":[[20.660,188],[20.661,2000],[20.666,80],[20.669,98],[20.671,296]],"bids":[[20.628,96],[20.624,88],[20.619,257],[20.617,117],[20.616,10]],"ts":1623545082754,"timestamp":1623545082754},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617732876614,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.55,7000],[2367.75,10435]],"bids":[[2365.95,20000],[2365.90,5550],[2365.85,7000],[2365.40,6342],[2364.75,20059]],"ts":1623545082764,"timestamp":1623545082764},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617730623031,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8312,4445],[0.8311,5965],[0.8310,4014],[0.8309,5010]],"ts":1623545082778,"timestamp":1623545082778},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617731473938,"asks":[[35520.0,64205],[35525,19899],[35532,28410],[35539,80000],[35545,31860]],"bids":[[35506,219],[35505,80000],[35503.00000000,3487],[35502.0,631],[35501,29910]],"ts":1623545082832,"timestamp":1623545082832},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617731938302,"asks":[[20.660,188],[20.661,2000],[20.664,253],[20.666,80],[20.669,98]],"bids":[[20.628,96],[20.624,88],[20.621,10],[20.619,257],[20.617,117]],"ts":1623545082859,"timestamp":1623545082859},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617732876615,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.55,7000],[2367.75,10435]],"bids":[[2365.95,20000],[2365.90,5550],[2365.85,7000],[2365.40,6342],[2364.75,20059]],"ts":1623545082864,"timestamp":1623545082864},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617730623032,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8312,4445],[0.8311,5965],[0.8310,4014],[0.8309,5010]],"ts":1623545082879,"timestamp":1623545082879},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617731473939,"asks":[[35520.0,64205],[35525,19899],[35532,28410],[35539,80000],[35545,31860]],"bids":[[35506,219],[35505,80000],[35503.00000000,3487],[35502.0,631],[35501,29910]],"ts":1623545082932,"timestamp":1623545082932},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617731938303,"asks":[[20.656,103],[20.658,10],[20.660,188],[20.661,2000],[20.664,263]],"bids":[[20.635,96],[20.628,96],[20.624,88],[20.621,286],[20.619,10]],"ts":1623545082963,"timestamp":1623545082963},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617732876616,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.55,7000],[2367.75,10435]],"bids":[[2365.95,20000],[2365.90,5550],[2365.85,7000],[2365.40,6342],[2364.75,20059]],"ts":1623545082964,"timestamp":1623545082964},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617730623033,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8312,4445],[0.8311,5965],[0.8310,4014],[0.8309,5010]],"ts":1623545082980,"timestamp":1623545082980},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617732876617,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.55,7000],[2367.75,10435]],"bids":[[2365.95,20000],[2365.90,5550],[2365.85,7000],[2365.40,6342],[2364.75,20059]],"ts":1623545083065,"timestamp":1623545083065},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617730623034,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8312,4445],[0.8311,5965],[0.8310,4014],[0.8309,5010]],"ts":1623545083080,"timestamp":1623545083080},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617731938304,"asks":[[20.657,188],[20.658,10],[20.661,2000],[20.662,385],[20.664,10]],"bids":[[20.628,96],[20.625,10],[20.624,88],[20.621,286],[20.619,10]],"ts":1623545083088,"timestamp":1623545083088},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617731473940,"asks":[[35520.0,64205],[35525,19899],[35531,12251],[35532,28410],[35539,80000]],"bids":[[35506,219],[35505,80000],[35503.00000000,3487],[35502.0,631],[35501,29910]],"ts":1623545083126,"timestamp":1623545083126},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617732876618,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.55,7000],[2367.75,10435]],"bids":[[2365.95,20000],[2365.90,5550],[2365.85,7000],[2365.40,6342],[2364.75,20059]],"ts":1623545083165,"timestamp":1623545083165},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617730623035,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8310,4014],[0.8309,5010],[0.8308,9715],[0.8307,16851]],"ts":1623545083184,"timestamp":1623545083184},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617731938305,"asks":[[20.657,188],[20.658,10],[20.661,2000],[20.662,385],[20.664,10]],"bids":[[20.628,96],[20.625,10],[20.624,88],[20.619,346],[20.617,117]],"ts":1623545083215,"timestamp":1623545083215},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617731473941,"asks":[[35520.0,64205],[35525,19899],[35531,12251],[35539,80000],[35545,31860]],"bids":[[35506,219],[35503.00000000,3487],[35501,29910],[35498,12290],[35492,19425]],"ts":1623545083227,"timestamp":1623545083227},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617732876619,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.55,7000],[2367.75,10435]],"bids":[[2365.95,20000],[2365.90,5550],[2365.85,7000],[2365.40,6342],[2364.75,20059]],"ts":1623545083265,"timestamp":1623545083265},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617730623036,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8310,4014],[0.8309,5010],[0.8308,9715],[0.8307,16851]],"ts":1623545083334,"timestamp":1623545083334},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617731938306,"asks":[[20.657,188],[20.658,10],[20.661,2000],[20.662,97],[20.664,10]],"bids":[[20.628,96],[20.625,10],[20.624,88],[20.619,10],[20.618,268]],"ts":1623545083346,"timestamp":1623545083346},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617731473942,"asks":[[35520.0,64205],[35525,19899],[35530,26670],[35531,12251],[35539,80000]],"bids":[[35506,219],[35503.00000000,3487],[35501,30541],[35498,12290],[35492,19425]],"ts":1623545083348,"timestamp":1623545083348},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617730623037,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8310,4014],[0.8309,5010],[0.8308,9715],[0.8307,16851]],"ts":1623545083435,"timestamp":1623545083435},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617732876620,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.55,7000],[2367.75,10435]],"bids":[[2365.90,5550],[2365.85,7000],[2365.40,6342],[2364.75,20059],[2364.70,6522]],"ts":1623545083440,"timestamp":1623545083440},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731473943,"asks":[[35520.0,64205],[35525,19899],[35531,12251],[35539,80000],[35545,31860]],"bids":[[35506,219],[35503.00000000,3487],[35498,12290],[35494,33900],[35493,80000]],"ts":1623545083458,"timestamp":1623545083458},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617731938307,"asks":[[20.654,89],[20.657,188],[20.658,272],[20.661,2000],[20.662,97]],"bids":[[20.634,118],[20.628,96],[20.625,10],[20.624,88],[20.619,10]],"ts":1623545083462,"timestamp":1623545083462},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617730623038,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8310,4014],[0.8309,5010],[0.8308,9715],[0.8307,16851]],"ts":1623545083535,"timestamp":1623545083535},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617732876621,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.55,7000],[2367.75,10435]],"bids":[[2365.90,5550],[2365.85,7000],[2365.40,6342],[2364.75,20059],[2364.70,6522]],"ts":1623545083540,"timestamp":1623545083540},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731473944,"asks":[[35520.0,64205],[35524,31349],[35525,19899],[35531,12251],[35534.00000000,3325]],"bids":[[35506,219],[35503.00000000,3487],[35500.0,631],[35498,12290],[35494,33900]],"ts":1623545083584,"timestamp":1623545083584},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617731938308,"asks":[[20.654,89],[20.657,188],[20.658,272],[20.661,2000],[20.662,97]],"bids":[[20.634,118],[20.628,96],[20.625,10],[20.624,88],[20.619,290]],"ts":1623545083589,"timestamp":1623545083589},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617730623039,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8310,4014],[0.8309,5010],[0.8308,9715],[0.8307,16851]],"ts":1623545083635,"timestamp":1623545083635},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617732876622,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.55,7000],[2367.75,10435]],"bids":[[2365.90,5550],[2365.85,7000],[2365.40,6342],[2364.75,20059],[2364.70,6522]],"ts":1623545083640,"timestamp":1623545083640},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731938309,"asks":[[20.654,89],[20.657,188],[20.658,272],[20.661,2000],[20.662,97]],"bids":[[20.634,118],[20.628,96],[20.625,10],[20.624,88],[20.619,290]],"ts":1623545083689,"timestamp":1623545083689},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617730623040,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8310,4014],[0.8309,5010],[0.8308,9715],[0.8307,16851]],"ts":1623545083735,"timestamp":1623545083735},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617732876623,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.55,7000],[2367.75,10435]],"bids":[[2365.90,5550],[2365.85,7000],[2365.40,6342],[2364.75,20059],[2364.70,6522]],"ts":1623545083740,"timestamp":1623545083740},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731473945,"asks":[[35520.0,64205],[35524,31349],[35525,19899],[35531,12251],[35534.00000000,3325]],"bids":[[35506,219],[35503.00000000,3487],[35500.0,631],[35498,12290],[35494,33900]],"ts":1623545083760,"timestamp":1623545083760},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617731938310,"asks":[[20.654,89],[20.657,188],[20.658,272],[20.661,2000],[20.662,97]],"bids":[[20.634,118],[20.628,96],[20.625,10],[20.624,88],[20.619,290]],"ts":1623545083789,"timestamp":1623545083789},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617730623041,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8310,4014],[0.8309,5010],[0.8308,9715],[0.8307,16851]],"ts":1623545083835,"timestamp":1623545083835},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617732876624,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.55,7000],[2367.75,10435]],"bids":[[2365.90,5550],[2365.85,7000],[2365.40,6342],[2364.75,20059],[2364.70,6522]],"ts":1623545083840,"timestamp":1623545083840},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731473946,"asks":[[35520.0,64205],[35524,31349],[35525,19899],[35531,12251],[35534.00000000,3325]],"bids":[[35506,219],[35503.00000000,3487],[35500.0,631],[35498,12290],[35494,33900]],"ts":1623545083860,"timestamp":1623545083860},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617731938311,"asks":[[20.654,89],[20.657,188],[20.658,272],[20.661,2000],[20.662,97]],"bids":[[20.634,118],[20.628,96],[20.625,10],[20.624,88],[20.619,290]],"ts":1623545083889,"timestamp":1623545083889},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617731473947,"asks":[[35520.0,64205],[35524,31349],[35525,19899],[35531,12251],[35534.00000000,3325]],"bids":[[35506,219],[35503.00000000,3487],[35500.0,631],[35498,12290],[35494,33900]],"ts":1623545083960,"timestamp":1623545083960},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617730623042,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8311,5935],[0.8310,4014],[0.8309,5010],[0.8308,9715]],"ts":1623545083960,"timestamp":1623545083960},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617732876625,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.55,7000],[2367.75,10435]],"bids":[[2365.95,20000],[2365.90,5550],[2365.85,7000],[2365.40,6342],[2364.75,20059]],"ts":1623545083956,"timestamp":1623545083956},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731938312,"asks":[[20.654,89],[20.657,188],[20.658,272],[20.661,2000],[20.662,97]],"bids":[[20.634,118],[20.628,96],[20.625,10],[20.624,88],[20.619,290]],"ts":1623545083989,"timestamp":1623545083989},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617732876626,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.55,7000],[2367.75,10435]],"bids":[[2365.95,20000],[2365.90,5550],[2365.85,7000],[2365.40,6342],[2364.75,20059]],"ts":1623545084057,"timestamp":1623545084057},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731473948,"asks":[[35520.0,64205],[35524,31349],[35525,19899],[35531,12251],[35534.00000000,3325]],"bids":[[35506,219],[35503.00000000,3487],[35500.0,631],[35498,12290],[35494,33900]],"ts":1623545084060,"timestamp":1623545084060},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617731938313,"asks":[[20.654,89],[20.657,188],[20.658,272],[20.661,2000],[20.662,97]],"bids":[[20.634,118],[20.628,96],[20.625,10],[20.624,88],[20.619,290]],"ts":1623545084090,"timestamp":1623545084090},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617730623043,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8311,5935],[0.8310,4014],[0.8309,5010],[0.8308,9715]],"ts":1623545084156,"timestamp":1623545084156},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617732876627,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.55,7000],[2367.75,10435]],"bids":[[2365.95,20000],[2365.90,5550],[2365.85,7000],[2365.40,6342],[2364.75,20059]],"ts":1623545084157,"timestamp":1623545084157},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731938314,"asks":[[20.654,89],[20.657,188],[20.658,272],[20.661,2000],[20.662,97]],"bids":[[20.634,118],[20.628,96],[20.625,10],[20.624,88],[20.619,290]],"ts":1623545084190,"timestamp":1623545084190},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617731473949,"asks":[[35520.0,64205],[35524,31349],[35525,19899],[35534.00000000,3325],[35537,33360]],"bids":[[35503.00000000,3487],[35500.0,631],[35498,12290],[35494,33900],[35493,80000]],"ts":1623545084235,"timestamp":1623545084235},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617730623044,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8311,5935],[0.8310,4014],[0.8309,5010],[0.8308,9715]],"ts":1623545084256,"timestamp":1623545084256},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617732876628,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.55,7000],[2367.75,10435]],"bids":[[2365.95,20000],[2365.90,5550],[2365.85,7000],[2365.40,6342],[2364.75,20059]],"ts":1623545084257,"timestamp":1623545084257},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731938315,"asks":[[20.653,188],[20.654,89],[20.658,272],[20.661,2000],[20.662,97]],"bids":[[20.634,118],[20.628,96],[20.625,10],[20.624,88],[20.619,290]],"ts":1623545084293,"timestamp":1623545084293},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617731473950,"asks":[[35520.0,64205],[35524,31349],[35525,19899],[35534.00000000,3325],[35537,33360]],"bids":[[35503.00000000,3487],[35500.0,631],[35498,12290],[35494,33900],[35493,80000]],"ts":1623545084335,"timestamp":1623545084335},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617730623045,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8311,5935],[0.8310,4014],[0.8309,5010],[0.8308,9715]],"ts":1623545084356,"timestamp":1623545084356},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617732876629,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.55,7000],[2367.75,10435]],"bids":[[2365.95,20000],[2365.90,5550],[2365.85,7000],[2365.40,6342],[2364.75,20059]],"ts":1623545084358,"timestamp":1623545084358},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731938316,"asks":[[20.653,188],[20.654,89],[20.658,272],[20.661,2000],[20.662,97]],"bids":[[20.634,118],[20.628,96],[20.625,10],[20.624,88],[20.619,290]],"ts":1623545084393,"timestamp":1623545084393},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617731473951,"asks":[[35520.0,64205],[35524,31349],[35525,19899],[35534.00000000,3325],[35537,33360]],"bids":[[35503.00000000,3487],[35500.0,631],[35498,12290],[35494,33900],[35493,80000]],"ts":1623545084436,"timestamp":1623545084436},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617730623046,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8311,5935],[0.8310,4014],[0.8309,5010],[0.8308,9715]],"ts":1623545084457,"timestamp":1623545084457},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617732876630,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.55,7000],[2367.75,10435]],"bids":[[2365.95,20000],[2365.90,5550],[2365.85,7000],[2365.40,6342],[2364.75,20059]],"ts":1623545084458,"timestamp":1623545084458},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731938317,"asks":[[20.653,188],[20.654,89],[20.658,272],[20.661,2000],[20.662,97]],"bids":[[20.634,118],[20.628,96],[20.625,10],[20.624,88],[20.619,290]],"ts":1623545084494,"timestamp":1623545084494},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617731473952,"asks":[[35520.0,64205],[35525,19899],[35530,35490],[35534.00000000,3325],[35539,80000]],"bids":[[35503.00000000,3487],[35500.0,631],[35498,12290],[35493,80000],[35492,19425]],"ts":1623545084542,"timestamp":1623545084542},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617730623047,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8311,5935],[0.8310,4014],[0.8309,5010],[0.8308,9715]],"ts":1623545084557,"timestamp":1623545084557},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617732876631,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.55,7000],[2367.75,10435]],"bids":[[2365.95,20000],[2365.90,5550],[2365.85,7000],[2365.40,6342],[2364.75,20059]],"ts":1623545084558,"timestamp":1623545084558},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731938318,"asks":[[20.653,188],[20.654,89],[20.658,272],[20.661,2000],[20.662,97]],"bids":[[20.634,118],[20.628,96],[20.625,10],[20.624,88],[20.619,290]],"ts":1623545084594,"timestamp":1623545084594},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617731473953,"asks":[[35520.0,64205],[35525,19899],[35530,35490],[35534.00000000,3325],[35539,80000]],"bids":[[35503.00000000,3487],[35500.0,631],[35499,33300],[35498,12290],[35492,19425]],"ts":1623545084646,"timestamp":1623545084646},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617732876632,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.55,7000],[2367.75,10435]],"bids":[[2365.95,20000],[2365.90,5550],[2365.85,7000],[2365.40,6342],[2364.75,20059]],"ts":1623545084658,"timestamp":1623545084658},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617730623048,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8311,5935],[0.8310,4014],[0.8309,5010],[0.8308,9715]],"ts":1623545084657,"timestamp":1623545084657},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617731938319,"asks":[[20.653,188],[20.654,89],[20.658,272],[20.661,2000],[20.662,97]],"bids":[[20.634,118],[20.628,96],[20.625,10],[20.624,88],[20.619,290]],"ts":1623545084694,"timestamp":1623545084694},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617730623049,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8311,5935],[0.8310,4014],[0.8309,5010],[0.8308,9715]],"ts":1623545084758,"timestamp":1623545084758},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617732876633,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.55,7000],[2367.75,10435]],"bids":[[2365.95,20000],[2365.90,5550],[2365.85,7000],[2365.40,6342],[2364.75,20059]],"ts":1623545084758,"timestamp":1623545084758},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731938320,"asks":[[20.653,188],[20.654,89],[20.658,272],[20.661,2000],[20.662,97]],"bids":[[20.634,118],[20.628,96],[20.625,10],[20.624,88],[20.619,290]],"ts":1623545084794,"timestamp":1623545084794},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617731473954,"asks":[[35520.0,64205],[35524,12275],[35525,19899],[35530,35490],[35534.00000000,3325]],"bids":[[35503.00000000,3487],[35500.0,631],[35499,33300],[35498,12290],[35492,19425]],"ts":1623545084797,"timestamp":1623545084797},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617730623050,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8311,5935],[0.8310,4014],[0.8309,5010],[0.8308,9715]],"ts":1623545084859,"timestamp":1623545084859},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617732876634,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.55,7000],[2367.75,10435]],"bids":[[2365.95,20000],[2365.90,5550],[2365.85,7000],[2365.40,6342],[2364.75,20059]],"ts":1623545084858,"timestamp":1623545084858},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731938321,"asks":[[20.653,188],[20.654,89],[20.658,272],[20.661,2000],[20.662,97]],"bids":[[20.634,118],[20.628,96],[20.625,10],[20.624,88],[20.619,290]],"ts":1623545084894,"timestamp":1623545084894},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617731473955,"asks":[[35520.0,64205],[35524,12275],[35525,19899],[35530,35490],[35534.00000000,3325]],"bids":[[35506,219],[35505,80000],[35503.00000000,3487],[35499,33300],[35498,12290]],"ts":1623545084936,"timestamp":1623545084936},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617730623051,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8311,5935],[0.8310,4014],[0.8309,5010],[0.8308,9715]],"ts":1623545084959,"timestamp":1623545084959},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617732876635,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.55,7000],[2367.75,10435]],"bids":[[2365.95,20000],[2365.90,5550],[2365.85,7000],[2365.40,6342],[2364.75,20059]],"ts":1623545084958,"timestamp":1623545084958},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731938322,"asks":[[20.653,188],[20.654,89],[20.658,272],[20.661,2000],[20.662,97]],"bids":[[20.634,118],[20.628,96],[20.625,10],[20.624,88],[20.619,290]],"ts":1623545084994,"timestamp":1623545084994},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617731473956,"asks":[[35520.0,64205],[35524,12275],[35525,19899],[35530,35490],[35534.00000000,3325]],"bids":[[35506,219],[35505,80000],[35503.00000000,3487],[35502.0,631],[35499,33300]],"ts":1623545085037,"timestamp":1623545085037},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617732876636,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.55,7000],[2367.75,10435]],"bids":[[2365.95,20000],[2365.90,5550],[2365.85,7000],[2365.40,6342],[2364.75,20059]],"ts":1623545085058,"timestamp":1623545085058},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617730623052,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8311,5935],[0.8310,4014],[0.8309,5010],[0.8308,9715]],"ts":1623545085059,"timestamp":1623545085059},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617731938323,"asks":[[20.653,188],[20.654,89],[20.658,272],[20.661,2000],[20.662,97]],"bids":[[20.634,118],[20.628,96],[20.625,10],[20.624,88],[20.619,290]],"ts":1623545085094,"timestamp":1623545085094},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617731473957,"asks":[[35520.0,64205],[35524,12275],[35525,19899],[35530,35490],[35534.00000000,3325]],"bids":[[35506,219],[35505,80000],[35503.00000000,3487],[35502.0,631],[35499,33300]],"ts":1623545085137,"timestamp":1623545085137},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617732876637,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.55,7000],[2367.75,10435]],"bids":[[2365.95,20000],[2365.90,5550],[2365.85,7000],[2365.40,6342],[2364.75,20059]],"ts":1623545085158,"timestamp":1623545085158},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617730623053,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8311,5935],[0.8310,4014],[0.8309,5010],[0.8308,9715]],"ts":1623545085159,"timestamp":1623545085159},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617731938324,"asks":[[20.653,188],[20.654,89],[20.658,272],[20.661,2000],[20.662,97]],"bids":[[20.634,118],[20.628,96],[20.625,10],[20.624,88],[20.619,290]],"ts":1623545085194,"timestamp":1623545085194},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617731473958,"asks":[[35520.0,64205],[35524,12275],[35525,19899],[35530,35490],[35534.00000000,3325]],"bids":[[35506,219],[35505,80000],[35503.00000000,3487],[35502.0,631],[35499,33300]],"ts":1623545085237,"timestamp":1623545085237},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617730623054,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8311,5935],[0.8310,4014],[0.8309,5010],[0.8308,9715]],"ts":1623545085259,"timestamp":1623545085259},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617732876638,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.55,7000],[2367.75,10435]],"bids":[[2365.95,20000],[2365.90,5550],[2365.85,7000],[2365.40,6342],[2364.75,20059]],"ts":1623545085258,"timestamp":1623545085258},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731938325,"asks":[[20.653,188],[20.654,89],[20.658,272],[20.661,2000],[20.662,97]],"bids":[[20.634,118],[20.628,96],[20.625,10],[20.624,88],[20.619,290]],"ts":1623545085294,"timestamp":1623545085294},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617731473959,"asks":[[35520.0,64205],[35524,12275],[35525,19899],[35530,35490],[35534.00000000,3325]],"bids":[[35506,219],[35505,80000],[35503.00000000,3487],[35502.0,631],[35499,33300]],"ts":1623545085337,"timestamp":1623545085337},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617730623055,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8311,5935],[0.8310,4014],[0.8309,5010],[0.8308,9715]],"ts":1623545085359,"timestamp":1623545085359},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617732876639,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.55,7000],[2367.75,10435]],"bids":[[2365.95,20000],[2365.90,5550],[2365.85,7000],[2365.40,6342],[2364.75,20059]],"ts":1623545085376,"timestamp":1623545085376},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731938326,"asks":[[20.653,188],[20.654,89],[20.658,272],[20.661,2000],[20.662,97]],"bids":[[20.634,118],[20.628,96],[20.625,10],[20.624,88],[20.619,290]],"ts":1623545085394,"timestamp":1623545085394},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617731473960,"asks":[[35520.0,64205],[35524,12275],[35525,19899],[35530,35490],[35534.00000000,3325]],"bids":[[35506,219],[35505,80000],[35503.00000000,3487],[35502.0,631],[35499,33300]],"ts":1623545085437,"timestamp":1623545085437},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617730623056,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8311,5935],[0.8310,4014],[0.8309,5010],[0.8308,9715]],"ts":1623545085459,"timestamp":1623545085459},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617732876640,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.55,7000],[2367.75,10435]],"bids":[[2365.95,20000],[2365.90,5550],[2365.85,7000],[2365.40,6342],[2364.75,20059]],"ts":1623545085476,"timestamp":1623545085476},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731938327,"asks":[[20.653,188],[20.654,89],[20.658,272],[20.661,2000],[20.662,97]],"bids":[[20.634,118],[20.628,96],[20.625,10],[20.624,88],[20.619,290]],"ts":1623545085494,"timestamp":1623545085494},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617731473961,"asks":[[35520.0,64205],[35524,12275],[35525,19899],[35530,35490],[35534.00000000,3325]],"bids":[[35506,219],[35505,80000],[35503.00000000,3487],[35502.0,631],[35499,33300]],"ts":1623545085537,"timestamp":1623545085537},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617730623057,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8311,5935],[0.8310,4014],[0.8309,5010],[0.8308,9715]],"ts":1623545085559,"timestamp":1623545085559},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617732876641,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.55,7000],[2367.75,10435]],"bids":[[2365.95,20000],[2365.90,5550],[2365.85,7000],[2365.40,6342],[2364.75,20059]],"ts":1623545085576,"timestamp":1623545085576},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731938328,"asks":[[20.653,188],[20.654,89],[20.658,272],[20.661,2000],[20.662,97]],"bids":[[20.634,118],[20.628,96],[20.625,10],[20.624,88],[20.619,290]],"ts":1623545085594,"timestamp":1623545085594},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617731473962,"asks":[[35520.0,64205],[35524,12275],[35525,19899],[35530,35490],[35534.00000000,3325]],"bids":[[35506,219],[35505,80000],[35503.00000000,3487],[35502.0,631],[35499,33300]],"ts":1623545085637,"timestamp":1623545085637},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617730623058,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8311,5935],[0.8310,4014],[0.8309,5010],[0.8308,9715]],"ts":1623545085660,"timestamp":1623545085660},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617731938329,"asks":[[20.653,188],[20.654,89],[20.658,272],[20.661,2000],[20.662,97]],"bids":[[20.634,118],[20.628,96],[20.625,10],[20.624,88],[20.619,290]],"ts":1623545085694,"timestamp":1623545085694},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617731473963,"asks":[[35520.0,64205],[35524,12275],[35525,19899],[35530,35490],[35534.00000000,3325]],"bids":[[35506,219],[35505,80000],[35503.00000000,3487],[35502.0,631],[35499,33300]],"ts":1623545085737,"timestamp":1623545085737},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617732876642,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.55,7000],[2367.75,10435]],"bids":[[2365.95,20000],[2365.90,5550],[2365.85,7000],[2365.40,6342],[2364.75,20059]],"ts":1623545085759,"timestamp":1623545085759},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617730623059,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8311,5935],[0.8310,4014],[0.8309,5010],[0.8308,9715]],"ts":1623545085760,"timestamp":1623545085760},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617731938330,"asks":[[20.653,188],[20.654,89],[20.658,272],[20.661,2000],[20.662,97]],"bids":[[20.634,118],[20.628,96],[20.625,10],[20.624,88],[20.619,290]],"ts":1623545085795,"timestamp":1623545085795},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617731473964,"asks":[[35520.0,64205],[35524,12275],[35525,19899],[35530,35490],[35534.00000000,3325]],"bids":[[35506,219],[35505,80000],[35503.00000000,3487],[35502.0,631],[35499,33300]],"ts":1623545085837,"timestamp":1623545085837},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617730623060,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8311,5935],[0.8310,4014],[0.8309,5010],[0.8308,9715]],"ts":1623545085860,"timestamp":1623545085860},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617732876643,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.55,7000],[2367.75,10435]],"bids":[[2365.95,20000],[2365.90,5550],[2365.85,7000],[2365.40,6342],[2364.75,20059]],"ts":1623545085859,"timestamp":1623545085859},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731938331,"asks":[[20.653,188],[20.654,89],[20.658,272],[20.661,2000],[20.662,97]],"bids":[[20.634,118],[20.628,96],[20.625,10],[20.624,88],[20.619,290]],"ts":1623545085895,"timestamp":1623545085895},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617731473965,"asks":[[35520.0,64205],[35524,12275],[35525,19899],[35530,35490],[35534.00000000,3325]],"bids":[[35506,219],[35505,80000],[35503.00000000,3487],[35502.0,631],[35499,33300]],"ts":1623545085937,"timestamp":1623545085937},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617732876644,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.55,7000],[2367.75,10435]],"bids":[[2365.95,20000],[2365.90,5550],[2365.85,7000],[2365.40,6342],[2364.75,20059]],"ts":1623545085959,"timestamp":1623545085959},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617730623061,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8311,5935],[0.8310,4014],[0.8309,5010],[0.8308,9715]],"ts":1623545085960,"timestamp":1623545085960},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617731938332,"asks":[[20.653,188],[20.654,89],[20.658,272],[20.661,2000],[20.662,97]],"bids":[[20.634,118],[20.628,96],[20.625,10],[20.624,88],[20.619,290]],"ts":1623545085995,"timestamp":1623545085995},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617731473966,"asks":[[35520.0,64205],[35524,12275],[35525,19899],[35530,35490],[35534.00000000,3325]],"bids":[[35506,219],[35505,80000],[35503.00000000,3487],[35502.0,631],[35499,33300]],"ts":1623545086038,"timestamp":1623545086038},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617732876645,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.55,7000],[2367.75,10435]],"bids":[[2365.95,20000],[2365.90,5550],[2365.85,7000],[2365.40,6342],[2364.75,20059]],"ts":1623545086059,"timestamp":1623545086059},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617730623062,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8311,5935],[0.8310,4014],[0.8309,5010],[0.8308,9715]],"ts":1623545086060,"timestamp":1623545086060},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617731938333,"asks":[[20.653,188],[20.654,89],[20.658,272],[20.661,2000],[20.662,97]],"bids":[[20.634,118],[20.628,96],[20.625,10],[20.624,88],[20.619,290]],"ts":1623545086095,"timestamp":1623545086095},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617731473967,"asks":[[35520.0,64205],[35524,12275],[35525,19899],[35530,35490],[35534.00000000,3325]],"bids":[[35506,219],[35505,80000],[35503.00000000,3487],[35502.0,631],[35499,33300]],"ts":1623545086138,"timestamp":1623545086138},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617732876646,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.55,7000],[2367.75,10435]],"bids":[[2365.95,20000],[2365.90,5550],[2365.85,7000],[2365.40,6342],[2364.75,20059]],"ts":1623545086159,"timestamp":1623545086159},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617730623063,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8311,5935],[0.8310,4014],[0.8309,5010],[0.8308,9715]],"ts":1623545086160,"timestamp":1623545086160},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617731938334,"asks":[[20.653,188],[20.654,89],[20.658,272],[20.661,2000],[20.662,97]],"bids":[[20.634,118],[20.628,96],[20.625,10],[20.624,88],[20.619,290]],"ts":1623545086195,"timestamp":1623545086195},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617731473968,"asks":[[35520.0,64205],[35524,12275],[35525,19899],[35530,35490],[35534.00000000,3325]],"bids":[[35506,219],[35505,80000],[35503.00000000,3487],[35502.0,631],[35499,33300]],"ts":1623545086238,"timestamp":1623545086238},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617730623064,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8311,5935],[0.8310,4014],[0.8309,5010],[0.8308,9715]],"ts":1623545086260,"timestamp":1623545086260},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617732876647,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.55,7000],[2367.75,10435]],"bids":[[2365.95,20000],[2365.90,5550],[2365.85,7000],[2365.40,6342],[2364.75,20059]],"ts":1623545086260,"timestamp":1623545086260},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731938335,"asks":[[20.653,188],[20.654,89],[20.658,272],[20.661,2000],[20.662,97]],"bids":[[20.634,118],[20.628,96],[20.625,10],[20.624,88],[20.619,290]],"ts":1623545086295,"timestamp":1623545086295},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617731473969,"asks":[[35520.0,64205],[35524,12275],[35525,19899],[35530,35490],[35534.00000000,3325]],"bids":[[35506,219],[35505,80000],[35503.00000000,3487],[35502.0,631],[35499,33300]],"ts":1623545086338,"timestamp":1623545086338},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617730623065,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8311,5935],[0.8310,4014],[0.8309,5010],[0.8308,9715]],"ts":1623545086360,"timestamp":1623545086360},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617732876648,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.55,7000],[2367.75,10435]],"bids":[[2365.95,20000],[2365.90,5550],[2365.85,7000],[2365.40,6342],[2364.75,20059]],"ts":1623545086360,"timestamp":1623545086360},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731938336,"asks":[[20.653,188],[20.654,89],[20.658,272],[20.661,2000],[20.662,97]],"bids":[[20.634,118],[20.628,96],[20.625,10],[20.624,88],[20.619,290]],"ts":1623545086396,"timestamp":1623545086396},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617731473970,"asks":[[35520.0,64205],[35524,12275],[35525,19899],[35530,35490],[35539,80000]],"bids":[[35506,219],[35505,80000],[35503.00000000,3487],[35502.0,631],[35499,33300]],"ts":1623545086453,"timestamp":1623545086453},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617730623066,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8311,5935],[0.8310,4014],[0.8309,5010],[0.8308,9715]],"ts":1623545086460,"timestamp":1623545086460},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617732876649,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.55,7000],[2367.75,10435]],"bids":[[2365.95,20000],[2365.90,5550],[2365.85,7000],[2365.40,6342],[2364.75,20059]],"ts":1623545086460,"timestamp":1623545086460},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731938337,"asks":[[20.653,188],[20.654,89],[20.658,272],[20.661,2000],[20.662,97]],"bids":[[20.634,118],[20.628,96],[20.625,10],[20.624,88],[20.619,290]],"ts":1623545086496,"timestamp":1623545086496},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617731473971,"asks":[[35520.0,64205],[35524,12275],[35525,19899],[35530,35490],[35539,80000]],"bids":[[35506,219],[35505,80000],[35503.00000000,3487],[35502.0,631],[35499,33300]],"ts":1623545086553,"timestamp":1623545086553},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617730623067,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8311,5935],[0.8310,4014],[0.8309,5010],[0.8308,9715]],"ts":1623545086560,"timestamp":1623545086560},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617732876650,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.55,7000],[2367.75,10435]],"bids":[[2365.95,20000],[2365.90,5550],[2365.85,7000],[2365.40,6342],[2364.75,20059]],"ts":1623545086561,"timestamp":1623545086561},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731938338,"asks":[[20.653,188],[20.654,89],[20.658,272],[20.661,2000],[20.662,97]],"bids":[[20.634,118],[20.628,96],[20.625,10],[20.624,88],[20.619,290]],"ts":1623545086596,"timestamp":1623545086596},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617730623068,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8311,5935],[0.8310,4014],[0.8309,5010],[0.8308,9715]],"ts":1623545086660,"timestamp":1623545086660},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617732876651,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.55,7000],[2367.75,10435]],"bids":[[2365.95,20000],[2365.90,5550],[2365.85,7000],[2365.40,6342],[2364.75,20059]],"ts":1623545086662,"timestamp":1623545086662},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731473972,"asks":[[35520.0,64205],[35524,12275],[35525,19899],[35530,35490],[35539,80000]],"bids":[[35506,219],[35505,80000],[35503.00000000,3487],[35502.0,631],[35498,12290]],"ts":1623545086661,"timestamp":1623545086661},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617731938339,"asks":[[20.653,188],[20.654,89],[20.658,272],[20.661,2000],[20.662,97]],"bids":[[20.634,118],[20.628,96],[20.625,10],[20.624,88],[20.619,290]],"ts":1623545086696,"timestamp":1623545086696},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617730623069,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8311,5935],[0.8310,4014],[0.8309,5010],[0.8308,9715]],"ts":1623545086761,"timestamp":1623545086761},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617732876652,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.55,7000],[2367.75,10435]],"bids":[[2365.95,20000],[2365.90,5550],[2365.85,7000],[2365.40,6342],[2364.75,20059]],"ts":1623545086763,"timestamp":1623545086763},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731473973,"asks":[[35520.0,64205],[35524,12275],[35525,19899],[35530,35490],[35539,80000]],"bids":[[35506,219],[35505,80000],[35503.00000000,3487],[35502.0,631],[35501,31049]],"ts":1623545086773,"timestamp":1623545086773},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617731938340,"asks":[[20.653,188],[20.654,89],[20.658,272],[20.661,2000],[20.662,97]],"bids":[[20.634,118],[20.628,96],[20.625,10],[20.624,88],[20.619,290]],"ts":1623545086796,"timestamp":1623545086796},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617730623070,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8311,5935],[0.8310,4014],[0.8309,5010],[0.8308,9715]],"ts":1623545086862,"timestamp":1623545086862},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617732876653,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.55,7000],[2367.75,10435]],"bids":[[2365.95,20000],[2365.90,5550],[2365.85,7000],[2365.40,6342],[2364.75,20059]],"ts":1623545086863,"timestamp":1623545086863},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731473974,"asks":[[35520.0,64205],[35524,12275],[35525,19899],[35530,35490],[35539,80000]],"bids":[[35506,219],[35505,80000],[35503.00000000,3487],[35502.0,631],[35501,31049]],"ts":1623545086873,"timestamp":1623545086873},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617731938341,"asks":[[20.653,188],[20.654,89],[20.658,272],[20.661,2000],[20.662,97]],"bids":[[20.634,118],[20.628,96],[20.625,10],[20.624,88],[20.619,290]],"ts":1623545086896,"timestamp":1623545086896},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617732876654,"asks":[[2367.15,20000],[2367.20,5622],[2367.35,10103],[2367.75,10435],[2368.00,6324]],"bids":[[2365.95,20000],[2365.85,7000],[2365.40,6342],[2364.75,20059],[2364.70,6522]],"ts":1623545086972,"timestamp":1623545086972},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731473975,"asks":[[35524,12275],[35525,19899],[35530,35490],[35539,80000],[35545,31860]],"bids":[[35506,219],[35505,80000],[35503.00000000,3487],[35502.0,631],[35501,31049]],"ts":1623545086975,"timestamp":1623545086975},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617730623071,"asks":[[0.8317,3734],[0.8318,5855],[0.8319,12484],[0.8320,5600],[0.8321,4750]],"bids":[[0.8316,240],[0.8311,5935],[0.8310,4014],[0.8309,5010],[0.8308,9715]],"ts":1623545086964,"timestamp":1623545086964},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617731938342,"asks":[[20.656,81],[20.658,272],[20.660,10],[20.661,2000],[20.664,10]],"bids":[[20.628,96],[20.624,88],[20.619,290],[20.617,117],[20.615,98]],"ts":1623545087005,"timestamp":1623545087005},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617730623072,"asks":[[0.8319,12484],[0.8320,5600],[0.8321,4750],[0.8322,4455],[0.8323,4755]],"bids":[[0.8316,240],[0.8311,5935],[0.8310,4014],[0.8309,9190],[0.8308,9715]],"ts":1623545087084,"timestamp":1623545087084},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617731473976,"asks":[[35524,12275],[35525,19899],[35536,29520],[35539,80000],[35550,29970]],"bids":[[35506,219],[35505,80000],[35504.0,631],[35503.00000000,3487],[35501,31049]],"ts":1623545087083,"timestamp":1623545087083},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617732876655,"asks":[[2367.20,20000],[2367.35,10103],[2367.60,5004],[2367.75,10435],[2368.60,5826]],"bids":[[2365.30,5922],[2364.75,20059],[2364.70,6522],[2364.45,5694],[2364.20,9520]],"ts":1623545087088,"timestamp":1623545087088},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731938343,"asks":[[20.657,188],[20.661,2000],[20.664,10],[20.669,98],[20.671,296]],"bids":[[20.637,114],[20.628,106],[20.624,88],[20.619,280],[20.617,117]],"ts":1623545087117,"timestamp":1623545087117},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617732876656,"asks":[[2367.35,10103],[2367.75,10435],[2368.60,5826],[2368.65,7104],[2369.05,6426]],"bids":[[2366.60,5766],[2365.30,5922],[2364.75,20059],[2364.70,6522],[2364.45,5694]],"ts":1623545087197,"timestamp":1623545087197},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731473977,"asks":[[35524,12275],[35525,19899],[35536,29520],[35539,80000],[35550,29970]],"bids":[[35506,219],[35505,80000],[35504.0,631],[35503.00000000,3487],[35498,12290]],"ts":1623545087202,"timestamp":1623545087202},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617731938344,"asks":[[20.657,188],[20.661,2000],[20.664,102],[20.666,10],[20.669,98]],"bids":[[20.628,106],[20.624,88],[20.622,289],[20.617,117],[20.615,98]],"ts":1623545087257,"timestamp":1623545087257},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617730623073,"asks":[[0.8319,12484],[0.8320,5600],[0.8321,4750],[0.8322,4455],[0.8323,4755]],"bids":[[0.8316,240],[0.8311,5935],[0.8310,8314],[0.8309,9190],[0.8308,9715]],"ts":1623545087261,"timestamp":1623545087261},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617731473978,"asks":[[35534.0,49263],[35536,29520],[35539,80000],[35550,29970],[35557,30840]],"bids":[[35506,25689],[35505,80631],[35503.00000000,3487],[35494,28200],[35492,19425]],"ts":1623545087309,"timestamp":1623545087309},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617732876657,"asks":[[2367.35,10103],[2367.75,10435],[2368.60,5826],[2369.05,6426],[2369.25,5544]],"bids":[[2366.60,5766],[2365.90,5850],[2365.7,117249],[2365.30,5922],[2365.20,5238]],"ts":1623545087335,"timestamp":1623545087335},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617730623074,"asks":[[0.8319,5505],[0.8320,5600],[0.8321,4750],[0.8322,4455],[0.8323,4755]],"bids":[[0.8316,20240],[0.8312,5635],[0.8311,5935],[0.8310,8314],[0.8309,9190]],"ts":1623545087366,"timestamp":1623545087366},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617731938345,"asks":[[20.657,188],[20.661,2000],[20.664,102],[20.666,10],[20.668,291]],"bids":[[20.634,10],[20.628,106],[20.624,88],[20.622,10],[20.617,117]],"ts":1623545087406,"timestamp":1623545087406},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617731473979,"asks":[[35533,32262],[35534.0,49263],[35538,35880],[35539,80000],[35547.00000000,3199]],"bids":[[35508,35760],[35507.0,631],[35505,80000],[35503.00000000,3487],[35494,28200]],"ts":1623545087414,"timestamp":1623545087414},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617732876658,"asks":[[2367.35,10103],[2367.75,10435],[2368.20,6149],[2368.60,5826],[2369.05,6426]],"bids":[[2366.65,20000],[2366.60,5766],[2366.40,5214],[2365.90,5850],[2365.7,117249]],"ts":1623545087461,"timestamp":1623545087461},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617730623075,"asks":[[0.8319,5505],[0.8320,5600],[0.8321,4750],[0.8322,4455],[0.8323,4755]],"bids":[[0.8316,20240],[0.8312,5635],[0.8311,5935],[0.8310,8314],[0.8309,9190]],"ts":1623545087519,"timestamp":1623545087519},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617731938346,"asks":[[20.657,188],[20.661,2000],[20.664,102],[20.666,10],[20.668,291]],"bids":[[20.643,107],[20.636,107],[20.634,10],[20.628,106],[20.624,393]],"ts":1623545087523,"timestamp":1623545087523},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617731473980,"asks":[[35533,32262],[35534.0,49263],[35538,35880],[35539,80000],[35547.00000000,3199]],"bids":[[35519.00000000,3801],[35509,11263],[35508,35979],[35505,80000],[35504.0,11200]],"ts":1623545087528,"timestamp":1623545087528},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617732876659,"asks":[[2367.35,10103],[2367.75,10435],[2368.20,6149],[2368.55,7000],[2368.60,5826]],"bids":[[2366.65,20000],[2366.60,12766],[2366.20,5664],[2365.90,5850],[2365.7,117249]],"ts":1623545087564,"timestamp":1623545087564},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617730623076,"asks":[[0.8321,4750],[0.8322,4455],[0.8323,4755],[0.8324,5905],[0.8325,9734]],"bids":[[0.8316,20240],[0.8313,4230],[0.8312,5635],[0.8311,10750],[0.8310,8314]],"ts":1623545087635,"timestamp":1623545087635},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617731938347,"asks":[[20.657,188],[20.661,2000],[20.664,102],[20.666,10],[20.668,291]],"bids":[[20.643,107],[20.636,107],[20.634,10],[20.628,106],[20.624,88]],"ts":1623545087627,"timestamp":1623545087627},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617732876660,"asks":[[2367.35,10103],[2367.75,10435],[2368.20,6149],[2368.55,7000],[2368.60,5826]],"bids":[[2366.65,20000],[2366.60,12766],[2366.20,5664],[2365.90,5850],[2365.7,117249]],"ts":1623545087664,"timestamp":1623545087664},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731473981,"asks":[[35533,32262],[35534.0,49263],[35538,35880],[35539,80000],[35547.00000000,3199]],"bids":[[35520,219],[35519.00000000,3801],[35513.0,631],[35509,11263],[35508,35760]],"ts":1623545087711,"timestamp":1623545087711},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617732876661,"asks":[[2367.35,10103],[2367.75,10435],[2368.20,6149],[2368.55,7000],[2368.60,5826]],"bids":[[2366.65,20000],[2366.60,12766],[2366.20,5664],[2365.90,5850],[2365.7,117249]],"ts":1623545087764,"timestamp":1623545087764},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617730623077,"asks":[[0.8321,4750],[0.8322,11407],[0.8323,4755],[0.8324,5905],[0.8325,9734]],"bids":[[0.8316,20240],[0.8313,4230],[0.8312,5635],[0.8311,10750],[0.8310,8314]],"ts":1623545087788,"timestamp":1623545087788},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617731938348,"asks":[[20.657,188],[20.661,2000],[20.664,102],[20.666,10],[20.668,291]],"bids":[[20.643,107],[20.636,107],[20.634,10],[20.629,89],[20.628,106]],"ts":1623545087795,"timestamp":1623545087795},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617732876662,"asks":[[2367.35,10103],[2367.75,10435],[2368.00,5508],[2368.60,5826],[2369.05,6426]],"bids":[[2366.65,20000],[2366.60,12766],[2365.90,5850],[2365.7,117249],[2365.30,5922]],"ts":1623545087886,"timestamp":1623545087886},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731938349,"asks":[[20.657,188],[20.661,2000],[20.664,102],[20.666,10],[20.668,291]],"bids":[[20.643,107],[20.636,107],[20.634,10],[20.629,89],[20.628,106]],"ts":1623545087895,"timestamp":1623545087895},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617731473982,"asks":[[35533,32262],[35534.0,49263],[35538,35880],[35539,80000],[35547.00000000,3199]],"bids":[[35520,219],[35519.00000000,3801],[35513.0,631],[35509,11263],[35508,35760]],"ts":1623545087899,"timestamp":1623545087899},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617730623078,"asks":[[0.8321,4750],[0.8322,11407],[0.8323,4755],[0.8324,5905],[0.8325,9734]],"bids":[[0.8316,20240],[0.8314,4710],[0.8313,4230],[0.8312,5635],[0.8311,10750]],"ts":1623545087903,"timestamp":1623545087903},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617731938350,"asks":[[20.657,188],[20.661,2000],[20.664,102],[20.666,10],[20.668,291]],"bids":[[20.643,107],[20.636,107],[20.634,10],[20.629,89],[20.628,106]],"ts":1623545087995,"timestamp":1623545087995},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617732876663,"asks":[[2367.35,10103],[2367.75,10435],[2368.00,5508],[2368.50,6360],[2368.60,5826]],"bids":[[2366.65,20000],[2366.60,12766],[2366.10,5472],[2365.90,5850],[2365.7,117249]],"ts":1623545087997,"timestamp":1623545087997},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731473983,"asks":[[35533,32262],[35534.0,49263],[35538,35880],[35539,80000],[35547.00000000,3199]],"bids":[[35520,219],[35519.00000000,3801],[35513.0,631],[35509,11263],[35508,35760]],"ts":1623545087999,"timestamp":1623545087999},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617730623079,"asks":[[0.8321,24750],[0.8322,11407],[0.8323,4755],[0.8324,5905],[0.8325,9734]],"bids":[[0.8316,20240],[0.8314,4710],[0.8313,4230],[0.8312,5635],[0.8311,10750]],"ts":1623545088034,"timestamp":1623545088034},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617732876664,"asks":[[2367.35,10103],[2367.75,10435],[2368.00,5508],[2368.50,6360],[2368.60,5826]],"bids":[[2366.65,20000],[2366.60,12766],[2366.10,5472],[2365.90,5850],[2365.7,117249]],"ts":1623545088097,"timestamp":1623545088097},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731473984,"asks":[[35533,32262],[35534.0,49263],[35538,35880],[35539,80000],[35547.00000000,3199]],"bids":[[35520,219],[35519.00000000,3801],[35513.0,631],[35509,11263],[35508,35760]],"ts":1623545088099,"timestamp":1623545088099},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617731938351,"asks":[[20.657,188],[20.661,2000],[20.666,252],[20.669,98],[20.672,426]],"bids":[[20.643,107],[20.636,107],[20.634,10],[20.629,89],[20.628,106]],"ts":1623545088103,"timestamp":1623545088103},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617730623080,"asks":[[0.8321,24750],[0.8322,11407],[0.8323,4755],[0.8324,5905],[0.8325,9734]],"bids":[[0.8316,240],[0.8314,4710],[0.8313,4230],[0.8312,5635],[0.8311,10750]],"ts":1623545088170,"timestamp":1623545088170},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617732876665,"asks":[[2367.35,10103],[2367.75,10435],[2368.00,5508],[2368.50,6360],[2368.60,5826]],"bids":[[2366.65,20000],[2366.60,12766],[2366.10,5472],[2365.90,5850],[2365.7,117249]],"ts":1623545088197,"timestamp":1623545088197},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731938352,"asks":[[20.657,188],[20.661,2000],[20.662,96],[20.666,242],[20.669,98]],"bids":[[20.641,103],[20.636,107],[20.632,10],[20.629,89],[20.628,106]],"ts":1623545088263,"timestamp":1623545088263},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617731473985,"asks":[[35533,32262],[35534.0,49263],[35538,35880],[35539,80000],[35547.00000000,3199]],"bids":[[35520,219],[35519.00000000,3801],[35513.0,631],[35509,11263],[35508,35760]],"ts":1623545088264,"timestamp":1623545088264},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617732876666,"asks":[[2367.35,10103],[2367.75,10435],[2368.00,5508],[2368.50,6360],[2368.60,5826]],"bids":[[2366.65,20000],[2366.60,12766],[2366.10,5472],[2365.90,5850],[2365.7,117249]],"ts":1623545088297,"timestamp":1623545088297},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731473986,"asks":[[35533,32262],[35534.0,49263],[35538,35880],[35539,80000],[35547.00000000,3199]],"bids":[[35520,219],[35519.00000000,3801],[35513.0,631],[35509,11263],[35508,35760]],"ts":1623545088364,"timestamp":1623545088364},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617730623081,"asks":[[0.8321,24750],[0.8322,11407],[0.8323,4755],[0.8324,5905],[0.8325,9734]],"bids":[[0.8316,240],[0.8314,4710],[0.8313,4230],[0.8312,5635],[0.8311,10750]],"ts":1623545088359,"timestamp":1623545088359},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617731938353,"asks":[[20.657,188],[20.661,2000],[20.662,10],[20.667,90],[20.669,98]],"bids":[[20.639,82],[20.630,10],[20.629,89],[20.628,96],[20.624,410]],"ts":1623545088384,"timestamp":1623545088384},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617731473987,"asks":[[35533,32262],[35534.0,49263],[35538,35880],[35539,80000],[35547.00000000,3199]],"bids":[[35520,219],[35519.00000000,3801],[35513.0,631],[35509,11263],[35508,35760]],"ts":1623545088464,"timestamp":1623545088464},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617730623082,"asks":[[0.8321,24750],[0.8322,11407],[0.8323,4755],[0.8324,5905],[0.8325,9734]],"bids":[[0.8316,240],[0.8314,4710],[0.8313,4230],[0.8312,5635],[0.8311,10750]],"ts":1623545088474,"timestamp":1623545088474},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617732876667,"asks":[[2367.35,10103],[2367.75,10435],[2368.00,5508],[2368.30,7000],[2368.50,6360]],"bids":[[2366.60,12766],[2366.10,5472],[2365.90,5850],[2365.7,117249],[2365.30,5922]],"ts":1623545088479,"timestamp":1623545088479},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731938354,"asks":[[20.657,188],[20.660,90],[20.661,2000],[20.662,292],[20.667,90]],"bids":[[20.639,82],[20.630,10],[20.629,89],[20.628,96],[20.624,410]],"ts":1623545088557,"timestamp":1623545088557},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617731473988,"asks":[[35533,32262],[35534.0,49263],[35538,35880],[35539,80000],[35547.00000000,3199]],"bids":[[35520,219],[35519.00000000,3801],[35513.0,631],[35509,11263],[35508,35760]],"ts":1623545088564,"timestamp":1623545088564},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617730623083,"asks":[[0.8321,24750],[0.8322,11407],[0.8323,4755],[0.8324,5905],[0.8325,9734]],"bids":[[0.8316,240],[0.8314,4710],[0.8313,4230],[0.8312,5635],[0.8311,10750]],"ts":1623545088574,"timestamp":1623545088574},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617732876668,"asks":[[2367.30,20000],[2367.35,10103],[2367.75,10435],[2368.30,7000],[2368.50,6360]],"bids":[[2366.60,12766],[2365.90,5850],[2365.80,7008],[2365.7,117249],[2365.30,5922]],"ts":1623545088589,"timestamp":1623545088589},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731938355,"asks":[[20.657,188],[20.660,90],[20.661,2000],[20.662,292],[20.667,90]],"bids":[[20.639,82],[20.630,10],[20.629,89],[20.628,96],[20.624,410]],"ts":1623545088657,"timestamp":1623545088657},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617731473989,"asks":[[35533,32262],[35534.0,49263],[35538,35880],[35539,80000],[35547.00000000,3199]],"bids":[[35520,219],[35519.00000000,3801],[35513.0,631],[35509,11263],[35508,35760]],"ts":1623545088665,"timestamp":1623545088665},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617730623084,"asks":[[0.8321,24750],[0.8322,11407],[0.8323,4755],[0.8324,5905],[0.8325,9734]],"bids":[[0.8316,240],[0.8314,4710],[0.8313,4230],[0.8312,5635],[0.8311,10750]],"ts":1623545088708,"timestamp":1623545088708},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617731938356,"asks":[[20.657,188],[20.660,90],[20.661,2000],[20.662,292],[20.667,90]],"bids":[[20.639,82],[20.630,10],[20.629,89],[20.628,96],[20.624,410]],"ts":1623545088757,"timestamp":1623545088757},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617732876669,"asks":[[2367.30,20000],[2367.35,10103],[2367.60,5400],[2367.75,10435],[2368.30,7000]],"bids":[[2366.60,12766],[2365.90,5850],[2365.80,7008],[2365.7,117249],[2365.30,5922]],"ts":1623545088761,"timestamp":1623545088761},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731473990,"asks":[[35533,32262],[35534.0,49263],[35538,35880],[35539,80000],[35547.00000000,3199]],"bids":[[35520,219],[35519.00000000,3801],[35513.0,631],[35509,11263],[35508,35760]],"ts":1623545088765,"timestamp":1623545088765},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617730623085,"asks":[[0.8321,24750],[0.8322,11407],[0.8323,4755],[0.8324,5905],[0.8325,9734]],"bids":[[0.8316,240],[0.8314,4710],[0.8313,4230],[0.8312,5635],[0.8311,10750]],"ts":1623545088809,"timestamp":1623545088809},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617732876670,"asks":[[2367.30,20000],[2367.35,10103],[2367.60,5400],[2367.75,10435],[2368.30,7000]],"bids":[[2366.60,12766],[2365.90,5850],[2365.80,7008],[2365.7,117249],[2365.30,5922]],"ts":1623545088861,"timestamp":1623545088861},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731473991,"asks":[[35533,32262],[35534.0,49263],[35538,35880],[35539,80000],[35547.00000000,3199]],"bids":[[35520,219],[35519.00000000,3801],[35513.0,631],[35509,11263],[35508,35760]],"ts":1623545088865,"timestamp":1623545088865},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617730623086,"asks":[[0.8321,24750],[0.8322,11407],[0.8323,4755],[0.8324,5905],[0.8325,9734]],"bids":[[0.8316,240],[0.8314,4710],[0.8313,4230],[0.8312,5635],[0.8311,10750]],"ts":1623545088909,"timestamp":1623545088909},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617731938357,"asks":[[20.660,90],[20.661,2000],[20.662,292],[20.667,90],[20.669,441]],"bids":[[20.639,82],[20.630,10],[20.629,89],[20.628,96],[20.624,410]],"ts":1623545088927,"timestamp":1623545088927},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617732876671,"asks":[[2367.30,20000],[2367.35,10103],[2367.60,5400],[2367.75,10435],[2368.30,7000]],"bids":[[2366.60,12766],[2365.90,5850],[2365.80,7008],[2365.7,117249],[2365.30,5922]],"ts":1623545088961,"timestamp":1623545088961},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731473992,"asks":[[35533,32262],[35534.0,49263],[35538,35880],[35539,80000],[35547.00000000,3199]],"bids":[[35520,219],[35519.00000000,3801],[35513.0,631],[35509,11263],[35508,35760]],"ts":1623545088997,"timestamp":1623545088997},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617730623087,"asks":[[0.8320,5100],[0.8321,4750],[0.8322,11407],[0.8323,4755],[0.8324,5905]],"bids":[[0.8316,20240],[0.8314,4710],[0.8313,4230],[0.8312,5635],[0.8311,10750]],"ts":1623545089084,"timestamp":1623545089084},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617731938358,"asks":[[20.659,188],[20.660,90],[20.661,2000],[20.662,282],[20.664,10]],"bids":[[20.640,83],[20.634,105],[20.629,89],[20.628,96],[20.624,410]],"ts":1623545089079,"timestamp":1623545089079},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617731473993,"asks":[[35533,32262],[35534.0,49263],[35538,35880],[35539,80000],[35547.00000000,3199]],"bids":[[35520,219],[35519.00000000,3801],[35513.0,631],[35509,11263],[35508,35760]],"ts":1623545089097,"timestamp":1623545089097},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617732876672,"asks":[[2367.30,20000],[2367.35,10103],[2367.75,10435],[2368.30,7000],[2368.50,6360]],"bids":[[2366.65,20000],[2366.60,12766],[2365.90,5850],[2365.80,7008],[2365.7,117249]],"ts":1623545089136,"timestamp":1623545089136},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731938359,"asks":[[20.659,188],[20.661,2000],[20.662,282],[20.669,343],[20.670,10]],"bids":[[20.641,118],[20.634,105],[20.629,89],[20.628,96],[20.624,410]],"ts":1623545089194,"timestamp":1623545089194},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617730623088,"asks":[[0.8320,5100],[0.8321,4750],[0.8322,11407],[0.8323,4755],[0.8324,5905]],"bids":[[0.8316,20240],[0.8315,4845],[0.8314,4710],[0.8313,4230],[0.8312,5635]],"ts":1623545089184,"timestamp":1623545089184},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617731473994,"asks":[[35533,32262],[35542,31710],[35557,30840],[35559.0,87579],[35562,30840]],"bids":[[35520,219],[35519.00000000,3801],[35513.0,631],[35511,27300],[35509,11263]],"ts":1623545089249,"timestamp":1623545089249},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617732876673,"asks":[[2367.35,10103],[2367.75,10435],[2368.00,4974],[2368.30,7000],[2368.50,6360]],"bids":[[2366.65,20000],[2366.60,12766],[2365.90,5850],[2365.7,117249],[2365.30,5922]],"ts":1623545089252,"timestamp":1623545089252},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617730623089,"asks":[[0.8320,5100],[0.8321,4750],[0.8322,11407],[0.8323,4755],[0.8324,5905]],"bids":[[0.8316,20240],[0.8314,4710],[0.8313,4230],[0.8312,5635],[0.8311,10750]],"ts":1623545089289,"timestamp":1623545089289},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617731938360,"asks":[[20.659,188],[20.661,2000],[20.662,282],[20.669,343],[20.672,436]],"bids":[[20.641,118],[20.634,105],[20.632,10],[20.629,89],[20.628,96]],"ts":1623545089315,"timestamp":1623545089315},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617731473995,"asks":[[35533,32262],[35542,31710],[35557,30840],[35562,30840],[35566,3942]],"bids":[[35520,219],[35519.00000000,3801],[35513.0,631],[35511,27300],[35509,11263]],"ts":1623545089360,"timestamp":1623545089360},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
{"data":{"sequence":1617730623090,"asks":[[0.8320,5100],[0.8321,4750],[0.8322,11407],[0.8323,4755],[0.8324,5905]],"bids":[[0.8316,20240],[0.8314,4710],[0.8313,4230],[0.8312,5635],[0.8311,10750]],"ts":1623545089405,"timestamp":1623545089405},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDM","type":"message"}
{"data":{"sequence":1617731938361,"asks":[[20.659,188],[20.661,2000],[20.662,282],[20.669,343],[20.672,436]],"bids":[[20.641,118],[20.634,105],[20.632,10],[20.629,89],[20.628,96]],"ts":1623545089415,"timestamp":1623545089415},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDM","type":"message"}
{"data":{"sequence":1617732876674,"asks":[[2367.35,10103],[2367.75,10435],[2368.00,4974],[2368.30,7000],[2368.50,6360]],"bids":[[2366.65,20000],[2366.60,12766],[2366.10,6851],[2365.90,5850],[2365.7,117249]],"ts":1623545089426,"timestamp":1623545089426},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDM","type":"message"}
{"data":{"sequence":1617731473996,"asks":[[35533,32262],[35542,31710],[35544.0,49859],[35557,30840],[35562,30840]],"bids":[[35520,219],[35519.00000000,3801],[35513.0,631],[35511,27300],[35509,11263]],"ts":1623545089469,"timestamp":1623545089469},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDM","type":"message"}
`

