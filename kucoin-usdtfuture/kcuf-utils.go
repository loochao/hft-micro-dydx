package kucoin_usdtfuture

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

var Depth5SampleLines = `{"data":{"sequence":1617730626597,"asks":[[35466.0,63],[35467.0,2],[35468,2266],[35469.0,2246],[35471,74]],"bids":[[35464,2],[35463,24],[35462,1856],[35461,4927],[35460,1791]],"ts":1623511660527,"timestamp":1623511660527},"subject":"level2","topic":"/contractMarket/level2Depth5:XBTUSDTM","type":"message"}
{"data":{"sequence":1617992464123,"asks":[[0.6285,293],[0.6286,4355],[0.6287,4484],[0.6288,3936],[0.6289,3512]],"bids":[[0.6280,3475],[0.6279,3438],[0.6278,2858],[0.6277,5050],[0.6276,4127]],"ts":1623511661603,"timestamp":1623511661603},"subject":"level2","topic":"/contractMarket/level2Depth5:BATUSDTM","type":"message"}
{"data":{"sequence":1619017805158,"asks":[[299.64,553],[299.71,56],[299.74,489],[299.77,473],[299.80,830]],"bids":[[299.45,20],[299.39,217],[299.29,212],[299.22,196],[299.17,194]],"ts":1623511662489,"timestamp":1623511662489},"subject":"level2","topic":"/contractMarket/level2Depth5:COMPUSDTM","type":"message"}
{"data":{"sequence":1617731068821,"asks":[[0.02681,444],[0.02682,623],[0.02683,595],[0.02684,712],[0.02685,729]],"bids":[[0.02679,233],[0.02678,2576],[0.02677,2806],[0.02676,1965],[0.02675,1617]],"ts":1623511663500,"timestamp":1623511663500},"subject":"level2","topic":"/contractMarket/level2Depth5:IOSTUSDTM","type":"message"}
{"data":{"sequence":1617159792806,"asks":[[0.490,3889],[0.491,4273],[0.492,8407],[0.493,53980],[0.494,36894]],"bids":[[0.489,1600],[0.488,6241],[0.487,6304],[0.486,9306],[0.485,59594]],"ts":1623511664527,"timestamp":1623511664527},"subject":"level2","topic":"/contractMarket/level2Depth5:OCEANUSDTM","type":"message"}
{"data":{"sequence":1617731141786,"asks":[[70.05,1941],[70.06,1985],[70.07,1773],[70.08,1751],[70.09,1873]],"bids":[[70.04,2027],[70.03,4203],[70.02,3685],[70.01,3125],[70.00,2742]],"ts":1623511665512,"timestamp":1623511665512},"subject":"level2","topic":"/contractMarket/level2Depth5:FILUSDTM","type":"message"}
{"data":{"sequence":1621086699949,"asks":[[8.673,4802],[8.674,4723],[8.675,3705],[8.676,3867],[8.677,3394]],"bids":[[8.671,713],[8.670,657],[8.669,1020],[8.668,3801],[8.667,4346]],"ts":1623511666506,"timestamp":1623511666506},"subject":"level2","topic":"/contractMarket/level2Depth5:QTUMUSDTM","type":"message"}
{"data":{"sequence":1620748056394,"asks":[[56.8,4750],[56.9,15554],[57.0,30687],[57.1,27290],[57.2,23954]],"bids":[[56.7,8540],[56.6,23339],[56.5,11714],[56.4,28370],[56.3,19519]],"ts":1623511667497,"timestamp":1623511667497},"subject":"level2","topic":"/contractMarket/level2Depth5:ICPUSDTM","type":"message"}
{"data":{"sequence":1617729911577,"asks":[[0.10423,17],[0.10424,200],[0.10427,581],[0.10428,1241],[0.10429,1552]],"bids":[[0.10417000,48],[0.10412000,668],[0.10410,989],[0.10409,960],[0.10408,676]],"ts":1623511668515,"timestamp":1623511668515},"subject":"level2","topic":"/contractMarket/level2Depth5:VETUSDTM","type":"message"}
{"data":{"sequence":1619017633810,"asks":[[6.549,1857],[6.550,1614],[6.551,1873],[6.552,1517],[6.553,1626]],"bids":[[6.543,1568],[6.542,1351],[6.541,1132],[6.540,1183],[6.539,983]],"ts":1623511669557,"timestamp":1623511669557},"subject":"level2","topic":"/contractMarket/level2Depth5:BANDUSDTM","type":"message"}
{"data":{"sequence":1617992745380,"asks":[[3.144,4227],[3.145,1788],[3.146,1858],[3.147,2133],[3.148,1608]],"bids":[[3.141,2085],[3.140,2333],[3.139,4364],[3.138,5487],[3.137,4281]],"ts":1623511670619,"timestamp":1623511670619},"subject":"level2","topic":"/contractMarket/level2Depth5:XTZUSDTM","type":"message"}
{"data":{"sequence":1617730755324,"asks":[[4.823,12],[4.824,15777],[4.825,22116],[4.826,21769],[4.827,25977]],"bids":[[4.821,10728],[4.820,14442],[4.819,13951],[4.818,19243],[4.817,15108]],"ts":1623511671480,"timestamp":1623511671480},"subject":"level2","topic":"/contractMarket/level2Depth5:EOSUSDTM","type":"message"}
{"data":{"sequence":1621086648423,"asks":[[8.302,1009],[8.303,1409],[8.304,1476],[8.305,1456],[8.306,1520]],"bids":[[8.298,452],[8.296,548],[8.295,627],[8.294,706],[8.293,1076]],"ts":1623511672522,"timestamp":1623511672522},"subject":"level2","topic":"/contractMarket/level2Depth5:SUSHIUSDTM","type":"message"}
{"data":{"sequence":1616616952950,"asks":[[0.003371,1694],[0.003372,1958],[0.003373,2512],[0.003374,3159],[0.003375,3469]],"bids":[[0.003368,1394],[0.003367,2288],[0.003366,2872],[0.003365,4377],[0.003364,2824]],"ts":1623511673494,"timestamp":1623511673494},"subject":"level2","topic":"/contractMarket/level2Depth5:BTTUSDTM","type":"message"}
{"data":{"sequence":1617729603462,"asks":[[36752,1520],[36766,1391],[36770,745],[36771,1389],[36780,737]],"bids":[[36739,1261],[36738,1196],[36728,891],[36722,1306],[36720,244]],"ts":1623511674567,"timestamp":1623511674567},"subject":"level2","topic":"/contractMarket/level2Depth5:YFIUSDTM","type":"message"}
{"data":{"sequence":1617731406441,"asks":[[0.8320,13035],[0.8321,13706],[0.8322,12426],[0.8323,15030],[0.8324,13227]],"bids":[[0.8317,24],[0.8316,7007],[0.83150000,14582],[0.8314,11523],[0.83130000,12821]],"ts":1623511675647,"timestamp":1623511675647},"subject":"level2","topic":"/contractMarket/level2Depth5:XRPUSDTM","type":"message"}
{"data":{"sequence":1620677135751,"asks":[[0.00000598,6919],[0.00000599,33433],[0.00000600,30425],[0.00000601,24617],[0.00000602,45297]],"bids":[[0.00000597,882],[0.00000596,41317],[0.00000595,17877],[0.00000594,18535],[0.00000593,189263]],"ts":1623511676537,"timestamp":1623511676537},"subject":"level2","topic":"/contractMarket/level2Depth5:SHIBUSDTM","type":"message"}
{"data":{"sequence":1619018819124,"asks":[[55.381,35],[55.382,578],[55.385,904],[55.387,578],[55.416,8816]],"bids":[[55.307,2],[55.306,453],[55.303,634],[55.294,4688],[55.292,2976]],"ts":1623511677587,"timestamp":1623511677587},"subject":"level2","topic":"/contractMarket/level2Depth5:ETCUSDTM","type":"message"}
{"data":{"sequence":1618414559195,"asks":[[245.16,2219],[245.21,1901],[245.24,2126],[245.30,2250],[245.33,2316]],"bids":[[245.03,16],[245.02,3937],[244.93,3464],[244.92,907],[244.85,834]],"ts":1623511678586,"timestamp":1623511678586},"subject":"level2","topic":"/contractMarket/level2Depth5:XMRUSDTM","type":"message"}
{"data":{"sequence":1617730832914,"asks":[[576.50,4365],[576.65,7338],[576.75,3385],[576.80,2234],[576.85,3965]],"bids":[[576.05,9169],[575.95,4098],[575.90,6086],[575.85000000,137],[575.80,6593]],"ts":1623511679567,"timestamp":1623511679567},"subject":"level2","topic":"/contractMarket/level2Depth5:BCHUSDTM","type":"message"}
{"data":{"sequence":1617729886708,"asks":[[5.640,500],[5.641,403],[5.642,313],[5.643,745],[5.644,668]],"bids":[[5.635,454],[5.634,541],[5.633,514],[5.632,677],[5.631,912]],"ts":1623511680536,"timestamp":1623511680536},"subject":"level2","topic":"/contractMarket/level2Depth5:LUNAUSDTM","type":"message"}
{"data":{"sequence":1617731887099,"asks":[[2412.4,16255],[2412.45,700],[2412.50,249],[2412.55,204],[2412.75,249]],"bids":[[2412.1,574],[2411.75,50],[2411.70,8068],[2411.45,930],[2411.20,291]],"ts":1623511681595,"timestamp":1623511681595},"subject":"level2","topic":"/contractMarket/level2Depth5:ETHUSDTM","type":"message"}
{"data":{"sequence":1619631970844,"asks":[[3.747,5069],[3.748,3],[3.75,4],[3.751,952],[3.755,143]],"bids":[[3.724,1800],[3.723,498],[3.722,424],[3.721,555],[3.72,598]],"ts":1623511682531,"timestamp":1623511682531},"subject":"level2","topic":"/contractMarket/level2Depth5:MIRUSDTM","type":"message"}
{"data":{"sequence":1616617227138,"asks":[[7.862,5300],[7.863,6986],[7.864,20656],[7.865,23744],[7.866,19605]],"bids":[[7.860,6304],[7.858,8152],[7.857,8579],[7.856,8106],[7.855,6688]],"ts":1623511683600,"timestamp":1623511683600},"subject":"level2","topic":"/contractMarket/level2Depth5:THETAUSDTM","type":"message"}
{"data":{"sequence":1617157080077,"asks":[[1.235,2248],[1.236,2097],[1.237,5398],[1.238,5163],[1.239,5909]],"bids":[[1.234,759],[1.233,3154],[1.232,3351],[1.231,2160],[1.230,3371]],"ts":1623511684522,"timestamp":1623511684522},"subject":"level2","topic":"/contractMarket/level2Depth5:ENJUSDTM","type":"message"}
{"data":{"sequence":1617730280701,"asks":[[0.2418,10670],[0.2419,12662],[0.2420,9265],[0.2421,12130],[0.24220,22198]],"bids":[[0.2415,8770],[0.2414,7243],[0.2413,6863],[0.2412,7152],[0.2411,15712]],"ts":1623511685669,"timestamp":1623511685669},"subject":"level2","topic":"/contractMarket/level2Depth5:FTMUSDTM","type":"message"}
{"data":{"sequence":1619628210961,"asks":[[2969.1,20],[2969.6,148],[2970.6,303],[2970.7,891],[2971.1,696]],"bids":[[2965.9,60],[2965.8,394],[2965.1,383],[2964.9,1106],[2964.7,369]],"ts":1623511686586,"timestamp":1623511686586},"subject":"level2","topic":"/contractMarket/level2Depth5:MKRUSDTM","type":"message"}
{"data":{"sequence":1617731038947,"asks":[[161.92,10],[161.93,2587],[161.98,3419],[162.00,63],[162.02,3294]],"bids":[[161.89,6],[161.88,4700],[161.85,3448],[161.83,791],[161.82,4389]],"ts":1623511687602,"timestamp":1623511687602},"subject":"level2","topic":"/contractMarket/level2Depth5:LTCUSDTM","type":"message"}
{"data":{"sequence":1617731309769,"asks":[[1.46360,1205],[1.46376,3895],[1.46386,58],[1.46411,57],[1.46415,13]],"bids":[[1.46217,909],[1.46209,5832],[1.46200,917],[1.46188,912],[1.46171,5728]],"ts":1623511688519,"timestamp":1623511688519},"subject":"level2","topic":"/contractMarket/level2Depth5:ADAUSDTM","type":"message"}
{"data":{"sequence":1617730186027,"asks":[[21.494,324],[21.503,344],[21.506,456],[21.507,117],[21.508,35]],"bids":[[21.486,294],[21.480,271],[21.475,9],[21.474,360],[21.473,82]],"ts":1623511689609,"timestamp":1623511689609},"subject":"level2","topic":"/contractMarket/level2Depth5:UNIUSDTM","type":"message"}
{"data":{"sequence":1618413870419,"asks":[[46.46,12],[46.47,1100],[46.48,1411],[46.49,1214],[46.50,2166]],"bids":[[46.44,1155],[46.43,1007],[46.42,1497],[46.41,1096],[46.40,1714]],"ts":1623511690588,"timestamp":1623511690588},"subject":"level2","topic":"/contractMarket/level2Depth5:NEOUSDTM","type":"message"}
{"data":{"sequence":1616616789884,"asks":[[0.2867,449],[0.2868,5625],[0.2869,7068],[0.2870,24063],[0.2871,9925]],"bids":[[0.2866,669],[0.2865,669],[0.2864,54680],[0.2863,71467],[0.2862,70631]],"ts":1623511691610,"timestamp":1623511691610},"subject":"level2","topic":"/contractMarket/level2Depth5:CHZUSDTM","type":"message"}
{"data":{"sequence":1617731061326,"asks":[[20.616,1261],[20.619,57],[20.620,1091],[20.624,919],[20.626,244]],"bids":[[20.615,706],[20.607,523],[20.603,1105],[20.599,460],[20.596,567]],"ts":1623511692638,"timestamp":1623511692638},"subject":"level2","topic":"/contractMarket/level2Depth5:DOTUSDTM","type":"message"}
{"data":{"sequence":1617730385602,"asks":[[0.60340,700],[0.60348,1740],[0.60352,2761],[0.60361,4391],[0.60362,2251]],"bids":[[0.60277,281],[0.60276,1331],[0.60255,3735],[0.60254,1702],[0.60248,1528]],"ts":1623511693574,"timestamp":1623511693574},"subject":"level2","topic":"/contractMarket/level2Depth5:GRTUSDTM","type":"message"}
{"data":{"sequence":1617730560239,"asks":[[1.2619,10],[1.2621,3795],[1.2623,112],[1.2624,2976],[1.2627,3338]],"bids":[[1.2616,10],[1.2614,1690],[1.2613,191],[1.2612,2071],[1.2609,1635]],"ts":1623511694597,"timestamp":1623511694597},"subject":"level2","topic":"/contractMarket/level2Depth5:MATICUSDTM","type":"message"}
{"data":{"sequence":1617729793928,"asks":[[340.18,15029],[340.26,36403],[340.29,18202],[340.32,18680],[340.35,1470]],"bids":[[339.97,2895],[339.96,2897],[339.95,2131],[339.94,515],[339.93,6051]],"ts":1623511695611,"timestamp":1623511695611},"subject":"level2","topic":"/contractMarket/level2Depth5:BNBUSDTM","type":"message"}
{"data":{"sequence":1621086411811,"asks":[[283.34,4478],[283.35,61],[283.44,4113],[283.46,3735],[283.52,1250]],"bids":[[283.15,2441],[283.07,1119],[283.04,2916],[283.02,754],[282.98,998]],"ts":1623511696620,"timestamp":1623511696620},"subject":"level2","topic":"/contractMarket/level2Depth5:AAVEUSDTM","type":"message"}
{"data":{"sequence":1617731521067,"asks":[[2.214,9144],[2.215,8725],[2.21600000,13170],[2.217,10719],[2.218,10736]],"bids":[[2.212,753],[2.211,6387],[2.210,5791],[2.20900000,6774],[2.208,6460]],"ts":1623511697656,"timestamp":1623511697656},"subject":"level2","topic":"/contractMarket/level2Depth5:CRVUSDTM","type":"message"}
{"data":{"sequence":1620887784685,"asks":[[162.88,1624],[162.95,1856],[162.97,1548],[162.99,3287],[163.00,1444]],"bids":[[162.81,118],[162.80,3165],[162.76,584],[162.75,6527],[162.74,2250]],"ts":1623511698640,"timestamp":1623511698640},"subject":"level2","topic":"/contractMarket/level2Depth5:DASHUSDTM","type":"message"}
{"data":{"sequence":1619629785279,"asks":[[0.0575,92],[0.05751,908],[0.05752,773],[0.05753,766],[0.05754,958]],"bids":[[0.05741,635],[0.05740,512],[0.05739,656],[0.05738,718],[0.05737,709]],"ts":1623511699669,"timestamp":1623511699669},"subject":"level2","topic":"/contractMarket/level2Depth5:DGBUSDTM","type":"message"}
{"data":{"sequence":1617729498880,"asks":[[411.340,1295],[411.574,129],[411.593,1458],[411.617,1170],[411.640,944]],"bids":[[411.163,53],[411.162,53],[411.160,2174],[411.063,132],[411.033,2319]],"ts":1623511700678,"timestamp":1623511700678},"subject":"level2","topic":"/contractMarket/level2Depth5:KSMUSDTM","type":"message"}
{"data":{"sequence":1618414744831,"asks":[[8.608,1200],[8.609,1099],[8.610,2024],[8.611,2626],[8.612,2470]],"bids":[[8.601,951],[8.600,1411],[8.599,1103],[8.598,1104],[8.597,1023]],"ts":1623511701615,"timestamp":1623511701615},"subject":"level2","topic":"/contractMarket/level2Depth5:SNXUSDTM","type":"message"}
{"data":{"sequence":1617730739907,"asks":[[0.31144,1],[0.31145,2],[0.31154000,36],[0.31156,1178],[0.311580,65]],"bids":[[0.31140,4],[0.31139,5278],[0.31138,4],[0.31135,7],[0.31126,106]],"ts":1623511702592,"timestamp":1623511702592},"subject":"level2","topic":"/contractMarket/level2Depth5:DOGEUSDTM","type":"message"}
{"data":{"sequence":1617731786951,"asks":[[0.969,7178],[0.970,9032],[0.971,6526],[0.972,10160],[0.973,47780]],"bids":[[0.967,1225],[0.966,1913],[0.965,6661],[0.964,11298],[0.963,9662]],"ts":1623511703699,"timestamp":1623511703699},"subject":"level2","topic":"/contractMarket/level2Depth5:ALGOUSDTM","type":"message"}
{"data":{"sequence":1618414840025,"asks":[[0.8949,716],[0.8950,722],[0.8951,928],[0.8952,777],[0.8953,7827]],"bids":[[0.8948,1920],[0.8944,1238],[0.8943,927],[0.8942,883],[0.8941,867]],"ts":1623511704592,"timestamp":1623511704592},"subject":"level2","topic":"/contractMarket/level2Depth5:ONTUSDTM","type":"message"}
{"data":{"sequence":1619017878562,"asks":[[14.899,20],[14.901,1523],[14.905,2904],[14.907,3307],[14.908,3051]],"bids":[[14.894,20],[14.893,617],[14.890,523],[14.888,480],[14.886,1953]],"ts":1623511705642,"timestamp":1623511705642},"subject":"level2","topic":"/contractMarket/level2Depth5:WAVESUSDTM","type":"message"}
{"data":{"sequence":1617731033894,"asks":[[36.412,3384],[36.422,51],[36.423,3749],[36.428,3298],[36.436,4166]],"bids":[[36.386,25],[36.374,289],[36.370,210],[36.369,4446],[36.365,2064]],"ts":1623511706684,"timestamp":1623511706684},"subject":"level2","topic":"/contractMarket/level2Depth5:SOLUSDTM","type":"message"}
{"data":{"sequence":1621086680388,"asks":[[0.32536,3481],[0.32545,4522],[0.32553,3745],[0.32559,1429],[0.32560,3106]],"bids":[[0.32528,1132],[0.32519,1192],[0.32509,181],[0.32507,1551],[0.32505,2257]],"ts":1623511707630,"timestamp":1623511707630},"subject":"level2","topic":"/contractMarket/level2Depth5:XLMUSDTM","type":"message"}
{"data":{"sequence":1617731342379,"asks":[[0.06820,4618],[0.06821,4508],[0.06822,4630],[0.06823,5533],[0.06824,5030]],"bids":[[0.06818,8],[0.06817,5747],[0.06816,4922],[0.06815,4969],[0.06814,8025]],"ts":1623511708652,"timestamp":1623511708652},"subject":"level2","topic":"/contractMarket/level2Depth5:TRXUSDTM","type":"message"}
{"data":{"sequence":1621086394087,"asks":[[2.664,1468],[2.665,1432],[2.666,1970],[2.667,2611],[2.668,917]],"bids":[[2.662,898],[2.661,776],[2.660,798],[2.659,639],[2.658,1512]],"ts":1623511709629,"timestamp":1623511709629},"subject":"level2","topic":"/contractMarket/level2Depth5:1INCHUSDTM","type":"message"}
{"data":{"sequence":1619628854443,"asks":[[0.07006,449],[0.07008,1118],[0.07009,1351],[0.07010,1025],[0.07011,4056]],"bids":[[0.07001,1554],[0.07000,1461],[0.06999,1269],[0.06998,1184],[0.06997,1172]],"ts":1623511710631,"timestamp":1623511710631},"subject":"level2","topic":"/contractMarket/level2Depth5:RVNUSDTM","type":"message"}
{"data":{"sequence":1617730771679,"asks":[[21.687,40],[21.688,16325],[21.692,19414],[21.693,39],[21.696,20040]],"bids":[[21.683,20],[21.675,14878],[21.671,5285],[21.670,2308],[21.669,10855]],"ts":1623511711677,"timestamp":1623511711677},"subject":"level2","topic":"/contractMarket/level2Depth5:LINKUSDTM","type":"message"}
{"data":{"sequence":1616618071164,"asks":[[11.658,3018],[11.663,3809],[11.664,110],[11.665,2915],[11.667,4354]],"bids":[[11.654,3784],[11.650,484],[11.649,3601],[11.647,1265],[11.646,1345]],"ts":1623511712705,"timestamp":1623511712705},"subject":"level2","topic":"/contractMarket/level2Depth5:ATOMUSDTM","type":"message"}
{"data":{"sequence":1620887561380,"asks":[[127.20,2956],[127.26,3698],[127.29,3453],[127.30,3530],[127.33,8001]],"bids":[[127.17,30],[127.16,8421],[127.15,3169],[127.14,5688],[127.13,2737]],"ts":1623511713671,"timestamp":1623511713671},"subject":"level2","topic":"/contractMarket/level2Depth5:ZECUSDTM","type":"message"}
{"data":{"sequence":1617995317858,"asks":[[0.1583,31],[0.1584,17924],[0.1585,23822],[0.1586,15188],[0.1587,42289]],"bids":[[0.1581,38547],[0.1580,49203],[0.1579,39762],[0.1578,34238],[0.1577,35522]],"ts":1623511714632,"timestamp":1623511714632},"subject":"level2","topic":"/contractMarket/level2Depth5:XEMUSDTM","type":"message"}
{"data":{"sequence":1617159581666,"asks":[[0.665,2880],[0.666,5023],[0.667,5987],[0.668,6813],[0.669,10392]],"bids":[[0.664,1091],[0.663,4785],[0.662,19592],[0.661,1700],[0.660,42396]],"ts":1623511715710,"timestamp":1623511715710},"subject":"level2","topic":"/contractMarket/level2Depth5:MANAUSDTM","type":"message"}
{"data":{"sequence":1617158325250,"asks":[[0.003130,7129],[0.003131,6722],[0.003132,1794],[0.003133,1530],[0.003134,2840]],"bids":[[0.003124,613],[0.003123,613],[0.003119,22450],[0.003118,7823],[0.003117,7612]],"ts":1623511716665,"timestamp":1623511716665},"subject":"level2","topic":"/contractMarket/level2Depth5:DENTUSDTM","type":"message"}
{"data":{"sequence":1617730748179,"asks":[[1.702,3979],[1.703,4243],[1.704,5631],[1.705,4831],[1.706,6031]],"bids":[[1.701,3002],[1.700,958],[1.699,1961],[1.698,2546],[1.697,2969]],"ts":1623511717680,"timestamp":1623511717680},"subject":"level2","topic":"/contractMarket/level2Depth5:SXPUSDTM","type":"message"}
{"data":{"sequence":1617730408564,"asks":[[13.26,3141],[13.27,2277],[13.28,3132],[13.29,2430],[13.30,1745]],"bids":[[13.25,2049],[13.24,5631],[13.23,2473],[13.22,5257],[13.21,6779]],"ts":1623511718666,"timestamp":1623511718666},"subject":"level2","topic":"/contractMarket/level2Depth5:AVAXUSDTM","type":"message"}
{"data":{"sequence":1617729979632,"asks":[[161.90,4337],[161.95,6550],[162.00,5212],[162.05,3558],[162.10,13584]],"bids":[[161.75,2252],[161.70,2260],[161.65,2110],[161.60,2756],[161.55,8799]],"ts":1623511719640,"timestamp":1623511719640},"subject":"level2","topic":"/contractMarket/level2Depth5:BSVUSDTM","type":"message"}
{"data":{"sequence":1620871789135,"asks":[[160.10,1426],[160.11,30],[160.12,117],[160.15,98],[160.21,137]],"bids":[[159.6900000000000000000000000,1],[159.68,2233],[159.62,455],[159.58,1394],[159.56,361]],"ts":1621608082211,"timestamp":1621608082211},"subject":"level2","topic":"/contractMarket/level2Depth5:ZECUSDTM","type":"message"}`
