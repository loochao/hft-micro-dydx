package main

import (
	"fmt"
	"math"
	"sort"
	"testing"
)

var maxOrderValues = map[string]float64{
	"1INCHUSDT":    1150,
	"AAVEUSDT":     1706,
	"ADAUSDT":      8204,
	"AKROUSDT":     760,
	"ALGOUSDT":     2043,
	"ALICEUSDT":    2038,
	"ALPHAUSDT":    1995,
	"ANKRUSDT":     1475,
	"ATOMUSDT":     2141,
	"AVAXUSDT":     1985,
	"AXSUSDT":      756,
	"BAKEUSDT":     1805,
	"BALUSDT":      993,
	"BANDUSDT":     1444,
	"BATUSDT":      1713,
	"BCHUSDT":      3009,
	"BELUSDT":      781,
	"BLZUSDT":      876,
	"BNBUSDT":      5231,
	"BTCUSDT":      16564,
	"BTSUSDT":      942,
	"BTTUSDT":      1526,
	"BZRXUSDT":     2700,
	"CELRUSDT":     1610,
	"CHRUSDT":      1463,
	"CHZUSDT":      2514,
	"COMPUSDT":     1016,
	"COTIUSDT":     831,
	"CRVUSDT":      10995,
	"CTKUSDT":      779,
	"CVCUSDT":      457,
	"DASHUSDT":     2108,
	"DENTUSDT":     1377,
	"DGBUSDT":      1065,
	"DODOUSDT":     4654,
	"DOGEUSDT":     4820,
	"DOTUSDT":      4954,
	"EGLDUSDT":     1389,
	"ENJUSDT":      1743,
	"EOSUSDT":      2776,
	"ETCUSDT":      4132,
	"ETHUSDT":      10357,
	"FILUSDT":      11589,
	"FLMUSDT":      1275,
	"FTMUSDT":      1714,
	"GRTUSDT":      2085,
	"HBARUSDT":     1463,
	"HNTUSDT":      1236,
	"HOTUSDT":      3346,
	"ICPUSDT":      4818,
	"ICXUSDT":      1683,
	"IOSTUSDT":     1527,
	"IOTAUSDT":     1413,
	"KAVAUSDT":     1488,
	"KNCUSDT":      7018,
	"KSMUSDT":      2078,
	"LINAUSDT":     13725,
	"LINKUSDT":     3887,
	"LITUSDT":      621,
	"LRCUSDT":      1035,
	"LTCUSDT":      5824,
	"LUNAUSDT":     3679,
	"MANAUSDT":     1456,
	"MATICUSDT":    4273,
	"MKRUSDT":      1332,
	"MTLUSDT":      1275,
	"NEARUSDT":     986,
	"NEOUSDT":      3163,
	"NKNUSDT":      648,
	"OCEANUSDT":    1132,
	"OGNUSDT":      1660,
	"OMGUSDT":      961,
	"ONEUSDT":      1351,
	"ONTUSDT":      2108,
	"QTUMUSDT":     2609,
	"REEFUSDT":     1260,
	"RENUSDT":      2007,
	"RLCUSDT":      1509,
	"RSRUSDT":      2923,
	"RUNEUSDT":     3551,
	"RVNUSDT":      1129,
	"SANDUSDT":     1586,
	"SCUSDT":       1107,
	"SFPUSDT":      1149,
	"SKLUSDT":      811,
	"SNXUSDT":      1123,
	"SOLUSDT":      3233,
	"SRMUSDT":      1212,
	"STMXUSDT":     832,
	"STORJUSDT":    1272,
	"SUSHIUSDT":    3481,
	"SXPUSDT":      14087,
	"THETAUSDT":    5034,
	"TOMOUSDT":     1153,
	"TRBUSDT":      806,
	"TRXUSDT":      8432,
	"UNFIUSDT":     761,
	"UNIUSDT":      2095,
	"VETUSDT":      6549,
	"WAVESUSDT":    1937,
	"XEMUSDT":      9627,
	"XLMUSDT":      1653,
	"XMRUSDT":      1945,
	"XRPUSDT":      25512,
	"XTZUSDT":      1389,
	"YFIIUSDT":     1117,
	"YFIUSDT":      2358,
	"ZECUSDT":      2764,
	"ZENUSDT":      1624,
	"ZILUSDT":      1498,
	"ZRXUSDT":      1503,
}



func TestMaxOrderSize(t *testing.T) {
	weights := make(map[string]float64)
	totalSize := 0.0
	for _, size := range maxOrderValues {
		totalSize += size
	}
	meanSize := totalSize/float64(len(maxOrderValues))
	symbols := make([]string, 0)
	for symbol, size := range maxOrderValues {
		weights[symbol] = math.Sqrt(size/meanSize)
		if weights[symbol] > 1 {
			weights[symbol] = 1.0
		}
		symbols = append(symbols, symbol)
	}
	for symbol, weight := range weights {
		fmt.Printf("%s: %.2f\n", symbol, weight)
	}
	sort.Strings(symbols)
	for _, symbol := range symbols {
		fmt.Printf("  %s: %s\n", symbol, symbol)
	}
}