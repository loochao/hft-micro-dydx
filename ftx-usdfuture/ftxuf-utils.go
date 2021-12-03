package ftx_usdfuture

import (
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"math"
	"time"
)

func ParseTicker(msg []byte, ticker *Ticker) (err error) {
	ticker.ParseTime = time.Now()
	//{"channel": "ticker", "market": "DOGE-PERP", "type": "update", "data": {"bid": 0.278362, "ask": 0.2784135, "bidSize": 107.0, "askSize": 5600.0, "last": 0.2783695, "time": 1624183024.08771}} 189
	collectEnd := 33
	collectStart := collectEnd
	bytesLen := len(msg)

	currentKey := common.JsonKeySymbol
	for collectEnd < bytesLen-2 {
		switch currentKey {
		case common.JsonKeySymbol:
			if msg[collectEnd] == '"' {
				ticker.Symbol = common.UnsafeBytesToString(msg[collectStart:collectEnd])
				currentKey = common.JsonKeyBidPrice
				collectEnd += 37
				collectStart = collectEnd
				continue
			}
			break
		case common.JsonKeyBidPrice:
			if msg[collectEnd] == ',' {
				ticker.Bid, err = common.ParseDecimal(msg[collectStart:collectEnd])
				if err != nil {
					return fmt.Errorf("JsonKeyBidPrice error %v start %d end %d %s", err, collectStart, collectEnd, msg[collectStart:collectEnd])
				}
				currentKey = common.JsonKeyAskPrice
				collectEnd += 9
				collectStart = collectEnd
				continue
			}
			break
		case common.JsonKeyAskPrice:
			if msg[collectEnd] == ',' {
				ticker.Ask, err = common.ParseDecimal(msg[collectStart:collectEnd])
				if err != nil {
					return fmt.Errorf("JsonKeyAskPrice error %v start %d end %d %s", err, collectStart, collectEnd, msg[collectStart:collectEnd])
				}
				currentKey = common.JsonKeyBidSize
				collectEnd += 13
				collectStart = collectEnd
				continue
			}
			break
		case common.JsonKeyBidSize:
			if msg[collectEnd] == ',' {
				ticker.BidSize, err = common.ParseDecimal(msg[collectStart:collectEnd])
				if err != nil {
					return fmt.Errorf("JsonKeyAskPrice error %v start %d end %d %s", err, collectStart, collectEnd, msg[collectStart:collectEnd])
				}
				currentKey = common.JsonKeyAskSize
				collectEnd += 13
				collectStart = collectEnd
				continue
			}
			break
		case common.JsonKeyAskSize:
			if msg[collectEnd] == ',' {
				ticker.AskSize, err = common.ParseDecimal(msg[collectStart:collectEnd])
				if err != nil {
					return fmt.Errorf("JsonKeyAskPrice error %v start %d end %d %s", err, collectStart, collectEnd, msg[collectStart:collectEnd])
				}
				collectEnd = bytesLen - 24
				currentKey = common.JsonKeyEventTime
			}
			break
		case common.JsonKeyEventTime:
			if msg[collectEnd] == ':' {
				var t float64
				t, err = common.ParseDecimal(msg[collectEnd+2 : bytesLen-2])
				if err != nil {
					return
				}
				ticker.EventTime = time.Unix(int64(t), int64((t-math.Floor(t))*1e9))
				return
			}
		}
		collectEnd += 1
	}
	return
}

