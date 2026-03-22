package binance_tusdspot

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	TransferSpotToTUSDFuture = 1
	TransferTUSDFutureToSpot = 2
	TransferSpotToCOINFuture = 3
	TransferCOINFutureToSpot = 4

	UniversalTransferTypeMainC2c        = "MAIN_C2C"
	UniversalTransferTypeMainUmFuture   = "MAIN_UMFUTURE"
	UniversalTransferTypeMainCmFuture   = "MAIN_CMFUTURE"
	UniversalTransferTypeMainMargin     = "MAIN_MARGIN"
	UniversalTransferTypeMainMining     = "MAIN_MINING"
	UniversalTransferTypeC2cMain        = "C2C_MAIN"
	UniversalTransferTypeC2cUmFuture    = "C2C_UMFUTURE"
	UniversalTransferTypeC2cMining      = "C2C_MINING"
	UniversalTransferTypeC2cMargin      = "C2C_MARGIN"
	UniversalTransferTypeUmFutureMain   = "UMFUTURE_MAIN"
	UniversalTransferTypeUmFutureC2c    = "UMFUTURE_C2C"
	UniversalTransferTypeUmFutureMargin = "UMFUTURE_MARGIN"
	UniversalTransferTypeCmFutureMain   = "CMFUTURE_MAIN"
	UniversalTransferTypeCmFutureMargin = "CMFUTURE_MARGIN"
	UniversalTransferTypeMarginMain     = "MARGIN_MAIN"
	UniversalTransferTypeMarginUmFuture = "MARGIN_UMFUTURE"
	UniversalTransferTypeMarginCmFuture = "MARGIN_CMFUTURE"
	UniversalTransferTypeMarginMining   = "MARGIN_MINING"
	UniversalTransferTypeMarginC2c      = "MARGIN_C2C"
	UniversalTransferTypeMiningMain     = "MINING_MAIN"
	UniversalTransferTypeMiningUmFuture = "MINING_UMFUTURE"
	UniversalTransferTypeMiningC2c      = "MINING_C2C"
	UniversalTransferTypeMiningMargin   = "MINING_MARGIN"
)

type API struct {
	client      http.Client
	credentials *common.Credentials
	mu          sync.Mutex
}

