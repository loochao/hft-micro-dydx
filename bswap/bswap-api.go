package bswap

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
	"strconv"
	"sync"
	"time"
)

type API struct {
	client      *http.Client
	credentials *common.Credentials
	mu          sync.Mutex
}

func (api *API) SendHTTPRequest(ctx context.Context, path string, params common.Params, result interface{}) error {
	path = "https://api.binance.com" + path
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

func (api *API) SendAuthenticatedHTTPRequest(ctx context.Context, method, path string, params common.Params, result interface{}) error {

	values := url.Values{}
	if params != nil {
		values = params.ToUrlValues()

	}
	values.Set("recvWindow", strconv.FormatInt(common.RecvWindow(60*time.Second), 10))
	values.Set("timestamp", strconv.FormatInt(time.Now().Unix()*1000, 10))

	api.mu.Lock()
	credentials := *api.credentials
	api.mu.Unlock()

	signature := values.Encode()
	hmacSigned := common.GetHMAC(common.HashSHA256, []byte(signature), []byte(credentials.Secret))
	hmacSignedStr := common.HexEncodeToString(hmacSigned)

	path = "https://api.binance.com" + path
	path = common.EncodeURLValues(path, values)
	path += fmt.Sprintf("&signature=%s", hmacSignedStr)
	req, err := http.NewRequest(method, path, nil)
	if err != nil {
		return err
	}
	req.Header.Add("X-MBX-APIKEY", credentials.Key)
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

func (api *API) GetPools(ctx context.Context) ([]Pool, error) {
	pools := make([]Pool, 0)
	err := api.SendAuthenticatedHTTPRequest(ctx, http.MethodGet, "/sapi/v1/bswap/pools", nil, &pools)
	if err != nil {
		return nil, err
	}
	return pools, nil
}

func (api *API) GetLiquidity(ctx context.Context) ([]Liquidity, error) {
	liquidity := make([]Liquidity, 0)
	err := api.SendAuthenticatedHTTPRequest(ctx, http.MethodGet, "/sapi/v1/bswap/liquidity", nil, &liquidity)
	if err != nil {
		return nil, err
	}
	return liquidity, nil
}

func (api *API) GetQuote(ctx context.Context, param QuoteParam) (*Quote, error) {
	quote := Quote{}
	err := api.SendAuthenticatedHTTPRequest(ctx, http.MethodGet, "/sapi/v1/bswap/quote", &param, &quote)
	if err != nil {
		return nil, err
	}
	return &quote, nil
}

func NewAPI(credentials *common.Credentials, proxy string) (*API, error) {
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
