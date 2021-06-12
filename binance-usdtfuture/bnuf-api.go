package binance_usdtfuture

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

func (api *API) SendHTTPRequest(ctx context.Context, path string, params common.Params, result interface{}) error {
	path = "https://fapi.binance.com" + path
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

	path = "https://fapi.binance.com" + path
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
	//if strings.Contains(path, "/fapi/v2/account") {
	//	logger.Debugf("%s", contents)
	//}
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

func (api *API) GetKLines(ctx context.Context, params KlineParams) ([]common.KLine, error) {
	var resp [][]interface{}

	err := api.SendHTTPRequest(
		ctx,
		"/fapi/v1/klines",
		&params,
		&resp,
	)
	if err != nil {
		return nil, err
	}
	kLines := make([]common.KLine, 0)
	for _, row := range resp {
		kLine := common.KLine{
			Symbol: params.Symbol,
		}
		open, ok := row[1].(string)
		if !ok {
			return nil, fmt.Errorf("can't convert open %v to string", row[1])
		}
		kLine.Open, err = strconv.ParseFloat(open, 64)
		if err != nil {
			return nil, err
		}
		high, ok := row[2].(string)
		if !ok {
			return nil, fmt.Errorf("can't convert high %v to string", row[2])
		}
		kLine.High, err = strconv.ParseFloat(high, 64)
		if err != nil {
			return nil, err
		}
		low, ok := row[3].(string)
		if !ok {
			return nil, fmt.Errorf("can't convert low %v to string", row[3])
		}
		kLine.Low, err = strconv.ParseFloat(low, 64)
		if err != nil {
			return nil, err
		}
		close_, ok := row[4].(string)
		if !ok {
			return nil, fmt.Errorf("can't convert close %v to string", row[4])
		}
		kLine.Close, err = strconv.ParseFloat(close_, 64)
		if err != nil {
			return nil, err
		}
		volume, ok := row[5].(string)
		if !ok {
			return nil, fmt.Errorf("can't convert volume %v to string", row[5])
		}
		kLine.Volume, err = strconv.ParseFloat(volume, 64)
		if err != nil {
			return nil, err
		}
		timestamp, ok := row[6].(float64)
		if !ok {
			return nil, fmt.Errorf("can't convert timestamp %v to float", row[6])
		}
		//需要额外加一毫秒
		kLine.Timestamp = time.Unix(int64(timestamp+1)/1000, 0)
		kLines = append(kLines, kLine)
	}
	return kLines, nil
}

func (api *API) GetPositions(ctx context.Context) ([]Position, error) {
	var positions []Position
	err := api.SendAuthenticatedHTTPRequest(
		ctx,
		http.MethodGet,
		"/fapi/v1/positionRisk",
		nil,
		&positions,
	)
	if err != nil {
		return nil, err
	}
	return positions, nil
}

func (api *API) GetServerTime(ctx context.Context) (*ServerTime, error) {
	var positions ServerTime
	err := api.SendAuthenticatedHTTPRequest(
		ctx,
		http.MethodGet,
		"/fapi/v1/time",
		nil,
		&positions,
	)
	if err != nil {
		return nil, err
	}
	return &positions, nil
}

func (api *API) PingServer(ctx context.Context) (*Ping, error) {
	var ping Ping
	return &ping, api.SendHTTPRequest(
		ctx,
		"/fapi/v1/ping",
		nil,
		&ping,
	)
}

func (api *API) GetAccount(ctx context.Context) (*Account, error) {
	var account Account
	err := api.SendAuthenticatedHTTPRequest(
		ctx,
		http.MethodGet,
		"/fapi/v2/account",
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
		"/fapi/v1/leverage",
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
		"/fapi/v1/marginType",
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
		"/fapi/v1/listenKey",
		nil,
		&listenKey,
	)
	if err != nil {
		logger.Debugf("Get ListenKey error %v", err)
		return nil, err
	}
	return &listenKey, nil
}

func (api *API) GetExchangeInfo(ctx context.Context) (*ExchangeInfo, error) {
	var resp ExchangeInfo
	if err := api.SendHTTPRequest(
		ctx,
		"/fapi/v1/exchangeInfo",
		nil,
		&resp,
	); err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		return nil, errors.New(resp.Msg)
	}
	return &resp, nil
}

func (api *API) GetPremiumIndex(ctx context.Context) ([]PremiumIndex, error) {
	resp := make([]PremiumIndex, 0)
	return resp, api.SendHTTPRequest(
		ctx,
		"/fapi/v1/premiumIndex",
		nil,
		&resp,
	)
}

func (api *API) GetHistoryKLines(ctx context.Context, symbol, interval string, startTime time.Time) ([]common.KLine, error) {
	kLines := make([]common.KLine, 0)
	retryCount := 10
	for {
		subCtxt, _ := context.WithTimeout(ctx, time.Second*15)
		o, err := api.GetKLines(subCtxt, KlineParams{
			Symbol:    symbol,
			Interval:  interval,
			StartTime: startTime.Unix() * 1000,
			Limit:     1000,
		})
		if err != nil {
			if retryCount <= 0 {
				return nil, err
			} else {
				retryCount--
				continue
			}
		}
		kLines = append(kLines, o...)
		if len(o) < 1000 {
			break
		}
		//startTime是include的
		startTime = o[len(o)-1].Timestamp.Add(time.Minute)
		time.Sleep(time.Second)
	}
	return kLines, nil
}

func (api *API) SubmitOrder(ctx context.Context, params NewOrderParams) (*Order, error) {
	var order Order
	err := api.SendAuthenticatedHTTPRequest(
		ctx,
		http.MethodPost,
		"/fapi/v1/order",
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
		"/fapi/v1/allOpenOrders",
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
		"/fapi/v1/order",
		&params,
		&order,
	)
	if err != nil {
		return nil, err
	}
	order.Symbol = params.Symbol
	return &order, nil
}

func (api *API) GetMultiAssetsMargin(ctx context.Context) (*MultiAssetsMargin, error) {
	var resp MultiAssetsMargin
	err := api.SendAuthenticatedHTTPRequest(
		ctx,
		http.MethodGet,
		"/fapi/v1/multiAssetsMargin",
		nil,
		&resp,
	)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (api *API) ChangeMultiAssetsMargin(ctx context.Context, params MultiAssetsMarginParam) (*Response, error) {
	var resp Response
	err := api.SendAuthenticatedHTTPRequest(
		ctx,
		http.MethodPost,
		"/fapi/v1/multiAssetsMargin",
		&params,
		&resp,
	)
	if err != nil {
		return nil, err
	}
	return &resp, nil
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
