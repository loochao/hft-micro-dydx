package main

import (
	"bufio"
	"compress/gzip"
	"context"
	"fmt"
	"github.com/geometrybase/hft-micro/bnspot"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/geometrybase/hft-micro/tdigest"
	"gopkg.in/yaml.v2"
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

	symbols := strings.Split(
		`BTCUSDT,LTCUSDT,ETHUSDT,NEOUSDT,QTUMUSDT,EOSUSDT,ZRXUSDT,OMGUSDT,LRCUSDT,TRXUSDT,KNCUSDT,IOTAUSDT,LINKUSDT,CVCUSDT,ETCUSDT,ZECUSDT,BATUSDT,DASHUSDT,XMRUSDT,ENJUSDT,XRPUSDT,STORJUSDT,BTSUSDT,ADAUSDT,XLMUSDT,WAVESUSDT,ICXUSDT,RLCUSDT,IOSTUSDT,BLZUSDT,ONTUSDT,ZILUSDT,ZENUSDT,THETAUSDT,VETUSDT,RENUSDT,MATICUSDT,ATOMUSDT,FTMUSDT,CHZUSDT,ALGOUSDT,DOGEUSDT,ANKRUSDT,TOMOUSDT,BANDUSDT,XTZUSDT,KAVAUSDT,BCHUSDT,SOLUSDT,HNTUSDT,COMPUSDT,MKRUSDT,SXPUSDT,SNXUSDT,DOTUSDT,RUNEUSDT,BALUSDT,YFIUSDT,SRMUSDT,CRVUSDT,SANDUSDT,OCEANUSDT,LUNAUSDT,RSRUSDT,TRBUSDT,EGLDUSDT,BZRXUSDT,KSMUSDT,SUSHIUSDT,YFIIUSDT,BELUSDT,UNIUSDT,AVAXUSDT,FLMUSDT,ALPHAUSDT,NEARUSDT,AAVEUSDT,FILUSDT,CTKUSDT,AXSUSDT,AKROUSDT,SKLUSDT,GRTUSDT,1INCHUSDT,LITUSDT,RVNUSDT,SFPUSDT,REEFUSDT,DODOUSDT,COTIUSDT,CHRUSDT,ALICEUSDT,HBARUSDT,MANAUSDT,STMXUSDT,UNFIUSDT,XEMUSDT,CELRUSDT,HOTUSDT,ONEUSDT,LINAUSDT,DENTUSDT,MTLUSDT,OGNUSDT,NKNUSDT,DGBUSDT`,
		",",
	)
	dateStrs := "20210503"

	quantiles := make(map[string]string)

	for _, symbol := range symbols {
		//computeInterval := time.Minute * 5
		//var lastEventTime *time.Time
		var lastBuyTrade *bnspot.Trade
		var lastSellTrade *bnspot.Trade
		var lastTrade *bnspot.Trade
		sellImpactTD, _ := tdigest.New()
		buyImpactTD, _ := tdigest.New()
		for _, dateStr := range strings.Split(dateStrs, ",") {
			file, err := os.Open(
				fmt.Sprintf("/Users/chenjilin/MarketData/bnspot-trade/%s-%s.bnspot.trade.jl.gz", dateStr, symbol),
			)
			if err != nil {
				logger.Debugf("os.Open() error %v", err)
				return
			}
			gr, err := gzip.NewReader(file)
			if err != nil {
				logger.Debugf("gzip.NewReader(file) error %v", err)
				return
			}
			scanner := bufio.NewScanner(gr)
			for scanner.Scan() {
				d, err := bnspot.ParseTrade(scanner.Bytes())
				if err != nil {
					//logger.Debugf("bnspot.ParseDepth20 error %v", err)
					continue
				}
				//fields := make(map[string]interface{})
				if lastTrade != nil {
					if d.IsTheBuyerTheMarketMaker {
						if lastSellTrade != nil &&
							d.EventTime.Sub(lastSellTrade.EventTime) < time.Millisecond &&
							lastTrade.IsTheBuyerTheMarketMaker {
							_ = sellImpactTD.Add((d.Price - lastSellTrade.Price) / lastSellTrade.Price)
							//fields["sellDelta"] = (d.Price - lastSellTrade.Price) / lastSellTrade.Price * 10000
							//fields["price"] = d.Price
						}
						lastSellTrade = d
					} else {
						if lastBuyTrade != nil &&
							d.EventTime.Sub(lastBuyTrade.EventTime) < time.Millisecond &&
							!lastTrade.IsTheBuyerTheMarketMaker {
							_ = buyImpactTD.Add((d.Price - lastBuyTrade.Price) / lastBuyTrade.Price)
							//fields["buyDelta"] = (d.Price - lastBuyTrade.Price) / lastBuyTrade.Price * 10000
							//fields["price"] = d.Price
						}
						lastBuyTrade = d
					}
				}
				//if lastEventTime != nil && d.EventTime.Truncate(computeInterval).Sub(*lastEventTime) > 0 {
				//	fields["buyDeltaQ9995"] = buyImpactTD.Quantile(0.9995)
				//	fields["buyDeltaQ995"] = buyImpactTD.Quantile(0.995)
				//	fields["buyDeltaQ95"] = buyImpactTD.Quantile(0.95)
				//	fields["buyDeltaQ80"] = buyImpactTD.Quantile(0.8)
				//	fields["sellDeltaQ0005"] = sellImpactTD.Quantile(0.0005)
				//	fields["sellDeltaQ005"] = sellImpactTD.Quantile(0.005)
				//	fields["sellDeltaQ05"] = sellImpactTD.Quantile(0.05)
				//	fields["sellDeltaQ20"] = sellImpactTD.Quantile(0.2)
				//}
				//if len(fields) > 0 {
				//	pt, err := client.NewPoint(
				//		"bnspot-trade-impact",
				//		map[string]string{
				//			"symbol": symbol,
				//		},
				//		fields,
				//		d.EventTime,
				//	)
				//	if err != nil {
				//		logger.Debugf("client.NewPoint error %v", err)
				//	}
				//	iw.PointCh <- pt
				//}
				//lastEventTime = &d.EventTime
				lastTrade = d
			}
			_ = gr.Close()
			_ = file.Close()
		}
		quantiles[symbol] = fmt.Sprintf(
			"%.5f,%.5f,%.5f,%.5f,%.5f,%.5f,%.5f,%.5f,%.5f,%.5f",
			sellImpactTD.Quantile(0.0005),
			sellImpactTD.Quantile(0.005),
			sellImpactTD.Quantile(0.05),
			sellImpactTD.Quantile(0.2),
			sellImpactTD.Quantile(0.5),
			buyImpactTD.Quantile(0.5),
			buyImpactTD.Quantile(0.8),
			buyImpactTD.Quantile(0.95),
			buyImpactTD.Quantile(0.995),
			buyImpactTD.Quantile(0.9995),
		)
		fmt.Printf(
			"%s:\t%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f\n",
			symbol,
			sellImpactTD.Quantile(0.0005),
			sellImpactTD.Quantile(0.005),
			sellImpactTD.Quantile(0.05),
			sellImpactTD.Quantile(0.2),
			sellImpactTD.Quantile(0.5),
			buyImpactTD.Quantile(0.5),
			buyImpactTD.Quantile(0.8),
			buyImpactTD.Quantile(0.95),
			buyImpactTD.Quantile(0.995),
			buyImpactTD.Quantile(0.9995),
		)
	}
	output, err := yaml.Marshal(quantiles)
	if err != nil {
		logger.Fatal(err)
	} else {
		fmt.Printf("%s", output)
	}
}
