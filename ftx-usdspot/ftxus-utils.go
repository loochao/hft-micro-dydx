package ftx_usdspot

import (
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"math"
	"time"
)

func ParseTicker(msg []byte, ticker *Ticker) (err error) {
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
				ticker.Time = time.Unix(int64(t), int64((t-math.Floor(t))*1e9))
				return
			}
		}
		collectEnd += 1
	}
	return
}

func UpdateDepth(msg []byte, depth *Depth) (err error) {
	//{"channel": "orderbook", "market": "DOGE/USD", "type": "partial", "data": {"time": 1631245567.2138145, "checksum": 647984333, "bids": [[0.2559625, 22500.0], [0.25595, 23430.0], [0.255943, 22500.0], [0.2559175, 10000.0], [0.255907, 1930.0], [0.255883, 43099.0], [0.255852, 2160.0], [0.255819, 32972.0], [0.2557825, 1594.0], [0.255773, 2694.0], [0.2557725, 42257.0], [0.255772, 5913.0], [0.255751, 600.0], [0.2557505, 62859.0], [0.25575, 19849.0], [0.2557295, 3000.0], [0.255729, 1456.0], [0.255685, 75243.0], [0.2556725, 1881.0], [0.255671, 1399.0], [0.2556705, 5911.0], [0.255659, 8563.0], [0.255653, 8158.0], [0.255615, 60.0], [0.2555315, 21200.0], [0.255515, 2799.0], [0.2554515, 479456.0], [0.2554505, 1.0], [0.25545, 30574.0], [0.2554425, 10.0], [0.2554405, 19.0], [0.25538, 559.0], [0.255373, 34200.0], [0.255361, 60.0], [0.255316, 66.0], [0.255286, 8170.0], [0.2552655, 11985.0], [0.2552585, 2498.0], [0.2551995, 4080.0], [0.2551845, 8155.0], [0.2551735, 28.0], [0.25517, 22100.0], [0.2550995, 4073.0], [0.2550645, 187244.0], [0.2550435, 8147.0], [0.255029, 3118.0], [0.2550285, 10.0], [0.2549415, 19612.0], [0.254927, 3118.0], [0.2549265, 5000.0], [0.2548275, 200.0], [0.254825, 3118.0], [0.2548155, 19.0], [0.254797, 23971.0], [0.254786, 76.0], [0.25478, 1.0], [0.2547595, 67.0], [0.254724, 236889.0], [0.254719, 18517.0], [0.254713, 55.0], [0.2546595, 7.0], [0.2546175, 24.0], [0.254614, 10.0], [0.254538, 150.0], [0.254523, 1886.0], [0.2545225, 275724.0], [0.2545035, 9.0], [0.254489, 15.0], [0.25448, 232414.0], [0.2544085, 4.0], [0.2543915, 27.0], [0.254327, 1500.0], [0.2542885, 18.0], [0.254241, 22.0], [0.2542405, 2.0], [0.2542125, 4.0], [0.254204, 67.0], [0.2542, 10.0], [0.2541925, 19.0], [0.2540915, 196456.0], [0.254089, 28.0], [0.254081, 10.0], [0.25407, 79.0], [0.2540655, 286682.0], [0.2540075, 5.0], [0.254, 8.0], [0.2538405, 46728.0], [0.25384, 976964.0], [0.25383, 2.0], [0.2538, 10.0], [0.253798, 4.0], [0.253771, 97901.0], [0.2537575, 1.0], [0.253741, 83.0], [0.253678, 200.0], [0.2536495, 67.0], [0.253632, 1.0], [0.253612, 13.0], [0.2536065, 79.0], [0.253593, 45.0]], "asks": [[0.2560325, 10000.0], [0.256044, 22500.0], [0.2560565, 600.0], [0.2560685, 600.0], [0.256069, 8947.0], [0.2560715, 361.0], [0.256078, 143974.0], [0.256083, 19542.0], [0.2560955, 23435.0], [0.2561225, 60.0], [0.2561325, 23432.0], [0.2561355, 1931.0], [0.256153, 43424.0], [0.2561535, 5822.0], [0.25616, 731.0], [0.256175, 1310.0], [0.256179, 3000.0], [0.2561795, 33114.0], [0.25618, 2230.0], [0.2561915, 12000.0], [0.256192, 60412.0], [0.2561935, 219967.0], [0.256196, 19848.0], [0.2561965, 13241.0], [0.256199, 114580.0], [0.2562035, 42147.0], [0.256232, 1383.0], [0.256271, 10.0], [0.256284, 5815.0], [0.25629, 4072.0], [0.256312, 1395.0], [0.256324, 1.0], [0.256335, 5793.0], [0.2563765, 60.0], [0.2563775, 2108.0], [0.256412, 976.0], [0.2564365, 2000.0], [0.256535, 66.0], [0.256547, 24800.0], [0.256571, 390.0], [0.2566215, 39.0], [0.25663, 60.0], [0.2566655, 11985.0], [0.2566855, 10.0], [0.2566965, 4883.0], [0.2567085, 27700.0], [0.256797, 19.0], [0.256844, 17.0], [0.256884, 60.0], [0.256887, 23971.0], [0.256891, 390.0], [0.2569125, 195560.0], [0.256989, 3118.0], [0.2570525, 2500.0], [0.257089, 1.0], [0.2570915, 11263.0], [0.2570955, 66.0], [0.2571, 10.0], [0.2571025, 2.0], [0.257112, 1.0], [0.257126, 200.0], [0.257187, 1.0], [0.2571945, 5.0], [0.257195, 3118.0], [0.257273, 191763.0], [0.25728, 19434.0], [0.2573275, 322408.0], [0.2574, 250496.0], [0.2574265, 19.0], [0.257435, 75.0], [0.2574515, 1.0], [0.257455, 28.0], [0.257479, 4.0], [0.257514, 10.0], [0.257548, 22671.0], [0.25755, 3.0], [0.257574, 13.0], [0.2576575, 66.0], [0.257688, 372822.0], [0.257696, 4.0], [0.2577425, 1.0], [0.257782, 1.0], [0.257798, 3.0], [0.257837, 82.0], [0.257871, 5.0], [0.2579135, 344587.0], [0.257925, 2.0], [0.2579285, 10.0], [0.257999, 1.0], [0.258025, 193.0], [0.258058, 19.0], [0.258122, 17.0], [0.258217, 1.0], [0.2582205, 66.0], [0.2582425, 680907.0], [0.2582655, 38.0], [0.2582755, 200.0], [0.258306, 1.0], [0.258315, 96921.0], [0.2583295, 3104.0]], "action": "partial"}}
	//{"channel": "orderbook", "market": "DOGE/USD", "type": "update", "data": {"time": 1631245567.3229442, "checksum": 1895275442, "bids": [[0.2559315, 1402.0], [0.253593, 0.0]], "asks": [[0.256547, 0.0], [0.2583325, 1.0]], "action": "update"}}
	msgLen := len(msg)
	if msgLen < 128 || msg[2] != 'c' || msg[13] != 'o' {
		return fmt.Errorf("bad msg %s", msg)
	}
	t := 0.0
	collectEnd := 0
	collectStart := 0
	bid := [2]float64{}
	ask := [2]float64{}
	if msg[45] == ',' {
		depth.Symbol = common.UnsafeBytesToString(msg[36:44])
		collectEnd += 56
	} else if msg[44] == ',' {
		depth.Symbol = common.UnsafeBytesToString(msg[36:43])
		collectEnd += 55
	} else if msg[46] == ',' {
		depth.Symbol = common.UnsafeBytesToString(msg[36:45])
		collectEnd += 57
	} else if msg[47] == ',' {
		depth.Symbol = common.UnsafeBytesToString(msg[36:46])
		collectEnd += 58
	} else if msg[48] == ',' {
		depth.Symbol = common.UnsafeBytesToString(msg[36:47])
		collectEnd += 59
	} else if msg[49] == ',' {
		depth.Symbol = common.UnsafeBytesToString(msg[36:48])
		collectEnd += 60
	} else if msg[50] == ',' {
		depth.Symbol = common.UnsafeBytesToString(msg[36:49])
		collectEnd += 61
	} else {
		return fmt.Errorf("missing symbol %s", msg)
	}
	currentKey := common.JsonKeyEventTime
	switch msg[collectEnd] {
	case 'p':
		depth.Bids = common.Bids{}
		depth.Asks = common.Asks{}
		collectEnd += 27
		collectStart = collectEnd
		collectEnd += 10
		break
	case 'u':
		collectEnd += 26
		collectStart = collectEnd
		collectEnd = collectEnd + 10
		break
	default:
		return fmt.Errorf("bad type %s", msg[collectStart:])
	}
	depth.ParseTime = time.Now()
	for collectEnd < msgLen-12 {
		switch currentKey {
		case common.JsonKeyEventTime:
			if msg[collectEnd] == ',' {
				t, err = common.ParseDecimal(msg[collectStart:collectEnd])
				if err != nil {
					return
				}
				depth.EventTime = time.Unix(0, int64(t*1000000000))
				currentKey = common.JsonKeyUnknown
				collectEnd += 14
			}
			break
		case common.JsonKeyBidPrice:
			if msg[collectEnd] == ',' {
				bid[0], err = common.ParseDecimal(msg[collectStart:collectEnd])
				if err != nil {
					return
				}
				//logger.Debugf("bid price %.8f", bid[0])
				collectEnd += 2
				collectStart = collectEnd
				currentKey = common.JsonKeyBidSize
			}
			break
		case common.JsonKeyBidSize:
			if msg[collectEnd] == ']' {
				bid[1], err = common.ParseDecimal(msg[collectStart:collectEnd])
				if err != nil {
					return err
				}
				//logger.Debugf("bid %v", bid)
				//logger.Debugf("bid size %.8f", bid[1])
				depth.Bids = depth.Bids.Update(bid)
				collectEnd += 1
				if msg[collectEnd] == ',' {
					//还有bid
					currentKey = common.JsonKeyBidPrice
					collectEnd += 3
					collectStart = collectEnd
				} else if msg[collectEnd] == ']' {
					//已经结束
					collectEnd += 12
					if msg[collectEnd] == '[' {
						//ask不为空
						currentKey = common.JsonKeyAskPrice
						collectEnd += 1
						collectStart = collectEnd
					} else if msg[collectEnd] == ']' {
						//ask为空, 解析结束
						return
					} else {
						return fmt.Errorf("bad ask %s", msg[collectStart:])
					}
				} else {
					return fmt.Errorf("bad bid %s", msg[collectStart:])
				}
			}
			break
		case common.JsonKeyAskPrice:
			if msg[collectEnd] == ',' {
				ask[0], err = common.ParseDecimal(msg[collectStart:collectEnd])
				if err != nil {
					return
				}
				//logger.Debugf("ask price %.8f", ask[0])
				collectEnd += 2
				collectStart = collectEnd
				currentKey = common.JsonKeyAskSize
			}
			break
		case common.JsonKeyAskSize:
			if msg[collectEnd] == ']' {
				ask[1], err = common.ParseDecimal(msg[collectStart:collectEnd])
				if err != nil {
					return
				}
				//logger.Debugf("ask size %.8f", ask[1])
				//logger.Debugf("ask %v", ask)
				depth.Asks = depth.Asks.Update(ask)
				collectEnd += 1
				if msg[collectEnd] == ',' {
					//还有ask
					currentKey = common.JsonKeyAskPrice
					collectEnd += 3
					collectStart = collectEnd
				} else if msg[collectEnd] == ']' {
					//ask结束
					return
				} else {
					return fmt.Errorf("bad ask end %s", msg[collectStart:])
				}
			}
			break
		case common.JsonKeyUnknown:
			if msg[collectEnd] == 's' && msg[collectEnd-3] == 'b' {
				if msg[collectEnd+5] == '[' {
					collectEnd += 6
					collectStart = collectEnd
					currentKey = common.JsonKeyBidPrice
				} else if msg[collectEnd+5] == ']' {
					//没有bids
					collectEnd += 17
					if msg[collectEnd] == '[' {
						//ask不为空
						currentKey = common.JsonKeyAskPrice
						collectEnd += 1
						collectStart = collectEnd
					} else if msg[collectEnd] == ']' {
						//ask为空, 解析结束
						return
					} else {
						return fmt.Errorf("bad ask %s", msg[collectStart:])
					}

				} else {
					return fmt.Errorf("bad bids %s", msg)
				}
			}
			break
		}
		collectEnd += 1
	}
	return nil
}
