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
	"github.com/geometrybase/hft-micro/tdigest"
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
	symbols := "FILUSDT"
	dateStrs := "20210428,20210429,20210430,20210501,20210502"
	binSize := 1000000.0
	minTradeValue := 1000.0
	for _, symbol := range strings.Split(symbols, ",") {
		timedMean := common.NewTimedMean(time.Hour*4)
		vpin := common.NewVPIN(binSize)
		mirTd, _ := tdigest.New()
		durationTd, _ := tdigest.New()
		sizeTD, _ := tdigest.New()
		var lastMir *float64
		var lastMirTrigger *time.Time
		for _, dateStr := range strings.Split(dateStrs, ",") {
			file, err := os.Open(
				fmt.Sprintf("/Users/chenjilin/MarketData/bnswap-trade/%s-%s.bnswap.trade.jl.gz", dateStr, symbol),
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
				d, err := bnswap.ParseTrade(scanner.Bytes())
				if err != nil {
					logger.Debugf("bnswap.ParseDepth20 error %v", err)
					continue
				}
				_ = sizeTD.Add(d.Price*d.Quantity)
				if d.Price*d.Quantity < minTradeValue {
					continue
				}
				timedMean.Insert(d.EventTime, d.Price)
				vpin.Insert(d.Quantity, d.Price)
				mir := common.ComputeMIR(timedMean.Values())
				fields := make(map[string]interface{})
				if lastMirTrigger == nil || lastMir == nil{
					lastMir = &mir
					lastMirTrigger = &d.EventTime
				}else if *lastMir * mir <= 0 {
					if mir != 0 {
						duration := d.EventTime.Sub(*lastMirTrigger).Seconds()
						_ = durationTd.Add(duration)
						fields["durationMid"] = durationTd.Quantile(0.5)
						fields["duration"] = duration
						lastMirTrigger = &d.EventTime
						lastMir = &mir
					}
				}
				_ = mirTd.Add(mir)
				fields["lastPrice"] = d.Price
				fields["mir"] = mir
				fields["vpin"] = vpin.Imbalance()
				fields["qTop"] = mirTd.Quantile(0.05)
				fields["qBot"] = mirTd.Quantile(0.95)
				fields["qValue"] = sizeTD.Quantile(0.8)
				pt, err := client.NewPoint(
					"bnswap-trade-mir",
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
		}
		time.Sleep(time.Second)
	}
}
