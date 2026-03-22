package main

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	binance_usdtfuture "github.com/geometrybase/hft-micro/binance-usdtfuture"
	coinbase_usdspot "github.com/geometrybase/hft-micro/coinbase-usdspot"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/influx/client"
	"github.com/geometrybase/hft-micro/logger"
	stream_stats "github.com/geometrybase/hft-micro/stream-stats"
	"os"
	"path"
	"sort"
	"strings"
	"time"
)



func main() {
	symbolsStr := flag.String("symbols", "MANA-USD,RLC-USD,GRT-USD,UNI-USD,ENJ-USD,ALGO-USD,BCH-USD,MATIC-USD,KEEP-USD,LTC-USD,FIL-USD,BTC-USD,XTZ-USD,DOGE-USD,OMG-USD,LRC-USD,ETC-USD,REN-USD,ZRX-USD,SUSHI-USD,BAT-USD,BAND-USD,LINK-USD,ANKR-USD,MKR-USD,ATOM-USD,SOL-USD,CRV-USD,CHZ-USD,NKN-USD,KNC-USD,DOT-USD,OGN-USD,EOS-USD,ICP-USD,GTC-USD,ZEC-USD,SNX-USD,BAL-USD,AAVE-USD,STORJ-USD,DASH-USD,XLM-USD,TRB-USD,YFI-USD,COMP-USD,ETH-USD,ADA-USD,1INCH-USD,SKL-USD", "symbols, separate by comma")
	flag.Parse()

	symbols := strings.Split(*symbolsStr, ",")
	sort.Strings(symbols)
	symbols = symbols[:]

	ctx := context.Background()
	iw, err := common.NewInfluxWriter(
		ctx,
		"http://localhost:8086",
		"",
		"",
		"hft",
		500,
	)
	if err != nil {
		panic(err)
	}
	defer iw.Stop()

	startTime, err := time.Parse("20060102", "20210725")
	if err != nil {
		logger.Fatal(err)
	}
	endTime, err := time.Parse("20060102", "20210726")
	if err != nil {
		logger.Fatal(err)
	}
	dateStrs := ""
	for i := startTime; i.Sub(endTime) <= 0; i = i.Add(time.Hour * 24) {
		dateStrs += i.Format("20060102,")
	}
	dateStrs = dateStrs[:len(dateStrs)-1]

	quantileLookback := time.Hour * 72
	quantileSubInterval := time.Hour
	quantilePath := "/home/clu/Downloads/cb-bn-leadlag"

	//bidTDs := make(map[string]*tdigest.TDigest)
	//askTDs := make(map[string]*tdigest.TDigest)

	for _, xSymbol := range symbols {
		ySymbol := strings.Replace(xSymbol, "-USD", "USDT", -1)
		counter := 0
		timedTDigest := stream_stats.NewTimedTDigest(quantileLookback, quantileSubInterval)

		shortLastEnter := 0.0
		longLastEnter := 0.0

		//bidTD, _ := tdigest.New()
		//askTD, _ := tdigest.New()

		xTicker := &coinbase_usdspot.Ticker{}
		yTicker := &binance_usdtfuture.BookTicker{}
		for _, dateStr := range strings.Split(dateStrs, ",") {
			logger.Debugf("/home/clu/MarketData/cbus-bnuf-ticker/%s/%s-%s,%s.jl.gz", dateStr, dateStr, xSymbol, ySymbol)
			file, err := os.Open(
				fmt.Sprintf("/home/clu/MarketData/cbus-bnuf-ticker/%s/%s-%s,%s.jl.gz", dateStr, dateStr, xSymbol, ySymbol),
			)
			if err != nil {
				logger.Debugf("os.Open() error %v", err)
				continue
			}
			gr, err := gzip.NewReader(file)
			if err != nil {
				logger.Debugf("gzip.NewReader(file) error %v", err)
				continue
			}
			b := make([]byte, 0, 512)
			_, err = gr.Read(b)
			if err != nil {
				logger.Debugf("gr.Read(b) error %v", err)
				continue
			}
			scanner := bufio.NewScanner(gr)
			var msg []byte
			for scanner.Scan() {
				counter++
				msg = scanner.Bytes()
				if msg[0] == 'X' {
					err = json.Unmarshal(msg[21:], xTicker)
					if err != nil {
						continue
					}
				} else if msg[0] == 'Y' {
					err = binance_usdtfuture.ParseBookTicker(msg[21:], yTicker)
					if err != nil {
						continue
					}
				} else {
					continue
				}
				if xTicker.Symbol == xSymbol && yTicker.Symbol == ySymbol {
					shortLastEnter = (yTicker.BestBidPrice - xTicker.BestAsk) / xTicker.BestAsk
					longLastEnter = (yTicker.BestAskPrice - xTicker.BestBid) / xTicker.BestBid
					if xTicker.Time.Sub(yTicker.EventTime) > 0 {
						_ = timedTDigest.Insert(xTicker.Time, (shortLastEnter+longLastEnter)*0.5)
					} else {
						_ = timedTDigest.Insert(yTicker.EventTime, (shortLastEnter+longLastEnter)*0.5)
					}
					if counter%1000 == 0 {
						fields := make(map[string]interface{})
						fields["enterMiddle"] = timedTDigest.Quantile(0.5)
						fields["shortLastEnter"] = shortLastEnter
						fields["longLastEnter"] = longLastEnter
						fields["xBidPrice"] = xTicker.BestBid
						fields["xAskPrice"] = xTicker.BestAsk
						pt, err := client.NewPoint(
							"cb-bn-leadlag",
							map[string]string{
								"xSymbol": xSymbol,
							},
							fields,
							yTicker.EventTime,
						)
						if err == nil {
							iw.PointCh <- pt
						}
					}
				}
			}
			_ = gr.Close()
			_ = file.Close()
		}
		data, err := json.Marshal(timedTDigest)
		if err != nil {
			logger.Debugf("%v", err)
			continue
		}
		file, err := os.OpenFile(path.Join(quantilePath, xSymbol+"-"+ySymbol+".json"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
		if err != nil {
			logger.Debugf("%v", err)
			continue
		}
		_, err = file.Write(data)
		if err != nil {
			logger.Debugf("%v", err)
			continue
		}
		err = file.Close()
		if err != nil {
			logger.Debugf("%v", err)
			continue
		}
	}
}
