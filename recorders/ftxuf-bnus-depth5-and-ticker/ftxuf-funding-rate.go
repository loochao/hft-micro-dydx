package main

import (
	"context"
	"fmt"
	ftxuf "github.com/geometrybase/hft-micro/ftx-usdfuture"
	"github.com/geometrybase/hft-micro/logger"
	"net/http"
	"time"
)

func streamFtxufFundingRate(ctx context.Context, api *ftxuf.API,channels map[string]chan *Message) {

	interval := time.Minute*5
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	afterFrTimer := time.NewTimer(time.Now().Truncate(time.Hour * 4).Add(time.Hour*4 + time.Second).Sub(time.Now()))
	defer afterFrTimer.Stop()

	logSilentTime := time.Now()

	var message *Message
	index := -1
	pool := [4096]*Message{}
	for i := 0; i < 4096; i++ {
		pool[i] = &Message{
			Source: []byte{'X', 'F'},
		}
	}
	var err error
	var msg []byte
	var subCtx context.Context

	for {
		select {
		case <-ctx.Done():
			return
		case <-afterFrTimer.C:
			for symbol, ch := range channels {
				subCtx, _ = context.WithTimeout(ctx, time.Minute)
				msg, err = api.SendRawHTTPRequest(subCtx, http.MethodGet, fmt.Sprintf("/futures/%s/stats", symbol), nil)
				if err != nil {
					logger.Debugf("api.GetCurrentFundingRate error %v", err)
				} else {
					index++
					if index == 4096 {
						index = 0
					}
					message = pool[index]
					message.Time = time.Now().UnixNano()
					message.Data = msg
					select {
					case ch <- message:
					default:
						if time.Now().Sub(logSilentTime) > 0 {
							logger.Debugf("ch <- message failed %s len(ch) = %d", symbol, len(ch))
							logSilentTime = time.Now().Add(time.Minute)
						}
					}
				}
			}
			afterFrTimer.Reset(time.Now().Truncate(time.Hour * 4).Add(time.Hour*4 + time.Second).Sub(time.Now()))
			break
		case <-timer.C:
			for symbol, ch := range channels {
				subCtx, _ = context.WithTimeout(ctx, time.Minute)
				msg, err = api.SendRawHTTPRequest(subCtx, http.MethodGet, fmt.Sprintf("/futures/%s/stats", symbol), nil)
				if err != nil {
					logger.Debugf("api.GetCurrentFundingRate error %v", err)
				} else {
					index++
					if index == 4096 {
						index = 0
					}
					message = pool[index]
					message.Time = time.Now().UnixNano()
					message.Data = msg
					select {
					case ch <- message:
					default:
						if time.Now().Sub(logSilentTime) > 0 {
							logger.Debugf("ch <- message failed %s len(ch) = %d", symbol, len(ch))
							logSilentTime = time.Now().Add(time.Minute)
						}
					}
				}
			}
			timer.Reset(time.Now().Truncate(interval).Add(interval).Sub(time.Now()))
			break
		}
	}
}
