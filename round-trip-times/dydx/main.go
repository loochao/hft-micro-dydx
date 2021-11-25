package main

import (
	"context"
	"fmt"
	dydx_usdfuture "github.com/geometrybase/hft-micro/dydx-usdfuture"
	"github.com/geometrybase/hft-micro/logger"
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
			logger.Debugf("%v", err)
		}else{
			totalTime += time.Now().Sub(startTime)
			counter ++
		}
	}
	fmt.Printf("\n\n")
	fmt.Printf("RTT %dus", totalTime/time.Duration(counter)/time.Microsecond)
	fmt.Printf("\n\n")
}
