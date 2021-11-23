package main

import (
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"testing"
)
func TestMathFloor(t *testing.T) {
	logger.Debugf("math.Floor(%f)=%f", 1.1, math.Floor(1.1))
	logger.Debugf("math.Floor(%f)=%f", -1.1, math.Floor(-1.1))
	logger.Debugf("math.Ceil(%f)=%f", -1.1, math.Ceil(-1.1))
}

//func TestGenerateTDs(t *testing.T) {
//	symbolMap := make(map[string]string)
//	maxPosSizes := make(map[string]float64)
//	symbols := make([]string, 0)
//	for xSymbol, xMaxPosSize := range okexv5_usdtswap.MaxSizes {
//		ySymbol := strings.Replace(xSymbol, "-USDT-SWAP", "USDTM", -1)
//		if xSymbol == "BTC-USDT-SWAP" {
//			ySymbol = "XBTUSDTM"
//		}
//		bnufSymbol := strings.Replace(xSymbol, "-USDT-SWAP", "USDT", -1)
//		_, ok1 := kucoin_usdtfuture.TickSizes[ySymbol]
//		_, ok2 := binance_usdtfuture.TickSizes[bnufSymbol]
//		if ok1 && ok2 {
//			symbols = append(symbols, xSymbol)
//			symbolMap[xSymbol] = ySymbol
//			maxPosSizes[xSymbol] = xMaxPosSize * okexv5_usdtswap.Multipliers[xSymbol] * 0.25
//		}
//	}
//	sort.Strings(symbols)
//	//fmt.Printf("\n\nxyPairs:\n")
//	//for _, xSymbol := range symbols {
//	//	fmt.Printf("  %s: %s\n", xSymbol, symbolMap[xSymbol])
//	//}
//	//fmt.Printf("\n\nmaxPosSizes:\n")
//	//for _, xSymbol := range symbols {
//	//	fmt.Printf("  %s: %.0f\n", xSymbol, maxPosSizes[xSymbol])
//	//}
//	var startTime, endTime time.Time
//	var err error
//
//	xRootPath := "/Volumes/CryptoData/okuf"
//	yRootPath := "/Volumes/CryptoData/kcuf"
//	if startTime, err = time.Parse("20060102", "20211013"); err != nil {
//		t.Fatal(err)
//	}
//	if endTime, err = time.Parse("20060102", "20211014"); err != nil {
//		t.Fatal(err)
//	}
//	dateStrs := ""
//	for i := startTime; i.Sub(endTime) <= 0; i = i.Add(time.Hour * 24) {
//		dateStrs += i.Format("20060102,")
//	}
//	dateStrs = dateStrs[:len(dateStrs)-1]
//
//	var xFile, yFile *os.File
//	var xGzipReader, yGzipReader *gzip.Reader
//	var xScanner, yScanner *bufio.Scanner
//	var xPath, yPath string
//	var xMsg, yMsg []byte
//	xDepth := &okexv5_usdtswap.Depth5{}
//	xBookTicker := &okexv5_usdtswap.Ticker{}
//	yDepth := &kucoin_usdtfuture.Depth5{}
//	yBookTicker := &kucoin_usdtfuture.Ticker{}
//	xTimestamp := int64(0)
//	yTimestamp := int64(0)
//	xTime := time.Time{}
//	yTime := time.Time{}
//	var xTicker, yTicer common.Ticker
//	for _, xSymbol := range symbols[:1] {
//		ySymbol := symbolMap[xSymbol]
//		for _, dateStr := range strings.Split(dateStrs, ",") {
//			xPath = path.Join(xRootPath, fmt.Sprintf("%s/%s-%s.jl.gz", dateStr, dateStr, xSymbol))
//			yPath = path.Join(yRootPath, fmt.Sprintf("%s/%s-%s.jl.gz", dateStr, dateStr, ySymbol))
//			logger.Debugf("%s %s", xPath, yPath)
//			if xFile, err = os.Open(xPath); err != nil {
//				logger.Debugf("os.Open() error %v", err)
//				continue
//			}
//			if xGzipReader, err = gzip.NewReader(xFile); err != nil {
//				logger.Debugf("gzip.NewReader() error %v", err)
//				continue
//			}
//			xScanner = bufio.NewScanner(xGzipReader)
//
//			if yFile, err = os.Open(yPath); err != nil {
//				logger.Debugf("os.Open() error %v", err)
//				continue
//			}
//			if yGzipReader, err = gzip.NewReader(yFile); err != nil {
//				logger.Debugf("gzip.NewReader() error %v", err)
//				continue
//			}
//			yScanner = bufio.NewScanner(yGzipReader)
//			for xScanner.Scan() {
//				xMsg = xScanner.Bytes()
//				if len(xMsg) <= 20 {
//					continue
//				}
//				if xTimestamp, err = common.ParseInt(xMsg[1:20]); err != nil {
//					logger.Debugf("%v", err)
//					continue
//				}
//				xTime = time.Unix(0, xTimestamp)
//				if xMsg[0] == 'D' {
//					if err = okexv5_usdtswap.ParseDepth5(xMsg[20:], xDepth); err != nil {
//						logger.Debugf("%v", err)
//						continue
//					}
//					if xDepth.InstId != xSymbol {
//						continue
//					}
//					xTicker = xDepth
//				}else if xMsg[0] == 'B' {
//					if err = okexv5_usdtswap.ParseTicker(xMsg[20:], xBookTicker); err != nil {
//						logger.Debugf("%v", err)
//						continue
//					}
//					if xBookTicker.InstId != xSymbol {
//						continue
//					}
//					xTicker = xBookTicker
//				}
//				for yTime.Sub(xTime) < 0 && yScanner.Scan() {
//					yMsg = yScanner.Bytes()
//					if len(yMsg) <= 20 {
//						continue
//					}
//					if yTimestamp, err = common.ParseInt(yMsg[1:20]); err != nil {
//						logger.Debugf("%v", err)
//						continue
//					}
//					yTime = time.Unix(0, yTimestamp)
//					if yMsg[0] == 'D' {
//						if err = kucoin_usdtfuture.ParseDepth5(yMsg[20:], yDepth); err != nil {
//							logger.Debugf("%v", err)
//							continue
//						}
//						if yDepth.Symbol != ySymbol {
//							continue
//						}
//						xTicker = xDepth
//					}else if yMsg[0] == 'B' {
//						if err = kucoin_usdtfuture.ParseTicker(xMsg[20:], xBookTicker); err != nil {
//							logger.Debugf("%v", err)
//							continue
//						}
//						if xBookTicker.InstId != xSymbol {
//							continue
//						}
//						xTicker = xBookTicker
//					}
//
//
//				}
//			}
//		}
//	}
//}
