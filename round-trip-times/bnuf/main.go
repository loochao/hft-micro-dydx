package main

import (
	"context"
	"fmt"
	binance_usdtfuture "github.com/geometrybase/hft-micro/binance-usdtfuture"
	"github.com/geometrybase/hft-micro/common"
	"github.com/montanaflynn/stats"
	"time"
)

func main() {
	api, err := binance_usdtfuture.NewAPI(&common.Credentials{}, "")
	if err != nil {
		panic(err)
	}
	fmt.Printf("\n\n")
	counter := 0
	rtts := make([]float64, 0)
	for counter < 100 {
		counter++
		startTime := time.Now()
		_, err = api.GetServerTime(context.Background())
		if err != nil {
			fmt.Printf("error %v", err)
		} else {
			diff := time.Now().Sub(startTime)
			fmt.Printf("BNUF %2d %v\n", counter, diff)
			rtts = append(rtts, diff.Seconds()*1000)
			counter++
		}
		time.Sleep(time.Second)
	}
	mean, err := stats.Mean(rtts)
	if err != nil {
		panic(err)
	}
	std, err := stats.StandardDeviation(rtts)
	if err != nil {
		panic(err)
	}
	median, err := stats.Median(rtts)
	if err != nil {
		panic(err)
	}
	max, err := stats.Max(rtts)
	if err != nil {
		panic(err)
	}
	min, err := stats.Max(rtts)
	if err != nil {
		panic(err)
	}
	fmt.Printf("\n\n")
	fmt.Printf("BNUF RTT MEAN=%.4fms STD=%.4fms MEDIAN=%.4fms MIN=%.4fms MAX=%.4fms", mean, std, median, min, max)
	fmt.Printf("\n\n")
}
