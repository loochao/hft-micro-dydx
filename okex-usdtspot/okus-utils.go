package okex_usdtspot

import (
	"context"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"strings"
	"time"
)

func WatchBalancesFromHttp(
	ctx context.Context,
	api *API,
	interval time.Duration,
	output chan []Balance,
) {
	logger.Debugf("START WatchBalancesFromHttp")
	defer logger.Debugf("EXIT WatchBalancesFromHttp")
	timer := time.NewTimer(interval)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			subCtx, _ := context.WithTimeout(ctx, time.Minute)
			balances, err := api.GetAccounts(subCtx)
			if err != nil {
				logger.Debugf("api.GetAccounts error %v", err)
			} else {
				select {
				case output <- balances:
				default:
					logger.Debugf("output <- balances failed, ch len %d", len(output))
				}
			}
			timer.Reset(time.Now().Truncate(interval).Add(interval).Sub(time.Now()))
		}
	}
}

func ParseDepth5(msg []byte, depth5 *Depth5) (err error) {
	bytesLen := len(msg)
	if bytesLen < 128 {
		return fmt.Errorf("bad msg %s", msg)
	}
	depth5.EventTime, err = time.Parse(okspotTimeLayout, common.UnsafeBytesToString(msg[bytesLen-28:bytesLen-4]))
	if err != nil {
		return fmt.Errorf("time.Parse %s error %v", msg[bytesLen-28:bytesLen-4], err)
	}
	if msg[bytesLen-53] == ':' {
		depth5.Symbol = common.UnsafeBytesToString(msg[bytesLen-51 : bytesLen-43])
	} else if msg[bytesLen-54] == ':' {
		depth5.Symbol = common.UnsafeBytesToString(msg[bytesLen-52 : bytesLen-43])
	} else if msg[bytesLen-55] == ':' {
		depth5.Symbol = common.UnsafeBytesToString(msg[bytesLen-53 : bytesLen-43])
	} else if msg[bytesLen-56] == ':' {
		depth5.Symbol = common.UnsafeBytesToString(msg[bytesLen-54 : bytesLen-43])
	} else {
		return fmt.Errorf("bad msg, can't find symbol %s", msg)
	}
	currentKey := common.JsonKeyAsks
	counter := 0
	offset := 42
	collectStart := offset
	if msg[offset-7] != 'k' && msg[offset-6] != 's' && msg[offset-5] != '"' {
		return fmt.Errorf("bad msg %s", msg)
	}
	for offset < bytesLen-54 {
		switch currentKey {
		case common.JsonKeyBids:
			if msg[offset] == '"' {
				if counter%3 < 2 {
					depth5.Bids[counter/3][counter%3], err = common.ParseFloat(msg[collectStart:offset])
					if err != nil {
						return fmt.Errorf("JsonKeyBids error %v mainLoop %d end %d %s", err, collectStart, offset, msg[collectStart:offset])
					}
				}
				counter += 1
				if counter >= 15 || (msg[offset+1] == ']' && msg[offset+2] == ']') {
					currentKey = common.JsonKeyEventTime
					offset = bytesLen
					//return
				} else if counter%3 == 0 {
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
			if msg[offset] == '"' {
				if counter%3 < 2 {
					depth5.Asks[counter/3][counter%3], err = common.ParseFloat(msg[collectStart:offset])
					if err != nil {
						return fmt.Errorf("JsonKeyAsks error %v mainLoop %d end %d %s", err, collectStart, offset, msg[collectStart:offset])
					}
				}
				counter += 1
				if counter >= 15 || (msg[offset+1] == ']' && msg[offset+2] == ']') {
					currentKey = common.JsonKeyBids
					offset += 14
					collectStart = offset
					counter = 0
				} else if counter%3 == 0 {
					offset += 5
					collectStart = offset
				} else {
					offset += 3
					collectStart = offset
				}
				continue
			}
			break
		}
		offset += 1
	}
	return nil
}

func ParseTicker(msg []byte, ticker *Ticker) (err error) {
	//logger.Debugf("%s", msg)
	counter := 0
	//{"table":"spot/ticker","data":[{"last":"16.486","open_24h":"16.206","best_bid":"16.475","high_24h":"16.619","low_24h":"15.945","open_utc0":"16.173","open_utc8":"16.288","base_volume_24h":"453392.90975651","quote_volume_24h":"7406915.38374752","best_ask":"16.496","instrument_id":"WAVES-USDT","timestamp":"2021-07-07T14:20:36.555Z","best_bid_size":"203.32785582","best_ask_size":"3.08008475","last_qty":"2.31531004"}]}
	bytesLen := len(msg)
	if bytesLen < 128 {
		return fmt.Errorf("bad msg %s", msg)
	}
	currentKey := common.JsonKeyUnknown
	offset := 61
	collectStart := offset
	for offset < bytesLen-10 {
		switch currentKey {
		case common.JsonKeySymbol:
			if msg[offset] == '"' {
				ticker.InstrumentID = common.UnsafeBytesToString(msg[collectStart:offset])
				currentKey = common.JsonKeyUnknown
				counter ++
			}
			break
		case common.JsonKeyBidPrice:
			if msg[offset] == '"' {
				ticker.BestBid, err = common.ParseDecimal(msg[collectStart:offset])
				if err != nil {
					return
				}
				currentKey = common.JsonKeyUnknown
				counter ++
			}
			break
		case common.JsonKeyBidSize:
			if msg[offset] == '"' {
				ticker.BestBidSize, err = common.ParseDecimal(msg[collectStart:offset])
				if err != nil {
					return
				}
				currentKey = common.JsonKeyUnknown
				counter ++
			}
			break
		case common.JsonKeyAskPrice:
			if msg[offset] == '"' {
				ticker.BestAsk, err = common.ParseDecimal(msg[collectStart:offset])
				if err != nil {
					return
				}
				currentKey = common.JsonKeyUnknown
				counter ++
			}
			break
		case common.JsonKeyAskSize:
			if msg[offset] == '"' {
				ticker.BestAskSize, err = common.ParseDecimal(msg[collectStart:offset])
				if err != nil {
					return
				}
				currentKey = common.JsonKeyUnknown
				counter ++
			}
			break
		case common.JsonKeyEventTime:
			if msg[offset] == '"' {
				ticker.Timestamp, err = time.Parse(okspotTimeLayout, common.UnsafeBytesToString(msg[collectStart:offset]))
				if err != nil {
					return
				}
				currentKey = common.JsonKeyUnknown
				counter ++
			}
			break
		case common.JsonKeyUnknown:
			if msg[offset] == '_' {
				if msg[offset+1] == 'b' && msg[offset+4] == '"' {
					currentKey = common.JsonKeyBidPrice
					offset += 7
					collectStart = offset
				} else if msg[offset+1] == 'a' && msg[offset+4] == '"' {
					currentKey = common.JsonKeyAskPrice
					offset += 7
					collectStart = offset
				} else if msg[offset+1] == 'b' && msg[offset+4] == '_' {
					//"best_bid_size":"203.32785582"
					currentKey = common.JsonKeyBidSize
					offset += 12
					collectStart = offset
				} else if msg[offset+1] == 'a' && msg[offset+4] == '_' {
					currentKey = common.JsonKeyAskSize
					offset += 12
					collectStart = offset
				} else if msg[offset+1] == 'i' {
					//"instrument_id":"WAVES-USDT"
					currentKey = common.JsonKeySymbol
					offset += 6
					collectStart = offset
				}
			} else if msg[offset] == 'p' && msg[offset+1] == '"' {
				//"timestamp":"2021-07-07T14:20:36.555Z"
				currentKey = common.JsonKeyEventTime
				offset += 4
				collectStart = offset
			}
		}
		offset += 1
	}
	if counter != 6 {
		return fmt.Errorf("missing fields, msg %s", msg)
	}
	return nil
}

func SystemStatusHttpLoop(
	ctx context.Context, api *API, interval time.Duration,
	output chan bool,
) {
	timer := time.NewTimer(time.Minute)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			subCtx, _ := context.WithTimeout(ctx, time.Minute)
			statuses, err := api.GetStatus(subCtx)
			if err != nil {
				logger.Debugf("api.GetStatus(subCtx) error %v", err)
				if !strings.Contains(err.Error(), "Too Many Requests") {
					select {
					case output <- true:
					default:
						logger.Debugf("output <- false, failed ch len %d", len(output))
					}
				}
			} else {
				ready := true
				for _, s := range statuses {
					if (s.ProductType == "0" || s.ProductType == "1") && s.Status == "1" {
						ready = false
					}
				}
				select {
				case output <- ready:
				default:
					logger.Debugf("output <- ready %v, failed ch len %d", ready, len(output))
				}
			}
			timer.Reset(time.Now().Truncate(interval).Add(interval).Sub(time.Now()))
		}
	}
}

