package ftx_usdspot

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type API struct {
	client     *http.Client
	key        string
	secret     string
	subAccount string
	mu         sync.Mutex
}

func (api *API) SendHTTPRequest(ctx context.Context, method, path string, param common.Params, result interface{}) error {
	path = "https://ftx.com/api" + path
	values := url.Values{}
	var err error
	if param != nil {
		values = param.ToUrlValues()
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
	var dataCap Response
	if err := json.Unmarshal(contents, &dataCap); err != nil {
		return err
	} else if !dataCap.Success {
		return errors.New(string(contents))
	}
	return json.Unmarshal(dataCap.Result, result)
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
		//logger.Debugf("%s", bodyStr)
		rBody = bytes.NewReader(bodyStr)
	}
	req, err := http.NewRequest(method, "https://ftx.com/api"+path, rBody)
	if err != nil {
		return err
	}

	timestamp := fmt.Sprintf("%d", time.Now().Unix()*1000)
	signature := fmt.Sprintf("%s%s%s%s", timestamp, method, "/api"+path, bodyStr)
	api.mu.Lock()
	hmacSigned := common.GetHMAC(common.HashSHA256, []byte(signature), []byte(api.secret))
	req.Header.Set("FTX-KEY", api.key)
	api.mu.Unlock()
	req.Header.Set("FTX-SIGN", common.HexEncodeToString(hmacSigned))
	req.Header.Set("FTX-TS", timestamp)
	if api.subAccount != "" {
		req.Header.Set("FTX-SUBACCOUNT", api.subAccount)
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
	//logger.Debugf("%s", contents)
	err = resp.Body.Close()
	if err != nil {
		return err
	}
	var dataCap Response
	if err := json.Unmarshal(contents, &dataCap); err != nil {
		return err
	} else if !dataCap.Success {
		return errors.New(dataCap.Error)
	}
	return json.Unmarshal(dataCap.Result, result)
}
func (api *API) GetAccount(ctx context.Context) (*Account, error) {
	account := Account{}
	return &account, api.SendAuthenticatedHTTPRequest(ctx, http.MethodGet, "/account", nil, nil, &account)
}

func (api *API) GetBalances(ctx context.Context) ([]Balance, error) {
	positions := make([]Balance, 0)
	return positions, api.SendAuthenticatedHTTPRequest(ctx, http.MethodGet, "/wallet/balances", nil, nil, &positions)
}

func (api *API) PlaceOrder(ctx context.Context, param NewOrderParam) (*Order, error) {
	order := Order{}
	return &order, api.SendAuthenticatedHTTPRequest(ctx, http.MethodPost, "/orders", nil, param, &order)
}

func (api *API) CancelOrderByClientID(ctx context.Context, clientID string) (*string, error) {
	var message string
	return &message, api.SendAuthenticatedHTTPRequest(ctx, http.MethodDelete, "/orders/by_client_id/"+clientID, nil, nil, &message)
}

func (api *API) CancelAllOrders(ctx context.Context, param CancelAllParam) (*string, error) {
	var message string
	return &message, api.SendAuthenticatedHTTPRequest(ctx, http.MethodDelete, "/orders", nil, param, &message)
}

func (api *API) ChangeLeverage(ctx context.Context, param LeverageParam) (*Leverage, error) {
	leverage := Leverage{
		Leverage: param.Leverage,
	}
	return &leverage, api.SendAuthenticatedHTTPRequest(ctx, http.MethodPost, "/account/leverage", nil, param, &leverage)
}

func (api *API) GetMarkets(ctx context.Context) ([]Market, error) {
	futures := make([]Market, 0)
	return futures, api.SendHTTPRequest(ctx, http.MethodGet, "/markets", nil, &futures)
}

func (api *API) GetFundingRates(ctx context.Context, param FundingRateParam) ([]FundingRate, error) {
	fundingRates := make([]FundingRate, 0)
	return fundingRates, api.SendHTTPRequest(ctx, http.MethodGet, "/funding_rates", &param, &fundingRates)
}


func NewAPI(key, secret, subAccount, proxy string) (*API, error) {
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
		client: &client,
		key:    key,
		secret: secret,
		subAccount: subAccount,
	}
	return &api, nil
}
