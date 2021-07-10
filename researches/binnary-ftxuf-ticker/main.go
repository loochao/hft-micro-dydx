package main

import (
	"bufio"
	"compress/gzip"
	"flag"
	"fmt"
	ftx_usdfuture "github.com/geometrybase/hft-micro/ftx-usdfuture"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {

	symbolsStr := flag.String("symbols", "1INCH-PERP,AAVE-PERP,ADA-PERP,ALGO-PERP,ALPHA-PERP,ATOM-PERP,AVAX-PERP,AXS-PERP,BAL-PERP,BAND-PERP,BAT-PERP,BCH-PERP,BNB-PERP,BTC-PERP,BTT-PERP,CHZ-PERP,COMP-PERP,CRV-PERP,DASH-PERP,DEFI-PERP,DENT-PERP,DODO-PERP,DOGE-PERP,DOT-PERP,EGLD-PERP,ENJ-PERP,EOS-PERP,ETC-PERP,ETH-PERP,FIL-PERP,FLM-PERP,FTM-PERP,GRT-PERP,HBAR-PERP,HNT-PERP,HOT-PERP,ICP-PERP,IOTA-PERP,KAVA-PERP,KNC-PERP,KSM-PERP,LINA-PERP,LINK-PERP,LRC-PERP,LTC-PERP,LUNA-PERP,MATIC-PERP,MKR-PERP,MTL-PERP,NEAR-PERP,NEO-PERP,OMG-PERP,ONT-PERP,QTUM-PERP,REEF-PERP,REN-PERP,RSR-PERP,RUNE-PERP,SAND-PERP,SC-PERP,SKL-PERP,SNX-PERP,SOL-PERP,SRM-PERP,STMX-PERP,STORJ-PERP,SUSHI-PERP,SXP-PERP,THETA-PERP,TOMO-PERP,TRX-PERP,UNI-PERP,VET-PERP,WAVES-PERP,XEM-PERP,XLM-PERP,XMR-PERP,XRP-PERP,XTZ-PERP,YFI-PERP,YFII-PERP,ZEC-PERP,ZIL-PERP,ZRX-PERP", "symbols, separate by comma")
	flag.Parse()
	symbols := strings.Split(*symbolsStr, ",")[:1]
	logger.Debugf("%d", len(symbols))
	startTime, err := time.Parse("20060102", "20210629")
	if err != nil {
		logger.Fatal(err)
	}
	endTime := time.Now()
	if err != nil {
		logger.Fatal(err)
	}
	dateStrs := ""
	for i := startTime; i.Sub(endTime) <= 0; i = i.Add(time.Hour * 24) {
		dateStrs += i.Format("20060102,")
	}
	dateStrs = dateStrs[:len(dateStrs)-1]
	dataPath := fmt.Sprintf("/Users/chenjilin/MarketData/ftxuf-ticker")
	const groupCount = 8192

	for _, symbol := range symbols {
		var csvFile *os.File
		var csvGW *gzip.Writer
		var lastMonth string
		for _, dateStr := range strings.Split(dateStrs, ",") {
			if csvFile == nil || lastMonth != dateStr[:6] {
				if csvGW != nil {
					_ = csvGW.Flush()
					_ = csvGW.Close()
				}
				if csvFile != nil {
					_ = csvFile.Close()
				}
				lastMonth = dateStr[:6]
				csvFile, err = os.OpenFile(fmt.Sprintf("%s/%s-%s-ticker.csv.gz", dataPath, dateStr[:6], symbol), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0755)
				if err != nil {
					logger.Debugf("os.OpenFile error %v", err)
					break
				}
				csvGW, err = gzip.NewWriterLevel(csvFile, gzip.BestCompression)
				if err != nil {
					logger.Debugf("gzip.NewWriterLevel error %v", err)
					break
				}
				_, err = csvGW.Write([]byte(fmt.Sprintf("time,bidPrice,bidSize,askPrice,askSize")))
				if err != nil {
					logger.Debugf("csvGW.Write error %v", err)
					break
				}
			}
			jlFile, err := os.Open(
				fmt.Sprintf("%s/%s/%s-%s.ticker.jl.gz", dataPath, dateStr, dateStr, symbol),
			)
			if err != nil {
				logger.Debugf("os.Open() error %v", err)
				continue
			}
			jlGr, err := gzip.NewReader(jlFile)
			if err != nil {
				logger.Debugf("gzip.NewReader error %v", err)
				continue
			}
			scanner := bufio.NewScanner(jlGr)
			var msg []byte
			var ticker = ftx_usdfuture.Ticker{}
			groupBytes := ""
			counter := 0
			for scanner.Scan() {
				msg = scanner.Bytes()
				err = ftx_usdfuture.ParseTicker(msg, &ticker)
				if err != nil {
					logger.Debugf("ftx_usdfuture.ParseTicker error %v", err)
					continue
				}
				groupBytes += fmt.Sprintf(
					"%d,%s,%s,%s,%s\n",
					ticker.Time.UnixNano(),
					strconv.FormatFloat(ticker.Bid, 'f', -1, 64),
					strconv.FormatFloat(ticker.BidSize, 'f', -1, 64),
					strconv.FormatFloat(ticker.Ask, 'f', -1, 64),
					strconv.FormatFloat(ticker.AskSize, 'f', -1, 64),
				)
				counter++
				if counter == groupCount {
					counter = 0
					_, err = csvGW.Write([]byte(groupBytes))
					if err != nil {
						logger.Debugf("csvGW.Write error %v", err)
						continue
					}
					groupBytes = ""
				}
			}
			if groupBytes != "" {
				_, err = csvGW.Write([]byte(groupBytes))
				if err != nil {
					logger.Debugf("csvGW.Write error %v", err)
				}
			}
			_ = jlGr.Close()
			_ = jlFile.Close()
		}
		if csvGW != nil {
			_ = csvGW.Flush()
			_ = csvGW.Close()
		}
		if csvFile != nil {
			_ = csvFile.Close()
		}
	}
}
