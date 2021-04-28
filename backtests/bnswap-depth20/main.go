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
		1000,
	)
	if err != nil {
		panic(err)
	}
	defer iw.Stop()

	symbol := "CRVUSDT"
	file, err := os.Open(
		fmt.Sprintf("/Users/chenjilin/MarketData/bnswap-depth20/2021042806-%s.depth20.jl.gzip", symbol),
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
	defer gr.Close()
	defer file.Close()

	scanner := bufio.NewScanner(gr)
	counter := 0
	for scanner.Scan() {
		d, err := bnswap.ParseDepth20(scanner.Bytes())
		if err != nil {
			logger.Debugf("bnswap.ParseDepth20 error %v", err)
			continue
		}
		counter ++
		bidSize := 0.0
		askSize := 0.0
		bidImpact := 0.0
		askImpact := 0.0
		for i := 0; i < 20; i ++ {
			bidSize += d.Bids[i][1]
			askSize += d.Asks[i][1]
			bidImpact += d.Bids[i][0]*d.Asks[i][1]
			askImpact += d.Asks[i][0]*d.Bids[i][1]
		}
		fields := make(map[string]interface{})
		fields["bidSize"] = bidSize
		fields["askSize"] = askSize
		fields["impactPrice"] = (bidImpact+askImpact)/(bidSize+askSize)
		fields["midPrice"] = (d.Bids[0][0]*d.Bids[0][1] + d.Asks[0][0]*d.Asks[0][1])/(d.Bids[0][1]+d.Asks[0][1])
		pt, err := client.NewPoint(
			"bnswap-depth20",
			map[string]string{
				"symbol": symbol,
			},
			fields,
			d.EventTime,
		)
		iw.PointCh <- pt
	}
}
