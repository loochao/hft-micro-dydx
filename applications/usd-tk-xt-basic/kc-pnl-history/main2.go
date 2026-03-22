package main

//type PnlRow struct {
//	ID                string  `json:"id"`
//	Symbol            string  `json:"symbol"`
//	RealisedPnl       float64 `json:"realisedPnl"`
//	RealisedGrossCost float64 `json:"realisedGrossCost"`
//	FundingFee        float64 `json:"fundingFee"`
//	DealComm          float64 `json:"dealComm"`
//	Type              string  `json:"type"`
//	Currency          string  `json:"currency"`
//	CloseDate         int64   `json:"closeDate"`
//	Offset            int     `json:"offset"`
//}
//
//type Data struct {
//	DataList [] PnlRow `json:"dataList"`
//	HasMore bool `json:"hasMore"`
//}
//
//type Response struct {
//}
//
//func main() {
//	pnls := map[string]float64{}
//	for i := 0; i < 1; i++ {
//		contents, err := ioutil.ReadFile(fmt.Sprintf("/home/clu/Projects/hft-micro/applications/usd-tk-xt-basic/kc-pnl-history/盈亏历史 (%d).csv", i))
//		if err != nil {
//			logger.Fatal(err)
//		}
//		lines := strings.Split(string(contents), "\r\n")
//		for j, l := range lines {
//			if j != 0 {
//				if l != "\"" {
//					l = strings.Replace(l, " ", "", -1)
//					l = strings.Replace(l, "&lt;", "", -1)
//					l = strings.Replace(l, "\"", "", -1)
//					l = strings.Replace(l, "\t", "", -1)
//					l = strings.Replace(l, "\t", "", -1)
//					l = strings.Replace(l, "永续/USDT", "|", -1)
//					l = strings.Replace(l, "USDT", "", -1)
//					l = strings.Split(l, "≈")[0]
//					segs := strings.Split(l, "|")
//					if len(segs) < 2 {
//						continue
//					}
//					v, err := common.ParseDecimal([]byte(segs[1]))
//					if err != nil {
//						logger.Fatal(err)
//					}
//					if _, ok := pnls[segs[0]]; ok {
//						pnls[segs[0]] += v
//					} else {
//						pnls[segs[0]] = v
//					}
//				}
//			}
//		}
//	}
//	total := 0.0
//	for s, pnl := range pnls {
//		total += pnl
//		fmt.Printf("%s: %.2f\n", s, pnl)
//	}
//
//	fmt.Printf("\n\nTOTAL: %f", total)
//}
