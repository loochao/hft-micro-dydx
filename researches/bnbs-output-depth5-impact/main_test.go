package main

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"testing"
)
var maxOrderValues = map[string]float64{
	"1INCHBUSD":    928,
	"AAVEBUSD":     1384,
	"ADABUSD":      7254,
	"ALGOBUSD":     1343,
	"ALICEBUSD":    1322,
	"ALPHABUSD":    1951,
	"ATOMBUSD":     1267,
	"AVAXBUSD":     1700,
	"AXSBUSD":      752,
	"BAKEBUSD":     1536,
	"BALBUSD":      944,
	"BANDBUSD":     1366,
	"BATBUSD":      967,
	"BCHBUSD":      2406,
	"BELBUSD":      754,
	"BNBBUSD":      5405,
	"BTCBUSD":      11362,
	"BTTBUSD":      2168,
	"BZRXBUSD":     1773,
	"CELRBUSD":     1385,
	"CHRBUSD":      1513,
	"CHZBUSD":      6276,
	"COMPBUSD":     1019,
	"COTIBUSD":     1293,
	"CRVBUSD":      3256,
	"CTKBUSD":      550,
	"DASHBUSD":     1599,
	"DGBBUSD":      1844,
	"DODOBUSD":     2236,
	"DOGEBUSD":     5712,
	"DOTBUSD":      3414,
	"EGLDBUSD":     1255,
	"ENJBUSD":      2517,
	"EOSBUSD":      4048,
	"ETCBUSD":      2710,
	"ETHBUSD":      6116,
	"FILBUSD":      4457,
	"FLMBUSD":      1053,
	"FTMBUSD":      1359,
	"GRTBUSD":      921,
	"GTCBUSD":      1860,
	"HBARBUSD":     1500,
	"HNTBUSD":      1023,
	"HOTBUSD":      4288,
	"ICPBUSD":      2558,
	"ICXBUSD":      1443,
	"IOSTBUSD":     1748,
	"IOTABUSD":     1090,
	"KAVABUSD":     883,
	"KEEPBUSD":     1738,
	"KNCBUSD":      1359,
	"KSMBUSD":      2760,
	"LINABUSD":     3948,
	"LINKBUSD":     4736,
	"LITBUSD":      850,
	"LRCBUSD":      650,
	"LTCBUSD":      3659,
	"LUNABUSD":     2064,
	"MANABUSD":     1440,
	"MATICBUSD":    3914,
	"MKRBUSD":      1228,
	"NEARBUSD":     1265,
	"NEOBUSD":      1554,
	"OCEANBUSD":    1144,
	"OMGBUSD":      785,
	"ONEBUSD":      1689,
	"ONTBUSD":      1396,
	"QTUMBUSD":     1572,
	"REEFBUSD":     2889,
	"RLCBUSD":      1398,
	"RSRBUSD":      1024,
	"RUNEBUSD":     3607,
	"RVNBUSD":      915,
	"SANDBUSD":     1338,
	"SCBUSD":       1090,
	"SFPBUSD":      1404,
	"SKLBUSD":      1348,
	"SNXBUSD":      1241,
	"SOLBUSD":      3104,
	"SRMBUSD":      717,
	"STMXBUSD":     1360,
	"SUSHIBUSD":    2217,
	"SXPBUSD":      1640,
	"THETABUSD":    1158,
	"TOMOBUSD":     961,
	"TRBBUSD":      1075,
	"TRXBUSD":      2427,
	"UNFIBUSD":     504,
	"UNIBUSD":      2063,
	"VETBUSD":      3990,
	"WAVESBUSD":    1550,
	"XEMBUSD":      1633,
	"XLMBUSD":      1774,
	"XMRBUSD":      1910,
	"XRPBUSD":      6572,
	"XTZBUSD":      2297,
	"YFIBUSD":      1870,
	"YFIIBUSD":     1018,
	"ZECBUSD":      1180,
	"ZENBUSD":      1615,
	"ZILBUSD":      2536,
	"ZRXBUSD":      717,
}


func TestMaxOrderValues(t *testing.T) {
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
		fmt.Printf("  %s: %s\n", symbol, strings.Replace(symbol, "BUSD", "USDT", -1))
	}
}