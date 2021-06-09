package bncoinfuture

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

const (
	OrderTimeInForceGTC  = "GTC"
	OrderTimeInForceIOC  = "IOC"
	OrderTimeInForceFOK  = "FOK"
	OrderTimeInForceGTX  = "GTX"
	OrderRespTypeAck     = "ACK"
	OrderRespTypeResult  = "RESULT"
	OrderRespTypeFull    = "FULL"
	OrderIsIsolatedTrue  = "TRUE"
	OrderIsIsolatedFalse = "FALSE"

	OrderSideBuy  = "BUY"
	OrderSideSell = "SELL"

	OrderTypeLimit           = "LIMIT"
	OrderTypeMarket          = "MARKET"
	OrderTypeStopLoss        = "STOP_LOSS"
	OrderTypeStopLossLimit   = "STOP_LOSS_LIMIT"
	OrderTypeTakeProfit      = "TAKE_PROFIT"
	OrderTypeTakeProfitLimit = "TAKE_PROFIT_LIMIT"
	OrderTypeLimitMarker     = "LIMIT_MAKER"

	OrderStatusNew             = "NEW"
	OrderStatusPartiallyFilled = "PARTIALLY_FILLED"
	OrderStatusFilled          = "FILLED"
	OrderStatusCancelled       = "CANCELED"
	OrderStatusPendingCancel   = "PENDING_CANCEL"
	OrderStatusReject          = "REJECTED"
	OrderStatusExpired         = "EXPIRED"
)

type API struct {
	client      *http.Client
	credentials *common.Credentials
	mu          sync.Mutex
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

	path = "https://dapi.binance.com" + path
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
	logger.Debugf("%s", contents)
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


func (api *API) ChangePositionMode(ctx context.Context, params ChangePositionModeParam) (*Response, error) {
	var resp Response
	err := api.SendAuthenticatedHTTPRequest(
		ctx,
		http.MethodPost,
		"/dapi/v1/positionSide/dual",
		&params,
		&resp,
	)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (api *API) GetPositions(ctx context.Context) ([]Position, error) {
	var positions []Position
	err := api.SendAuthenticatedHTTPRequest(
		ctx,
		http.MethodGet,
		"/dapi/v1/positionRisk",
		nil,
		&positions,
	)
	if err != nil {
		return nil, err
	}
	return positions, nil
}

func (api *API) GetAccount(ctx context.Context) (*Account, error) {
	var account Account
	err := api.SendAuthenticatedHTTPRequest(
		ctx,
		http.MethodGet,
		"/dapi/v2/account",
		nil,
		&account,
	)
	if err != nil {
		return nil, err
	}
	return &account, nil
}



func (api *API) UpdateLeverage(ctx context.Context, params UpdateLeverageParams) (*Response, error) {
	var resp Response
	err := api.SendAuthenticatedHTTPRequest(
		ctx,
		http.MethodPost,
		"/dapi/v1/leverage",
		&params,
		&resp,
	)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (api *API) UpdateMarginType(ctx context.Context, params UpdateMarginTypeParams) (*Response, error) {
	var resp Response
	err := api.SendAuthenticatedHTTPRequest(
		ctx,
		http.MethodPost,
		"/dapi/v1/marginType",
		&params,
		&resp,
	)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (api *API) GetListenKey(ctx context.Context) (*ListenKey, error) {
	var listenKey ListenKey
	err := api.SendAuthenticatedHTTPRequest(
		ctx,
		http.MethodPost,
		"/dapi/v1/listenKey",
		nil,
		&listenKey,
	)
	if err != nil {
		logger.Debugf("Get ListenKey error %v", err)
		return nil, err
	}
	return &listenKey, nil
}


func (api *API) SubmitOrder(ctx context.Context, params NewOrderParams) (*Order, error) {
	var order Order
	err := api.SendAuthenticatedHTTPRequest(
		ctx,
		http.MethodPost,
		"/dapi/v1/order",
		&params,
		&order,
	)
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func (api *API) CancelAllOpenOrders(ctx context.Context, params CancelAllOrderParams) (*CancelAllOrderResponse, error) {
	var order CancelAllOrderResponse
	err := api.SendAuthenticatedHTTPRequest(
		ctx,
		http.MethodDelete,
		"/dapi/v1/allOpenOrders",
		&params,
		&order,
	)
	if err != nil {
		return nil, err
	}
	order.Symbol = params.Symbol
	return &order, nil
}

func (api *API) CancelOrder(ctx context.Context, params CancelOrderParam) (*Order, error) {
	var order Order
	err := api.SendAuthenticatedHTTPRequest(
		ctx,
		http.MethodDelete,
		"/dapi/v1/order",
		&params,
		&order,
	)
	if err != nil {
		return nil, err
	}
	order.Symbol = params.Symbol
	return &order, nil
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
