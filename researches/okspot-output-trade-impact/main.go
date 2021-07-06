package main

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/geometrybase/hft-micro/okex-usdtspot"
	"github.com/geometrybase/hft-micro/tdigest"
	"os"
	"strings"
	"time"
)

func main() {
	symbols := strings.Split(
		`BTC-USDT,ETH-USDT,BAT-USDT,SRM-USDT,DOT-USDT,BAND-USDT,YFI-USDT,TRB-USDT,YFII-USDT,RSR-USDT,SUSHI-USDT,KSM-USDT,UNI-USDT,AVAX-USDT,SOL-USDT,EGLD-USDT,AAVE-USDT,FIL-USDT,NEAR-USDT,GRT-USDT,1INCH-USDT,BCH-USDT,DOGE-USDT,ADA-USDT,ALGO-USDT,ATOM-USDT,BAL-USDT,COMP-USDT,CRV-USDT,FLM-USDT,FTM-USDT,SNX-USDT,WAVES-USDT,XTZ-USDT,ZIL-USDT,DASH-USDT,LRC-USDT,XRP-USDT,ZEC-USDT,NEO-USDT,QTUM-USDT,IOTA-USDT,LTC-USDT,ETC-USDT,EOS-USDT,OMG-USDT,STORJ-USDT,LINK-USDT,ZRX-USDT,CVC-USDT,KNC-USDT,ICX-USDT,TRX-USDT,XMR-USDT,XLM-USDT,IOST-USDT,THETA-USDT,MKR-USDT,ZEN-USDT,ONT-USDT`,
		",",
	)
	dateStrs := "20210507"

	quantiles := make(map[string]string)
	for _, symbol := range symbols {
		var lastTrade *okex_usdtspot.Trade
		sellImpactTD, _ := tdigest.New()
		buyImpactTD, _ := tdigest.New()
		var wsTrade okex_usdtspot.WSTrades
		for _, dateStr := range strings.Split(dateStrs, ",") {
			file, err := os.Open(
				fmt.Sprintf("/Users/chenjilin/MarketData/okex-usdtspot-trade/%s-%s.okex-usdtspot.trade.jl.gz", dateStr, symbol),
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
				err = json.Unmarshal(scanner.Bytes(), &wsTrade)
				if err != nil {
					continue
				}
				for _, d := range wsTrade.Data {
					if lastTrade != nil {
						if d.Side == "sell" {
							if lastTrade.Side == "sell" &&
								d.EventTime.Sub(lastTrade.EventTime) < time.Millisecond {
								_ = sellImpactTD.Add((d.Price - lastTrade.Price) / lastTrade.Price)
							}
						} else {
							if lastTrade.Side == "buy" &&
								d.EventTime.Sub(lastTrade.EventTime) < time.Millisecond {
								_ = buyImpactTD.Add((d.Price - lastTrade.Price) / lastTrade.Price)
							}
						}
					}
					d := d
					lastTrade = &d
				}
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
			sellImpactTD.Quantile(0.00005),
			sellImpactTD.Quantile(0.0005),
			sellImpactTD.Quantile(0.005),
			sellImpactTD.Quantile(0.05),
			sellImpactTD.Quantile(0.5),
			buyImpactTD.Quantile(0.5),
			buyImpactTD.Quantile(0.95),
			buyImpactTD.Quantile(0.995),
			buyImpactTD.Quantile(0.9995),
			buyImpactTD.Quantile(0.99995),
		)
	}
}
