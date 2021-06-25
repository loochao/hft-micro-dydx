package main

import (
	"fmt"
	"math"
	"sort"
	"testing"
)

var maxOrderSizes = map[string]float64{
	"1INCHUSDT": 562,
	"AAVEUSDT":  821,
	"ADAUSDT":   3992,
	"AKROUSDT":  329,
	"ALGOUSDT":  928,
	"ALICEUSDT": 1203,
	"ALPHAUSDT": 833,
	"ANKRUSDT":  683,
	"ATOMUSDT":  965,
	"AVAXUSDT":  837,
	"AXSUSDT":   393,
	"BAKEUSDT":  676,
	"BALUSDT":   558,
	"BANDUSDT":  720,
	"BATUSDT":   737,
	"BCHUSDT":   1546,
	"BELUSDT":   398,
	"BLZUSDT":   346,
	"BNBUSDT":   3472,
	"BTCUSDT":   6569,
	"BTSUSDT":   541,
	"BTTUSDT":   622,
	"BZRXUSDT":  1366,
	"CELRUSDT":  731,
	"CHRUSDT":   652,
	"CHZUSDT":   1170,
	"COMPUSDT":  485,
	"COTIUSDT":  453,
	"CRVUSDT":   4411,
	"CTKUSDT":   421,
	"CVCUSDT":   231,
	"DASHUSDT":  873,
	"DENTUSDT":  683,
	"DGBUSDT":   562,
	"DODOUSDT":  1983,
	"DOGEUSDT":  1964,
	"DOTUSDT":   1805,
	"EGLDUSDT":  723,
	"ENJUSDT":   776,
	"EOSUSDT":   1406,
	"ETCUSDT":   2104,
	"ETHUSDT":   4176,
	"FILUSDT":   4287,
	"FLMUSDT":   499,
	"FTMUSDT":   824,
	"GRTUSDT":   989,
	"HBARUSDT":  733,
	"HNTUSDT":   581,
	"HOTUSDT":   1696,
	"ICPUSDT":   1864,
	"ICXUSDT":   842,
	"IOSTUSDT":  744,
	"IOTAUSDT":  776,
	"KAVAUSDT":  801,
	"KNCUSDT":   2674,
	"KSMUSDT":   1051,
	"LINAUSDT":  5614,
	"LINKUSDT":  1753,
	"LITUSDT":   272,
	"LRCUSDT":   512,
	"LTCUSDT":   2441,
	"LUNAUSDT":  1659,
	"MANAUSDT":  714,
	"MATICUSDT": 1921,
	"MKRUSDT":   585,
	"MTLUSDT":   594,
	"NEARUSDT":  456,
	"NEOUSDT":   1582,
	"NKNUSDT":   326,
	"OCEANUSDT": 505,
	"OGNUSDT":   781,
	"OMGUSDT":   414,
	"ONEUSDT":   585,
	"ONTUSDT":   801,
	"QTUMUSDT":  1375,
	"REEFUSDT":  444,
	"RENUSDT":   1128,
	"RLCUSDT":   554,
	"RSRUSDT":   1425,
	"RUNEUSDT":  1623,
	"RVNUSDT":   502,
	"SANDUSDT":  734,
	"SCUSDT":    520,
	"SFPUSDT":   496,
	"SKLUSDT":   335,
	"SNXUSDT":   511,
	"SOLUSDT":   1258,
	"SRMUSDT":   578,
	"STMXUSDT":  397,
	"STORJUSDT": 609,
	"SUSHIUSDT": 1528,
	"SXPUSDT":   5929,
	"THETAUSDT": 2111,
	"TOMOUSDT":  679,
	"TRBUSDT":   368,
	"TRXUSDT":   3829,
	"UNFIUSDT":  304,
	"UNIUSDT":   1001,
	"VETUSDT":   2654,
	"WAVESUSDT": 840,
	"XEMUSDT":   4613,
	"XLMUSDT":   872,
	"XMRUSDT":   978,
	"XRPUSDT":   9686,
	"XTZUSDT":   619,
	"YFIIUSDT":  481,
	"YFIUSDT":   1063,
	"ZECUSDT":   1183,
	"ZENUSDT":   819,
	"ZILUSDT":   677,
	"ZRXUSDT":   677,
}


func TestMaxOrderSize(t *testing.T) {
	weights := make(map[string]float64)
	totalSize := 0.0
	for _, size := range maxOrderSizes {
		totalSize += size
	}
	meanSize := totalSize/float64(len(maxOrderSizes))
	symbols := make([]string, 0)
	for symbol, size := range maxOrderSizes {
		weights[symbol] = math.Sqrt(size/meanSize)
		if weights[symbol] > 1 {
			weights[symbol] = 1.0
		}
		symbols = append(symbols, symbol)
	}
	//for symbol, weight := range weights {
	//	fmt.Printf("%s: %.2f\n", symbol, weight)
	//}
	sort.Strings(symbols)
	for _, symbol := range symbols {
		fmt.Printf("  %s: %s\n", symbol, symbol)
	}
}