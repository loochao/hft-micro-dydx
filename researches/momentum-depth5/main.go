package main

import (
	"bufio"
	"compress/gzip"
	"context"
	"fmt"
	binance_usdtfuture "github.com/geometrybase/hft-micro/binance-usdtfuture"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/influx/client"
	"github.com/geometrybase/hft-micro/logger"
	stream_stats "github.com/geometrybase/hft-micro/stream-stats"
	"os"
	"path"
	"strings"
	"time"
)

var symbolsMap = map[string]string{
	"BTCUSDT": "XBTUSDTM",
	//"IOSTUSDT":  "IOSTUSDTM",
	//"UNIUSDT":   "UNIUSDTM",
	//"ICPUSDT":   "ICPUSDTM",
	//"THETAUSDT": "THETAUSDTM",
	//"YFIUSDT":   "YFIUSDTM",
	//"OCEANUSDT": "OCEANUSDTM",
	//"XMRUSDT":   "XMRUSDTM",
	//"SXPUSDT":   "SXPUSDTM",
	//"BCHUSDT":   "BCHUSDTM",
	//"TRXUSDT":   "TRXUSDTM",
	//"XEMUSDT":   "XEMUSDTM",
	//"ETHUSDT":   "ETHUSDTM",
	//"MKRUSDT":   "MKRUSDTM",
	//"FTMUSDT":   "FTMUSDTM",
	//"ATOMUSDT":  "ATOMUSDTM",
	//"BANDUSDT":  "BANDUSDTM",
	//"DOTUSDT":   "DOTUSDTM",
	//"FILUSDT":   "FILUSDTM",
	//"AVAXUSDT":  "AVAXUSDTM",
	//"QTUMUSDT":  "QTUMUSDTM",
	//"COMPUSDT":  "COMPUSDTM",
	//"ZECUSDT":   "ZECUSDTM",
	//"ADAUSDT":   "ADAUSDTM",
	//"DOGEUSDT":  "DOGEUSDTM",
	//"XLMUSDT":   "XLMUSDTM",
	//"EOSUSDT":   "EOSUSDTM",
	//"LTCUSDT":   "LTCUSDTM",
	//"VETUSDT":   "VETUSDTM",
	//"ONTUSDT":   "ONTUSDTM",
	//"RVNUSDT":   "RVNUSDTM",
	//"MATICUSDT": "MATICUSDTM",
	//"1INCHUSDT": "1INCHUSDTM",
	//"XRPUSDT":   "XRPUSDTM",
	//"NEOUSDT":   "NEOUSDTM",
	//"ALGOUSDT":  "ALGOUSDTM",
	//"MANAUSDT":  "MANAUSDTM",
	//"WAVESUSDT": "WAVESUSDTM",
	//"KSMUSDT":   "KSMUSDTM",
	//"AAVEUSDT":  "AAVEUSDTM",
	//"LINKUSDT":  "LINKUSDTM",
	//"BATUSDT":   "BATUSDTM",
	//"DENTUSDT":  "DENTUSDTM",
	//"LUNAUSDT":  "LUNAUSDTM",
	//"ETCUSDT":   "ETCUSDTM",
	//"CHZUSDT":   "CHZUSDTM",
	//"CRVUSDT":   "CRVUSDTM",
	//"DASHUSDT":  "DASHUSDTM",
	//"SNXUSDT":   "SNXUSDTM",
	//"GRTUSDT":   "GRTUSDTM",
	//"BTTUSDT":   "BTTUSDTM",
	//"SUSHIUSDT": "SUSHIUSDTM",
	//"ENJUSDT":   "ENJUSDTM",
	//"XTZUSDT":   "XTZUSDTM",
	//"DGBUSDT":   "DGBUSDTM",
	//"SOLUSDT":   "SOLUSDTM",
	//"BNBUSDT":   "BNBUSDTM",
}

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

	startTime, err := time.Parse("20060102", "20210622")
	if err != nil {
		logger.Fatal(err)
	}
	endTime, err := time.Parse("20060102", "20210704")
	if err != nil {
		logger.Fatal(err)
	}
	dateStrs := ""
	for i := startTime; i.Sub(endTime) <= 0; i = i.Add(time.Hour * 24) {
		dateStrs += i.Format("20060102,")
	}
	dateStrs = dateStrs[:len(dateStrs)-1]

	//quantileLookback := time.Hour * 72
	//quantileSubInterval := time.Hour
	//xMultiplier := 1.0
	//yMultiplier := 1.0
	//depthTakerImpact := 3000.0

	csvPath := "/Users/chenjilin/Downloads"

	for ySymbol, xSymbol := range symbolsMap {
		counter := 0
		//upSideJumpTimedTDigest := stream_stats.NewTimedTDigest(quantileLookback, quantileSubInterval)
		//downSideJumpTimedTDigest := stream_stats.NewTimedTDigest(quantileLookback, quantileSubInterval)
		//
		outputCsv, err := os.OpenFile(path.Join(csvPath, xSymbol+"-"+ySymbol+"-depth.csv"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0775)
		if err != nil {
			logger.Debugf("%v", err)
			return
		}
		//
		//downSideJump := 0.0
		//upSideJump := 0.0

		//xDepth := &kucoin_usdtfuture.Depth5{}
		var yDepth *binance_usdtfuture.Depth5
		//var yLastDepth *binance_usdtfuture.Depth5
		//var xWalkedDepth *common.WalkedDepthBBMAA
		//var yWalkedDepth *common.WalkedDepthBBMAA
		////var xLastWalkedDepth *common.WalkedDepthBBMAA
		//var yLastWalkedDepth *common.WalkedDepthBBMAA
		mircoPriceTimedDelta := stream_stats.NewTimedDelta(time.Second * 300)
		//bidTimedDelta := stream_stats.NewTimedDelta(time.Second * 300)
		mircoPriceTimedTD := stream_stats.NewTimedTDigest(time.Hour*24, time.Minute*5)
		//bidTimedTD := stream_stats.NewTimedTDigest(time.Hour, time.Minute*5)
		//downSideJumpTimedTDigest := stream_stats.NewTimedTDigest(quantileLookback, quantileSubInterval)
		for _, dateStr := range strings.Split(dateStrs, ",") {
			logger.Debugf("/Users/chenjilin/MarketData/bnuf-kcuf-depth5/%s/%s-%s,%s.depth5.jl.gz", dateStr, dateStr, ySymbol, xSymbol)
			file, err := os.Open(
				fmt.Sprintf("/Users/chenjilin/MarketData/bnuf-kcuf-depth5/%s/%s-%s,%s.depth5.jl.gz", dateStr, dateStr, ySymbol, xSymbol),
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
				if msg[0] == 'K' {
					continue
					//err = kucoin_usdtfuture.ParseDepth5(msg[1:], xDepth)
					//if err != nil {
					//	//logger.Debugf("%v", err)
					//	continue
					//}
					////xLastWalkedDepth = xWalkedDepth
					//xWalkedDepth = new(common.WalkedDepthBBMAA)
					//err = common.WalkDepthBBMAA(xDepth, xMultiplier, depthTakerImpact, xWalkedDepth)
					//if err != nil {
					//	logger.Debugf("%v", err)
					//	continue
					//}
				} else if msg[0] == 'B' {
					//yLastDepth = yDepth
					yDepth = new(binance_usdtfuture.Depth5)
					err = binance_usdtfuture.ParseDepth5(msg[1:], yDepth)
					if err != nil {
						logger.Debugf("%v", err)
						continue
					}
					mircoPriceTimedDelta.Insert(
						yDepth.EventTime,
						(yDepth.Asks[0][0]*yDepth.Bids[0][1]+yDepth.Bids[0][0]*yDepth.Asks[0][1])/(yDepth.Bids[0][1]+yDepth.Asks[0][1]),
					)
					_ = mircoPriceTimedTD.Insert(yDepth.EventTime, mircoPriceTimedDelta.Delta())
					//yLastWalkedDepth = yWalkedDepth
					//yWalkedDepth = new(common.WalkedDepthBBMAA)
					//err = common.WalkDepthBBMAA(yDepth, yMultiplier, depthTakerImpact, yWalkedDepth)
					//if err != nil {
					//	logger.Debugf("%v", err)
					//	continue
					//}
					if counter%100 == 0 {
						fields := make(map[string]interface{})
						fields["mircoPriceTimedDelta"] = mircoPriceTimedDelta.Delta()
						fields["mircoPriceTimedDelta995"] = mircoPriceTimedTD.Quantile(0.995)
						fields["mircoPriceTimedDelta95"] = mircoPriceTimedTD.Quantile(0.95)
						fields["mircoPriceTimedDelta80"] = mircoPriceTimedTD.Quantile(0.80)
						fields["mircoPriceTimedDelta20"] = mircoPriceTimedTD.Quantile(0.20)
						fields["mircoPriceTimedDelta05"] = mircoPriceTimedTD.Quantile(0.05)
						fields["mircoPriceTimedDelta005"] = mircoPriceTimedTD.Quantile(0.005)
						fields["mircoPrice"] = (yDepth.Asks[0][0]*yDepth.Bids[0][1]+yDepth.Bids[0][0]*yDepth.Asks[0][1])/(yDepth.Bids[0][1]+yDepth.Asks[0][1])
						pt, err := client.NewPoint(
							"momentum-depth5",
							map[string]string{
								"xSymbol": xSymbol,
							},
							fields,
							yDepth.EventTime,
						)
						if err == nil {
							iw.PointCh <- pt
						}
					}
				}

				//if yDepth != nil && yLastDepth != nil {
				//bidSize := yLastDepth.Bids[0][1] + yLastDepth.Bids[1][1] + yLastDepth.Bids[2][1] + yLastDepth.Bids[3][1] + yLastDepth.Bids[4][1]
				//askSize := yLastDepth.Asks[0][1] + yLastDepth.Asks[1][1] + yLastDepth.Asks[2][1] + yLastDepth.Asks[3][1] + yLastDepth.Asks[4][1]
				//data := fmt.Sprintf(
				//	"%d,%.6f,%.6f,%.6f,%.6f\n",
				//	yLastDepth.EventTime.UnixNano(),
				//	(yLastDepth.Bids[0][0]+yLastDepth.Asks[0][0])*0.5,
				//	yLastDepth.Asks[0][0]-yLastDepth.Bids[0][0],
				//	bidSize/(bidSize+askSize),
				//	(yDepth.Bids[0][0]+yDepth.Asks[0][0])*0.5,
				//)
				//_, _ = outputCsv.Write(([]byte)(data))
				//downSideJump = (yWalkedDepth.BidPrice - yLastWalkedDepth.BidPrice) / yLastWalkedDepth.BidPrice
				//upSideJump = (yWalkedDepth.AskPrice - yLastWalkedDepth.AskPrice) / yLastWalkedDepth.AskPrice
				//
				//_ = downSideJumpTimedTDigest.Insert(yWalkedDepth.Time, downSideJump)
				//_ = upSideJumpTimedTDigest.Insert(yWalkedDepth.Time, upSideJump)
				//if counter%1000 == 0 {
				//	fields := make(map[string]interface{})
				//	fields["downSideJumpTop95"] = downSideJumpTimedTDigest.Quantile(0.95)
				//	fields["downSideJumpTop50"] = downSideJumpTimedTDigest.Quantile(0.50)
				//	fields["downSideJumpTop05"] = downSideJumpTimedTDigest.Quantile(0.05)
				//	fields["upSideJumpTop95"] = upSideJumpTimedTDigest.Quantile(0.95)
				//	fields["upSideJumpTop50"] = upSideJumpTimedTDigest.Quantile(0.50)
				//	fields["upSideJumpTop05"] = upSideJumpTimedTDigest.Quantile(0.05)
				//	fields["downSideJump"] = downSideJump
				//	fields["upSideJump"] = upSideJump
				//	fields["yMidPrice"] = yWalkedDepth.MidPrice
				//	pt, err := client.NewPoint(
				//		"momentum-depth5",
				//		map[string]string{
				//			"xSymbol": xSymbol,
				//		},
				//		fields,
				//		yWalkedDepth.Time,
				//	)
				//	if err == nil {
				//		iw.PointCh <- pt
				//	}
				//}
				//}
			}
			_ = gr.Close()
			_ = file.Close()
		}
		err = outputCsv.Close()
		if err != nil {
			logger.Debugf("%v", err)
			continue
		}
	}

}

