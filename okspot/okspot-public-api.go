package okspot

import (
	"context"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"strconv"
	"time"
)

func (api *API) GetRecentCandles(ctx context.Context, params MarketDataParams) ([]common.KLine, error) {
	candleInterval := time.Second * time.Duration(params.Granularity)
	if params.Start == nil {
		return nil, fmt.Errorf("bad nil start time for get history candles")
	}
	if params.End == nil {
		return nil, fmt.Errorf("bad nil end time for get history candles")
	}
	if params.Limit == nil {
		var limit int64 = 200
		params.Limit = &limit
	}
	if *params.Limit > 200 {
		*params.Limit = 200
	}
	finalEnd := *params.End
	candles := make([]common.KLine, 0)
	errorCounter := 0
	for errorCounter < 3 {
		*params.End = params.Start.Add(candleInterval * time.Duration(*params.Limit-1))
		marketData, err := api.GetMarketData(ctx, params)
		if err != nil {
			logger.Errorf(
				"GetHistoricalMarketData error %v", err,
			)
			errorCounter++
			time.Sleep(time.Second)
			continue
		}
		//logger.Debugf("%v -> %v %d ", *params.End, *params.Start, len(marketData))
		//默认是新数据在前边, 反向
		for i := len(marketData) - 1; i >= 0; i-- {
			data := marketData[i]
			timeData, err := time.Parse(okspotTimeLayout, data[0])
			if err != nil {
				logger.Errorf(
					"parse %s: %v", data[0], err,
				)
				continue
			}
			//时间是startTime
			timeData = timeData.Add(candleInterval)
			ohlcv := common.KLine{
				Timestamp: timeData,
			}
			ohlcv.Open, err = strconv.ParseFloat(data[1], 64)
			if err != nil {
				logger.Debugf("ParseFloat %s for open error %v", data[1], err)
				continue
			}
			ohlcv.High, err = strconv.ParseFloat(data[2], 64)
			if err != nil {
				logger.Debugf("ParseFloat %s for high error %v", data[2], err)
				continue
			}
			ohlcv.Low, err = strconv.ParseFloat(data[3], 64)
			if err != nil {
				logger.Debugf("ParseFloat %s for low error %v", data[3], err)
				continue
			}
			ohlcv.Close, err = strconv.ParseFloat(data[4], 64)
			if err != nil {
				logger.Debugf("ParseFloat %s for close error %v", data[4], err)
				continue
			}
			ohlcv.Volume, err = strconv.ParseFloat(data[5], 64)
			if err != nil {
				logger.Debugf("ParseFloat %s for volume error %v", data[5], err)
				continue
			}
			if len(candles) > 0 &&
				ohlcv.Timestamp.Sub(candles[len(candles)-1].Timestamp).Seconds() > 0 &&
				ohlcv.Timestamp.Sub(finalEnd).Seconds() <= 0 {
				candles = append(candles, ohlcv)
			} else if len(candles) == 0 &&
				ohlcv.Timestamp.Sub(finalEnd).Seconds() <= 0 {
				candles = append(candles, ohlcv)
			}
		}
		if len(marketData) < int(*params.Limit/2) {
			break
		}
		if len(candles) > 0 {
			if candles[len(candles)-1].Timestamp.Sub(finalEnd) >= 0 {
				break
			}
			startTime := candles[len(candles)-1].Timestamp.Add(-candleInterval * 1)
			params.Start = &startTime
		}
	}
	return candles, nil
}

func (api *API) GetMarketData(ctx context.Context, params MarketDataParams) (marketData []MarketData, _ error) {
	queryStr := fmt.Sprintf("&granularity=%d", params.Granularity)
	if params.Start != nil {
		queryStr += fmt.Sprintf("&start=%s&", params.Start.UTC().Format(okspotTimeLayout))
	}
	if params.End != nil {
		queryStr += fmt.Sprintf("&end=%s", params.End.UTC().Format(okspotTimeLayout))
	}
	queryStr = queryStr[1:]
	requestUrl := fmt.Sprintf(
		"/api/spot/v3/instruments/%s/candles?%s",
		params.InstrumentId,
		queryStr,
	)
	//logger.Debugf("%s%s", api.apiUrl, requestUrl)
	return marketData, api.SendHTTPRequest(ctx, requestUrl, &marketData)
}

func (api *API) GetHistoricalMarketData(ctx context.Context, params MarketDataParams) (marketData []MarketData, _ error) {
	queryStr := ""
	if params.Start != nil {
		queryStr += fmt.Sprintf("&start=%s", params.Start.UTC().Format(okspotTimeLayout))
	}
	if params.End != nil {
		queryStr += fmt.Sprintf("&end=%s", params.End.UTC().Format(okspotTimeLayout))
	}
	queryStr += fmt.Sprintf("&granularity=%d", params.Granularity)
	if params.Limit != nil {
		queryStr += fmt.Sprintf("&limit=%d", *params.Limit)
	}
	queryStr = queryStr[1:]
	requestUrl := fmt.Sprintf(
		"/api/spot/v3/instruments/%s/history/candles?%s",
		params.InstrumentId,
		queryStr,
	)
	//logger.Debugf("%s%s", api.apiUrl, requestUrl)
	return marketData, api.SendHTTPRequest(ctx, requestUrl, &marketData)
}

func (api *API) GetInstruments(ctx context.Context) ([]Instrument, error) {
	var instruments []Instrument
	return instruments, api.SendHTTPRequest(ctx, "/api/spot/v3/instruments", &instruments)
}

func (api *API) GetTickers(ctx context.Context) ([]WSTicker, error) {
	var instruments []WSTicker
	return instruments, api.SendHTTPRequest(ctx, "/api/spot/v3/instruments/ticker", &instruments)
}
