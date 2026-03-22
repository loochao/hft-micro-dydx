package main

import (
	"bufio"
	"compress/gzip"
	"context"
	"fmt"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"os"
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

	symbol := "FILUSDT"
	dateStr := "20210428"
	file, err := os.Open(
		fmt.Sprintf("/home/clu/MarketData/bnswap-trade/%s-%s.bnswap.trade.jl.gz", dateStr, symbol),
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
	buyVolume := common.NewTimedSum(time.Second*300)
	sellVolume := common.NewTimedSum(time.Second*300)
	for scanner.Scan() {
		trade, err := bnswap.ParseTrade(scanner.Bytes())
		if err != nil {
			logger.Fatal(err)
		}
		if trade.IsTheBuyerTheMarketMaker {
			sellVolume.Insert(trade.EventTime, trade.Quantity)
		}else{
			buyVolume.Insert(trade.EventTime, trade.Quantity)
		}
		if sellVolume.Sum() < 0 {
			logger.Debugf("%s", scanner.Bytes())
			logger.Fatalf("negative sell %v %v %v", sellVolume.Sum(), sellVolume.Len(), sellVolume.Range())
		}
		if buyVolume.Sum() < 0 {
			logger.Debugf("%s", scanner.Bytes())
			logger.Fatalf("negative buy %v %v %v", buyVolume.Sum(), buyVolume.Len(), buyVolume.Range())
		}
	}
	_ = gr.Close()
	_ = file.Close()
}

