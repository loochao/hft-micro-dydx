package main

import (
	"context"
	"fmt"
	dydx_usdfuture "github.com/geometrybase/hft-micro/dydx-usdfuture"
	"time"
)

func main() {
	api, err := dydx_usdfuture.NewAPI(dydx_usdfuture.Credentials{}, "")
	if err != nil {
		panic(err)
	}

	totalTime := time.Duration(0)
	counter := 0
	for counter < 100 {
		counter++
		startTime := time.Now()
		_, err = api.GetServerTime(context.Background())
		if err != nil {
			fmt.Printf("error %v", err)
		} else {
			diff := time.Now().Sub(startTime)
			fmt.Printf("%2d %v\n", counter, diff)
			totalTime += diff
			counter++
		}
		time.Sleep(time.Second)
	}
	fmt.Printf("\n\n")
	fmt.Printf("RTT %dus", totalTime/time.Duration(counter)/time.Microsecond)
	fmt.Printf("\n\n")
}
