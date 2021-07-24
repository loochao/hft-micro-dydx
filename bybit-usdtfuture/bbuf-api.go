package bybit_usdtfuture

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
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

type API struct {
	client      *http.Client
	key         string
	secret      string
	url         string
	clients     []*http.Client
	clientIndex int32
	useProxies  bool
}

func (api *API) GetServerTime(ctx context.Context) (*time.Time, error) {
	req, err := http.NewRequest(http.MethodGet, api.url+"/v2/public/time", nil)
	if err != nil {
		return nil, err
	}
	var client = api.client
	if api.useProxies {
		index := int(atomic.AddInt32(&api.clientIndex, 1))%len(api.clients)
		client = api.clients[index]
		logger.Debugf("PROXY CLIENT %d", index)
	}
	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	reader := resp.Body
	contents, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	//logger.Debugf("%s", contents)
	err = resp.Body.Close()
	if err != nil {
		return nil, err
	}
	var dataCap ResponseCap
	if err := json.Unmarshal(contents, &dataCap); err != nil {
		return nil, err
	} else if dataCap.RetCode != 0 {
		return nil, errors.New(dataCap.RetMsg)
	}
	t := time.Unix(0, int64(dataCap.TimeNow*1000000000))
	return &t, nil
}

func (api *API) SendHTTPRequest(ctx context.Context, method, path string, param common.Params, result interface{}) error {
	path = api.url + path
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
	var client = api.client
	if api.useProxies {
		index := int(atomic.AddInt32(&api.clientIndex, 1))%len(api.clients)
		client = api.clients[index]
		logger.Debugf("PROXY CLIENT %d", index)
	}
	resp, err := client.Do(req.WithContext(ctx))
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
	var dataCap ResponseCap
	if err := json.Unmarshal(contents, &dataCap); err != nil {
		return err
	} else if dataCap.RetCode != 0 {
		return errors.New(dataCap.RetMsg)
	}
	return json.Unmarshal(dataCap.Result, result)
}

func (api *API) SendAuthenticatedHTTPRequest(ctx context.Context, method, path string, param Param, result interface{}) error {
	values := url.Values{}
	if param != nil {
		values = param.ToUrlValues()
	}
	values.Set("api_key", api.key)
	values.Set("recv_window", "5000")
	values.Set("timestamp", strconv.FormatInt(time.Now().UnixNano()/1000000, 10))
	hmacSigned := common.GetHMAC(common.HashSHA256, []byte(values.Encode()), []byte(api.secret))
	values.Set("sign", common.HexEncodeToString(hmacSigned))
	path = common.EncodeURLValues(path, values)
	//logger.Debugf("%s", api.url+path)
	req, err := http.NewRequest(method, api.url+path, nil)
	if err != nil {
		return err
	}
	var client = api.client
	if api.useProxies {
		index := int(atomic.AddInt32(&api.clientIndex, 1))%len(api.clients)
		client = api.clients[index]
		logger.Debugf("PROXY CLIENT %d", index)
	}
	resp, err := client.Do(req.WithContext(ctx))
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
	var dataCap ResponseCap
	if err := json.Unmarshal(contents, &dataCap); err != nil {
		return err
	} else if dataCap.RetCode != 0 {
		return fmt.Errorf("RetCode: %d, RetMsg: \"%s\"", dataCap.RetCode, dataCap.RetMsg)
	}
	if result != nil {
		return json.Unmarshal(dataCap.Result, result)
	} else {
		return nil
	}
}

func (api *API) GetSymbols(ctx context.Context) ([]Symbol, error) {
	symbols := make([]Symbol, 0)
	return symbols, api.SendHTTPRequest(ctx, http.MethodGet, "/v2/public/symbols", nil, &symbols)
}

func (api *API) GetPrevFundingRate(ctx context.Context, param PrevFundingRateParam) (*FundingRate, error) {
	fr := &FundingRate{}
	return fr, api.SendHTTPRequest(ctx, http.MethodGet, "/public/linear/funding/prev-funding-rate", &param, fr)
}

func (api *API) GetBalance(ctx context.Context, param BalanceParam) (*Balance, error) {
	balances := make(map[string]Balance, 0)
	err := api.SendAuthenticatedHTTPRequest(ctx, http.MethodGet, "/v2/private/wallet/balance", &param, &balances)
	if err != nil {
		return nil, err
	}
	if balance, ok := balances[param.Coin]; ok {
		return &balance, nil
	} else {
		return nil, fmt.Errorf("balance for %s not found", param.Coin)
	}
}

func (api *API) GetPositions(ctx context.Context) ([]PositionData, error) {
	positions := make([]PositionData, 0)
	return positions, api.SendAuthenticatedHTTPRequest(ctx, http.MethodGet, "/private/linear/position/list", nil, &positions)
}

func (api *API) SetAutoAddMargin(ctx context.Context, param SetAutoAddMarginParam) error {
	return api.SendAuthenticatedHTTPRequest(ctx, http.MethodPost, "/private/linear/position/set-auto-add-margin", &param, nil)
}

func (api *API) SwitchIsolated(ctx context.Context, param SwitchIsolatedParam) error {
	return api.SendAuthenticatedHTTPRequest(ctx, http.MethodPost, "/private/linear/position/switch-isolated", &param, nil)
}

func (api *API) SetLeverage(ctx context.Context, param SetLeverageParam) error {
	return api.SendAuthenticatedHTTPRequest(ctx, http.MethodPost, "/private/linear/position/set-leverage", &param, nil)
}

func (api *API) PlaceOrder(ctx context.Context, param NewOrderParam) (*Order, error) {
	order := &Order{}
	return order, api.SendAuthenticatedHTTPRequest(ctx, http.MethodPost, "/private/linear/order/create", &param, &order)
}

func (api *API) CancelAllOrders(ctx context.Context, param CancelAllParam) ([]string, error) {
	ids := make([]string, 0)
	return ids, api.SendAuthenticatedHTTPRequest(ctx, http.MethodPost, "/private/linear/order/cancel-all", &param, &ids)
}

func (api *API) CancelOrder(ctx context.Context, param CancelParam) (*CancelOrderResp, error) {
	res := &CancelOrderResp{}
	return res, api.SendAuthenticatedHTTPRequest(ctx, http.MethodPost, "/private/linear/order/cancel", &param, &res)
}

func NewAPI(key, secret, apiUrl, proxy string) (*API, error) {
	var client http.Client
	var clients = make([]*http.Client, 0)
	if os.Getenv("BBUF_API_PROXIES") == "" {
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
	} else {
		proxies := strings.Split(os.Getenv("BBUF_API_PROXIES"), ",")
		for _, proxy := range proxies {
			logger.Debugf("PROXY %s", proxy)
			proxyUrl, err := url.Parse(proxy)
			if err != nil {
				return nil, err
			}
			client := http.Client{
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
			clients = append(clients, &client)
		}

	}
	if apiUrl == "" {
		apiUrl = "https://api.bybit.com"
	}
	api := API{
		client:      &client,
		key:         key,
		secret:      secret,
		url:         apiUrl,
		clientIndex: 0,
		clients:     clients,
		useProxies:  os.Getenv("BBUF_API_PROXIES") != "",
	}
	return &api, nil
}