func (api *API) SendHTTPRequest(ctx context.Context, path string, params common.Params, result interface{}) error {
	path = "https://api.binance.com" + path
	values := url.Values{}
	var err error
	if params != nil {
		values = params.ToUrlValues()
	}
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

func (api *API) GetKlines(ctx context.Context, params KlineParams) ([]common.KLine, error) {
	var resp [][]interface{}

	err := api.SendHTTPRequest(
		ctx,
		"/api/v3/klines",
		&params,
		&resp,
	)
	if err != nil {
		return nil, err
	}
	kLines := make([]common.KLine, 0)
	for _, row := range resp {
		kline := common.KLine{}
		open, ok := row[1].(string)
		if !ok {
			return nil, fmt.Errorf("can't convert open %v to string", row[1])
		}
		kline.Open, err = strconv.ParseFloat(open, 64)
		if err != nil {
			return nil, err
		}
		high, ok := row[2].(string)
		if !ok {
			return nil, fmt.Errorf("can't convert high %v to string", row[2])
		}
		kline.High, err = strconv.ParseFloat(high, 64)
		if err != nil {
			return nil, err
		}
		low, ok := row[3].(string)
		if !ok {
			return nil, fmt.Errorf("can't convert low %v to string", row[3])
		}
		kline.Low, err = strconv.ParseFloat(low, 64)
		if err != nil {
			return nil, err
		}
		close_, ok := row[4].(string)
		if !ok {
			return nil, fmt.Errorf("can't convert close %v to string", row[4])
		}
		kline.Close, err = strconv.ParseFloat(close_, 64)
		if err != nil {
			return nil, err
		}
		volume, ok := row[5].(string)
		if !ok {
			return nil, fmt.Errorf("can't convert volume %v to string", row[5])
		}
		kline.Volume, err = strconv.ParseFloat(volume, 64)
		if err != nil {
			return nil, err
		}
		timestamp, ok := row[6].(float64)
		if !ok {
			return nil, fmt.Errorf("can't convert timestamp %v to float64", row[6])
		}
		//需要额外加一毫秒
		kline.Timestamp = time.Unix(0, int64(timestamp+1)*1000000)
		kLines = append(kLines, kline)
	}
	return kLines, nil
}

func (api *API) GetExchangeInfo(ctx context.Context) (*ExchangeInfo, error) {
	ei := ExchangeInfo{}
	err := api.SendHTTPRequest(
		ctx,
		"/api/v3/exchangeInfo",
		nil,
		&ei,
	)
	return &ei, err
}

func (api *API) GetHistoryKLines(ctx context.Context, symbol, interval string, startTime time.Time) ([]common.KLine, error) {
	kLines := make([]common.KLine, 0)
	retryCount := 10
	for {
		subCtxt, _ := context.WithTimeout(ctx, time.Second*15)
		o, err := api.GetKlines(subCtxt, KlineParams{
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
		startTime = time.Unix(o[len(o)-1].Timestamp.Unix()/1000+60, 0)
		time.Sleep(time.Second)
	}
	return kLines, nil
}

func (api *API) SendAuthenticatedHTTPRequest(ctx context.Context, method, path string, params common.Params, result interface{}) (map[string]int64, error) {

	var err error
	values := url.Values{}
	if params != nil {
		values = params.ToUrlValues()
	}
	if path != "/api/v3/userDataStream" {
		values.Set("recvWindow", strconv.FormatInt(common.RecvWindow(15*time.Second), 10))
		values.Set("timestamp", strconv.FormatInt(time.Now().Unix()*1000, 10))
	}

	api.mu.Lock()
	credentials := *api.credentials
	api.mu.Unlock()
	signature := values.Encode()
	hmacSigned := common.GetHMAC(common.HashSHA256, []byte(signature), []byte(credentials.Secret))
	hmacSignedStr := hex.EncodeToString(hmacSigned)

	if path != "/api/v3/userDataStream" {
		values.Set("signature", hmacSignedStr)
	}
	path = "https://api.binance.com" + path
	path = common.EncodeURLValues(path, values)
	req, err := http.NewRequest(method, path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("X-MBX-APIKEY", credentials.Key)
	resp, err := api.client.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	reader := resp.Body

	limits := make(map[string]int64)
	for key, values := range resp.Header {
		key := strings.ToLower(key)
		if strings.Contains(key, "x-mbx-used-weight") && len(values) > 0 {
			limits[key], _ = common.ParseInt([]byte(values[0]))
		} else if strings.Contains(key, "x-mbx-order-count") && len(values) > 0 {
			limits[key], _ = common.ParseInt([]byte(values[0]))
		} else if strings.Contains(key, "retry-after") && len(values) > 0 {
			limits[key], _ = common.ParseInt([]byte(values[0]))
		}
	}
	contents, err := ioutil.ReadAll(reader)
	//if strings.Contains(path, "/api/v3/account"){
	//	logger.Debugf("%s", contents)
	//}
	if err != nil {
		return limits, err
	}
	err = resp.Body.Close()
	if err != nil {
		return limits, err
	}
	var errCap common.ErrorCap
	if err := json.Unmarshal(contents, &errCap); err == nil {
		if errCap.Code != 0 && errCap.Code != 200 {
			return limits, errors.New(errCap.Msg)
		}
	}
	return limits, json.Unmarshal(contents, result)
}

func (api *API) GetListenKey(ctx context.Context) (*ListenKey, map[string]int64, error) {
	var listenKey ListenKey
	limits, err := api.SendAuthenticatedHTTPRequest(
		ctx,
		http.MethodPost,
		"/api/v3/userDataStream",
		nil,
		&listenKey,
	)
	if err != nil {
		logger.Debugf("Get ListenKey error %v", err)
	}
	return &listenKey, limits, err
}

func (api *API) GetAccount(ctx context.Context) (*Account, map[string]int64, error) {
	var account Account
	limits, err := api.SendAuthenticatedHTTPRequest(
		ctx,
		http.MethodGet,
		"/api/v3/account",
		nil,
		&account,
	)
	return &account, limits, err
}

func (api *API) NewFutureAccountTransfer(ctx context.Context, params FutureAccountTransferParams) (*TransferResponse, map[string]int64, error) {
	var tRes TransferResponse
	limits, err := api.SendAuthenticatedHTTPRequest(
		ctx,
		http.MethodPost,
		"/sapi/v1/futures/transfer",
		&params,
		&tRes,
	)
	return &tRes, limits, err
}

func (api *API) SubmitOrder(ctx context.Context, params NewOrderParams) (*NewOrderResponse, map[string]int64, error) {
	var order NewOrderResponse
	params.NewOrderRespType = OrderRespTypeFull
	limits, err := api.SendAuthenticatedHTTPRequest(
		ctx,
		http.MethodPost,
		"/api/v3/order",
		&params,
		&order,
	)
	return &order, limits, err
}

func (api *API) CancelAllOrder(ctx context.Context, params CancelAllOrderParams) ([]CancelOrderResponse, map[string]int64, error) {
	var orders []CancelOrderResponse
	limits, err := api.SendAuthenticatedHTTPRequest(
		ctx,
		http.MethodDelete,
		"/api/v3/openOrders",
		&params,
		&orders,
	)
	return orders, limits, err
}

func (api *API) PingServer(ctx context.Context) (*Ping, error) {
	var ping Ping
	return &ping, api.SendHTTPRequest(
		ctx,
		"/api/v3/ping",
		nil,
		&ping,
	)
}

func (api *API) GetTicker(ctx context.Context, params TickerParam) (*Ticker, error) {
	var ticker Ticker
	return &ticker, api.SendHTTPRequest(
		ctx,
		"/api/v3/ticker/price",
		&params,
		&ticker,
	)
}

func NewAPI(credentials *common.Credentials, proxy string) (*API, error) {
	var client http.Client
	if proxy != "" {
		proxyUrl, err := url.Parse(proxy)
		if err != nil {
			return nil, err
		}
		client = http.Client{
			Timeout: 60 * time.Second,
			Transport: &http.Transport{
				Proxy:                 http.ProxyURL(proxyUrl),
				MaxIdleConns:          100,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   60 * time.Second,
				ExpectContinueTimeout: 10 * time.Second,
				DialContext: (&net.Dialer{
					Timeout:   60 * time.Second,
					KeepAlive: 90 * time.Second,
				}).DialContext,
			},
		}
	} else {
		client = http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:          100,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
				DialContext: (&net.Dialer{
					Timeout:   10 * time.Second,
					KeepAlive: 90 * time.Second,
				}).DialContext,
			},
		}
	}
	api := API{
		client:      client,
		credentials: credentials,
		mu:          sync.Mutex{},
	}
	return &api, nil
}
