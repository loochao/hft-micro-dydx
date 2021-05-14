package main

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/ftxperp"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/geometrybase/hft-micro/tdigest"
	"os"
	"sort"
	"strings"
	"time"
)

func main() {


	matchSymbols := make([]string, 0)
	pairMaps := make(map[string]string)
	for ftxMarket := range ftxperp.PriceIncrements {
		bnSymbol := strings.Replace(ftxMarket, "-PERP", "USDT", -1)
		if _, ok := bnswap.TickSizes[bnSymbol]; ok && len(bnSymbol) >= 7{
			pairMaps[ftxMarket] = bnSymbol
			matchSymbols = append(matchSymbols, ftxMarket)
		}
	}

	logger.Debugf("\n\n%s\n\n", matchSymbols)


	sort.Strings(matchSymbols)
	fmt.Printf("\n\n")
	for _, fs := range matchSymbols {
		bs := pairMaps[fs]
		fmt.Printf("%s: %s\n", fs, bs)
	}
	fmt.Printf("\n\n")


	symbols := strings.Split(
		`DOGE-PERP`,
		",",
	)
	symbols = matchSymbols
	dateStrs := "20210509,20210510,20210511,20210512,20210513"
	for _, symbol := range symbols {
		var lastBuyTrade *ftxperp.Trade
		var lastSellTrade *ftxperp.Trade
		var lastTrade *ftxperp.Trade
		sellImpactTD, _ := tdigest.New()
		buyImpactTD, _ := tdigest.New()
		for _, dateStr := range strings.Split(dateStrs, ",") {
			file, err := os.Open(
				fmt.Sprintf("/Users/chenjilin/MarketData/ftxperp-trade/%s-%s.ftxperp.trade.jl.gz", dateStr, symbol),
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
				tradeData := ftxperp.TradesData{}
				err = json.Unmarshal(scanner.Bytes(), &tradeData)
				if err != nil {
					continue
				}
				for _, d := range tradeData.Data {
					d := d
					if lastTrade != nil {
						if d.Side == ftxperp.TradeSideSell {
							if lastSellTrade != nil &&
								d.Time.Sub(lastSellTrade.Time) < time.Second &&
								lastTrade.Side == ftxperp.TradeSideSell {
								_ = sellImpactTD.Add((d.Price - lastSellTrade.Price) / lastSellTrade.Price)
							}
							lastSellTrade = &d
						} else {
							if lastBuyTrade != nil &&
								d.Time.Sub(lastBuyTrade.Time) < time.Second &&
								lastTrade.Side == ftxperp.TradeSideBuy {
								_ = buyImpactTD.Add((d.Price - lastBuyTrade.Price) / lastBuyTrade.Price)
							}
							lastBuyTrade = &d
						}
					}
					lastTrade = &d
				}



			}
			_ = gr.Close()
			_ = file.Close()
		}
		fmt.Printf(
			"%s:\t%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f\n",
			symbol,
			sellImpactTD.Quantile(0.005),
			sellImpactTD.Quantile(0.05),
			sellImpactTD.Quantile(0.1),
			sellImpactTD.Quantile(0.2),
			sellImpactTD.Quantile(0.5),
			buyImpactTD.Quantile(0.5),
			buyImpactTD.Quantile(0.8),
			buyImpactTD.Quantile(0.9),
			buyImpactTD.Quantile(0.95),
			buyImpactTD.Quantile(0.995),
		)
	}
}
