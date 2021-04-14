package hbswap

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
	"sync"
	"time"
)

type API struct {
	client http.Client
	//signer *KcSigner
	mu sync.Mutex
}

func (api *API) SendHTTPRequest(ctx context.Context, method, path string, params common.Params, result interface{}) error {
	path = "https://api.hbdm.vn" + path
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
	logger.Debugf("%s", contents)
	err = resp.Body.Close()
	if err != nil {
		return err
	}
	var dataCap DataCap
	if err := json.Unmarshal(contents, &dataCap); err != nil {
		return err
	} else if dataCap.Status != "ok" {
		return errors.New(fmt.Sprintf("%d %s", dataCap.ErrCode, dataCap.ErrMsg))
	}
	if dataCap.Data == nil {
		return json.Unmarshal(contents, result)
	}
	return json.Unmarshal(dataCap.Data, result)
}

//func (api *API) SendAuthenticatedHTTPRequest(ctx context.Context, method, path string, params common.Params, body, result interface{}) error {
//
//	values := url.Values{}
//	if params != nil {
//		values = params.ToUrlValues()
//
//	}
//	path = common.EncodeURLValues(path, values)
//	var rBody io.Reader
//	var bodyStr []byte
//	var err error
//	if body != nil {
//		bodyStr, err = json.Marshal(body)
//		if err != nil {
//			return err
//		}
//		logger.Debugf("%s", bodyStr)
//		rBody = bytes.NewReader(bodyStr)
//	}
//	headers := api.signer.Headers(fmt.Sprintf("%s%s%s", method, path, bodyStr))
//	path = "https://api-futures.kucoin.com" + path
//	req, err := http.NewRequest(method, path, rBody)
//	if err != nil {
//		return err
//	}
//	for key, value := range headers {
//		req.Header.Set(key, value)
//	}
//	req.Header.Set("Content-Type", "application/json")
//
//	resp, err := api.client.Do(req.WithContext(ctx))
//	if err != nil {
//		return err
//	}
//	reader := resp.Body
//	contents, err := ioutil.ReadAll(reader)
//	if err != nil {
//		return err
//	}
//	err = resp.Body.Close()
//	if err != nil {
//		return err
//	}
//	var dataCap DataCap
//	if err := json.Unmarshal(contents, &dataCap); err != nil {
//		return err
//	} else if dataCap.Code != 200000 {
//		return errors.New(dataCap.Msg)
//	}
//	return json.Unmarshal(dataCap.Data, result)
//}

func (api *API) GetHeartbeat(ctx context.Context) (*DataCap, error) {
	dataCap := &DataCap{}
	return dataCap, api.SendHTTPRequest(ctx, http.MethodGet, "/api/v1/timestamp", nil, dataCap)
}

func (api *API) GetContracts(ctx context.Context) ([]Contract, error) {
	data := make([]Contract, 0)
	return data, api.SendHTTPRequest(ctx, http.MethodGet, "/linear-swap-api/v1/swap_contract_info", nil, &data)
}

func (api *API) GetKlines(ctx context.Context, param KlinesParam) ([]common.KLine, error) {
	klines := make([]Kline, 0)
	err := api.SendHTTPRequest(ctx, http.MethodGet, "/linear-swap-ex/market/history/kline", &param, &klines)
	if err != nil {
		return nil, err
	}
	sort.Slice(klines, func(i, j int) bool {
		return klines[i].ID < klines[j].ID
	})
	bars := make([]common.KLine, len(klines))
	for i, kline := range klines {
		bars[i].Symbol = param.ContractCode
		bars[i].Timestamp = time.Unix(kline.ID, 0).Add(KlinePeriodDuration[param.Period])
		bars[i].Open = kline.Open
		bars[i].High = kline.High
		bars[i].Low = kline.Low
		bars[i].Close = kline.Close
		bars[i].Volume = kline.Vol
	}
	return bars, nil
}

func (api *API) GetFundingRates(ctx context.Context) ([]FundingRate, error) {
	data := make([]FundingRate, 0)
	return data, api.SendHTTPRequest(ctx, http.MethodGet, "/linear-swap-api/v1/swap_batch_funding_rate", nil, &data)
}

//func (api *API) GetAccountOverView(ctx context.Context, param AccountParam) (*Account, error) {
//	account := &Account{}
//	return account, api.SendAuthenticatedHTTPRequest(ctx, http.MethodGet, "/api/v1/account-overview", &param, nil, account)
//}
//
//func (api *API) GetPositions(ctx context.Context) ([]Position, error) {
//	positions := make([]Position, 0)
//	return positions, api.SendAuthenticatedHTTPRequest(ctx, http.MethodGet, "/api/v1/positions", nil, nil, &positions)
//}
//
//func (api *API) SubmitOrder(ctx context.Context, param NewOrderParam) (*OrderResponse, error) {
//	or := OrderResponse{}
//	return &or, api.SendAuthenticatedHTTPRequest(ctx, http.MethodPost, "/api/v1/orders", nil, param, &or)
//}
//
//func (api *API) ChangeAutoDepositStatus(ctx context.Context, param AutoDepositStatusParam) (bool, error) {
//	status := param.Status
//	return status, api.SendAuthenticatedHTTPRequest(ctx, http.MethodPost, "/api/v1/position/margin/auto-deposit-status", nil, param, &status)
//}
//
//func (api *API) CancelAllOrders(ctx context.Context, param CancelAllOrdersParam) (*CancelAllOrdersResponse, error) {
//	or := CancelAllOrdersResponse{}
//	return &or, api.SendAuthenticatedHTTPRequest(ctx, http.MethodDelete, "/api/v1/orders", &param, nil, &or)
//}
//
//func (api *API) GetPublicConnectToken(ctx context.Context) (*ConnectToken, error) {
//	pct := &ConnectToken{}
//	return pct, api.SendHTTPRequest(ctx, http.MethodPost, "/api/v1/bullet-public", nil, pct)
//}
//
//func (api *API) GetPrivateConnectToken(ctx context.Context) (*ConnectToken, error) {
//	pct := &ConnectToken{}
//	return pct, api.SendAuthenticatedHTTPRequest(ctx, http.MethodPost, "/api/v1/bullet-private", nil, nil, pct)
//}
//
//func (api *API) GetCurrentFundingRate(ctx context.Context, symbol string) (*CurrentFundingRate, error) {
//	pct := &CurrentFundingRate{}
//	err := api.SendHTTPRequest(ctx, http.MethodGet, fmt.Sprintf("/api/v1/funding-rate/%s/current", symbol), nil, pct)
//	if err != nil {
//		return nil, err
//	}
//	pct.Symbol = symbol
//	return pct, nil
//}
//


func NewAPI(key, secret, proxy string) (*API, error) {
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
		//signer: NewKcSigner(key, secret, passphrase),
		mu: sync.Mutex{},
	}
	return &api, nil
}
