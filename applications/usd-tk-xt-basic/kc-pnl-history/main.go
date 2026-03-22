package main

import (
	"encoding/json"
	"fmt"
	"github.com/geometrybase/hft-micro/logger"
	"io/ioutil"
	"math"
	"sort"
	"strings"
)

type PnlRow struct {
	ID                string  `json:"id"`
	Symbol            string  `json:"symbol"`
	RealisedPnl       float64 `json:"realisedPnl"`
	RealisedGrossCost float64 `json:"realisedGrossCost"`
	FundingFee        float64 `json:"fundingFee"`
	DealComm          float64 `json:"dealComm"`
	Type              string  `json:"type"`
	Currency          string  `json:"currency"`
	CloseDate         int64   `json:"closeDate"`
	Offset            int     `json:"offset"`
}

type Data struct {
	DataList []PnlRow `json:"dataList"`
	HasMore  bool     `json:"hasMore"`
}

type Response struct {
	Success bool   `json:"success"`
	Code    string `json:"code"`
	Msg     string `json:"msg"`
	Retry   bool   `json:"retry"`
	Data    Data   `json:"data"`
}

func main() {
	pnls := map[string]float64{}
	pnlDetails := map[string][]float64{}
	contents, err := ioutil.ReadFile("/home/clu/Projects/hft-micro/applications/usd-tk-xt-basic/kc-pnl-history/outputs/history20210911.json")
	if err != nil {
		logger.Fatal(err)
	}
	history := Response{}
	err = json.Unmarshal(contents, &history)
	if err != nil {
		logger.Fatal(err)
	}
	logger.Debugf("TOTAL LEN %d", len(history.Data.DataList))
	for _, h := range history.Data.DataList {
		if _, ok := pnls[h.Symbol]; ok {
			pnls[h.Symbol] += h.RealisedPnl + h.FundingFee + h.DealComm
			pnlDetails[h.Symbol] = append(pnlDetails[h.Symbol], pnls[h.Symbol])
		} else {
			pnls[h.Symbol] = h.RealisedPnl + h.FundingFee + h.DealComm
			pnlDetails[h.Symbol] = []float64{pnls[h.Symbol]}
		}
	}
	symbols := make([]string, 0)
	for s := range pnls {
		symbols = append(symbols, s)
	}
	sort.Strings(symbols)
	for _, s := range symbols {
		pnl := pnls[s]
		//if pnl < 0 {
			fmt.Printf("%s: %.f %.2f\n", s, pnl, pnlDetails[s])
		//}
	}
	fmt.Printf("\n\n")
	total := 0.0
	for _, s := range symbols {
		pnl := pnls[s]
		total += pnl
		fmt.Printf("%s: %.2f\n", s, pnl)
		if pnl < 0 {
			delete(pnls, s)
			continue
		}
	}
	fmt.Printf("\n\nTOTAL: %f", total)

	sum := 0.0
	count := 0
	for _, s := range symbols {
		pnl := pnls[s]
		sum += pnl
		count += len(pnlDetails[s])
	}
	sort.Strings(symbols)
	mean := sum / float64(len(symbols))
	meanCount := count/len(symbols)


	fmt.Printf("\nsharpeRatios:\n")
	sharpeRatios := make(map[string]float64)
	sharpeRatioSum := 0.0
	for _, s := range symbols {
		pnl := pnls[s]
		stddev := StdDev(pnlDetails[s])
		//if stddev != 0 {
		//	sharpeRatios[s] = pnl / stddev * float64(len(pnlDetails[s]))
		//	fmt.Printf("  %s: %.2f %f \n", s, pnl/stddev, pnlDetails[s])
		//} else {
		//	sharpeRatios[s] = float64(len(pnlDetails[s]))
		//	fmt.Printf("  %s: %.2f %f \n", s, 1.0, pnlDetails[s])
		//}
		if stddev != 0 {
			sharpeRatios[s] = pnl / stddev * (float64(len(pnlDetails[s]))/float64(meanCount))
			fmt.Printf("  %s: %.2f\n", s, sharpeRatios[s])
		} else {
			sharpeRatios[s] = float64(len(pnlDetails[s]))/float64(meanCount)
			fmt.Printf("  %s: %.2f\n", s, sharpeRatios[s])
		}
		sharpeRatioSum += sharpeRatios[s]
	}

	fmt.Printf("\nMEAN %.2f\n\n", mean)
	fmt.Printf("\nxyPairs:\n")
	for _, s := range symbols {
		fmt.Printf("  %s: %s\n", s, strings.Replace(s, "USDTM", "USDT", -1))
	}
	fmt.Printf("\ntargetWeights:\n")
	for _, s := range symbols {
		//pnl := pnls[s]
		//weight := math.Sqrt(pnl / mean)
		weight := math.Sqrt(sharpeRatios[s]/(sharpeRatioSum/float64(len(symbols))))
		if weight > 1.0 {
			weight = 1.0
		}
		fmt.Printf("  %s: %.2f\n", s, weight)
	}

}

func StdDev(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	sum := 0.0
	for i := 0; i < len(values); i++ {
		sum += values[i]
	}
	mean := sum / float64(len(values))
	sd := 0.0
	for j := 0; j < len(values); j++ {
		sd += math.Pow(values[j]-mean, 2)
	}
	return math.Sqrt(sd / float64(len(values)))
}