func ParseOrderBook(msg []byte, orderBook *OrderBook) error {
	//{"channel": "orderbook", "market": "DOGE-PERP", "type": "partial", "data": {"time": 1637091435.6559627, "checksum": 244155079, "bids": [[0.2406795, 86.0], [0.240657, 10379.0], [0.240648, 9663.0], [0.2406335, 58500.0], [0.2406325, 10385.0], [0.2405975, 24175.0], [0.240587, 354.0], [0.240558, 12166.0], [0.2405245, 156448.0], [0.2405235, 15000.0], [0.240519, 46125.0], [0.2405105, 10775.0], [0.240508, 33322.0], [0.240501, 83548.0], [0.240499, 81714.0], [0.24049, 69.0], [0.2404855, 4150.0], [0.240485, 16200.0], [0.240479, 119379.0], [0.240475, 130.0], [0.240474, 13753.0], [0.2404605, 11312.0], [0.2404515, 93705.0], [0.2404385, 19534.0], [0.240437, 1000.0], [0.240436, 78331.0], [0.240435, 73544.0], [0.2404345, 46027.0], [0.240434, 249143.0], [0.2404295, 3000.0], [0.240419, 5503.0], [0.2404185, 19535.0], [0.2404065, 1043.0], [0.240404, 5408.0], [0.2404025, 1044.0], [0.2404005, 8253.0], [0.2404, 5504.0], [0.240399, 8679.0], [0.2403975, 13023.0], [0.2403895, 1000.0], [0.2403865, 8254.0], [0.240386, 8254.0], [0.2403845, 11004.0], [0.2403825, 4000.0], [0.2403765, 24955.0], [0.240376, 61600.0], [0.240367, 86026.0], [0.240366, 139646.0], [0.240351, 11005.0], [0.2403485, 8255.0], [0.240346, 76567.0], [0.240345, 1.0], [0.2403445, 8114.0], [0.240341, 4154.0], [0.2403375, 5505.0], [0.2403365, 20783.0], [0.240333, 47606.0], [0.24033, 5505.0], [0.2403295, 1.0], [0.240326, 1039.0], [0.240325, 8680.0], [0.240321, 5506.0], [0.240319, 8115.0], [0.240316, 1039.0], [0.240311, 4324.0], [0.2403025, 40820.0], [0.2402955, 1046.0], [0.240292, 6.0], [0.2402915, 1.0], [0.2402835, 1040.0], [0.240282, 1.0], [0.2402785, 6551.0], [0.240264, 1000.0], [0.240263, 65171.0], [0.2402625, 1043.0], [0.240262, 24962.0], [0.240252, 277.0], [0.2402515, 8117.0], [0.240244, 6234.0], [0.2402355, 5508.0], [0.240233, 5508.0], [0.2402215, 8260.0], [0.2402175, 1.0], [0.2402065, 167.0], [0.240205, 13764.0], [0.2401965, 4155.0], [0.240195, 10852.0], [0.2401945, 1040.0], [0.240192, 1000.0], [0.2401875, 107701.0], [0.2401845, 1040.0], [0.2401665, 12125.0], [0.2401655, 10391.0], [0.24016, 7294.0], [0.240158, 1.0], [0.2401545, 1040.0], [0.2401475, 6234.0], [0.2401465, 478508.0], [0.240144, 8677.0], [0.240143, 130752.0]], "asks": [[0.24068, 83907.0], [0.240684, 20753.0], [0.2406845, 2.0], [0.240696, 1000.0], [0.240729, 1.0], [0.2407475, 5496.0], [0.240768, 1000.0], [0.240769, 10386.0], [0.240773, 6232.0], [0.240791, 10380.0], [0.240804, 130.0], [0.240807, 2878.0], [0.2408075, 15000.0], [0.2408155, 1046.0], [0.240821, 3093.0], [0.2408235, 157620.0], [0.240824, 119380.0], [0.2408255, 4324.0], [0.240827, 8239.0], [0.2408355, 24914.0], [0.240836, 33432.0], [0.2408385, 1043.0], [0.24084, 1000.0], [0.2408405, 15134.0], [0.240842, 1039.0], [0.240845, 164173.0], [0.240852, 25951.0], [0.240857, 78331.0], [0.240858, 16200.0], [0.240866, 8309.0], [0.2408745, 48344.0], [0.240875, 623.0], [0.2408815, 4153.0], [0.2408825, 19576.0], [0.240894, 1000.0], [0.2408985, 1044.0], [0.2409025, 6227.0], [0.24091, 672241.0], [0.240911, 5492.0], [0.2409285, 1036.0], [0.24093, 63630.0], [0.240931, 65171.0], [0.240938, 10888.0], [0.240943, 1000.0], [0.2409455, 1046.0], [0.2409475, 5396.0], [0.2409505, 189.0], [0.240953, 8674.0], [0.240954, 76374.0], [0.2409625, 277.0], [0.240963, 9.0], [0.2409735, 6530.0], [0.24098, 623.0], [0.240981, 8234.0], [0.2409825, 1043.0], [0.2409835, 99041.0], [0.2409925, 8676.0], [0.240993, 29822.0], [0.241002, 19517.0], [0.2410085, 20700.0], [0.2410125, 47606.0], [0.2410185, 5490.0], [0.2410225, 1044.0], [0.241026, 10840.0], [0.2410285, 35394.0], [0.241033, 8674.0], [0.2410335, 5394.0], [0.2410345, 4150.0], [0.2410385, 7634.0], [0.2410415, 5489.0], [0.24105, 10835.0], [0.2410575, 1036.0], [0.2410675, 2165.0], [0.2410755, 1046.0], [0.241081, 10976.0], [0.241085, 623.0], [0.241088, 130752.0], [0.241095, 277.0], [0.241105, 1039.0], [0.241109, 10823.0], [0.2411125, 10970.0], [0.241115, 1039.0], [0.2411175, 5487.0], [0.2411185, 139646.0], [0.2411265, 1043.0], [0.2411305, 6767.0], [0.241132, 16461.0], [0.241139, 7631.0], [0.2411415, 8228.0], [0.2411465, 1044.0], [0.241147, 8673.0], [0.2411485, 8228.0], [0.2411645, 19504.0], [0.241165, 8227.0], [0.2411725, 5486.0], [0.24119, 5486.0], [0.241192, 8707.0], [0.2411945, 5486.0], [0.2412055, 1046.0], [0.241213, 8226.0]], "action": "partial"}}
	//{"channel": "orderbook", "market": "DOGE-PERP", "type": "update", "data": {"time": 1637091435.6967268, "checksum": 239280271, "bids": [[0.2404605, 8312.0]], "asks": [[0.2407255, 4155.0], [0.2407605, 3000.0], [0.240773, 0.0], [0.241213, 0.0]], "action": "update"}}
	//{"channel": "orderbook", "market": "DOGE-PERP", "type": "update", "data": {"time": 1637091437.7401257, "checksum": 408629872, "bids": [[0.240419, 5503.0]], "asks": [], "action": "update"}}
	//{"channel": "orderbook", "market": "DOGE-PERP", "type": "update", "data": {"time": 1637091437.8839934, "checksum": 3493437894, "bids": [], "asks": [[0.2408395, 0.0], [0.2411415, 8228.0]], "action": "update"}}

	orderBook.ParseTime = time.Now()
	offset := 0

	if msg[45] == ',' {
		offset = 56
		orderBook.Market = common.UnsafeBytesToString(msg[36:44])
	} else if msg[46] == ',' {
		offset = 57
		orderBook.Market = common.UnsafeBytesToString(msg[36:45])
	} else if msg[44] == ',' {
		offset = 55
		orderBook.Market = common.UnsafeBytesToString(msg[36:43])
	} else if msg[47] == ',' {
		offset = 58
		orderBook.Market = common.UnsafeBytesToString(msg[36:46])
	} else if msg[48] == ',' {
		offset = 59
		orderBook.Market = common.UnsafeBytesToString(msg[36:47])
	} else if msg[49] == ',' {
		offset = 60
		orderBook.Market = common.UnsafeBytesToString(msg[36:48])
	} else if msg[50] == ',' {
		offset = 61
		orderBook.Market = common.UnsafeBytesToString(msg[36:49])
	} else {
		return fmt.Errorf("market not found for %s", msg)
	}

	collectStart := offset
	if msg[offset] == 'p' {
		orderBook.Bids = orderBook.Bids[:0]
		orderBook.Asks = orderBook.Asks[:0]
		orderBook.hasPartial = true
		offset += 41
		collectStart += 27
	} else if msg[offset] == 'u' {
		offset += 40
		collectStart += 26
	} else {
		return fmt.Errorf("bad msg type for %s", msg)
	}
	var err error
	bytesLen := len(msg)
	currentKey := common.JsonKeyEventTime
	var t float64
	var bid, ask [2]float64
	for offset < bytesLen-2 {
		switch currentKey {
		case common.JsonKeyEventTime:
			if msg[offset] == ',' {
				t, err = common.ParseFloat(msg[collectStart:offset])
				if err != nil {
					return fmt.Errorf("time common.ParseDecimal error %v msg %s", err, msg[collectStart:offset])
				}
				orderBook.Time = time.Unix(0, int64(t*1000000000))
				//从checksum的第4位开始
				offset += 18
				currentKey = common.JsonKeyID
				continue
			}
			break
		case common.JsonKeyID:
			if msg[offset] == ',' {
				offset += 11
				if msg[offset] == '[' {
					currentKey = common.JsonKeyBidPrice
					offset += 1
					collectStart = offset
				} else if msg[offset] == ']' {
					//empty bid
					offset += 12
					if msg[offset] == '[' {
						currentKey = common.JsonKeyAskPrice
						offset += 1
						collectStart = offset
					} else if msg[offset] == ']' {
						//empty ask
						return nil
					} else {
						return fmt.Errorf("bad ask location for %s", msg)
					}
				} else {
					return fmt.Errorf("bad bid location for %s", msg)
				}
			}
			break
		case common.JsonKeyBidPrice:
			if msg[offset] == ',' {
				bid[0], err = common.ParseDecimal(msg[collectStart:offset])
				if err != nil {
					return fmt.Errorf("bidPrice common.ParseDecimal error %v msg %s", err, msg[collectStart:offset])
				}
				offset += 2
				collectStart = offset
				currentKey = common.JsonKeyBidSize
				continue
			}
			break
		case common.JsonKeyBidSize:
			if msg[offset] == ']' {
				bid[1], err = common.ParseDecimal(msg[collectStart:offset])
				if err != nil {
					return fmt.Errorf("bidSize common.ParseDecimal error %v msg %s", err, msg[collectStart:offset])
				}
				orderBook.Bids = orderBook.Bids.Update(bid)
				offset += 1
				if msg[offset] == ',' {
					offset += 3
					collectStart = offset
					currentKey = common.JsonKeyBidPrice
					continue
				} else if msg[offset] == ']' {
					//empty bid
					offset += 12
					if msg[offset] == '[' {
						currentKey = common.JsonKeyAskPrice
						offset += 1
						collectStart = offset
						continue
					} else if msg[offset] == ']' {
						//empty ask
						return nil
					} else {
						return fmt.Errorf("bad ask location for %s", msg)
					}
				} else {
					return fmt.Errorf("bad size following for %s", msg)
				}
			}
			break
		case common.JsonKeyAskPrice:
			if msg[offset] == ',' {
				ask[0], err = common.ParseDecimal(msg[collectStart:offset])
				if err != nil {
					return fmt.Errorf("askPrice common.ParseDecimal error %v msg %s", err, msg[collectStart:offset])
				}
				offset += 2
				collectStart = offset
				currentKey = common.JsonKeyAskSize
				continue
			}
			break
		case common.JsonKeyAskSize:
			if msg[offset] == ']' {
				ask[1], err = common.ParseDecimal(msg[collectStart:offset])
				if err != nil {
					return fmt.Errorf("askSize common.ParseDecimal error %v msg %s", err, msg[collectStart:offset])
				}
				orderBook.Asks = orderBook.Asks.Update(ask)
				offset += 1
				if msg[offset] == ',' {
					offset += 3
					collectStart = offset
					currentKey = common.JsonKeyAskPrice
					continue
				} else if msg[offset] == ']' {
					return nil
				} else {
					return fmt.Errorf("bad ask size following for %s", msg)
				}
			}
			break
		}
		offset += 1
	}
	return nil
}
