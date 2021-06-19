package binance_busdfuture

import (
	"context"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
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
	return  nil
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
			subCtx, _ := context.WithTimeout(ctx, time.Minute)
			account, err := api.GetAccount(subCtx)
			if err != nil {
				logger.Debugf("api.GetAccount(subCtx) error %v", err)
			} else {
				output <- *account
			}
			timer.Reset(time.Now().Truncate(interval).Add(interval).Sub(time.Now()))
		}
	}
}

func WatchPositionsFromHttp(
	ctx context.Context, api *API, symbols []string, interval time.Duration,
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
			eventTime := time.Now()
			positions, err := api.GetPositions(subCtx)
			if err != nil {
				logger.Debugf("WatchPositionsFromHttp GetPositions error %v", err)
			} else {
				//有一种情况是有的合约的仓位是拉不到的, 拉不到的都是空仓
				positionBySymbols := make(map[string]Position)
				for _, symbol := range symbols {
					positionBySymbols[symbol] = Position{
						Symbol:       symbol,
						PositionSide: "BOTH",
						EventTime:    eventTime,
						ParseTime:    time.Now(),
					}
				}
				for _, position := range positions {
					position := position
					position.ParseTime = time.Now()
					position.EventTime = eventTime
					//if position.PositionAmt == 0 {
					//	position.PositionSide = "BOTH"
					//}
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

func GetOrderLimits(
	ctx context.Context, api *API, symbols []string,
) (tickSizes, stepSizes, minSizes, minNotional, multiplierUps, multiplierDowns map[string]float64, err error) {
	exchangeInfo, err := api.GetExchangeInfo(ctx)
	if err != nil {
		return tickSizes, stepSizes, minSizes, minNotional, multiplierUps, multiplierDowns, err
	}
	tickSizes = make(map[string]float64)
	stepSizes = make(map[string]float64)
	minSizes = make(map[string]float64)
	multiplierUps = make(map[string]float64)
	multiplierDowns = make(map[string]float64)
	minNotional = make(map[string]float64)
	for _, symbol := range exchangeInfo.Symbols {
		if symbol.ContractType != "PERPETUAL" && symbol.Status != "TRADING" {
			continue
		}
		if !common.StringDataContains(symbols, symbol.Symbol) {
			continue
		}
		symbols = append(symbols, symbol.Symbol)
		for _, filter := range symbol.Filters {
			switch filter.FilterType {
			case "PRICE_FILTER":
				tickSizes[symbol.Symbol] = filter.TickSize
			case "MARKET_LOT_SIZE":
				stepSizes[symbol.Symbol] = filter.StepSize
				minSizes[symbol.Symbol] = filter.MinQty
			case "PERCENT_PRICE":
				multiplierUps[symbol.Symbol] = filter.MultiplierUp
				multiplierDowns[symbol.Symbol] = filter.MultiplierDown
			case "MIN_NOTIONAL":
				minNotional[symbol.Symbol] = filter.Notional
			}
		}
	}
	for _, symbol := range symbols {
		if _, ok := tickSizes[symbol]; !ok {
			return nil, nil, nil, nil, nil, nil, fmt.Errorf("NO SWAP TICKSIZE FOR %s", symbol)
		}
		if _, ok := stepSizes[symbol]; !ok {
			return nil, nil, nil, nil, nil, nil, fmt.Errorf("NO SWAP STEPSIZE FOR %s", symbol)
		}
		if _, ok := minSizes[symbol]; !ok {
			return nil, nil, nil, nil, nil, nil, fmt.Errorf("NO SWAP  MINSIZE FOR %s", symbol)
		}
		if _, ok := minNotional[symbol]; !ok {
			return nil, nil, nil, nil, nil, nil, fmt.Errorf("NO SWAP  MIN NOTIONAL FOR %s", symbol)
		}
		if _, ok := multiplierUps[symbol]; !ok {
			return nil, nil, nil, nil, nil, nil, fmt.Errorf("NO SWAP  MULTIPLIER UPS FOR %s", symbol)
		}
		if _, ok := multiplierDowns[symbol]; !ok {
			return nil, nil, nil, nil, nil, nil, fmt.Errorf("NO SWAP  MULTIPLIER DOWNS FOR %s", symbol)
		}
	}
	logger.Debugf("BNSWAP TICK SIZES %v", tickSizes)
	logger.Debugf("BNSWAP STEP SIZES %v", stepSizes)
	logger.Debugf("BNSWAP MIN SIZES %v", minSizes)
	logger.Debugf("BNSWAP MIN NOTIONAL %v", minNotional)
	logger.Debugf("BNSWAP MULTIPLIER UPS %v", multiplierUps)
	logger.Debugf("BNSWAP MULTIPLIER DOWNS %v", multiplierDowns)
	return tickSizes, stepSizes, minSizes, minNotional, multiplierUps, multiplierDowns, nil
}

func UpdateLeverageAndMarginType(ctx context.Context, api *API, symbols []string, leverage int64, marginType string) {
	for _, symbol := range symbols {
		res, err := api.UpdateLeverage(ctx, UpdateLeverageParams{
			Symbol:   symbol,
			Leverage: leverage,
		})
		if err != nil {
			logger.Debugf("UPDATE LEVERAGE FOR %s ERROR %v", symbol, err)
		} else {
			logger.Debugf("UPDATE LEVERAGE FOR %s RESPONSE %v", symbol, res)
		}
		time.Sleep(time.Second)
		res, err = api.UpdateMarginType(ctx, UpdateMarginTypeParams{
			Symbol:     symbol,
			MarginType: marginType,
		})
		if err != nil {
			logger.Debugf("UPDATE MARGIN TYPE FOR %s ERROR %v", symbol, err)
		} else {
			logger.Debugf("UPDATE MARGIN TYPE FOR %s RESPONSE %v", symbol, res)
		}
		time.Sleep(time.Second)
	}
}

func WatchPremiumIndexesFromHttp(
	ctx context.Context, api *API, symbols []string, interval time.Duration,
	output chan map[string]PremiumIndex,
) {
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			subCtx, _ := context.WithTimeout(ctx, time.Minute)
			indexes, err := api.GetPremiumIndex(subCtx)
			if err != nil {
				logger.Debugf("WatchPositionsFromHttp GetPositions error %v", err)
			} else {
				indexMap := make(map[string]PremiumIndex)
				for _, symbol := range symbols {
					indexMap[symbol] = PremiumIndex{
						Symbol: symbol,
					}
				}
				for _, i := range indexes {
					if _, ok := indexMap[i.Symbol]; ok {
						indexMap[i.Symbol] = i
					}
				}
				output <- indexMap
			}
			timer.Reset(time.Now().Truncate(interval).Add(interval).Sub(time.Now()))
		}
	}
}

func ParseDepth5(bytes []byte, depth5 *Depth5) error {
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

func SystemStatusLoop(
	ctx context.Context, api *API, interval time.Duration,
	output chan bool,
) {
	logger.Debugf("START SystemStatusLoop")
	defer logger.Debugf("EXIT SystemStatusLoop")
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
