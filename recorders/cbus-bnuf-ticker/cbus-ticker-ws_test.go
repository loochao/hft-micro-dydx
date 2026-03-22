package main

import (
	"context"
	"github.com/geometrybase/hft-micro/logger"
	"testing"
)

func TestNewCbusTickerWS(t *testing.T) {
	channels := make(map[string]chan *Message)
	channels["STORJ-USD"] = make(chan *Message, 1000)
	tickerWS := NewCbusTickerWS(
		context.Background(),
		"socks5://127.0.0.1:1083",
		channels,
	)
	for {
		select {
		case <-tickerWS.Done():
			return
		case m := <-channels["STORJ-USD"]:
			logger.Debugf("%v", m)
		}
	}
}
