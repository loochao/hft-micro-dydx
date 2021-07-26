package gate_usdtfuture

import (
	"context"
	"encoding/json"
	"github.com/geometrybase/hft-micro/common"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"time"
)

type API struct {
	client http.Client
}

func (api *API) SendHTTPRequest(ctx context.Context, method, path string, params common.Params, result interface{}) error {
	path = "https://api.gateio.ws" + path
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
	return json.Unmarshal(contents, result)
}

func (api *API) GetProducts(ctx context.Context) ([]Contract, error) {
	products := make([]Contract, 0)
	return products, api.SendHTTPRequest(ctx, http.MethodGet, "/api/v4/futures/usdt/contracts", nil, &products)
}

func NewAPI(proxy string) (*API, error) {
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
	}
	return &api, nil
}
