package main

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/kcspot"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/geometrybase/hft-micro/tdigest"
	"os"
	"strings"
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
		`BTC-USDT,ETH-USDT,BCH-USDT,BCHSV-USDT,LINK-USDT,UNI-USDT,YFI-USDT,EOS-USDT,DOT-USDT,FIL-USDT,ADA-USDT,XRP-USDT,LTC-USDT,TRX-USDT,GRT-USDT,SUSHI-USDT,XLM-USDT,1INCH-USDT,ZEC-USDT,DASH-USDT,AAVE-USDT,KSM-USDT,DOGE-USDT,LUNA-USDT,VET-USDT,BNB-USDT,SXP-USDT,IOST-USDT,CRV-USDT,ALGO-USDT,AVAX-USDT,FTM-USDT,THETA-USDT,ATOM-USDT,BTT-USDT,CHZ-USDT,ENJ-USDT,MANA-USDT,BAT-USDT,XEM-USDT,XTZ-USDT`,
		",",
	)

	dateStrs := "20210506"

	quantiles := make(map[string]string)
	tradeCounts := make(map[string]int)

	for _, symbol := range symbols {
		sellImpactTD, _ := tdigest.New()
		buyImpactTD, _ := tdigest.New()
		tradeCounts[symbol] = 0
		for _, dateStr := range strings.Split(dateStrs, ",") {
			file, err := os.Open(
				fmt.Sprintf("/Users/chenjilin/MarketData/kcspot-trade/%s-%s.kcspot.trade.jl.gz", dateStr, symbol),
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
			var takerFirstMatch *kcspot.Trade
			var takerLastMatch *kcspot.Trade
			for scanner.Scan() {
				wsTrade := kcspot.WSTrade{}
				err := json.Unmarshal(scanner.Bytes(), &wsTrade)
				if err != nil {
					//logger.Debugf("json.Unmarshal %v %s", err, scanner.Bytes())
					continue
				}
				if takerFirstMatch != nil && takerLastMatch != nil && takerFirstMatch.TakerOrderId != wsTrade.Data.TakerOrderId {
					if takerLastMatch.Price-takerFirstMatch.Price != 0 {
						if takerLastMatch.Side == "buy" {
							_ = buyImpactTD.Add((takerLastMatch.Price - takerFirstMatch.Price) / takerFirstMatch.Price)
						} else {
							_ = sellImpactTD.Add((takerLastMatch.Price - takerFirstMatch.Price) / takerFirstMatch.Price)
						}
					}
					takerFirstMatch = &wsTrade.Data
				}
				if takerFirstMatch == nil {
					takerFirstMatch = &wsTrade.Data
				}
				takerLastMatch = &wsTrade.Data
				tradeCounts[symbol]++
			}
			_ = gr.Close()
			_ = file.Close()
		}
		quantiles[symbol] = fmt.Sprintf(
			"%.5f,%.5f,%.5f,%.5f,%.5f,%.5f,%.5f,%.5f,%.5f,%.5f",
			sellImpactTD.Quantile(0.005),
			sellImpactTD.Quantile(0.05),
			sellImpactTD.Quantile(0.2),
			sellImpactTD.Quantile(0.25),
			sellImpactTD.Quantile(0.5),
			buyImpactTD.Quantile(0.5),
			buyImpactTD.Quantile(0.75),
			buyImpactTD.Quantile(0.8),
			buyImpactTD.Quantile(0.95),
			buyImpactTD.Quantile(0.995),
		)
		fmt.Printf(
			"%s:\t%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f\n",
			symbol,
			sellImpactTD.Quantile(0.005),
			sellImpactTD.Quantile(0.05),
			sellImpactTD.Quantile(0.2),
			sellImpactTD.Quantile(0.25),
			sellImpactTD.Quantile(0.5),
			buyImpactTD.Quantile(0.5),
			buyImpactTD.Quantile(0.75),
			buyImpactTD.Quantile(0.8),
			buyImpactTD.Quantile(0.95),
			buyImpactTD.Quantile(0.995),
		)
		//output, err := yaml.Marshal(quantiles)
		//if err != nil {
		//	logger.Fatal(err)
		//} else {
		//	fmt.Printf("%s", output)
		//}

	}
	logger.Debugf("%v", tradeCounts)
}
