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
	symbols := "TRXUSDT"
	dateStr := "20210428"
	for _, symbol := range strings.Split(symbols, ",") {
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
		var lastSellPrice, lastBuyPrice float64
		var midPrice float64
		var buyTWM = common.NewTimedWeightedMean(time.Second * 60)
		var sellTWM = common.NewTimedWeightedMean(time.Second * 60)
		entrySize := 0.0
		entryPrice := 0.0
		netWorth := 1.0
		entryStep := 0.1
		entryTarget := 1.0
		lastEntryPrice := 1.0
		lastTradeTime := time.Now()
		tradeInterval := time.Second * 10
		commission := -0.0002
		unRealisedPnl := 0.0
		tradeSilentTime := time.Unix(0, 0)
		tradeSilent := time.Minute * 5
		for scanner.Scan() {
			d, err := bnswap.ParseTrade(scanner.Bytes())
			if err != nil {
				logger.Debugf("bnswap.ParseDepth20 error %v", err)
				continue
			}
			if d.IsTheBuyerTheMarketMaker {
				lastSellPrice = d.Price
				sellTWM.Insert(d.EventTime, d.Quantity, d.Price)
			} else {
				lastBuyPrice = d.Price
				buyTWM.Insert(d.EventTime, d.Quantity, d.Price)
			}

			if lastBuyPrice == 0 || lastSellPrice == 0 {
				continue
			}

			if buyTWM.TotalWeight > sellTWM.TotalWeight &&
				buyTWM.TotalWeight < 1.25*sellTWM.TotalWeight &&
				d.EventTime.Sub(tradeSilentTime) > 0 {
				if entrySize < 0 {
					pnlPct := -(lastBuyPrice - entryPrice) / entryPrice
					netWorth += pnlPct*-entrySize + (-entrySize)*commission
					netWorth += entryStep * commission
					entryPrice = lastBuyPrice
					lastEntryPrice = lastBuyPrice
					lastTradeTime = d.EventTime
					if entrySize <= -entryTarget &&
						pnlPct > 0.001 {
						logger.Debugf("CLOSE SHORT ADD SILENT %f %f", entrySize, pnlPct)
						entrySize = 0
						tradeSilentTime = d.EventTime.Add(tradeSilent)
					} else {
						netWorth += entryStep * commission
						entrySize = entryStep
					}
				} else if entrySize == 0 {
					netWorth += entryStep * commission
					entryPrice = lastBuyPrice
					lastEntryPrice = lastBuyPrice
					entrySize = entryStep
					lastTradeTime = d.EventTime
				} else if (lastBuyPrice-lastEntryPrice)/lastEntryPrice > 0.0005 &&
					d.EventTime.Sub(lastTradeTime) > tradeInterval &&
					entrySize < entryTarget {
					netWorth += entryStep * commission
					entryPrice = (lastBuyPrice*entryStep + entrySize*entryPrice) / (entryStep + entrySize)
					lastEntryPrice = lastBuyPrice
					entrySize += entryStep
					lastTradeTime = d.EventTime
					//} else if (lastSellPrice-lastEntryPrice)/lastEntryPrice < -0.002 &&
					//	d.EventTime.Sub(lastTradeTime) > tradeInterval {
					//	netWorth += (lastSellPrice-entryPrice)/entryPrice*entrySize + entrySize*commission
					//	entryPrice = lastSellPrice
					//	lastEntryPrice = lastSellPrice
					//	entrySize = entryStep
					//	lastTradeTime = d.EventTime.Add(tradeInterval*4)
					//} else if entrySize > entryTarget &&
					//	d.EventTime.Sub(lastTradeTime) > tradeInterval {
					//	netWorth += (lastSellPrice-entryPrice)/entryPrice*entrySize + entrySize*commission
					//	entryPrice = lastSellPrice
					//	lastEntryPrice = lastSellPrice
					//	entrySize = entryStep
					//	lastTradeTime = d.EventTime.Add(time.Hour*999)
				}
			} else if sellTWM.TotalWeight > buyTWM.TotalWeight &&
				sellTWM.TotalWeight < 1.5*buyTWM.TotalWeight &&
				d.EventTime.Sub(tradeSilentTime) > 0 {
				if entrySize > 0 {
					pnlPct := (lastSellPrice - entryPrice) / entryPrice
					netWorth += pnlPct*entrySize + entrySize*commission
					entryPrice = lastSellPrice
					lastEntryPrice = lastSellPrice
					lastTradeTime = d.EventTime
					if entrySize >= entryTarget &&
						pnlPct > 0.001 {
						logger.Debugf("CLOSE LONG AND ADD SILENT %f %f", entrySize, pnlPct)
						entrySize = 0
						tradeSilentTime = d.EventTime.Add(tradeSilent)
					} else {
						netWorth += entryStep * commission
						entrySize = -entryStep
					}
				} else if entrySize == 0 {
					netWorth += entryStep * commission
					entryPrice = lastSellPrice
					lastEntryPrice = lastSellPrice
					entrySize = -entryStep
					lastTradeTime = d.EventTime
				} else if (lastSellPrice-lastEntryPrice)/lastEntryPrice < -0.0005 &&
					d.EventTime.Sub(lastTradeTime) > tradeInterval &&
					entrySize > -entryTarget {
					netWorth += entryStep * commission
					entryPrice = (-entryStep*lastSellPrice + entrySize*entryPrice) / (-entryStep + entrySize)
					lastEntryPrice = lastSellPrice
					entrySize -= entryStep
					lastTradeTime = d.EventTime
					//} else if (lastBuyPrice-lastEntryPrice)/lastEntryPrice > 0.002 &&
					//	d.EventTime.Sub(lastTradeTime) > tradeInterval {
					//	netWorth += (lastBuyPrice-entryPrice)/entryPrice*entrySize + -entrySize*commission
					//	entryPrice = lastBuyPrice
					//	lastEntryPrice = lastBuyPrice
					//	entrySize = -entryStep
					//	lastTradeTime = d.EventTime.Add(4*tradeInterval)
				}
			}

			midPrice = (lastSellPrice + lastBuyPrice) * 0.5
			unRealisedPnl = 0.0
			if entrySize > 0 {
				unRealisedPnl = (lastSellPrice - entryPrice) / entryPrice * entrySize
			} else {
				unRealisedPnl = (lastBuyPrice - entryPrice) / entryPrice * entrySize
			}

			fields := make(map[string]interface{})
			fields["midPrice"] = midPrice
			fields["netWorth"] = netWorth + unRealisedPnl
			fields["entrySize"] = entrySize
			fields["unRealisedPnl"] = unRealisedPnl
			fields["spread"] = (buyTWM.Median() - sellTWM.Median()) / midPrice
			fields["lastSellPrice"] = lastSellPrice
			fields["lastBuyPrice"] = lastBuyPrice
			fields["vMeanSellPrice"] = sellTWM.Median()
			fields["vMeanBuyPrice"] = buyTWM.Median()
			fields["totalBuyValue"] = buyTWM.TotalValue
			fields["totalBuyWeight"] = buyTWM.TotalWeight
			fields["totalSellValue"] = sellTWM.TotalValue
			fields["totalSellWeight"] = sellTWM.TotalWeight
			pt, err := client.NewPoint(
				"bnswap-trade",
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
		logger.Debugf("%s %f", symbol, netWorth)
		time.Sleep(time.Second)
	}
}
