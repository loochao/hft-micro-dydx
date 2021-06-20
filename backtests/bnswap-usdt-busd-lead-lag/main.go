package main

import (
	"bufio"
	"compress/gzip"
	"context"
	bnuf "github.com/geometrybase/hft-micro/binance-usdtfuture"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"os"
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

	//counter := 0
	logCounter := 0
	for depthScanner.Scan() {
		err = bnuf.ParseDepth5(depthScanner.Bytes(), tempDepth)
		if err != nil {
			logger.Debugf("bnuf.ParseDepth5 error %v", err)
			continue
		}
		if tempDepth.Symbol == usdtSymbol {
			*usdtDepth = *tempDepth
			//logger.Debugf("%v", busdDepth.EventTime.Sub(usdtDepth.EventTime))
		} else if tempDepth.Symbol == busdSymbol {
			*busdDepth = *tempDepth
			if math.Abs(busdDepth.Bids[0][0] - usdtDepth.Bids[0][0]) > 1.5 {
				logCounter = 20
			}
			//}
		}else{
			continue
		}
		if logCounter > 0 {
			logCounter--
			logger.Debugf("%v\t%f %f %f %f", busdDepth.EventTime.Sub(usdtDepth.EventTime), busdDepth.Bids[0][0] - usdtDepth.Bids[0][0], busdDepth.Asks[0][0], busdDepth.Bids[0][0],usdtDepth.Bids[0][0])
			if logCounter <= 0 {
				logger.Debugf("")
			}
		}


		//counter ++
		//if counter > 10000 {
		//	return
		//}

		//fields := make(map[string]interface{})
		//if positionSize > 0 {
		//	fields["netWorth"] = netWorth + positionSize*(bestBidPrice-positionCost)/positionCost
		//} else if positionSize < 0 {
		//	fields["netWorth"] = netWorth + positionSize*(bestAskPrice-positionCost)/positionCost
		//} else {
		//	fields["netWorth"] = netWorth
		//}
		//fields["positionSize"] = positionSize
		//if positionCost != 0 {
		//	fields["positionCost"] = positionCost
		//}
		//if lastFilledPrice != 0 {
		//	fields["lastFilledPrice"] = lastFilledPrice
		//}
		//fields["bestBidPrice"] = bestBidPrice
		//fields["bestAskPrice"] = bestAskPrice
		//fields["depthSize"] = depthSize
		//fields["depthTimedSizeDelta"] = depthTimedSizeDelta.Sum()
		//fields["depthDeltaRatio"] = depthDeltaRatio
		//fields["depthDeltaRatioQ9995"] = depthDeltaRatioTD.Quantile(0.9995)
		//fields["depthDeltaRatioQ995"] = depthDeltaRatioTD.Quantile(0.995)
		//fields["depthDeltaRatioQ99"] = depthDeltaRatioTD.Quantile(0.99)
		//fields["depthDeltaRatioQ95"] = depthDeltaRatioTD.Quantile(0.95)
		//fields["depthDeltaRatioQ80"] = depthDeltaRatioTD.Quantile(0.80)
		//
		//fields["depthDir"] = depthDir
		//fields["depthDirQ9995"] = depthDirTD.Quantile(0.9995)
		//fields["depthDirQ995"] = depthDirTD.Quantile(0.995)
		//fields["depthDirQ80"] = depthDirTD.Quantile(0.80)
		//fields["depthDirQ0005"] = depthDirTD.Quantile(0.0005)
		//fields["depthDirQ005"] = depthDirTD.Quantile(0.005)
		//fields["depthDirQ20"] = depthDirTD.Quantile(0.20)
		//pt, err := client.NewPoint(
		//	"bnswap-depth-racing",
		//	map[string]string{
		//		"symbol": symbol,
		//	},
		//	fields,
		//	d.EventTime,
		//)
		//iw.PointCh <- pt
	}
	_ = depthGzReader.Close()
	_ = depthFile.Close()
}
