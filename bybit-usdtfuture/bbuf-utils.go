package bybit_usdtfuture

import (
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"time"
)

//{"topic":"orderBookL2_25.ETHUSDT","type":"snapshot","data":{"order_book":[{"price":"1956.80","symbol":"ETHUSDT","id":"19568000","side":"Buy","size":62.109997},{"price":"1956.85","symbol":"ETHUSDT","id":"19568500","side":"Buy","size":21.6},{"price":"1956.90","symbol":"ETHUSDT","id":"19569000","side":"Buy","size":71.53},{"price":"1956.95","symbol":"ETHUSDT","id":"19569500","side":"Buy","size":91.22},{"price":"1957.00","symbol":"ETHUSDT","id":"19570000","side":"Buy","size":66.88},{"price":"1957.05","symbol":"ETHUSDT","id":"19570500","side":"Buy","size":20.65},{"price":"1957.10","symbol":"ETHUSDT","id":"19571000","side":"Buy","size":36.12},{"price":"1957.15","symbol":"ETHUSDT","id":"19571500","side":"Buy","size":46.39},{"price":"1957.20","symbol":"ETHUSDT","id":"19572000","side":"Buy","size":16.54},{"price":"1957.25","symbol":"ETHUSDT","id":"19572500","side":"Buy","size":33.29},{"price":"1957.30","symbol":"ETHUSDT","id":"19573000","side":"Buy","size":72.62},{"price":"1957.35","symbol":"ETHUSDT","id":"19573500","side":"Buy","size":77.35},{"price":"1957.40","symbol":"ETHUSDT","id":"19574000","side":"Buy","size":63.510002},{"price":"1957.45","symbol":"ETHUSDT","id":"19574500","side":"Buy","size":42.300003},{"price":"1957.50","symbol":"ETHUSDT","id":"19575000","side":"Buy","size":70.56},{"price":"1957.55","symbol":"ETHUSDT","id":"19575500","side":"Buy","size":73.65},{"price":"1957.60","symbol":"ETHUSDT","id":"19576000","side":"Buy","size":123.07},{"price":"1957.65","symbol":"ETHUSDT","id":"19576500","side":"Buy","size":126.479996},{"price":"1957.70","symbol":"ETHUSDT","id":"19577000","side":"Buy","size":70.85},{"price":"1957.75","symbol":"ETHUSDT","id":"19577500","side":"Buy","size":84.67},{"price":"1957.80","symbol":"ETHUSDT","id":"19578000","side":"Buy","size":3.9},{"price":"1957.85","symbol":"ETHUSDT","id":"19578500","side":"Buy","size":6.16},{"price":"1957.90","symbol":"ETHUSDT","id":"19579000","side":"Buy","size":9.94},{"price":"1958.00","symbol":"ETHUSDT","id":"19580000","side":"Buy","size":8.1},{"price":"1958.40","symbol":"ETHUSDT","id":"19584000","side":"Buy","size":215.75},{"price":"1958.45","symbol":"ETHUSDT","id":"19584500","side":"Sell","size":720.13995},{"price":"1958.50","symbol":"ETHUSDT","id":"19585000","side":"Sell","size":129.13},{"price":"1958.55","symbol":"ETHUSDT","id":"19585500","side":"Sell","size":113.799995},{"price":"1958.60","symbol":"ETHUSDT","id":"19586000","side":"Sell","size":119.3},{"price":"1958.65","symbol":"ETHUSDT","id":"19586500","side":"Sell","size":58.92},{"price":"1958.70","symbol":"ETHUSDT","id":"19587000","side":"Sell","size":65.340004},{"price":"1958.75","symbol":"ETHUSDT","id":"19587500","side":"Sell","size":67.78},{"price":"1958.80","symbol":"ETHUSDT","id":"19588000","side":"Sell","size":21.44},{"price":"1958.85","symbol":"ETHUSDT","id":"19588500","side":"Sell","size":28.17},{"price":"1958.90","symbol":"ETHUSDT","id":"19589000","side":"Sell","size":73.06},{"price":"1958.95","symbol":"ETHUSDT","id":"19589500","side":"Sell","size":137.38},{"price":"1959.00","symbol":"ETHUSDT","id":"19590000","side":"Sell","size":51.730003},{"price":"1959.05","symbol":"ETHUSDT","id":"19590500","side":"Sell","size":14.190001},{"price":"1959.10","symbol":"ETHUSDT","id":"19591000","side":"Sell","size":22.94},{"price":"1959.15","symbol":"ETHUSDT","id":"19591500","side":"Sell","size":10.68},{"price":"1959.20","symbol":"ETHUSDT","id":"19592000","side":"Sell","size":94.270004},{"price":"1959.25","symbol":"ETHUSDT","id":"19592500","side":"Sell","size":78.06},{"price":"1959.30","symbol":"ETHUSDT","id":"19593000","side":"Sell","size":18.03},{"price":"1959.35","symbol":"ETHUSDT","id":"19593500","side":"Sell","size":8.24},{"price":"1959.40","symbol":"ETHUSDT","id":"19594000","side":"Sell","size":46.03},{"price":"1959.45","symbol":"ETHUSDT","id":"19594500","side":"Sell","size":59.420002},{"price":"1959.50","symbol":"ETHUSDT","id":"19595000","side":"Sell","size":67.1},{"price":"1959.55","symbol":"ETHUSDT","id":"19595500","side":"Sell","size":74.07},{"price":"1959.60","symbol":"ETHUSDT","id":"19596000","side":"Sell","size":18.4},{"price":"1959.65","symbol":"ETHUSDT","id":"19596500","side":"Sell","size":66.55}]},"cross_seq":"3501312250","timestamp_e6":"1626591301507553"}

