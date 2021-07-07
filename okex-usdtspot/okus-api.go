package okex_usdtspot

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
	"sync"
	"time"
)

const (
	okspotWsSubsection         = "spot/"
	okspotWsCandle             = "candle"
	okspotWsCandle60s          = okspotWsSubsection + okspotWsCandle + "60s"
	okspotWsCandle180s         = okspotWsSubsection + okspotWsCandle + "180s"
	okspotWsCandle300s         = okspotWsSubsection + okspotWsCandle + "300s"
	okspotWsCandle900s         = okspotWsSubsection + okspotWsCandle + "900s"
	okspotWsCandle1800s        = okspotWsSubsection + okspotWsCandle + "1800s"
	okspotWsCandle3600s        = okspotWsSubsection + okspotWsCandle + "3600s"
	okspotWsCandle7200s        = okspotWsSubsection + okspotWsCandle + "7200s"
	okspotWsCandle14400s       = okspotWsSubsection + okspotWsCandle + "14400s"
	okspotWsCandle21600s       = okspotWsSubsection + okspotWsCandle + "21600"
	okspotWsCandle43200s       = okspotWsSubsection + okspotWsCandle + "43200s"
	okspotWsCandle86400s       = okspotWsSubsection + okspotWsCandle + "86400s"
	okspotWsCandle604900s      = okspotWsSubsection + okspotWsCandle + "604800s"
	okspotWsTicker             = okspotWsSubsection + "ticker"
	okspotWsTrade              = okspotWsSubsection + "trade"
	okspotWsDepth5             = okspotWsSubsection + "depth5"
	okspotWsAccount            = okspotWsSubsection + "account"
	okspotWsMarginAccount      = okspotWsSubsection + "margin_account"
	okspotWsOrder              = okspotWsSubsection + "order"
	okspotTimeLayout           = "2006-01-02T15:04:05.999Z"
	okspotAPIUrl               = ""
	WsUrl                      = "wss://real.okex.com:8443/ws/v3"
	AWSWsUrl                   = "wss: //awspush.okex.com:8443/ws/v3"
	ApiUrl                     = "https://www.okex.com"
	AWSApiUrl                  = "https://aws.okex.com"
	OrderTypeNormalOrder       = "0"
	OrderTypePostOnly          = "1"
	OrderTypeFillOrKill        = "2"
	OrderTypeImmediateOrCancel = "3"
	OrderSideBuy               = "buy"
	OrderSideSell              = "sell"
	OrderLimit                 = "limit"
	OrderMarket                = "market"
	OrderStateFailed           = "-2"
	OrderStateCanceled         = "-1"
	OrderStateOpen             = "0"
	OrderStatePartiallyFilled  = "1"
	OrderStateFullyFilled      = "2"
	OrderStateSubmitting       = "3"
	OrderStateCancelling       = "4"
	ExchangeID                 = common.OkexUsdtSpot
)

