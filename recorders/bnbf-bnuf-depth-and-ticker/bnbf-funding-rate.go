package main

import (
	"context"
	bnbf "github.com/geometrybase/hft-micro/binance-busdfuture"
	"github.com/geometrybase/hft-micro/logger"
	"strings"
	"time"
)

func streamBnbfFundingRate(ctx context.Context, api *bnbf.API,channels map[string]chan *Message) {

	interval := time.Minute*5
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	afterFrTimer := time.NewTimer(time.Now().Truncate(time.Hour * 4).Add(time.Hour*4 + time.Second).Sub(time.Now()))
	defer afterFrTimer.Stop()

	logSilentTime := time.Now()
	const bufferSize = 1024
	var message *Message
	index := -1
	pool := [bufferSize]*Message{}
	for i := 0; i < bufferSize; i++ {
		pool[i] = &Message{
			Source: []byte{'X', 'F'},
		}
	}
	var err error
	var msg []byte
	var subCtx context.Context
	var indexes []bnbf.PremiumIndex

	for {
		select {
		case <-ctx.Done():
			return
		case <-afterFrTimer.C:

			subCtx, _ = context.WithTimeout(ctx, time.Minute)
			indexes, err = api.GetPremiumIndex(subCtx)
			if err != nil {
				logger.Debugf("GetPremiumIndex error %v", err)
			} else {
				for _, fr := range indexes {
					if ch, ok := channels[strings.ToLower(fr.Symbol)]; ok {
						msg, err = fr.MarshalJSON()
						if err == nil {
							index++
							if index == bufferSize {
								index = 0
							}
							message = pool[index]
							message.Time = time.Now().UnixNano()
							message.Data = msg
							select {
							case ch <- message:
							default:
								if time.Now().Sub(logSilentTime) > 0 {
									logger.Debugf("ch <- message failed %s len(ch) = %d", fr.Symbol, len(ch))
									logSilentTime = time.Now().Add(time.Minute)
								}
							}
						}
					}
				}
			}


			afterFrTimer.Reset(time.Now().Truncate(time.Hour * 4).Add(time.Hour*4 + time.Second).Sub(time.Now()))
			break
		case <-timer.C:

			subCtx, _ = context.WithTimeout(ctx, time.Minute)
			indexes, err = api.GetPremiumIndex(subCtx)
			if err != nil {
				logger.Debugf("GetPremiumIndex error %v", err)
			} else {
				for _, fr := range indexes {
					if ch, ok := channels[strings.ToLower(fr.Symbol)]; ok {
						msg, err = fr.MarshalJSON()
						if err == nil {
							index++
							if index == bufferSize {
								index = 0
							}
							message = pool[index]
							message.Time = time.Now().UnixNano()
							message.Data = msg
							select {
							case ch <- message:
							default:
								if time.Now().Sub(logSilentTime) > 0 {
									logger.Debugf("ch <- message failed %s len(ch) = %d", fr.Symbol, len(ch))
									logSilentTime = time.Now().Add(time.Minute)
								}
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
