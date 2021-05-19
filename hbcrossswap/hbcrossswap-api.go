package hbcrossswap

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
	client    http.Client
	accessKey string
	secretKey string
	mu        sync.Mutex
}

func (api *API) GetHeartBeat(ctx context.Context) (*HeartBeat, error) {
	path := "https://api.hbdm.com/heartbeat/"
	req, err := http.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	resp, err := api.client.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	reader := resp.Body
	contents, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	err = resp.Body.Close()
	if err != nil {
		return nil, err
	}
	var dataCap DataCap
	if err := json.Unmarshal(contents, &dataCap); err != nil {
		return nil, err
	} else if dataCap.Status != "ok" {
		return nil, errors.New(fmt.Sprintf("%d %s", dataCap.ErrCode, dataCap.ErrMsg))
	}
	hb := &HeartBeat{}
	if dataCap.Data == nil {
		return nil, json.Unmarshal(contents, hb)
	}
	return hb, json.Unmarshal(dataCap.Data, hb)
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
	//logger.Debugf("%s", contents)
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

func (api *API) SendAuthenticatedHTTPRequest(ctx context.Context, method, path string, params common.Params, postBody, result interface{}) error {
	values := url.Values{}
	if params != nil {
		values = params.ToUrlValues()
	}
	values.Set("AccessKeyId", api.accessKey)
	values.Set("SignatureMethod", "HmacSHA256")
	values.Set("SignatureVersion", "2")
	values.Set("Timestamp", time.Now().UTC().Format("2006-01-02T15:04:05"))
	payload := fmt.Sprintf("%s\napi.hbdm.vn\n%s\n%s", method, path, values.Encode())
	hmac := common.GetHMAC(common.HashSHA256, []byte(payload), []byte(api.secretKey))
	values.Set("Signature", common.Base64Encode(hmac))
	path = common.EncodeURLValues("https://api.hbdm.vn"+path, values)
	//logger.Debugf("%s", path)
	var rBody io.Reader
	if postBody != nil {
		bodyStr, err := json.Marshal(postBody)
		if err != nil {
			return err
		}
		logger.Debugf("%s", bodyStr)
		rBody = bytes.NewReader(bodyStr)
	}
	req, err := http.NewRequest(method, path, rBody)
	if err != nil {
		return err
	}
	if method == http.MethodGet {
		req.Header.Set("Content-EventType", "application/x-www-form-urlencoded")
	} else {
		req.Header.Set("Content-EventType", "application/json")
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
	//logger.Debugf("%s", contents)
	var dataCap DataCap
	if err := json.Unmarshal(contents, &dataCap); err != nil {
		return err
	} else if dataCap.Status != "ok" {
		return errors.New(fmt.Sprintf("%d %s", dataCap.ErrCode, dataCap.ErrMsg))
	}
	return json.Unmarshal(dataCap.Data, result)
}

func (api *API) GetTimestamp(ctx context.Context) (*DataCap, error) {
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
		bars[i].Symbol = param.Symbol
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

func (api *API) GetAccounts(ctx context.Context) ([]Account, error) {
	accounts := make([]Account, 0)
	return accounts, api.SendAuthenticatedHTTPRequest(ctx, http.MethodPost, "/linear-swap-api/v1/swap_cross_account_info", nil, nil, &accounts)
}

func (api *API) GetPositions(ctx context.Context) ([]Position, error) {
	positions := make([]Position, 0)
	return positions, api.SendAuthenticatedHTTPRequest(ctx, http.MethodPost, "/linear-swap-api/v1/swap_cross_position_info", nil, nil, &positions)
}

func (api *API) SubmitOrder(ctx context.Context, order NewOrderParam) (*NewOrderResponse, error) {
	nor := &NewOrderResponse{}
	return nor, api.SendAuthenticatedHTTPRequest(ctx, http.MethodPost, "/linear-swap-api/v1/swap_cross_order", nil, order, &nor)
}

func (api *API) CancelAllOrders(ctx context.Context, param CancelAllParam) (*CancelAllResponse, error) {
	nor := &CancelAllResponse{}
	return nor, api.SendAuthenticatedHTTPRequest(ctx, http.MethodPost, "/linear-swap-api/v1/swap_cross_cancelall", nil, param, &nor)
}

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
				TLSHandshakeTimeout:   60 * time.Second,
				ExpectContinueTimeout: 10 * time.Second,
				DialContext: (&net.Dialer{
					Timeout:   60 * time.Second,
					KeepAlive: 90 * time.Second,
				}).DialContext,
			},
		}
	}
	api := API{
		client:    client,
		accessKey: key,
		secretKey: secret,
		mu:        sync.Mutex{},
	}
	return &api, nil
}
