package kucoin_usdtfuture

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"sort"
	"sync"
	"time"
)

type API struct {
	client http.Client
	signer *KcSigner
	mu     sync.Mutex
}

func (api *API) SendHTTPRequest(ctx context.Context, method, path string, params common.Params, result interface{}) error {
	path = "https://api-futures.kucoin.com" + path
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

func (api *API) SendAuthenticatedHTTPRequest(ctx context.Context, method, path string, params common.Params, body, result interface{}) error {

	values := url.Values{}
	if params != nil {
		values = params.ToUrlValues()

	}
	path = common.EncodeURLValues(path, values)
	var rBody io.Reader
	var bodyStr []byte
	var err error
	if body != nil {
		bodyStr, err = json.Marshal(body)
		if err != nil {
			return err
		}
		rBody = bytes.NewReader(bodyStr)
	}
	headers := api.signer.Headers(fmt.Sprintf("%s%s%s", method, path, bodyStr))
	path = "https://api-futures.kucoin.com" + path
	req, err := http.NewRequest(method, path, rBody)
	if err != nil {
		return err
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	req.Header.Set("Content-Type", "application/json")

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
		logger.Debugf("%s", contents)
		return err
	} else if dataCap.Code != 200000 {
		logger.Debugf("%s", contents)
		return errors.New(dataCap.Msg)
	}
	err = json.Unmarshal(dataCap.Data, result)
	if err != nil {
		logger.Debugf("%s", contents)
		return err
	}
	return nil
}

func (api *API) GetContracts(ctx context.Context) ([]Contract, error) {
	contracts := make([]Contract, 0)
	return contracts, api.SendHTTPRequest(ctx, http.MethodGet, "/api/v1/contracts/active", nil, &contracts)
}

func (api *API) GetTicker(ctx context.Context, param TickerParam) (*Ticker, error) {
	ticker := &Ticker{}
	return ticker, api.SendHTTPRequest(ctx, http.MethodGet, "/api/v1/ticker", &param, &ticker)
}

func (api *API) GetAccountOverView(ctx context.Context, param AccountParam) (*Account, error) {
	account := &Account{}
	return account, api.SendAuthenticatedHTTPRequest(ctx, http.MethodGet, "/api/v1/account-overview", &param, nil, account)
}

func (api *API) GetPositions(ctx context.Context) ([]Position, error) {
	positions := make([]Position, 0)
	return positions, api.SendAuthenticatedHTTPRequest(ctx, http.MethodGet, "/api/v1/positions", nil, nil, &positions)
}

func (api *API) SubmitOrder(ctx context.Context, param NewOrderParam) (*OrderResponse, error) {
	or := OrderResponse{}
	return &or, api.SendAuthenticatedHTTPRequest(ctx, http.MethodPost, "/api/v1/orders", nil, param, &or)
}

func (api *API) ChangeAutoDepositStatus(ctx context.Context, param AutoDepositStatusParam) (bool, error) {
	status := param.Status
	return status, api.SendAuthenticatedHTTPRequest(ctx, http.MethodPost, "/api/v1/position/margin/auto-deposit-status", nil, param, &status)
}

func (api *API) CancelAllOrders(ctx context.Context, param CancelAllOrdersParam) (*CancelAllOrdersResponse, error) {
	or := CancelAllOrdersResponse{}
	return &or, api.SendAuthenticatedHTTPRequest(ctx, http.MethodDelete, "/api/v1/orders", &param, nil, &or)
}

func (api *API) GetPublicConnectToken(ctx context.Context) (*ConnectToken, error) {
	pct := &ConnectToken{}
	return pct, api.SendHTTPRequest(ctx, http.MethodPost, "/api/v1/bullet-public", nil, pct)
}

func (api *API) GetSystemStatus(ctx context.Context) (*SystemStatus, error) {
	pct := &SystemStatus{}
	return pct, api.SendHTTPRequest(ctx, http.MethodGet, "/api/v1/status", nil, pct)
}

func (api *API) GetPrivateConnectToken(ctx context.Context) (*ConnectToken, error) {
	pct := &ConnectToken{}
	return pct, api.SendAuthenticatedHTTPRequest(ctx, http.MethodPost, "/api/v1/bullet-private", nil, nil, pct)
}

func (api *API) GetCurrentFundingRate(ctx context.Context, symbol string) (*CurrentFundingRate, error) {
	pct := &CurrentFundingRate{}
	err := api.SendHTTPRequest(ctx, http.MethodGet, fmt.Sprintf("/api/v1/funding-rate/%s/current", symbol), nil, pct)
	if err != nil {
		return nil, err
	}
	pct.Symbol = symbol
	return pct, nil
}

func (api *API) GetKlines(ctx context.Context, param KlinesParam) ([]common.KLine, error) {
	candlesRaw := make([][7]float64, 0)
	err := api.SendHTTPRequest(ctx, http.MethodGet, "/api/v1/kline/query", &param, &candlesRaw)
	if err != nil {
		return nil, err
	}
	sort.Slice(candlesRaw, func(i, j int) bool {
		return candlesRaw[i][0] < candlesRaw[j][0]
	})
	klines := make([]common.KLine, len(candlesRaw))
	for i, row := range candlesRaw {
		klines[i].Symbol = param.Symbol
		klines[i].Timestamp = time.Unix(0, int64(row[0])*1000000).Add(GranularityDurations[param.Granularity])
		klines[i].Open = row[1]
		klines[i].High = row[2]
		klines[i].Low = row[3]
		klines[i].Close = row[4]
		klines[i].Volume = row[5]
	}
	return klines, nil
}

func NewAPI(key, secret, passphrase, proxy string) (*API, error) {
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
		signer: NewKcSigner(key, secret, passphrase),
		mu:     sync.Mutex{},
	}
	return &api, nil
}
