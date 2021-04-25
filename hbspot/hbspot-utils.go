package hbspot

import (
	"context"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"strings"
	"time"
	"unsafe"
)

//{"ch":"market.btcusdt.depth.step1","ts":1618475611868,"tick":{"bids":[[62753.2,2.140127],[62753.1,1.005768],[62751.4,0.01],[62750.2,1.588851],[62750.1,0.132173],[62747.1,1.04731],[62747.0,1.357035],[62746.0,0.001031],[62744.8,0.44207],[62744.6,0.064435],[62743.0,0.051222],[62741.8,8.0E-4],[62739.5,0.450211],[62739.0,0.026874],[62737.6,0.2],[62737.0,0.001401],[62736.9,0.1],[62736.7,0.047803],[62735.4,1.6E-4],[62733.6,0.135775]],"asks":[[62753.3,0.038953],[62754.3,0.03],[62758.0,0.09781],[62758.8,0.045154],[62759.5,0.01],[62760.8,0.134133],[62761.8,0.03],[62761.9,0.132173],[62763.4,8.95E-4],[62763.8,0.123199],[62764.2,0.06],[62765.0,0.010786],[62765.8,0.002],[62766.0,0.162855],[62767.2,0.001596],[62767.6,1.145],[62768.9,2.733775],[62769.1,0.159325],[62770.7,0.017235],[62771.5,0.04]],"version":125019588599,"ts":1618475611865}}

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
				orderBook.Bids[counter/2][counter%2], err = common.ParseFloat(bytes[collectStart:offset])
				if err != nil {
					return nil, fmt.Errorf("JsonKeyBids error %v mainLoop %d end %d %s", err, collectStart, offset, bytes[collectStart:offset])
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
				orderBook.Asks[counter/2][counter%2], err = common.ParseFloat(bytes[collectStart:offset])
				if err != nil {
					return nil, fmt.Errorf("JsonKeyAsks error %v mainLoop %d end %d %s", err, collectStart, offset, bytes[collectStart:offset])
				}
				counter += 1
				if counter >= 40 {
					currentKey = common.JsonKeyVersion
					offset += 13
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
				orderBook.Version, err = common.ParseInt(bytes[collectStart:offset])
				if err != nil {
					return nil, fmt.Errorf("JsonKeyVersion error %v mainLoop %d end %d %s", err, collectStart, offset, bytes[collectStart:offset])
				}
				currentKey = common.JsonKeyEventTime
				offset += 6
				collectStart = offset
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
			offset = bytesLen
			continue
		case common.JsonKeySymbol:
			if bytes[offset] == '.' {
				symbol := bytes[collectStart:offset]
				orderBook.Symbol = *(*string)(unsafe.Pointer(&symbol))
				offset += 50
				collectStart = offset
				currentKey = common.JsonKeyBids
				counter = 0
			}
			break
		}
		offset += 1
	}
	return &orderBook, nil
}

func WatchBalancesFromHttp(
	ctx context.Context, api *API,
	symbols []string, interval time.Duration,
	output chan map[string]Balance,
) {
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	subCtx, _ := context.WithTimeout(ctx, time.Minute)
	accounts, err := api.GetAccounts(subCtx)
	if err != nil {
		logger.Fatal(err)
	}
	spotAccountID := int64(0)
	for _, a := range accounts {
		if a.Type == "spot" {
			spotAccountID = a.ID
		}
	}
	if spotAccountID == 0 {
		logger.Fatal("NO SPOT ACCOUNT FOUND!")
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			subCtx, _ := context.WithTimeout(ctx, time.Minute)
			account, err := api.GetAccount(subCtx, spotAccountID)
			if err != nil {
				logger.Debugf("WatchPositionsFromHttp GetAccount error %v", err)
			} else {
				balanceBySymbol := make(map[string]Balance)
				for _, symbol := range symbols {
					balanceBySymbol[symbol] = Balance{
						Symbol:   symbol,
						Currency: strings.Replace(symbol, "usdt", "", -1),
					}
				}
				balanceBySymbol["usdtusdt"] = Balance{
					Symbol:   "usdtusdt",
					Currency: "usdtusdt",
				}
				for _, wsBalance := range account.Balances {
					symbol := wsBalance.Currency + "usdt"
					if balance, ok := balanceBySymbol[symbol]; ok {
						switch wsBalance.Type {
						case "trade":
							balance.Trade = wsBalance.Balance
						case "frozen":
							balance.Frozen = wsBalance.Balance
						default:
						}
						balance.Available = balance.Trade
						balance.Balance = balance.Trade + balance.Frozen
						balanceBySymbol[symbol] = balance
					}
				}
				output <- balanceBySymbol
			}
			timer.Reset(time.Now().Truncate(interval).Add(interval).Sub(time.Now()))
		}
	}
}

func WatchAccountFromHttp(
	ctx context.Context, api *API, accountID int64, interval time.Duration,
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
			account, err := api.GetAccount(subCtx, accountID)
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
) (tickSizes, stepSizes, minSizes, minNotional map[string]float64, pricePrecisions, amountPrecisions map[string]int, err error) {
	var allSymbols []Symbol
	subCtx, _ := context.WithTimeout(ctx, time.Minute)
	allSymbols, err = api.GetSymbols(subCtx)
	if err != nil {
		return
	}
	tickSizes = make(map[string]float64)
	stepSizes = make(map[string]float64)
	minSizes = make(map[string]float64)
	minNotional = make(map[string]float64)
	pricePrecisions = make(map[string]int)
	amountPrecisions = make(map[string]int)
	symbolsMap := make(map[string]string)
	for _, symbol := range symbols {
		symbolsMap[symbol] = symbol
	}
	for _, symbol := range allSymbols {
		if _, ok := symbolsMap[symbol.Symbol]; ok {
			delete(symbolsMap, symbol.Symbol)
			tickSizes[symbol.Symbol] = math.Pow(10, -float64(symbol.PricePrecision))
			stepSizes[symbol.Symbol] = math.Pow(10, -float64(symbol.AmountPrecision))
			pricePrecisions[symbol.Symbol] = symbol.PricePrecision
			amountPrecisions[symbol.Symbol] = symbol.AmountPrecision
			minSizes[symbol.Symbol] = symbol.MinOrderAmt
			minNotional[symbol.Symbol] = symbol.MinOrderValue
		}
	}
	if len(symbolsMap) != 0 {
		err = fmt.Errorf("NO ORDER LIMITS FOR %v", symbolsMap)
	} else {
		logger.Debugf("TICK SIZES %v", tickSizes)
		logger.Debugf("STEP SIZES %v", stepSizes)
		logger.Debugf("MIN  SIZES %v", minSizes)
		logger.Debugf("MIN  NOTIONAL %v", minNotional)
	}
	return
}


func SystemStatusLoop(
	ctx context.Context,
	api *API,
	interval time.Duration,
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
			marketStatus, err := api.GetMarketStatus(subCtx)
			if err != nil {
				logger.Debugf("api.GetHeartBeat error %v", err)
				select {
				case output <- false:
				default:
					logger.Debugf("output <- false failed, ch len %d", len(output))
				}
			} else {
				if marketStatus.MarketStatus == 1 {
					select {
					case output <- true:
					default:
						logger.Debugf("output <- true failed, ch len %d", len(output))
					}
				} else {
					select {
					case output <- false:
					default:
						logger.Debugf("output <- false failed, ch len %d", len(output))
					}
				}
			}
			timer.Reset(time.Now().Truncate(interval).Add(interval).Sub(time.Now()))
		}
	}
}
