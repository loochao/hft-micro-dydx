package kcspot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type API struct {
	client http.Client
	signer *KcSigner
	mu     sync.Mutex
}

func (api *API) SendHTTPRequest(ctx context.Context, method, path string, params common.Params, result interface{}) error {
	path = "https://api.kucoin.com" + path
	values := url.Values{}
	var err error
	if params != nil {
		values = params.ToUrlValues()
	}
	path = common.EncodeURLValues(path, values)
	req, err := http.NewRequest(method, path, nil)
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
	var dataCap DataCap
	if err := json.Unmarshal(contents, &dataCap); err != nil {
		return err
	} else if dataCap.Code != 200000 {
		return errors.New(dataCap.Msg)
	}
	return json.Unmarshal(dataCap.Data, result)
}

func (api *API) SendAuthenticatedHTTPRequest(ctx context.Context, method, path string, params common.Params, result interface{}) error {

	values := url.Values{}
	if params != nil {
		values = params.ToUrlValues()

	}
	//values.Set("recvWindow", strconv.FormatInt(common.RecvWindow(60*time.Second), 10))
	//values.Set("timestamp", strconv.FormatInt(time.Now().Unix()*1000, 10))

	path = common.EncodeURLValues(path, values)
	headers := api.signer.Headers(fmt.Sprintf("%s%s", method, path))

	path = "https://api.kucoin.com" + path
	req, err := http.NewRequest(method, path, nil)
	if err != nil {
		return err
	}
	logger.Debugf("%v", headers)
	for key, value := range headers {
		req.Header.Add(key, value)
	}
	logger.Debugf("%v", req.Header)
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
	var dataCap DataCap
	if err := json.Unmarshal(contents, &dataCap); err != nil {
		return err
	} else if dataCap.Code != 200000 {
		return errors.New(dataCap.Msg)
	}
	return json.Unmarshal(dataCap.Data, result)
}

func (api *API) GetAccounts(ctx context.Context, param AccountsParam) ([]Account, error) {
	accounts := make([]Account, 0)
	return accounts, api.SendAuthenticatedHTTPRequest(ctx, http.MethodGet, "/api/v1/accounts", &param, &accounts)
}

func (api *API) GetSymbols(ctx context.Context) ([]Symbol, error) {
	symbols := make([]Symbol, 0)
	return symbols, api.SendHTTPRequest(ctx, http.MethodGet, "/api/v1/symbols", nil, &symbols)
}

func (api *API) GetPublicConnectToken(ctx context.Context) (*ConnectToken, error) {
	pct := &ConnectToken{}
	return pct, api.SendHTTPRequest(ctx, http.MethodPost, "/api/v1/bullet-public", nil, pct)
}

func (api *API) GetPrivateConnectToken(ctx context.Context) (*ConnectToken, error) {
	pct := &ConnectToken{}
	return pct, api.SendHTTPRequest(ctx, http.MethodPost, "/api/v1/bullet-private", nil, pct)
}

func (api *API) GetCandles(ctx context.Context, param CandlesParam) ([]common.KLine, error) {
	candlesRaw := make([][7]string, 0)
	err := api.SendHTTPRequest(ctx, http.MethodGet, "/api/v1/market/candles", &param, &candlesRaw)
	if err != nil {
		return nil, err
	}
	sort.Slice(candlesRaw, func(i, j int) bool {
		return strings.Compare(candlesRaw[i][0], candlesRaw[j][0]) < 0
	})
	klines := make([]common.KLine, len(candlesRaw))
	for i, row := range candlesRaw {
		klines[i].Symbol = param.Symbol
		timestamp, err := strconv.ParseInt(row[0], 10, 64)
		if err != nil {
			return nil, err
		}
		klines[i].Timestamp = time.Unix(timestamp, 0).Add(CandleTypeDurations[param.Type])
		klines[i].Open, err = strconv.ParseFloat(row[1], 64)
		if err != nil {
			return nil, err
		}
		klines[i].Close, err = strconv.ParseFloat(row[2], 64)
		if err != nil {
			return nil, err
		}
		klines[i].High, err = strconv.ParseFloat(row[3], 64)
		if err != nil {
			return nil, err
		}
		klines[i].Low, err = strconv.ParseFloat(row[4], 64)
		if err != nil {
			return nil, err
		}
		klines[i].Volume, err = strconv.ParseFloat(row[5], 64)
		if err != nil {
			return nil, err
		}
	}
	return klines, nil
}

func NewAPI(signer *KcSigner, proxy string) (*API, error) {
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
		client: client,
		signer: signer,
		mu:     sync.Mutex{},
	}
	return &api, nil
}
