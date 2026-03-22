package main

import (
	"context"
	"fmt"
	dydx_usdfuture "github.com/geometrybase/hft-micro/dydx-usdfuture"
	"github.com/montanaflynn/stats"
	"time"
)

func main() {
	api, err := dydx_usdfuture.NewAPI(dydx_usdfuture.Credentials{}, "")
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
			fmt.Printf("DYDX %2d %v\n", counter, diff)
			rtts = append(rtts, diff.Seconds()*1000)
			counter++
		}
		time.Sleep(time.Second)
	}
	mean, err := stats.Mean(rtts)
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
	min, err := stats.Min(rtts)
	if err != nil {
		panic(err)
	}
	std, err := stats.StandardDeviation(rtts)
	if err != nil {
		panic(err)
	}
	fmt.Printf("\n\n")
	fmt.Printf("DYDX RTT MEAN=%.4f MEDIAN=%.4f MIN=%.4f MAX=%.4f STD=%.4f", mean, median, min, max, std)
	fmt.Printf("\n\n")
}
