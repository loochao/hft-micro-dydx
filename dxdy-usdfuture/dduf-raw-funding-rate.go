package dxdy_usdtfuture

import (
	"context"
	"encoding/json"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func StreamRawFundingRate(
	ctx context.Context,
	proxyAddress string,
	prefix []byte,
	channels map[string]chan *common.RawMessage,
) {
	api, err := NewAPI(&common.Credentials{}, proxyAddress)
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
			Prefix: prefix,
		}
	}
	var msg []byte
	var subCtx context.Context
	var markets map[string]Market

	for {
		select {
		case <-ctx.Done():
			return
		case <-afterFrTimer.C:
			timer.Reset(time.Now().Truncate(interval).Add(interval).Sub(time.Now()))
			subCtx, _ = context.WithTimeout(ctx, time.Minute)
			markets, err = api.GetMarkets(subCtx)
			for _, market := range markets {
				if ch, ok := channels[market.Market]; ok {
					msg, err = json.Marshal(market)
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
								logger.Debugf("ch <- message failed %s len(ch) = %d", market.Market, len(ch))
								logSilentTime = time.Now().Add(time.Minute)
							}
						}
					}
				}
			}
			afterFrTimer.Reset(time.Now().Truncate(time.Hour * 4).Add(time.Hour*4 + time.Second).Sub(time.Now()))
			break
		case <-timer.C:
			subCtx, _ = context.WithTimeout(ctx, time.Minute)
			markets, err = api.GetMarkets(subCtx)
			for _, market := range markets {
				if ch, ok := channels[market.Market]; ok {
					msg, err = json.Marshal(market)
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
								logger.Debugf("ch <- message failed %s len(ch) = %d", market.Market, len(ch))
								logSilentTime = time.Now().Add(time.Minute)
							}
						}
					}
				}
			}
			timer.Reset(time.Now().Truncate(interval).Add(interval).Sub(time.Now()))
			break
		}
	}
}
