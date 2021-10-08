package dydx_usdfuture

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

type API struct {
	client      *http.Client
	credentials Credentials
	mu          sync.Mutex
}

func (api *API) SendHTTPRequest(ctx context.Context, path string, params common.Params, result interface{}) error {
	path = "https://api.dydx.exchange" + path
	values := url.Values{}
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
	//logger.Debugf("%s", contents)
	var errCap common.ErrorCap
	if err := json.Unmarshal(contents, &errCap); err == nil {
		if errCap.Code != 0 && errCap.Code != 200 {
			return errors.New(errCap.Msg)
		}
	}
	return json.Unmarshal(contents, result)
}

func (api *API) SendAuthenticatedHTTPRequest(ctx context.Context, method, path string, params common.Params, data interface{}, result interface{}) error {
	values := url.Values{}
	if params != nil {
		values = params.ToUrlValues()
	}
	path = common.EncodeURLValues(path, values)

	api.mu.Lock()
	credentials := api.credentials
	api.mu.Unlock()
	isoTimestamp := time.Now().UTC().Format(TimeLayout)
	var bodyStr string
	if data != nil {
		bodyData, err := json.Marshal(data)
		if err != nil {
			return err
		}
		bodyStr = string(bodyData)
	}
	signature := fmt.Sprintf(
		"%s%s%s%s",
		isoTimestamp,
		method,
		path,
		bodyStr,
	)
	secret, err := base64.URLEncoding.DecodeString(credentials.ApiSecret)
	if err != nil {
		return err
	}
	h := hmac.New(sha256.New, secret)
	h.Write([]byte(signature))
	hmacSigned := h.Sum(nil)
	signStr := base64.URLEncoding.EncodeToString(hmacSigned)
	path = "https://api.dydx.exchange" + path
	req, err := http.NewRequest(method, path, strings.NewReader(bodyStr))
	if err != nil {
		return err
	}
	req.Header.Add("DYDX-SIGNATURE", signStr)
	req.Header.Add("DYDX-API-KEY", credentials.ApiKey)
	req.Header.Add("DYDX-PASSPHRASE", credentials.ApiPassphrase)
	req.Header.Add("DYDX-TIMESTAMP", isoTimestamp)
	resp, err := api.client.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}
	reader := resp.Body
	contents, err := ioutil.ReadAll(reader)
	logger.Debugf("%s %s %s", method, path, bodyStr)
	logger.Debugf("%s", contents)
	if err != nil {
		return err
	}
	err = resp.Body.Close()
	if err != nil {
		return err
	}
	var errorsCap ErrorsCap
	if err := json.Unmarshal(contents, &errorsCap); err == nil {
		if errorsCap.Errors != nil {
			return errors.New(string(contents))
		}
	}
	return json.Unmarshal(contents, result)
}

func (api *API) GetMarkets(ctx context.Context) (map[string]Market, error) {
	exchangeInfo := struct {
		Markets map[string]Market `json:"markets"`
	}{}
	return exchangeInfo.Markets, api.SendHTTPRequest(
		ctx,
		"/v3/markets",
		nil,
		&exchangeInfo,
	)
}

func (api *API) GetAccounts(ctx context.Context) ([]Account, error) {
	ar := &AccountsResp{}
	return ar.Accounts, api.SendAuthenticatedHTTPRequest(
		ctx,
		http.MethodGet,
		"/v3/accounts",
		nil,
		nil,
		ar,
	)
}

func (api *API) GetAccount(ctx context.Context) (*Account, error) {
	ar := &AccountResp{}
	return &ar.Account, api.SendAuthenticatedHTTPRequest(
		ctx,
		http.MethodGet,
		"/v3/accounts/"+api.credentials.AccountID,
		nil,
		nil,
		ar,
	)
}

func (api *API) GetOrders(ctx context.Context) ([]Order, error) {
	or := &OrdersResp{}
	return or.Orders, api.SendAuthenticatedHTTPRequest(
		ctx,
		http.MethodGet,
		"/v3/orders",
		nil,
		nil,
		or,
	)
}
func (api *API) CreateOrder(ctx context.Context, params *NewOrderParams) (*Order, error) {
	if os.Getenv("DYDX_PYTHON_URL") == "" {
		panic("DYDX_PYTHON_URL not fund")
	} else {
		postData, err := json.Marshal(params)
		if err != nil {
			return nil, err
		}
		req, err := http.NewRequest(http.MethodPost, os.Getenv("DYDX_PYTHON_URL"), bytes.NewReader(postData))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := api.client.Do(req.WithContext(ctx))
		if err != nil {
			return nil, err
		}
		reader := resp.Body
		contents, err := ioutil.ReadAll(reader)
		logger.Debugf("%s", contents)
		if err != nil {
			return nil, err
		}
		err = resp.Body.Close()
		if err != nil {
			return nil, err
		}
		var errorsCap ErrorsCap
		if err := json.Unmarshal(contents, &errorsCap); err == nil {
			if errorsCap.Errors != nil {
				return nil, errors.New(string(contents))
			}
		}
		or := &CreateOrderResp{}
		err = json.Unmarshal(contents, or)
		return &or.Order, err
	}
}

func (api *API) CancelOrders(ctx context.Context, params *CancelOrdersParam) ([]Order, error) {
	or := &CancelOrdersResp{}
	return or.CancelOrders, api.SendAuthenticatedHTTPRequest(
		ctx,
		http.MethodDelete,
		"/v3/orders",
		params,
		nil,
		or,
	)
}

func NewAPI(credentials Credentials, proxy string) (*API, error) {
	var client *http.Client
	if proxy != "" {
		proxyUrl, err := url.Parse(proxy)
		if err != nil {
			return nil, err
		}
		client = &http.Client{
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
		client = &http.Client{
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