func UpdateOrderBook(msg []byte, orderBook *OrderBook) (err error) {
	currentKey := common.JsonKeyUnknown
	offset := 0
	collectStart := 0
	msgLen := len(msg)
	symbolLen := 0
	lastSide := 0
	timestamp := int64(0)
	bid := [2]float64{}
	ask := [2]float64{}
	var lastSize, lastPrice float64
	if msg[32] == '"' {
		orderBook.Symbol = common.UnsafeBytesToString(msg[25:32])
		symbolLen = 7
		offset = 32
	} else if msg[31] == '"' {
		orderBook.Symbol = common.UnsafeBytesToString(msg[25:31])
		symbolLen = 6
		offset = 31
	} else if msg[33] == '"' {
		orderBook.Symbol = common.UnsafeBytesToString(msg[25:33])
		symbolLen = 8
		offset = 33
	} else if msg[34] == '"' {
		orderBook.Symbol = common.UnsafeBytesToString(msg[25:34])
		symbolLen = 9
		offset = 34
	} else if msg[35] == '"' {
		orderBook.Symbol = common.UnsafeBytesToString(msg[25:35])
		symbolLen = 10
		offset = 35
	} else {
		return fmt.Errorf("symbol not found for msg %s", msg)
	}
	offset += 10
	if msg[offset] == 's' {
		orderBook.Bids = orderBook.Bids[:0]
		orderBook.Asks = orderBook.Asks[:0]
		offset += 42
		collectStart = offset
		currentKey = common.JsonKeyPrice
		for offset < msgLen-31 {
			switch currentKey {
			case common.JsonKeyPrice:
				if msg[offset] == '"' {
					lastPrice, err = common.ParseDecimal(msg[collectStart:offset])
					if err != nil {
						return
					}
					offset += 23 + symbolLen
					currentKey = common.JsonKeySide
					continue
				}
			case common.JsonKeySide:
				if msg[offset] == 's' {
					offset += 7
					if msg[offset] == 'B' {
						lastSide = 0
						currentKey = common.JsonKeySize
						offset += 12
						collectStart = offset
						continue
					} else if msg[offset] == 'S' {
						lastSide = 1
						currentKey = common.JsonKeySize
						offset += 13
						collectStart = offset
						continue
					} else {
						return fmt.Errorf("bad msg, bad side %s", msg[offset:])
					}
				}
			case common.JsonKeySize:
				if msg[offset] == '}' {
					lastSize, err = common.ParseDecimal(msg[collectStart:offset])
					if err != nil {
						return
					}
					if lastSide == 0 {
						bid[0] = lastPrice
						bid[1] = lastSize
						orderBook.Bids = orderBook.Bids.Update(bid)
						//logger.Debugf("snapshot bid %f", bid)
					} else {
						ask[0] = lastPrice
						ask[1] = lastSize
						orderBook.Asks = orderBook.Asks.Update(ask)
						//logger.Debugf("snapshot ask %f", ask)
					}
					offset += 1
					if msg[offset] == ']' {
						offset += 19
						currentKey = common.JsonKeyEventTime
						continue
					} else if msg[offset] == ',' {
						currentKey = common.JsonKeyPrice
						offset += 11
						collectStart = offset
						continue
					} else {
						return fmt.Errorf("bad msg %s", msg[offset:])
					}
				}
			case common.JsonKeyEventTime:
				if msg[offset] == 't' && msg[offset-1] == '"' {
					offset += 15
					timestamp, err = common.ParseInt(msg[offset : offset+16])
					if err != nil {
						return
					}
					orderBook.EventTime = time.Unix(0, timestamp*1000)
					orderBook.ParseTime = time.Now()
					return
				}
			}
			offset++
		}
	} else if msg[offset] == 'd' {
		//{"topic":"orderBookL2_25.XTZUSDT","type":"delta","data":{"delete":[],"update":[{"price":"2.426","symbol":"XTZUSDT","id":"24260","side":"Buy","size":2902.7},{"price":"2.431","symbol":"XTZUSDT","id":"24310","side":"Sell","size":5981}],"insert":[]},"cross_seq":"1168950555","timestamp_e6":"1626652805434387"}
		offset += 25
		if msg[offset] == '{' {
			currentKey = common.JsonKeyPrice
			offset += 10
			collectStart = offset
		deleteLoop:
			for offset < msgLen-31 {
				switch currentKey {
				case common.JsonKeyPrice:
					if msg[offset] == '"' {
						lastPrice, err = common.ParseDecimal(msg[collectStart:offset])
						if err != nil {
							return
						}
						offset += 23 + symbolLen
						currentKey = common.JsonKeySide
						continue
					}
				case common.JsonKeySide:
					if msg[offset] == 's' {
						offset += 7
						if msg[offset] == 'B' {
							bid[0] = lastPrice
							bid[1] = 0
							orderBook.Bids = orderBook.Bids.Update(bid)
							offset += 5
							//logger.Debugf("%s", msg)
							//logger.Debugf("delete bid %f", bid)
							if msg[offset] == ']' {
								offset += 12
								break deleteLoop
							} else if msg[offset] == ',' {
								offset += 11
								collectStart = offset
								currentKey = common.JsonKeyPrice
								continue
							} else {
								return fmt.Errorf("bad msg @ delete %s", msg[offset:])
							}
						} else if msg[offset] == 'S' {
							ask[0] = lastPrice
							ask[1] = 0
							orderBook.Asks = orderBook.Asks.Update(ask)
							offset += 6
							//logger.Debugf("delete ask %f", ask)
							if msg[offset] == ']' {
								offset += 12
								break deleteLoop
							} else if msg[offset] == ',' {
								offset += 11
								collectStart = offset
								currentKey = common.JsonKeyPrice
								continue
							} else {
								return fmt.Errorf("bad msg @ delete %s", msg[offset:])
							}
						} else {
							return fmt.Errorf("bad msg, bad side %s", msg[offset:])
						}
					}
				}
				offset++
			}
		} else if msg[offset] == ']' {
			offset += 12
		} else {
			return fmt.Errorf("bad msg @ delta delete %s", msg[offset:])
		}


		//update
		if msg[offset] == '{' {
			currentKey = common.JsonKeyPrice
			offset += 10
			collectStart = offset
		updateLoop:
			for offset < msgLen-31 {
				switch currentKey {
				case common.JsonKeyPrice:
					if msg[offset] == '"' {
						lastPrice, err = common.ParseDecimal(msg[collectStart:offset])
						if err != nil {
							return
						}
						offset += 23 + symbolLen
						currentKey = common.JsonKeySide
						continue
					}
				case common.JsonKeySide:
					if msg[offset] == 's' {
						offset += 7
						if msg[offset] == 'B' {
							lastSide = 0
							currentKey = common.JsonKeySize
							offset += 12
							collectStart = offset
							continue
						} else if msg[offset] == 'S' {
							lastSide = 1
							currentKey = common.JsonKeySize
							offset += 13
							collectStart = offset
							continue
						} else {
							return fmt.Errorf("bad msg, bad side %s", msg[offset:])
						}
					}
				case common.JsonKeySize:
					if msg[offset] == '}' {
						lastSize, err = common.ParseDecimal(msg[collectStart:offset])
						if err != nil {
							return
						}
						if lastSide == 0 {
							bid[0] = lastPrice
							bid[1] = lastSize
							orderBook.Bids = orderBook.Bids.Update(bid)
							//logger.Debugf("update bid %f", bid)
						} else {
							ask[0] = lastPrice
							ask[1] = lastSize
							orderBook.Asks = orderBook.Asks.Update(ask)
							//logger.Debugf("update ask %f", ask)
						}
						offset += 1
						if msg[offset] == ']' {
							offset += 12
							break updateLoop
						} else if msg[offset] == ',' {
							currentKey = common.JsonKeyPrice
							offset += 11
							collectStart = offset
							continue
						} else {
							return fmt.Errorf("bad msg %s", msg[offset:])
						}
					}
				}
				offset++
			}
		} else if msg[offset] == ']' {
			offset += 12
		} else {
			return fmt.Errorf("bad msg @ delta update %s", msg[offset:])
		}

		// insert loop
		if msg[offset] == '{' {
			currentKey = common.JsonKeyPrice
			offset += 10
			collectStart = offset
			for offset < msgLen-31 {
				switch currentKey {
				case common.JsonKeyPrice:
					if msg[offset] == '"' {
						lastPrice, err = common.ParseDecimal(msg[collectStart:offset])
						if err != nil {
							return
						}
						offset += 23 + symbolLen
						currentKey = common.JsonKeySide
						continue
					}
				case common.JsonKeySide:
					if msg[offset] == 's' {
						offset += 7
						if msg[offset] == 'B' {
							lastSide = 0
							currentKey = common.JsonKeySize
							offset += 12
							collectStart = offset
							continue
						} else if msg[offset] == 'S' {
							lastSide = 1
							currentKey = common.JsonKeySize
							offset += 13
							collectStart = offset
							continue
						} else {
							return fmt.Errorf("bad msg, bad side %s", msg[offset:])
						}
					}
				case common.JsonKeySize:
					if msg[offset] == '}' {
						lastSize, err = common.ParseDecimal(msg[collectStart:offset])
						if err != nil {
							return
						}
						if lastSide == 0 {
							bid[0] = lastPrice
							bid[1] = lastSize
							orderBook.Bids = orderBook.Bids.Update(bid)
							//logger.Debugf("insert bid %f", bid)
						} else {
							ask[0] = lastPrice
							ask[1] = lastSize
							orderBook.Asks = orderBook.Asks.Update(ask)
							//logger.Debugf("insert ask %f", ask)
						}
						offset += 1
						if msg[offset] == ']' {
							offset += 19
							currentKey = common.JsonKeyEventTime
							continue
						} else if msg[offset] == ',' {
							currentKey = common.JsonKeyPrice
							offset += 11
							collectStart = offset
							continue
						} else {
							return fmt.Errorf("bad msg %s", msg[offset:])
						}
					}
				case common.JsonKeyEventTime:
					if msg[offset] == 't' && msg[offset-1] == '"' {
						offset += 15
						timestamp, err = common.ParseInt(msg[offset : offset+16])
						if err != nil {
							return
						}
						orderBook.EventTime = time.Unix(0, timestamp*1000)
						orderBook.ParseTime = time.Now()
						return
					}
				}
				offset++
			}
		} else if msg[offset] == ']' {
			offset += 19
		} else {
			return fmt.Errorf("bad msg @ delta insert %s", msg[offset:])
		}
		currentKey = common.JsonKeyEventTime
		for offset < msgLen-31 {
			if msg[offset] == 't' && msg[offset-1] == '"' {
				offset += 15
				timestamp, err = common.ParseInt(msg[offset : offset+16])
				//logger.Debugf("%s", msg[offset:offset+16])
				if err != nil {
					return
				}
				orderBook.EventTime = time.Unix(0, timestamp*1000)
				orderBook.ParseTime = time.Now()
				return
			}
			offset++
		}
	}
	return fmt.Errorf("bad msg %s", msg)
}
