package okexv5_usdtswap

import (
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"time"
)

func ParseDepth5(msg []byte, depth5 *Depth5) (err error) {
	depth5.ParseTime = time.Now()
	//{"arg":{"channel":"books5","instId":"DOGE-USDT"},"data":[{"asks":[["0.257659","0.000011","0","1"],["0.257744","1380","0","1"],["0.257749","300","0","1"],["0.257769","1942.701948","0","1"],["0.25777","1000","0","1"]],"bids":[["0.257634","949.769316","0","1"],["0.257633","1380","0","1"],["0.257627","20250","0","1"],["0.25762","2929.149403","0","1"],["0.257614","5350","0","1"]],"instId":"DOGE-USDT","ts":"1636741161692"}]}
	msgLen := len(msg)
	if msgLen < 128 {
		return fmt.Errorf("bad msg %s", msg)
	}
	currentKey := common.JsonKeyUnknown
	counter := 0
	offset := 4
	collectStart := offset
	for offset < msgLen-2 {
		switch currentKey {
		case common.JsonKeyBids:
			if msg[offset] == '"' {
				if counter%4 < 2 {
					depth5.Bids[counter/4][counter%4], err = common.ParseFloat(msg[collectStart:offset])
					if err != nil {
						return fmt.Errorf("JsonKeyBids error %v %s", err, msg[collectStart:offset])
					}
				}
				counter += 1
				if msg[offset+1] == ']' && msg[offset+2] == ']' {
					currentKey = common.JsonKeySymbol
					offset += 14
					collectStart = offset
				} else if counter%4 == 0 {
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
				if counter%4 < 2 {
					depth5.Asks[counter/4][counter%4], err = common.ParseFloat(msg[collectStart:offset])
					if err != nil {
						return fmt.Errorf("JsonKeyAsks error %v %s", err, msg[collectStart:offset])
					}
				}
				counter += 1
				if msg[offset+1] == ']' && msg[offset+2] == ']' {
					currentKey = common.JsonKeyBids
					offset += 14
					collectStart = offset
					counter = 0
				} else if counter%4 == 0 {
					offset += 5
					collectStart = offset
				} else {
					offset += 3
					collectStart = offset
				}
				continue
			}
			break
		case common.JsonKeySymbol:
			if msg[offset] == '"' {
				depth5.InstId = common.UnsafeBytesToString(msg[collectStart:offset])
				offset = msgLen - 4
				collectStart = msgLen - 17
				var t int64
				t, err = common.ParseInt(msg[collectStart:offset])
				if err == nil {
					depth5.EventTime = time.Unix(0, t*1000000)
				}
				return
			}
			break
		default:
			if msg[offset] == '"' && msg[offset-4] == 'a' {
				currentKey = common.JsonKeyAsks
				offset += 5
				collectStart = offset
			}
		}
		offset += 1
	}
	return fmt.Errorf("bad end %s", msg)
}

func ParseTicker(msg []byte, ticker *Ticker) (err error) {
	ticker.ParseTime = time.Now()
	//{"arg":{"channel":"tickers","instId":"DOGE-USDT"},"data":[{"instType":"SPOT","instId":"DOGE-USDT","last":"0.254381","lastSz":"600","askPx":"0.254381","askSz":"1400","bidPx":"0.25438","bidSz":"400","open24h":"0.263668","high24h":"0.268614","low24h":"0.248601","sodUtc0":"0.260658","sodUtc8":"0.253989","volCcy24h":"125310776.54685","vol24h":"486148293.462458","ts":"1636737706397"}]}
	bytesLen := len(msg)
	if bytesLen < 128 {
		return fmt.Errorf("bad msg %s", msg)
	}
	currentKey := common.JsonKeyUnknown
	offset := 5
	collectStart := offset
	for offset < bytesLen {
		switch currentKey {
		case common.JsonKeySymbol:
			if msg[offset] == '"' {
				ticker.InstId = common.UnsafeBytesToString(msg[collectStart:offset])
				currentKey = common.JsonKeyUnknown
			}
			break
		case common.JsonKeyBidPrice:
			if msg[offset] == '"' {
				ticker.BidPx, err = common.ParseDecimal(msg[collectStart:offset])
				if err != nil {
					return
				}
				currentKey = common.JsonKeyBidSize
				offset += 11
				collectStart = offset
			}
			break
		case common.JsonKeyBidSize:
			if msg[offset] == '"' {
				ticker.BidSz, err = common.ParseDecimal(msg[collectStart:offset])
				if err != nil {
					return
				}
				offset = bytesLen - 4
				collectStart = bytesLen - 17
				var t int64
				t, err = common.ParseInt(msg[collectStart:offset])
				if err == nil {
					ticker.EventTime = time.Unix(0, t*1000000)
				}
				return
			}
			break
		case common.JsonKeyAskPrice:
			if msg[offset] == '"' {
				ticker.AskPx, err = common.ParseDecimal(msg[collectStart:offset])
				if err != nil {
					return
				}
				currentKey = common.JsonKeyAskSize
				offset += 11
				collectStart = offset
			}
			break
		case common.JsonKeyAskSize:
			if msg[offset] == '"' {
				ticker.AskSz, err = common.ParseDecimal(msg[collectStart:offset])
				if err != nil {
					return
				}
				currentKey = common.JsonKeyBidPrice
				offset += 11
				collectStart = offset
			}
			break
		case common.JsonKeyUnknown:
			if msg[offset] == '"' {
				if msg[offset-1] == 'd' && msg[offset-6] == 'i' {
					currentKey = common.JsonKeySymbol
					offset += 3
					collectStart = offset
				} else if msg[offset-1] == 'x' && msg[offset-5] == 'a' {
					currentKey = common.JsonKeyAskPrice
					offset += 3
					collectStart = offset
				}
			}
		}
		offset += 1
	}
	return fmt.Errorf("msg not end in ts %s", msg)
}

func ParseTrade(msg []byte, trade *Trade) (err error) {
	//{"arg":{"channel":"trades","instId":"DOGE-USDT"},"data":[{"instId":"DOGE-USDT","tradeId":"106645495","px":"0.256222","sz":"14.19554","side":"sell","ts":"1636778780284"}]}
	bytesLen := len(msg)
	if bytesLen < 64 {
		return fmt.Errorf("too short %s", msg)
	}
	currentKey := common.JsonKeyUnknown
	offset := 5
	collectStart := offset
	hasSymbol := false
	for offset < bytesLen {
		switch currentKey {
		case common.JsonKeySymbol:
			if msg[offset] == '"' {
				trade.InstId = common.UnsafeBytesToString(msg[collectStart:offset])
				hasSymbol = true
				currentKey = common.JsonKeyUnknown
				offset += 20
			}
			break
		case common.JsonKeyPrice:
			if msg[offset] == '"' {
				trade.Px, err = common.ParseDecimal(msg[collectStart:offset])
				if err != nil {
					return
				}
				currentKey = common.JsonKeySize
				offset += 8
				collectStart = offset
			}
			break
		case common.JsonKeySize:
			if msg[offset] == '"' {
				trade.Sz, err = common.ParseDecimal(msg[collectStart:offset])
				if err != nil {
					return
				}
				offset += 10
				if offset < bytesLen {
					if msg[offset] == 's' {
						trade.Side = "sell"
					} else {
						trade.Side = "buy"
					}
				}
				offset = bytesLen - 4
				collectStart = bytesLen - 17
				var t int64
				t, err = common.ParseInt(msg[collectStart:offset])
				if err == nil {
					trade.TS = time.Unix(0, t*1000000)
				}
				return
			}
			break
		case common.JsonKeyUnknown:
			if msg[offset] == '"' {
				if msg[offset-1] == 'd' && msg[offset-6] == 'i' {
					if hasSymbol {
						offset += 20
					} else {
						currentKey = common.JsonKeySymbol
						offset += 3
						collectStart = offset
					}
				} else if msg[offset-1] == 'x' && msg[offset-2] == 'p' {
					currentKey = common.JsonKeyPrice
					offset += 3
					collectStart = offset
				}
			}
		}
		offset += 1
	}
	return fmt.Errorf("msg not end in ts %s", msg)
}

func ParseFundingRate(msg []byte, fundingRate *FundingRate) (err error) {
	//{"arg":{"channel":"funding-rate","instId":"BTC-USDT-SWAP"},"data":[{"fundingRate":"-0.00003062","fundingTime":"1636848000000","instId":"BTC-USDT-SWAP","instType":"SWAP","nextFundingRate":"-0.00013"}]}
	bytesLen := len(msg)
	if bytesLen < 64 {
		return fmt.Errorf("too short %s", msg)
	}
	currentKey := common.JsonKeyUnknown
	offset := 5
	collectStart := offset
	var t int64
	for offset < bytesLen {
		switch currentKey {
		case common.JsonKeySymbol:
			if msg[offset] == '"' {
				fundingRate.InstId = common.UnsafeBytesToString(msg[collectStart:offset])
				if offset > 64 {
					offset += 39
					collectStart = offset
					currentKey = common.JsonKeyNextFundingRate
				} else {
					offset += 23
					currentKey = common.JsonKeyUnknown
				}
			}
			break
		case common.JsonKeyNextFundingRate:
			if msg[offset] == '"' {
				fundingRate.NextFundingRate, err = common.ParseDecimal(msg[collectStart:offset])
				if err != nil {
					return
				}
				return
			}
			break
		case common.JsonKeyFundingRate:
			if msg[offset] == '"' {
				fundingRate.FundingRate, err = common.ParseDecimal(msg[collectStart:offset])
				if err != nil {
					return
				}
				currentKey = common.JsonKeyNextFundingTime
				offset += 17
				collectStart = offset
				offset += 12
			}
			break
		case common.JsonKeyNextFundingTime:
			if msg[offset] == '"' {
				t, err = common.ParseInt(msg[collectStart:offset])
				if err != nil {
					return
				}
				fundingRate.FundingTime = time.Unix(0, t*1000000)
				currentKey = common.JsonKeySymbol
				offset += 12
				collectStart = offset
			}
			break
		case common.JsonKeyUnknown:
			if msg[offset] == '"' {
				if msg[offset-1] == 'd' && msg[offset-6] == 'i' {
					currentKey = common.JsonKeySymbol
					offset += 3
					collectStart = offset
				} else if msg[offset-1] == 'e' && msg[offset-4] == 'R' && msg[offset-11] == 'f' {
					currentKey = common.JsonKeyFundingRate
					offset += 3
					//logger.Debugf("fr %s", msg[offset:])
					collectStart = offset
				}
			}
		}
		offset += 1
	}
	return fmt.Errorf("msg not end in nfr %s", msg)
}
