package bncoinfuture

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	KlineInterval1m            = "1m"
	KlineInterval3m            = "3m"
	KlineInterval5m            = "5m"
	KlineInterval15m           = "15m"
	KlineInterval30m           = "30m"
	KlineInterval1h            = "1h"
	KlineInterval2h            = "2h"
	KlineInterval4h            = "4h"
	KlineInterval6h            = "6h"
	KlineInterval8h            = "8h"
	KlineInterval12h           = "12h"
	KlineInterval1d            = "1d"
	KlineInterval3d            = "3d"
	KlineInterval1w            = "1w"
	KlineInterval1M            = "1M"
)

func (api *API) SendHTTPRequest(ctx context.Context, path string, params common.Params, result interface{}) error {
	path = "https://dapi.binance.com" + path
	values := url.Values{}
	if params != nil {
		values = params.ToUrlValues()
	}
	values.Set("recvWindow", strconv.FormatInt(common.RecvWindow(15*time.Second), 10))
	values.Set("timestamp", strconv.FormatInt(time.Now().Unix()*1000, 10))
	path = common.EncodeURLValues(path, values)

	req, err := http.NewRequest("GET", path, nil)
	if err != nil {
		return err
	}
	resp, err := api.client.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}
	reader := resp.Body
	contents, err := ioutil.ReadAll(reader)
	if err != nil {
		return err
	}
	logger.Debugf("%s", contents)
	err = resp.Body.Close()
	if err != nil {
		return err
	}
	var errCap common.ErrorCap
	if err := json.Unmarshal(contents, &errCap); err == nil {
		if errCap.Code != 0 && errCap.Code != 200 {
			return errors.New(errCap.Msg)
		}
	}
	return json.Unmarshal(contents, result)
}


func (api *API) PingServer(ctx context.Context) (*Ping, error) {
	var ping Ping
	return &ping, api.SendHTTPRequest(
		ctx,
		"/dapi/v1/ping",
		nil,
		&ping,
	)
}

func (api *API) GetServerTime(ctx context.Context) (*ServerTime, error) {
	var positions ServerTime
	err := api.SendAuthenticatedHTTPRequest(
		ctx,
		http.MethodGet,
		"/dapi/v1/time",
		nil,
		&positions,
	)
	if err != nil {
		return nil, err
	}
	return &positions, nil
}

func (api *API) GetKLines(ctx context.Context, params KlineParams) ([]common.KLine, error) {
	var resp [][12]interface{}
	err := api.SendHTTPRequest(
		ctx,
		"/dapi/v1/klines",
		&params,
		&resp,
	)
	if err != nil {
		return nil, err
	}
	kLines := make([]common.KLine, 0)
	for _, row := range resp {
		kLine := common.KLine{
			Symbol: params.Symbol,
		}
		open, ok := row[1].(string)
		if !ok {
			return nil, fmt.Errorf("can't convert open %v to string", row[1])
		}
		kLine.Open, err = strconv.ParseFloat(open, 64)
		if err != nil {
			return nil, err
		}
		high, ok := row[2].(string)
		if !ok {
			return nil, fmt.Errorf("can't convert high %v to string", row[2])
		}
		kLine.High, err = strconv.ParseFloat(high, 64)
		if err != nil {
			return nil, err
		}
		low, ok := row[3].(string)
		if !ok {
			return nil, fmt.Errorf("can't convert low %v to string", row[3])
		}
		kLine.Low, err = strconv.ParseFloat(low, 64)
		if err != nil {
			return nil, err
		}
		close_, ok := row[4].(string)
		if !ok {
			return nil, fmt.Errorf("can't convert close %v to string", row[4])
		}
		kLine.Close, err = strconv.ParseFloat(close_, 64)
		if err != nil {
			return nil, err
		}
		volume, ok := row[5].(string)
		if !ok {
			return nil, fmt.Errorf("can't convert volume %v to string", row[5])
		}
		kLine.Volume, err = strconv.ParseFloat(volume, 64)
		if err != nil {
			return nil, err
		}
		timestamp, ok := row[6].(float64)
		if !ok {
			return nil, fmt.Errorf("can't convert timestamp %v to float", row[6])
		}
		//需要额外加一毫秒
		kLine.Timestamp = time.Unix(int64(timestamp+1)/1000, 0)
		kLines = append(kLines, kLine)
	}
	return kLines, nil
}




func (api *API) GetExchangeInfo(ctx context.Context) (*ExchangeInfo, error) {
	var resp ExchangeInfo
	if err := api.SendHTTPRequest(
		ctx,
		"/dapi/v1/exchangeInfo",
		nil,
		&resp,
	); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (api *API) GetPremiumIndex(ctx context.Context) ([]PremiumIndex, error) {
	resp := make([]PremiumIndex, 0)
	return resp, api.SendHTTPRequest(
		ctx,
		"/dapi/v1/premiumIndex",
		nil,
		&resp,
	)
}

func (api *API) GetHistoryKLines(ctx context.Context, symbol, interval string, startTime time.Time) ([]common.KLine, error) {
	kLines := make([]common.KLine, 0)
	retryCount := 10
	for {
		subCtxt, _ := context.WithTimeout(ctx, time.Second*15)
		o, err := api.GetKLines(subCtxt, KlineParams{
			Symbol:    symbol,
			Interval:  interval,
			StartTime: startTime.Unix() * 1000,
			Limit:     1000,
		})
		if err != nil {
			if retryCount <= 0 {
				return nil, err
			} else {
				retryCount--
				continue
			}
		}
		kLines = append(kLines, o...)
		if len(o) < 1000 {
			break
		}
		//startTime是include的
		startTime = o[len(o)-1].Timestamp.Add(time.Minute)
		time.Sleep(time.Second)
	}
	return kLines, nil
}

