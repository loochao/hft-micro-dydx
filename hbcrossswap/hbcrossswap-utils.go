package hbcrossswap

import (
	"context"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"time"
	"unsafe"
)

//{"ch":"market.BTC-USDT.depth.step6","ts":1618410970115,"tick":{"mrid":28158325357,"id":1618410970,"bids":[[63402.5,88],[63402.2,6],[63402.1,42],[63401.4,50],[63400.7,24],[63400,238],[63398.9,1],[63398.6,31],[63398.5,300],[63397.3,39],[63397.1,200],[63397,115],[63396.3,51],[63394.6,200],[63393.2,1000],[63392.5,1],[63392,177],[63391.6,115],[63391.5,115],[63391.4,115]],"asks":[[63402.6,20318],[63402.8,46],[63405,1583],[63405.2,300],[63406.7,108],[63406.8,484],[63406.9,325],[63407,58],[63407.1,1120],[63407.2,16590],[63407.3,1016],[63407.4,797],[63407.5,270],[63407.6,753],[63407.7,1178],[63407.8,521],[63407.9,330],[63408,170],[63408.1,1064],[63408.2,606]],"ts":1618410970112,"version":1618410970,"ch":"market.BTC-USDT.depth.step6"}}

func ParseDepth20(bytes []byte) (*Depth20, error) {
	var err error
	orderBook := Depth20{
		Bids:      [20][2]float64{},
		Asks:      [20][2]float64{},
		ParseTime: time.Now(),
	}
	if bytes[12] != 't' && bytes[13] != '.' {
		return nil, fmt.Errorf("bad bytes %s", bytes)
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
				orderBook.Bids[counter/2][counter%2], err = common.ParseBinanceFloat(bytes[collectStart:offset])
				if err != nil {
					return nil, fmt.Errorf("JsonKeyBids error %v start %d end %d %s", err, collectStart, offset, bytes[collectStart:offset])
				}
				counter += 1
				if counter >= 40 {
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
				orderBook.Asks[counter/2][counter%2], err = common.ParseBinanceFloat(bytes[collectStart:offset])
				if err != nil {
					return nil, fmt.Errorf("JsonKeyAsks error %v start %d end %d %s", err, collectStart, offset, bytes[collectStart:offset])
				}
				counter += 1
				if counter >= 40 {
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
				orderBook.Version, err = common.ParseBinanceInt(bytes[collectStart:offset])
				if err != nil {
					return nil, fmt.Errorf("JsonKeyVersion error %v start %d end %d %s", err, collectStart, offset, bytes[collectStart:offset])
				}
				offset = bytesLen
				continue
			}
			break
		case common.JsonKeyEventTime:
			offset += 13
			timestamp, err := common.ParseBinanceInt(bytes[collectStart:offset])
			if err != nil {
				return nil, fmt.Errorf("JsonKeyEventTime error %v start %d end %d %s", err, collectStart, offset, bytes[collectStart:offset])
			}
			orderBook.EventTime = time.Unix(0, timestamp*1000000)
			offset += 11
			collectStart = offset
			currentKey = common.JsonKeyVersion
			continue
		case common.JsonKeyID:
			if bytes[offset] == ',' {
				orderBook.ID, err = common.ParseBinanceInt(bytes[collectStart:offset])
				if err != nil {
					return nil, fmt.Errorf("JsonKeyID error %v start %d end %d %s", err, collectStart, offset, bytes[collectStart:offset])
				}
				offset += 10
				collectStart = offset
				currentKey = common.JsonKeyBids
				counter = 0
			}
			break
		case common.JsonKeyMRID:
			if bytes[offset] == ',' {
				orderBook.MRID, err = common.ParseBinanceInt(bytes[collectStart:offset])
				if err != nil {
					return nil, fmt.Errorf("JsonKeyMRID error %v start %d end %d %s", err, collectStart, offset, bytes[collectStart:offset])
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
	return &orderBook, nil
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
			subCtx, _ := context.WithTimeout(ctx, time.Minute*10)
			positions, err := api.GetPositions(subCtx)
			if err != nil {
				logger.Debugf("WatchPositionsFromHttp GetPositions error %v", err)
			} else {
				//有一种情况是有的合约的仓位是拉不到的, 拉不到的都是空仓
				positionBySymbols := make(map[string]Position)
				for _, symbol := range symbols {
					positionBySymbols[symbol] = Position{
						Symbol: symbol,
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
			subCtx, _ := context.WithTimeout(ctx, time.Minute*10)
			accounts, err := api.GetAccounts(subCtx)
			if err != nil {
				logger.Debugf("WatchAccountFromHttp GetAccountOverView error %v", err)
			} else {
				for _, account := range accounts {
					if account.MarginAsset == "USDT" {
						output <- account
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
) (tickSizes, contractSizes map[string]float64, err error) {
	var contracts []Contract
	contracts, err = api.GetContracts(ctx)
	if err != nil {
		return
	}
	tickSizes = make(map[string]float64)
	contractSizes = make(map[string]float64)
	symbolsMap := make(map[string]string)
	for _, symbol := range symbols {
		symbolsMap[symbol] = symbol
	}
	for _, contract := range contracts {
		if _, ok := symbolsMap[contract.Symbol]; ok {
			delete(symbolsMap, contract.Symbol)
			contractSizes[contract.Symbol] = contract.ContractSize
			tickSizes[contract.Symbol] = contract.PriceTick
		}
	}
	if len(symbolsMap) != 0 {
		err = fmt.Errorf("NO ORDER LIMITS FOR %v", symbolsMap)
	} else {
		logger.Debugf("TICK SIZES %v", tickSizes)
		logger.Debugf("CONTRACT SIZE %v", contractSizes)
	}
	return
}

func WatchFundingRate(
	ctx context.Context, api *API, symbols []string, interval time.Duration,
	output chan map[string]FundingRate,
) {
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	symbolMap := make(map[string]string)
	for _, symbol := range symbols {
		symbolMap[symbol] = symbol
	}
	frMap := make(map[string]FundingRate)
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			frs, err := api.GetFundingRates(ctx)
			if err != nil {
				logger.Debugf("GetFundingRates error %v", err)
			} else {
				for _, fr := range frs {
					if _, ok := symbolMap[fr.Symbol]; ok {
						frMap[fr.Symbol] = fr
					}
				}
				select {
				case <-time.After(time.Millisecond):
					logger.Debug("SEND FR MAP OUT TIMEOUT IN 1MS")
				case output <- frMap:
				}
			}
			timer.Reset(time.Now().Truncate(interval).Add(interval).Sub(time.Now()))
		}
	}
}
