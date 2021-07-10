package main

import (
	"bufio"
	"compress/gzip"
	"context"
	"flag"
	"fmt"
	binance_usdtfuture "github.com/geometrybase/hft-micro/binance-usdtfuture"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/influx/client"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/geometrybase/hft-micro/tdigest"
	"os"
	"strings"
	"time"
)

func main() {

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

	leadSymbol := flag.String("leadSymbol", "BTCUSDT", "symbols, separate by comma")
	//lagSymbolsStr := flag.String("lagSymbols", "1000SHIBUSDT,1INCHUSDT,AAVEUSDT,ADAUSDT,AKROUSDT,ALGOUSDT,ALICEUSDT,ALPHAUSDT,ANKRUSDT,ATOMUSDT,AVAXUSDT,AXSUSDT,BAKEUSDT,BALUSDT,BANDUSDT,BATUSDT,BCHUSDT,BELUSDT,BLZUSDT,BNBUSDT,BTCDOMUSDT,BTSUSDT,BTTUSDT,BZRXUSDT,CELRUSDT,CHRUSDT,CHZUSDT,COMPUSDT,COTIUSDT,CRVUSDT,CTKUSDT,CVCUSDT,DASHUSDT,DEFIUSDT,DENTUSDT,DGBUSDT,DODOUSDT,DOGEUSDT,DOTUSDT,EGLDUSDT,ENJUSDT,EOSUSDT,ETCUSDT,ETHUSDT,FILUSDT,FLMUSDT,FTMUSDT,GRTUSDT,GTCUSDT,HBARUSDT,HNTUSDT,HOTUSDT,ICPUSDT,ICXUSDT,IOSTUSDT,IOTAUSDT,KAVAUSDT,KEEPUSDT,KNCUSDT,KSMUSDT,LINAUSDT,LINKUSDT,LITUSDT,LRCUSDT,LTCUSDT,LUNAUSDT,MANAUSDT,MATICUSDT,MKRUSDT,MTLUSDT,NEARUSDT,NEOUSDT,NKNUSDT,OCEANUSDT,OGNUSDT,OMGUSDT,ONEUSDT,ONTUSDT,QTUMUSDT,REEFUSDT,RENUSDT,RLCUSDT,RSRUSDT,RUNEUSDT,RVNUSDT,SANDUSDT,SCUSDT,SFPUSDT,SKLUSDT,SNXUSDT,SOLUSDT,SRMUSDT,STMXUSDT,STORJUSDT,SUSHIUSDT,SXPUSDT,THETAUSDT,TOMOUSDT,TRBUSDT,TRXUSDT,UNFIUSDT,UNIUSDT,VETUSDT,WAVESUSDT,XEMUSDT,XLMUSDT,XMRUSDT,XRPUSDT,XTZUSDT,YFIIUSDT,YFIUSDT,ZECUSDT,ZENUSDT,ZILUSDT,ZRXUSDT", "symbols, separate by comma")
	lagSymbolsStr := flag.String("lagSymbols", "ETHUSDT,1INCHUSDT,AAVEUSDT,ADAUSDT,AKROUSDT,ALGOUSDT,ALICEUSDT,ALPHAUSDT,ANKRUSDT,ATOMUSDT,AVAXUSDT,AXSUSDT,BAKEUSDT,BALUSDT,BANDUSDT,BATUSDT,BCHUSDT,BELUSDT,BLZUSDT,BNBUSDT,BTCDOMUSDT,BTSUSDT,BTTUSDT,BZRXUSDT,CELRUSDT,CHRUSDT,CHZUSDT,COMPUSDT,COTIUSDT,CRVUSDT,CTKUSDT,CVCUSDT,DASHUSDT,DEFIUSDT,DENTUSDT,DGBUSDT,DODOUSDT,DOGEUSDT,DOTUSDT,EGLDUSDT,ENJUSDT,EOSUSDT,ETCUSDT,ETHUSDT,FILUSDT,FLMUSDT,FTMUSDT,GRTUSDT,GTCUSDT,HBARUSDT,HNTUSDT,HOTUSDT,ICPUSDT,ICXUSDT,IOSTUSDT,IOTAUSDT,KAVAUSDT,KEEPUSDT,KNCUSDT,KSMUSDT,LINAUSDT,LINKUSDT,LITUSDT,LRCUSDT,LTCUSDT,LUNAUSDT,MANAUSDT,MATICUSDT,MKRUSDT,MTLUSDT,NEARUSDT,NEOUSDT,NKNUSDT,OCEANUSDT,OGNUSDT,OMGUSDT,ONEUSDT,ONTUSDT,QTUMUSDT,REEFUSDT,RENUSDT,RLCUSDT,RSRUSDT,RUNEUSDT,RVNUSDT,SANDUSDT,SCUSDT,SFPUSDT,SKLUSDT,SNXUSDT,SOLUSDT,SRMUSDT,STMXUSDT,STORJUSDT,SUSHIUSDT,SXPUSDT,THETAUSDT,TOMOUSDT,TRBUSDT,TRXUSDT,UNFIUSDT,UNIUSDT,VETUSDT,WAVESUSDT,XEMUSDT,XLMUSDT,XMRUSDT,XRPUSDT,XTZUSDT,YFIIUSDT,YFIUSDT,ZECUSDT,ZENUSDT,ZILUSDT,ZRXUSDT", "symbols, separate by comma")
	flag.Parse()
	symbols := strings.Split(*lagSymbolsStr, ",")[:1]
	logger.Debugf("%d", len(symbols))
	startTime, err := time.Parse("20060102", "20210703")
	if err != nil {
		logger.Fatal(err)
	}
	endTime, err := time.Parse("20060102", "20210706")
	if err != nil {
		logger.Fatal(err)
	}
	dateStrs := ""
	for i := startTime; i.Sub(endTime) <= 0; i = i.Add(time.Hour * 24) {
		dateStrs += i.Format("20060102,")
	}
	dateStrs = dateStrs[:len(dateStrs)-1]
	for _, symbol := range symbols {

		timedMean := common.NewTimedMean(time.Second * 5)
		td, _ := tdigest.New()
		positionSize := 0.0
		positionPrice := 0.0
		netWorth := 1.0
		enterSilentTime := time.Time{}
		enterSilent := time.Minute
		enterValue := 0.1
		commission := -0.0002
		for _, dateStr := range strings.Split(dateStrs, ",") {
			file1, err := os.Open(
				fmt.Sprintf("/Users/chenjilin/MarketData/bnuf-ticker/%s/%s-%s.ticker.jl.gz", dateStr, dateStr, symbol),
			)
			if err != nil {
				logger.Debugf("os.Open() error %v", err)
				continue
			}
			gr1, err := gzip.NewReader(file1)
			if err != nil {
				logger.Debugf("gzip.NewReader(file) error %v", err)
				continue
			}
			lagScanner := bufio.NewScanner(gr1)
			file2, err := os.Open(
				fmt.Sprintf("/Users/chenjilin/MarketData/bnuf-ticker/%s/%s-%s.ticker.jl.gz", dateStr, dateStr, *leadSymbol),
			)
			if err != nil {
				logger.Debugf("os.Open() error %v", err)
				continue
			}
			gr2, err := gzip.NewReader(file2)
			if err != nil {
				logger.Debugf("gzip.NewReader(file) error %v", err)
				continue
			}
			leadScanner := bufio.NewScanner(gr2)
			var msg []byte
			var leadBookTicker = binance_usdtfuture.BookTicker{}
			var lastLagBookTicker *binance_usdtfuture.BookTicker
			var currentLagBookTicker = binance_usdtfuture.BookTicker{}
			counter := 0
			for leadScanner.Scan() {
				msg = leadScanner.Bytes()
				err = binance_usdtfuture.ParseBookTicker(msg, &leadBookTicker)
				if err != nil {
					logger.Debugf("binance_busdspot.ParseBookTicker error %v", err)
					continue
				}
				//logger.Debugf("%v", leadBookTicker)
				if currentLagBookTicker.EventTime.Sub(leadBookTicker.EventTime) < 0 {
					for lagScanner.Scan() {
						if lastLagBookTicker == nil {
							lastLagBookTicker = new(binance_usdtfuture.BookTicker)
						}
						*lastLagBookTicker = currentLagBookTicker
						msg = lagScanner.Bytes()
						err = binance_usdtfuture.ParseBookTicker(msg, &currentLagBookTicker)
						if err != nil {
							logger.Debugf("binance_busdspot.ParseBookTicker error %v", err)
							break
						}
						if currentLagBookTicker.EventTime.Sub(leadBookTicker.EventTime) > 0 {
							break
						}
					}
				}
				if currentLagBookTicker.EventTime.Sub(leadBookTicker.EventTime) > 0 &&
					lastLagBookTicker != nil &&
					leadBookTicker.EventTime.Sub(lastLagBookTicker.EventTime) < time.Second &&
					leadBookTicker.EventTime.Sub(lastLagBookTicker.EventTime) > -time.Second {
					leadMidPrice := (leadBookTicker.BestBidPrice*leadBookTicker.BestAskQty + leadBookTicker.BestAskPrice*leadBookTicker.BestBidQty) / (leadBookTicker.BestAskQty + leadBookTicker.BestBidQty)
					lagMidPrice := (lastLagBookTicker.BestBidPrice*lastLagBookTicker.BestAskQty + lastLagBookTicker.BestAskPrice*lastLagBookTicker.BestBidQty) / (lastLagBookTicker.BestAskQty + lastLagBookTicker.BestBidQty)
					mean := timedMean.Insert(leadBookTicker.EventTime, lagMidPrice/leadMidPrice)
					_ = td.Add((lagMidPrice/leadMidPrice - mean) / mean * 10000)

					if leadBookTicker.EventTime.Sub(enterSilentTime) > 0 {
						if (lagMidPrice/leadMidPrice-mean)/mean*10000 > 10 {
							enterSilentTime = lastLagBookTicker.EventTime.Add(enterSilent)
							size := enterValue / lastLagBookTicker.BestBidPrice
							if positionSize >= 0 {
								if positionSize == 0 || positionPrice < lastLagBookTicker.BestBidPrice {
									positionPrice = (positionSize*positionPrice + enterValue) / (positionSize + size)
									netWorth += commission * enterValue
									positionSize += size
								}
							} else {
								//先平仓
								netWorth += positionSize * (lastLagBookTicker.BestBidPrice - positionPrice)
								netWorth += -positionSize * lastLagBookTicker.BestBidPrice * commission
								//再加仓
								netWorth += commission * enterValue
								positionPrice = lastLagBookTicker.BestBidPrice
								positionSize = size
							}
						} else if (lagMidPrice/leadMidPrice-mean)/mean*10000 < -10 {
							enterSilentTime = lastLagBookTicker.EventTime.Add(enterSilent)
							size := -enterValue / lastLagBookTicker.BestAskPrice
							if positionSize <= 0 {
								if positionSize == 0 || positionPrice > lastLagBookTicker.BestAskPrice {
									positionPrice = (positionSize*positionPrice - enterValue) / (positionSize + size)
									netWorth += commission * enterValue
									positionSize += size
								}
							} else {
								//先平仓
								netWorth += positionSize * (lastLagBookTicker.BestAskPrice - positionPrice)
								netWorth += positionSize * lastLagBookTicker.BestAskPrice * commission
								//再加仓
								netWorth += commission * enterValue
								positionPrice = lastLagBookTicker.BestAskPrice
								positionSize = size
							}
						}
					}

					counter++
					if counter%100 == 0 {
						//logger.Debugf("%v", currentLagBookTicker)
						//logger.Debugf("%v", lastLagBookTicker)
						//return
						fields := make(map[string]interface{})
						fields["lagMidPrice"] = lagMidPrice
						fields["leadMidPrice"] = leadMidPrice
						fields["mean"] = mean
						if positionPrice > 0 {
							fields["positionSize"] = positionSize
							fields["positionPrice"] = positionPrice
						}
						if positionSize > 0 {
							fields["netWorth"] = netWorth+positionSize*(lastLagBookTicker.BestAskPrice-positionPrice)
						}else if positionSize < 0 {
							fields["netWorth"] = netWorth+positionSize*(lastLagBookTicker.BestBidPrice-positionPrice)
						}else{
							fields["netWorth"] = netWorth
						}
						fields["delta"] = (lagMidPrice/leadMidPrice - mean) / mean * 10000
						fields["delta80"] = td.Quantile(0.8)
						fields["delta995"] = td.Quantile(0.995)
						fields["delta95"] = td.Quantile(0.95)
						fields["delta05"] = td.Quantile(0.05)
						fields["delta005"] = td.Quantile(0.005)
						fields["delta20"] = td.Quantile(0.2)
						fields["delta50"] = td.Quantile(0.5)
						pt, err := client.NewPoint(
							"bnuf-btc-leadlag-ticker",
							map[string]string{
								"symbol": symbol,
							},
							fields,
							leadBookTicker.EventTime,
						)
						if err == nil {
							iw.PointCh <- pt
						}
					}
				}
			}
			_ = gr1.Close()
			_ = file1.Close()
			_ = gr2.Close()
			_ = file2.Close()
		}
	}
}
