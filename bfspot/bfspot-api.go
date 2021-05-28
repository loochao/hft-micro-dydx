package bfspot

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type API struct {
	client      http.Client
	credentials *common.Credentials
	mu          sync.Mutex
}

func (api *API) SendHTTPRequest(ctx context.Context, path string, params common.Params, result interface{}) error {
	path = "https://api-pub.bitfinex.com" + path
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

func (api *API) GetTradingParis(ctx context.Context) (interface{}, error) {
	//ei := interface{}{}
	err := api.SendHTTPRequest(
		ctx,
		"/v2/conf/pub:list:pair:exchange",
		nil,
		nil,
	)
	return nil, err
}


//func (api *API) SendAuthenticatedHTTPRequest(ctx context.Context, method, path string, params common.Params, result interface{}) (map[string]int64, error) {
//
//	var err error
//	values := url.Values{}
//	if params != nil {
//		values = params.ToUrlValues()
//	}
//	if path != "/api/v3/userDataStream" {
//		values.Set("recvWindow", strconv.FormatInt(common.RecvWindow(15*time.Second), 10))
//		values.Set("timestamp", strconv.FormatInt(time.Now().Unix()*1000, 10))
//	}
//
//	api.mu.Lock()
//	credentials := *api.credentials
//	api.mu.Unlock()
//	signature := values.Encode()
//	hmacSigned := common.GetHMAC(common.HashSHA256, []byte(signature), []byte(credentials.Secret))
//	hmacSignedStr := hex.EncodeToString(hmacSigned)
//
//	if path != "/api/v3/userDataStream" {
//		values.Set("signature", hmacSignedStr)
//	}
//	path = "https://api.binance.com" + path
//	path = common.EncodeURLValues(path, values)
//	req, err := http.NewRequest(method, path, nil)
//	if err != nil {
//		return nil, err
//	}
//	req.Header.Add("X-MBX-APIKEY", credentials.Key)
//	resp, err := api.client.Do(req.WithContext(ctx))
//	if err != nil {
//		return nil, err
//	}
//	reader := resp.Body
//
//	limits := make(map[string]int64)
//	for key, values := range resp.Header {
//		key := strings.ToLower(key)
//		if strings.Contains(key, "x-mbx-used-weight") && len(values) > 0 {
//			limits[key], _ = common.ParseInt([]byte(values[0]))
//		} else if strings.Contains(key, "x-mbx-order-count") && len(values) > 0 {
//			limits[key], _ = common.ParseInt([]byte(values[0]))
//		} else if strings.Contains(key, "retry-after") && len(values) > 0 {
//			limits[key], _ = common.ParseInt([]byte(values[0]))
//		}
//	}
//	contents, err := ioutil.ReadAll(reader)
//	//if strings.Contains(path, "/api/v3/account"){
//	//	logger.Debugf("%s", contents)
//	//}
//	if err != nil {
//		return limits, err
//	}
//	err = resp.Body.Close()
//	if err != nil {
//		return limits, err
//	}
//	var errCap common.ErrorCap
//	if err := json.Unmarshal(contents, &errCap); err == nil {
//		if errCap.Code != 0 && errCap.Code != 200 {
//			return limits, errors.New(errCap.Msg)
//		}
//	}
//	return limits, json.Unmarshal(contents, result)
//}

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
