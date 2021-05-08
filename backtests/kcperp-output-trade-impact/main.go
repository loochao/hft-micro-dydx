package main

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/kcperp"
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

	//symbols := strings.Split(
	//	`XBTUSDTM,ETHUSDTM,BCHUSDTM,BSVUSDTM,LINKUSDTM,UNIUSDTM,YFIUSDTM,EOSUSDTM,DOTUSDTM,FILUSDTM,ADAUSDTM,XRPUSDTM,LTCUSDTM,TRXUSDTM,GRTUSDTM,SUSHIUSDTM,XLMUSDTM,1INCHUSDTM,ZECUSDTM,DASHUSDTM,AAVEUSDTM,KSMUSDTM,DOGEUSDTM,LUNAUSDTM,VETUSDTM,BNBUSDTM,SXPUSDTM,IOSTUSDTM,CRVUSDTM,ALGOUSDTM,AVAXUSDTM,FTMUSDTM,THETAUSDTM,ATOMUSDTM,BTTUSDTM,CHZUSDTM,ENJUSDTM,MANAUSDTM,BATUSDTM,XEMUSDTM,XTZUSDTM`,
	//	",",
	//)
	symbols := strings.Split(
		`XBTUSDTM,ETHUSDTM,BCHUSDTM,LINKUSDTM,UNIUSDTM,YFIUSDTM,EOSUSDTM,DOTUSDTM,FILUSDTM,ADAUSDTM,XRPUSDTM,LTCUSDTM,TRXUSDTM,GRTUSDTM,SUSHIUSDTM,XLMUSDTM,1INCHUSDTM,ZECUSDTM,DASHUSDTM,AAVEUSDTM,KSMUSDTM,DOGEUSDTM,LUNAUSDTM,VETUSDTM,BNBUSDTM,SXPUSDTM,IOSTUSDTM,CRVUSDTM,ALGOUSDTM,AVAXUSDTM,FTMUSDTM,THETAUSDTM,ATOMUSDTM,BTTUSDTM,CHZUSDTM,ENJUSDTM,MANAUSDTM,BATUSDTM,XEMUSDTM,XTZUSDTM`,
		",",
	)

	//symbols := strings.Split(
	//	`DOGEUSDTM`,
	//	",",
	//)
	takerTradeCounts := make(map[string]int)
	makerTradeCounts := make(map[string]int)
	totalTradeCount := 0

	dateStrs := "20210505,20210506,20210507"

	quantiles := make(map[string]string)

	totalValue := 0.0

	for _, symbol := range symbols {
		sellImpactTD, _ := tdigest.New()
		buyImpactTD, _ := tdigest.New()
		for _, dateStr := range strings.Split(dateStrs, ",") {
			file, err := os.Open(
				fmt.Sprintf("/Users/chenjilin/MarketData/kcperp-trade/%s-%s.kcperp.trade.jl.gz", dateStr, symbol),
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
			var takerFirstMatch *kcperp.Match
			var takerLastMatch *kcperp.Match
			for scanner.Scan() {
				match := kcperp.MatchWS{}
				err := json.Unmarshal(scanner.Bytes(), &match)
				if err != nil {
					//logger.Debugf("json.Unmarshal %v %s", err, scanner.Bytes())
					continue
				}
				if takerFirstMatch != nil && takerLastMatch != nil && takerFirstMatch.TakerOrderId != match.Data.TakerOrderId {
					if takerLastMatch.Price-takerFirstMatch.Price != 0 {
						if takerLastMatch.Side == "buy" {
							_ = buyImpactTD.Add((takerLastMatch.Price - takerFirstMatch.Price) / takerFirstMatch.Price)
						} else {
							_ = sellImpactTD.Add((takerLastMatch.Price - takerFirstMatch.Price) / takerFirstMatch.Price)
						}
					}
					takerFirstMatch = &match.Data
				}
				if takerFirstMatch == nil {
					takerFirstMatch = &match.Data
				}
				takerLastMatch = &match.Data
				if _, ok := makerTradeCounts[takerLastMatch.MakerUserID]; !ok {
					makerTradeCounts[takerLastMatch.MakerUserID] = 0
				}
				makerTradeCounts[takerLastMatch.MakerUserID] ++
				if _, ok := takerTradeCounts[takerLastMatch.TakerUserID]; !ok {
					takerTradeCounts[takerLastMatch.TakerUserID] = 0
				}
				takerTradeCounts[takerLastMatch.TakerUserID] ++
				totalTradeCount++
				//if takerLastMatch.TakerUserID == "6083f90aae8ec10007fbba17" {
				//	totalValue += takerLastMatch.Price*takerLastMatch.Size*100
				//}
			}
			_ = gr.Close()
			_ = file.Close()
		}
		quantiles[symbol] = fmt.Sprintf(
			"%.5f,%.5f,%.5f,%.5f,%.5f,%.5f,%.5f,%.5f,%.5f,%.5f",
			sellImpactTD.Quantile(0.05),
			sellImpactTD.Quantile(0.2),
			sellImpactTD.Quantile(0.3),
			sellImpactTD.Quantile(0.4),
			sellImpactTD.Quantile(0.5),
			buyImpactTD.Quantile(0.5),
			buyImpactTD.Quantile(0.6),
			buyImpactTD.Quantile(0.7),
			buyImpactTD.Quantile(0.8),
			buyImpactTD.Quantile(0.95),
		)
		fmt.Printf(
			"%s:\t%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f\n",
			symbol,
			sellImpactTD.Quantile(0.05),
			sellImpactTD.Quantile(0.2),
			sellImpactTD.Quantile(0.3),
			sellImpactTD.Quantile(0.4),
			sellImpactTD.Quantile(0.5),
			buyImpactTD.Quantile(0.5),
			buyImpactTD.Quantile(0.6),
			buyImpactTD.Quantile(0.7),
			buyImpactTD.Quantile(0.8),
			buyImpactTD.Quantile(0.95),
		)
		//output, err := yaml.Marshal(quantiles)
		//if err != nil {
		//	logger.Fatal(err)
		//} else {
		//	fmt.Printf("%s", output)
		//}

	}
	//for takerUserID, count := range takerTradeCounts {
	//	if float64(count)/float64(totalTradeCount) > 0.01 {
	//		fmt.Printf("taker %s %d %f\n", takerUserID, count,float64(count)/float64(totalTradeCount))
	//	}
	//}
	//fmt.Printf("\n\n")
	//for makerUserID, count := range makerTradeCounts {
	//	if float64(count)/float64(totalTradeCount) > 0.01 {
	//		fmt.Printf("maker %s %d %f\n", makerUserID, count,float64(count)/float64(totalTradeCount))
	//	}
	//}
	logger.Debugf("TOTAL VALUE %f", totalValue)
}
