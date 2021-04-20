package main

import (
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

var ch = make(chan interface{})

func go1() {
	for {
		select {
		case ch <- nil:
		}
	}
}

func go2() {
	for {
		select {
		case ch <- nil:
		}
	}
}

func main() {
	fundingTime1 := time.Now().Truncate(time.Hour*8)
	fundingTime2 := time.Now().Truncate(time.Hour*8).Add(time.Hour*8)

	logger.Debugf("%v", time.Now().Sub(fundingTime1))
	logger.Debugf("%v", fundingTime2.Sub(time.Now()))
	//ch1 := make(chan interface{})
	//ch2 := make(chan interface{})
	//close(ch1)
	//select {
	//case <-ch1:
	//	logger.Debugf("CLOSE CH")
	//case <-time.After(time.Millisecond):
	//	logger.Debugf("TIME AFTER")
	//case ch2 <- nil:
	//	logger.Debugf("OUTPUT CH")
	//}
	//go go1()
	//go go2()
	//t := time.NewTimer(time.Second * 5)
	//for {
	//	select {
	//	case <-t.C:
	//		logger.Debugf("EXIT")
	//		return
	//	case <-ch:
	//	}
	//}
}

//结论 和主线程共存运行的，可以不管退出时chan会不会堵，但是在程序中要重启的go routine 一定要有context, 不然有可能因为发不出消息卡住
