package kucoin_usdtfuture

import (
	"context"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"net/http"
	"time"
)

func StreamRawFundingRate(
	ctx context.Context,
	proxyAddress string,
	source []byte,
	channels map[string]chan *common.RawMessage,
) {

	api, err := NewAPI("", "", "", proxyAddress)
	if err != nil {
		logger.Fatal(err)
	}

	interval := time.Minute * 5
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	afterFrTimer := time.NewTimer(time.Now().Truncate(time.Hour * 4).Add(time.Hour*4 + time.Second).Sub(time.Now()))
	defer afterFrTimer.Stop()

	logSilentTime := time.Now()

	var message *common.RawMessage
	index := -1
	pool := [4096]*common.RawMessage{}
	for i := 0; i < 4096; i++ {
		pool[i] = &common.RawMessage{
			Prefix: source,
		}
	}
	var msg []byte
	var subCtx context.Context

	for {
		select {
		case <-ctx.Done():
			return
		case <-afterFrTimer.C:
			for symbol, ch := range channels {
				subCtx, _ = context.WithTimeout(ctx, time.Minute)
				msg, err = api.GetRawData(subCtx, http.MethodGet, fmt.Sprintf("/api/v1/funding-rate/%s/current", symbol), nil)
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
				msg, err = api.GetRawData(subCtx, http.MethodGet, fmt.Sprintf("/api/v1/funding-rate/%s/current", symbol), nil)
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
