package hbspot

import (
	"bytes"
	"context"
	"encoding/json"
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
	mu sync.Mutex
}

func (api *API) SendHTTPRequest(ctx context.Context, method, path string, params common.Params, result interface{}) error {
	path = "https://api-aws.huobi.pro" + path
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
		return fmt.Errorf("unmarshal error %v %s", err, contents)
	} else if dataCap.Status != "ok" {
		return fmt.Errorf("status %s code %s msg %s", dataCap.Status, dataCap.ErrCode, dataCap.ErrMsg)
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
	payload := fmt.Sprintf("%s\napi-aws.huobi.pro\n%s\n%s", method, path, values.Encode())
	hmac := common.GetHMAC(common.HashSHA256, []byte(payload), []byte(api.secretKey))
	values.Set("Signature", common.Base64Encode(hmac))
	path = common.EncodeURLValues("https://api-aws.huobi.pro"+path, values)
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
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req.Header.Set("Content-Type", "application/json")
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
		return fmt.Errorf("unmarshal error %v %s", err, contents)
	} else if dataCap.Status != "ok" {
		return fmt.Errorf("status %s code %s msg %s", dataCap.Status, dataCap.ErrCode, dataCap.ErrMsg)
	}
	return json.Unmarshal(dataCap.Data, result)
}


func (api *API) GetSymbols(ctx context.Context) ([]Symbol, error) {
	data := make([]Symbol, 0)
	return data, api.SendHTTPRequest(ctx, http.MethodGet, "/v1/common/symbols", nil, &data)
}

func (api *API) GetKlines(ctx context.Context, param KlinesParam) ([]common.KLine, error) {
	klines := make([]Kline, 0)
	err := api.SendHTTPRequest(ctx, http.MethodGet, "/market/history/kline", &param, &klines)
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
		bars[i].Volume = kline.Volume
	}
	return bars, nil
}


func (api *API) GetAccounts(ctx context.Context) ([]Account, error) {
	accounts := make([]Account, 0)
	return accounts, api.SendAuthenticatedHTTPRequest(ctx, http.MethodGet, "/v1/account/accounts", nil, nil, &accounts)
}

func (api *API) GetAccount(ctx context.Context, accountID int64) (*Account, error) {
	account := &Account{}
	return account, api.SendAuthenticatedHTTPRequest(ctx, http.MethodGet, fmt.Sprintf( "/v1/account/accounts/%d/balance", accountID), nil, nil, account)
}

func (api *API) SubmitOrder(ctx context.Context, order NewOrderParam) (*string, error) {
	var clientOrderID string
	return &clientOrderID, api.SendAuthenticatedHTTPRequest(ctx, http.MethodPost, "/v1/order/orders/place", nil, order, &clientOrderID)
}

func (api *API) CancelAllOrders(ctx context.Context, param CancelAllParam) (*CancelAllResponse, error) {
	nor := &CancelAllResponse{}
	return nor, api.SendAuthenticatedHTTPRequest(ctx, http.MethodPost, "/v1/order/orders/batchCancelOpenOrders", nil, param, &nor)
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
