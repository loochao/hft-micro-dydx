package main

import (
	"bufio"
	"compress/gzip"
	"context"
	"fmt"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/influx/client"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"strings"
	"time"
)

//func NewInfluxWriter(ctx context.Context, address, username, password, database string, batchSize int) (*InfluxWriter, error) {

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

	//symbols := `BTCUSDT,LTCUSDT,ETHUSDT,NEOUSDT,QTUMUSDT,EOSUSDT,ZRXUSDT,OMGUSDT,LRCUSDT,TRXUSDT,KNCUSDT,IOTAUSDT,LINKUSDT,CVCUSDT,ETCUSDT,ZECUSDT,BATUSDT,DASHUSDT,XMRUSDT,ENJUSDT,XRPUSDT,STORJUSDT,BTSUSDT,ADAUSDT,XLMUSDT,WAVESUSDT,ICXUSDT,RLCUSDT,IOSTUSDT,BLZUSDT,ONTUSDT,ZILUSDT,ZENUSDT,THETAUSDT,VETUSDT,RENUSDT,MATICUSDT,ATOMUSDT,FTMUSDT,CHZUSDT,ALGOUSDT,DOGEUSDT,ANKRUSDT,TOMOUSDT,BANDUSDT,XTZUSDT,KAVAUSDT,BCHUSDT,SOLUSDT,HNTUSDT,COMPUSDT,MKRUSDT,SXPUSDT,SNXUSDT,DOTUSDT,RUNEUSDT,BALUSDT,YFIUSDT,SRMUSDT,CRVUSDT,SANDUSDT,OCEANUSDT,LUNAUSDT,RSRUSDT,TRBUSDT,EGLDUSDT,BZRXUSDT,KSMUSDT,SUSHIUSDT,YFIIUSDT,BELUSDT,UNIUSDT,AVAXUSDT,FLMUSDT,ALPHAUSDT,NEARUSDT,AAVEUSDT,FILUSDT,CTKUSDT,AXSUSDT,AKROUSDT,SKLUSDT,GRTUSDT,1INCHUSDT,LITUSDT,RVNUSDT,SFPUSDT,REEFUSDT,DODOUSDT,COTIUSDT,CHRUSDT,ALICEUSDT,HBARUSDT,MANAUSDT,STMXUSDT,UNFIUSDT,XEMUSDT,CELRUSDT,HOTUSDT,ONEUSDT,LINAUSDT,DENTUSDT,MTLUSDT,OGNUSDT,NKNUSDT,DGBUSDT`
	symbols := "OMGUSDT"
	dateStr := "20210428"
	for _, symbol := range strings.Split(symbols, ",") {
		file, err := os.Open(
			fmt.Sprintf("/Users/chenjilin/MarketData/bnswap-depth20/%s-%s.depth20.jl.gz", dateStr, symbol),
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
		counter := 0
		value := 1.0
		entrySize := 0.0
		entryPrice := 0.0
		levelDecay := 0.995
		midPrice := 0.0
		mircoPrice := 0.0
		bidAskRatio := 1.0
		askBidRatio := 1.0
		bidSize := 0.0
		askSize := 0.0
		commission := -0.0002
		decay := 1.0
		bidAskRatioTM := common.NewTimedMedian(time.Second * 300)
		askBidRatioTM := common.NewTimedMedian(time.Second * 300)
		for scanner.Scan() {
			d, err := bnswap.ParseDepth20(scanner.Bytes())
			if err != nil {
				logger.Debugf("bnswap.ParseDepth20 error %v", err)
				continue
			}
			counter++
			bidSize = 0.0
			askSize = 0.0
			decay = 1.0
			for i := 0; i < 20; i++ {
				bidSize += d.Bids[i][1] * decay
				askSize += d.Asks[i][1] * decay
				decay *= levelDecay
			}
			mircoPrice = (d.Asks[0][0]*d.Bids[0][1] + d.Bids[0][0]*d.Asks[0][1]) / (d.Bids[0][1] + d.Asks[0][1])
			midPrice = (d.Bids[0][0] + d.Asks[0][0]) / 2
			bidAskRatio = bidAskRatioTM.Insert(bidSize/askSize, d.EventTime)
			askBidRatio = askBidRatioTM.Insert(askSize/bidSize, d.EventTime)
			if bidAskRatio > 2 {
				if entrySize == -1 {
					value += (entryPrice-midPrice)/entryPrice + 2*commission
					logger.Debugf("SWAP TO LONG %f -> %f %f", entryPrice, midPrice, (entryPrice-midPrice)/entryPrice + 2*commission)
					entryPrice = midPrice
					entrySize = 1
				} else if entrySize == 0 {
					value += commission
					entryPrice = midPrice
					entrySize = 1
				}
			} else if askBidRatio > 2 {
				if entrySize == 1 {
					value += (midPrice-entryPrice)/entryPrice + 2*commission
					logger.Debugf("SWAP TO SHORT %f -> %f %f", entryPrice, midPrice, (midPrice-entryPrice)/entryPrice + 2*commission)
					entryPrice = midPrice
					entrySize = -1
				} else if entrySize == 0 {
					value += commission
					entryPrice = midPrice
					entrySize = -1
				}
			} else if entrySize > 0 && bidAskRatio < 1 {
				value += (midPrice-entryPrice)/entryPrice + commission
				logger.Debugf("CLOSE LONG %f -> %f %f", entryPrice, midPrice, (midPrice-entryPrice)/entryPrice+commission)
				entryPrice = midPrice
				entrySize = 0
			} else if entrySize < 0 && askBidRatio < 1 {
				value += (entryPrice-midPrice)/entryPrice + commission
				logger.Debugf("CLOSE SHORT %f -> %f %f", entryPrice, midPrice, (entryPrice-midPrice)/entryPrice + commission)
				entryPrice = midPrice
				entrySize = 0
			}
			fields := make(map[string]interface{})
			fields["bidSize"] = bidSize
			fields["askSize"] = askSize
			fields["mircoPrice"] = mircoPrice
			fields["midPrice"] = midPrice
			fields["value"] = value
			fields["bidAskRatio"] = bidAskRatio
			fields["askBidRatio"] = askBidRatio
			fields["entrySize"] = entrySize
			if entrySize != 0 {
				fields["entryPrice"] = entryPrice
			}else{
				fields["entryPrice"] = midPrice
			}
			pt, err := client.NewPoint(
				"bnswap-lob",
				map[string]string{
					"symbol": symbol,
				},
				fields,
				d.EventTime,
			)
			iw.PointCh <- pt
		}
		_ = gr.Close()
		_ = file.Close()
		logger.Debugf("%s %f", symbol, value)
		time.Sleep(time.Second)
	}
}