func GetOkOrderLimits(ctx context.Context, api *API, symbols []string) (
	tickSizes map[string]float64, stepSizes map[string]float64, minSizes map[string]float64, err error,
) {
	var instruments []Instrument
	instruments, err = api.GetInstruments(ctx)
	if err != nil {
		return
	}
	tickSizes = make(map[string]float64)
	stepSizes = make(map[string]float64)
	minSizes = make(map[string]float64)
	unmatchedSymbols := make(map[string]bool)
	for _, symbol := range symbols {
		unmatchedSymbols[symbol] = true
	}
	for _, instrument := range instruments {
		if len(instrument.InstrumentId) < 5 {
			continue
		}
		if instrument.InstrumentId[len(instrument.InstrumentId)-5:] != "-USDT" {
			continue
		}
		if _, ok := unmatchedSymbols[instrument.InstrumentId]; ok {
			delete(unmatchedSymbols, instrument.InstrumentId)
			tickSizes[instrument.InstrumentId] = instrument.TickSize
			stepSizes[instrument.InstrumentId] = instrument.SizeIncrement
			minSizes[instrument.InstrumentId] = instrument.MinSize
		}
	}
	if len(unmatchedSymbols) > 0 {
		err = fmt.Errorf("not matched symbols %v", unmatchedSymbols)
	}
	logger.Debugf("SPOT TICK SIZES %v", tickSizes)
	logger.Debugf("SPOT STEP SIZES %v", stepSizes)
	logger.Debugf("SPOT MIN SIZES %v", minSizes)
	return
}
