package main

import (
	"bufio"
	"compress/gzip"
	"context"
	bnuf "github.com/geometrybase/hft-micro/binance-usdtfuture"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/influx/client"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"time"
)


func main() {
	ctx := context.Background()
	iw, err := common.NewInfluxWriter(
		ctx,
		"http://localhost:8086",
		"",
		"",
		"hft",
		100,
	)
	if err != nil {
		panic(err)
	}
	defer iw.Stop()

	usdtDepth := &bnuf.Depth5{}
	busdDepth := &bnuf.Depth5{}
	//usdtSymbol := "BTCUSDT"
	//busdSymbol := "BTCBUSD"
	//tempDepth := &bnuf.Depth5{}
	//depthFile, err := os.Open(
	//	"/Users/chenjilin/Downloads/20210620-BTCUSDT,BTCBUSD.depth5.jl.gz",
	//)

	usdtSymbol := "ETHUSDT"
	busdSymbol := "ETHBUSD"
	tempDepth := &bnuf.Depth5{}
	depthFile, err := os.Open(
		"/Users/chenjilin/Downloads/20210620-ETHUSDT,ETHBUSD.depth5.jl.gz",
	)
	if err != nil {
		logger.Debugf("os.Open() error %v", err)
		return
	}
	depthGzReader, err := gzip.NewReader(depthFile)
	if err != nil {
		logger.Debugf("gzip.NewReader(depthFile) error %v", err)
		return
	}
	depthScanner := bufio.NewScanner(depthGzReader)

	busdBidWalk := common.NewTimedWalkingDistance(time.Second)
	usdtBidWalk := common.NewTimedWalkingDistance(time.Second)
	busdAskWalk := common.NewTimedWalkingDistance(time.Second)
	usdtAskWalk := common.NewTimedWalkingDistance(time.Second)
	for depthScanner.Scan() {
		err = bnuf.ParseDepth5(depthScanner.Bytes(), tempDepth)
		if err != nil {
			logger.Debugf("bnuf.ParseDepth5 error %v", err)
			continue
		}
		if tempDepth.Symbol == usdtSymbol {
			*usdtDepth = *tempDepth
			usdtBidWalk.Insert(usdtDepth.EventTime, usdtDepth.Bids[0][0])
			usdtAskWalk.Insert(usdtDepth.EventTime, usdtDepth.Asks[0][0])
			continue
		} else if tempDepth.Symbol == busdSymbol {
			*busdDepth = *tempDepth
			busdBidWalk.Insert(busdDepth.EventTime, busdDepth.Bids[0][0])
			busdAskWalk.Insert(busdDepth.EventTime, busdDepth.Asks[0][0])
		}else{
			continue
		}
		fields := make(map[string]interface{})
		fields["usdtBidWalk"] = usdtBidWalk.WalkDistance()
		fields["usdtAskWalk"] = usdtAskWalk.WalkDistance()
		fields["busdBidWalk"] = busdBidWalk.WalkDistance()
		fields["busdAskWalk"] = busdAskWalk.WalkDistance()
		fields["busdBidPrice"] = busdDepth.Bids[0][0]
		fields["usdtBidPrice"] = usdtDepth.Bids[0][0]
		pt, err := client.NewPoint(
			"bnswap-usdt-busd-lead-lag",
			map[string]string{
				"bSymbol": busdSymbol,
			},
			fields,
			busdDepth.EventTime,
		)
		if err != nil {
			logger.Fatal(err)
		}
		iw.PointCh <- pt
	}
	_ = depthGzReader.Close()
	_ = depthFile.Close()
}