var ErrorCodes = map[string]error{
	"0":     errors.New("successful"),
	"1":     errors.New("invalid parameter in url normally"),
	"30001": errors.New("request header \"OK_ACCESS_KEY\" cannot be blank"),
	"30002": errors.New("request header \"OK_ACCESS_SIGN\" cannot be blank"),
	"30003": errors.New("request header \"OK_ACCESS_TIMESTAMP\" cannot be blank"),
	"30004": errors.New("request header \"OK_ACCESS_PASSPHRASE\" cannot be blank"),
	"30005": errors.New("invalid OK_ACCESS_TIMESTAMP"),
	"30006": errors.New("invalid OK_ACCESS_KEY"),
	"30007": errors.New("invalid Content_Type, please use \"application/json\" format"),
	"30008": errors.New("timestamp request expired"),
	"30009": errors.New("system error"),
	"30010": errors.New("api validation failed"),
	"30011": errors.New("invalid IP"),
	"30012": errors.New("invalid authorization"),
	"30013": errors.New("invalid sign"),
	"30014": errors.New("request too frequent"),
	"30015": errors.New("request header \"OK_ACCESS_PASSPHRASE\" incorrect"),
	"30016": errors.New("you are using v1 apiKey, please use v1 endpoint. If you would like to use v3 endpoint, please subscribe to v3 apiKey"),
	"30017": errors.New("apikey's broker id does not match"),
	"30018": errors.New("apikey's domain does not match"),
	"30020": errors.New("body cannot be blank"),
	"30021": errors.New("json data format error"),
	"30023": errors.New("required parameter cannot be blank"),
	"30024": errors.New("parameter value error"),
	"30025": errors.New("parameter category error"),
	"30026": errors.New("requested too frequent; endpoint limit exceeded"),
	"30027": errors.New("login failure"),
	"30028": errors.New("unauthorized execution"),
	"30029": errors.New("account suspended"),
	"30030": errors.New("endpoint request failed. Please try again"),
	"30031": errors.New("token does not exist"),
	"30032": errors.New("pair does not exist"),
	"30033": errors.New("exchange domain does not exist"),
	"30034": errors.New("exchange ID does not exist"),
	"30035": errors.New("trading is not supported in this website"),
	"30036": errors.New("no relevant data"),
	"30037": errors.New("endpoint is offline or unavailable"),
	"30038": errors.New("user does not exist"),
	"32001": errors.New("futures account suspended"),
	"32002": errors.New("futures account does not exist"),
	"32003": errors.New("canceling, please wait"),
	"32004": errors.New("you have no unfilled orders"),
	"32005": errors.New("max order quantity"),
	"32006": errors.New("the order price or trigger price exceeds USD 1 million"),
	"32007": errors.New("leverage level must be the same for orders on the same side of the contract"),
	"32008": errors.New("max. positions to open (cross margin)"),
	"32009": errors.New("max. positions to open (fixed margin)"),
	"32010": errors.New("leverage cannot be changed with open positions"),
	"32011": errors.New("futures status error"),
	"32012": errors.New("futures order update error"),
	"32013": errors.New("token type is blank"),
	"32014": errors.New("your number of contracts closing is larger than the number of contracts available"),
	"32015": errors.New("margin ratio is lower than 100% before opening positions"),
	"32016": errors.New("margin ratio is lower than 100% after opening position"),
	"32017": errors.New("no BBO"),
	"32018": errors.New("the order quantity is less than 1, please try again"),
	"32019": errors.New("the order price deviates from the price of the previous minute by more than 3%"),
	"32020": errors.New("the price is not in the range of the price limit"),
	"32021": errors.New("leverage error"),
	"32022": errors.New("this function is not supported in your country or region according to the regulations"),
	"32023": errors.New("this account has outstanding loan"),
	"32024": errors.New("order cannot be placed during delivery"),
	"32025": errors.New("order cannot be placed during settlement"),
	"32026": errors.New("your account is restricted from opening positions"),
	"32027": errors.New("cancelled over 20 orders"),
	"32028": errors.New("account is suspended and liquidated"),
	"32029": errors.New("order info does not exist"),
	"33001": errors.New("margin account for this pair is not enabled yet"),
	"33002": errors.New("margin account for this pair is suspended"),
	"33003": errors.New("no loan balance"),
	"33004": errors.New("loan amount cannot be smaller than the minimum limit"),
	"33005": errors.New("repayment amount must exceed 0"),
	"33006": errors.New("loan order not found"),
	"33007": errors.New("status not found"),
	"33008": errors.New("loan amount cannot exceed the maximum limit"),
	"33009": errors.New("user ID is blank"),
	"33010": errors.New("you cannot cancel an order during session 2 of call auction"),
	"33011": errors.New("no new market data"),
	"33012": errors.New("order cancellation failed"),
	"33013": errors.New("order placement failed"),
	"33014": errors.New("order does not exist"),
	"33015": errors.New("exceeded maximum limit"),
	"33016": errors.New("margin trading is not open for this token"),
	"33017": errors.New("insufficient balance"),
	"33018": errors.New("this parameter must be smaller than 1"),
	"33020": errors.New("request not supported"),
	"33021": errors.New("token and the pair do not match"),
	"33022": errors.New("pair and the order do not match"),
	"33023": errors.New("you can only place market orders during call auction"),
	"33024": errors.New("trading amount too small"),
	"33025": errors.New("base token amount is blank"),
	"33026": errors.New("transaction completed"),
	"33027": errors.New("cancelled order or order cancelling"),
	"33028": errors.New("the decimal places of the trading price exceeded the limit"),
	"33029": errors.New("the decimal places of the trading size exceeded the limit"),
	"34001": errors.New("withdrawal suspended"),
	"34002": errors.New("please add a withdrawal address"),
	"34003": errors.New("sorry, this token cannot be withdrawn to xx at the moment"),
	"34004": errors.New("withdrawal fee is smaller than minimum limit"),
	"34005": errors.New("withdrawal fee exceeds the maximum limit"),
	"34006": errors.New("withdrawal amount is lower than the minimum limit"),
	"34007": errors.New("withdrawal amount exceeds the maximum limit"),
	"34008": errors.New("insufficient balance"),
	"34009": errors.New("your withdrawal amount exceeds the daily limit"),
	"34010": errors.New("transfer amount must be larger than 0"),
	"34011": errors.New("conditions not met"),
	"34012": errors.New("the minimum withdrawal amount for NEO is 1, and the amount must be an integer"),
	"34013": errors.New("please transfer"),
	"34014": errors.New("transfer limited"),
	"34015": errors.New("subaccount does not exist"),
	"34016": errors.New("transfer suspended"),
	"34017": errors.New("account suspended"),
	"34018": errors.New("incorrect trades password"),
	"34019": errors.New("please bind your email before withdrawal"),
	"34020": errors.New("please bind your funds password before withdrawal"),
	"34021": errors.New("not verified address"),
	"34022": errors.New("withdrawals are not available for sub accounts"),
	"35001": errors.New("contract subscribing does not exist"),
	"35002": errors.New("contract is being settled"),
	"35003": errors.New("contract is being paused"),
	"35004": errors.New("pending contract settlement"),
	"35005": errors.New("perpetual swap trading is not enabled"),
	"35008": errors.New("margin ratio too low when placing order"),
	"35010": errors.New("closing position size larger than available size"),
	"35012": errors.New("placing an order with less than 1 contract"),
	"35014": errors.New("order size is not in acceptable range"),
	"35015": errors.New("leverage level unavailable"),
	"35017": errors.New("changing leverage level"),
	"35019": errors.New("order size exceeds limit"),
	"35020": errors.New("order price exceeds limit"),
	"35021": errors.New("order size exceeds limit of the current tier"),
	"35022": errors.New("contract is paused or closed"),
	"35030": errors.New("place multiple orders"),
	"35031": errors.New("cancel multiple orders"),
	"35061": errors.New("invalid instrument_id"),
}

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
		mu:          sync.Mutex{},
	}
	return &api, nil
}
