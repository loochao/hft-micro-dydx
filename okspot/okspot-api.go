package okspot

import (
	"bytes"
	"compress/gzip"
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
	"strconv"
	"strings"
	"time"
)

type API struct {
	client      *http.Client
	apiUrl      string
	credentials *Credentials
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
	var reader io.ReadCloser
	contentTypeDifferent := false
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err = gzip.NewReader(resp.Body)
		if err != nil {
			return err
		}
		err = reader.Close()
		if err != nil {
			return err
		}
	case "json":
		reader = resp.Body
	default:
		switch {
		case strings.Contains(resp.Header.Get("Content-Type"), "application/json"):
			reader = resp.Body
		default:
			logger.Warnf("request response content type differs from JSON; received %v", resp.Header.Get("Content-Type"))
			reader = resp.Body
			contentTypeDifferent = true
		}
	}
	contents, err := ioutil.ReadAll(reader)
	if err != nil {
		return err
	}
	if contentTypeDifferent {
		logger.Debugf("CONTENTS %s", string(contents))
	}
	err = resp.Body.Close()
	if err != nil {
		return err
	}
	var errCap ErrorCap
	if err := json.Unmarshal(contents, &errCap); err == nil {
		if errCap.ErrorMessage != "" {
			return fmt.Errorf("error: %v", errCap.ErrorMessage)
		}
		if errCap.ErrorCode > 0 {
			return fmt.Errorf("sendHTTPRequest error - %s",
				ErrorCodes[strconv.FormatInt(errCap.ErrorCode, 10)])
		}
		if !errCap.Result {
			return errors.New("unspecified error occurred")
		}
	}
	err = json.Unmarshal(contents, result)
	if err != nil {
		err = fmt.Errorf("JSON DECODE ERROR: \"%v\" CONTENT: %s", err, string(contents))
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
		//logger.Debugf("%s", payload)
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
	req.Header.Add("OK-ACCESS-KEY", api.credentials.Key)

	resp, err := api.client.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}
	var reader io.ReadCloser
	contentTypeDifferent := false
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return err
		}
		err = reader.Close()
		if err != nil {
			return err
		}
	case "json":
		reader = resp.Body
	default:
		switch {
		case strings.Contains(resp.Header.Get("Content-Type"), "application/json"):
			reader = resp.Body
		default:
			logger.Warnf("request response content type differs from JSON; received %v", resp.Header.Get("Content-Type"))
			reader = resp.Body
			contentTypeDifferent = true
		}
	}
	contents, err := ioutil.ReadAll(reader)
	if err != nil {
		return err
	}
	if contentTypeDifferent {
		logger.Debugf("CONTENTS %s", string(contents))
	}
	err = resp.Body.Close()
	if err != nil {
		return err
	}
	var errCap ErrorCap
	if err := json.Unmarshal(contents, &errCap); err == nil {
		if errCap.ErrorMessage != "" {
			logger.Debugf("ERROR CONTENTS %s %s", string(contents), path)
			return fmt.Errorf("error: %d %v", errCap.ErrorCode, errCap.ErrorMessage)
		}
		if errCap.ErrorCode > 0 {
			logger.Debugf("ERROR CONTENTS %s %s", string(contents), path)
			return fmt.Errorf("sendHTTPRequest error - %s",
				ErrorCodes[strconv.FormatInt(errCap.ErrorCode, 10)])
		}
		if !errCap.Result {
			return errors.New("unspecified error occurred")
		}
	}
	err = json.Unmarshal(contents, result)
	if err != nil {
		err = fmt.Errorf("JSON DECODE ERROR: \"%v\" CONTENT: %s", err, string(contents))
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
	}
	return &api, nil
}
