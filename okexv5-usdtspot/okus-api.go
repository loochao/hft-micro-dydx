package okexv5_usdtspot

import (
	"bytes"
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
	"sync"
	"time"
)

type API struct {
	client      *http.Client
	credentials *Credentials
	mu          sync.Mutex
}

func (api *API) SendHTTPRequest(ctx context.Context, requestPath string, result interface{}) (err error) {
	path := "https://www.okex.com" + requestPath

	//logger.Debugf("%v", path)
	req, err := http.NewRequest("GET", path, nil)
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	resp, err := api.client.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}
	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	//logger.Debugf("%s", contents)
	err = resp.Body.Close()
	if err != nil {
		return err
	}
	var errCap CommonCapture
	if err = json.Unmarshal(contents, &errCap); err != nil {
		return err
	}
	if errCap.Code != "0" {
		return fmt.Errorf("sendHTTPRequest error - path %s code %s msg %s", path, errCap.Code, errCap.Msg)
	}
	if errCap.Data == nil{
		return errors.New("unspecified error occurred")
	}
	//logger.Debugf("%s", errCap.Data)
	err = json.Unmarshal(errCap.Data, result)
	if err != nil {
		err = fmt.Errorf("json.Unmarshal(errCap.Data, result): \"%v\" content: %s", err, errCap.Data)
	}
	return err
}

func (api *API) SendAuthenticatedHTTPRequest(ctx context.Context, httpMethod, requestPath string, data, result interface{}) (err error) {

	utcTime := time.Now().UTC().Format(time.RFC3339)
	payload := []byte("")

	if data != nil {
		payload, err = json.Marshal(data)
		if err != nil {
			return errors.New("sendHTTPRequest: Unable to JSON request")
		}
	}

	path := "https://www.okex.com" + requestPath
	req, err := http.NewRequest(httpMethod, path, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	hmac := common.GetHMAC(common.HashSHA256,
		[]byte(utcTime+httpMethod+requestPath+string(payload)),
		[]byte(api.credentials.Secret))
	req.Header.Add("OK-ACCESS-KEY", api.credentials.Key)
	req.Header.Add("OK-ACCESS-SIGN", common.Base64Encode(hmac))
	req.Header.Add("OK-ACCESS-TIMESTAMP", utcTime)
	req.Header.Add("OK-ACCESS-PASSPHRASE", api.credentials.Passphrase)
	resp, err := api.client.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}
	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if requestPath == "/api/v5/trade/order" {
		logger.Debugf("ORDER-PAYLOAD %s", payload)
		//logger.Debugf("ORDER-RESULT %s", contents)
	}
	err = resp.Body.Close()
	if err != nil {
		return err
	}
	var errCap CommonCapture
	if err = json.Unmarshal(contents, &errCap); err != nil {
		return err
	}
	if errCap.Code != "0" {
		return fmt.Errorf("sendHTTPRequest error - path %s code %s", path, errCap.Code)
	}
	if errCap.Data == nil{
		return errors.New("unspecified error occurred")
	}

	//logger.Debugf("%s", errCap.Data)
	err = json.Unmarshal(errCap.Data, result)
	if err != nil {
		err = fmt.Errorf("json.Unmarshal(errCap.Data, result): \"%v\" content: %s", err, errCap.Data)
	}
	return err
}

func NewAPI(credentials *Credentials, proxy string) (*API, error) {
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
