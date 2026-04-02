package dydx_v4_usdfuture

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/geometrybase/hft-micro/logger"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type V4API struct {
	client           *http.Client
	address          string
	subaccountNumber int
}

func NewV4API(address string, subaccountNumber int, proxy string) (*V4API, error) {
	var transport *http.Transport
	if proxy != "" {
		proxyURL, err := url.Parse(proxy)
		if err != nil {
			return nil, fmt.Errorf("url.Parse proxy error %v", err)
		}
		transport = &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		}
	} else {
		transport = &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		}
	}
	return &V4API{
		client: &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
		},
		address:          address,
		subaccountNumber: subaccountNumber,
	}, nil
}

func (api *V4API) doGet(ctx context.Context, path string, result interface{}) error {
	fullURL := IndexerRestURL + path
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return err
	}
	resp, err := api.client.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(contents))
	}
	return json.Unmarshal(contents, result)
}

func (api *V4API) GetPerpetualMarkets(ctx context.Context) (map[string]PerpetualMarket, error) {
	var resp PerpetualMarketsResp
	err := api.doGet(ctx, "/v4/perpetualMarkets", &resp)
	if err != nil {
		return nil, err
	}
	return resp.Markets, nil
}

func (api *V4API) GetSubaccount(ctx context.Context) (*Subaccount, error) {
	path := fmt.Sprintf("/v4/addresses/%s/subaccountNumber/%d", api.address, api.subaccountNumber)
	var resp SubaccountResp
	err := api.doGet(ctx, path, &resp)
	if err != nil {
		return nil, err
	}
	return &resp.Subaccount, nil
}

func (api *V4API) GetOrderbook(ctx context.Context, ticker string) (*OrderbookResp, error) {
	path := fmt.Sprintf("/v4/orderbooks/perpetualMarket/%s", ticker)
	var resp OrderbookResp
	err := api.doGet(ctx, path, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (api *V4API) GetOrders(ctx context.Context) ([]V4Order, error) {
	path := fmt.Sprintf("/v4/orders?address=%s&subaccountNumber=%d&limit=100", api.address, api.subaccountNumber)
	var resp []V4Order
	err := api.doGet(ctx, path, &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (api *V4API) GetHistoricalFunding(ctx context.Context, ticker string) ([]HistoricalFundingEntry, error) {
	path := fmt.Sprintf("/v4/historicalFunding/%s", ticker)
	var resp HistoricalFundingResp
	err := api.doGet(ctx, path, &resp)
	if err != nil {
		return nil, err
	}
	return resp.HistoricalFunding, nil
}

func (api *V4API) CheckHealth(ctx context.Context) error {
	path := "/v4/perpetualMarkets"
	var resp PerpetualMarketsResp
	err := api.doGet(ctx, path, &resp)
	if err != nil {
		return err
	}
	if len(resp.Markets) == 0 {
		return errors.New("no markets returned")
	}
	return nil
}

func ParseOrderbookToDepth(resp *OrderbookResp, market string) *V4Depth {
	depth := &V4Depth{
		Market:           market,
		ParseTime:        time.Now(),
		WithSnapshotData: true,
	}
	depth.Bids = make([][2]float64, 0, len(resp.Bids))
	for _, bid := range resp.Bids {
		price, _ := strconv.ParseFloat(bid.Price, 64)
		size, _ := strconv.ParseFloat(bid.Size, 64)
		if size > 0 {
			depth.Bids = append(depth.Bids, [2]float64{price, size})
		}
	}
	depth.Asks = make([][2]float64, 0, len(resp.Asks))
	for _, ask := range resp.Asks {
		price, _ := strconv.ParseFloat(ask.Price, 64)
		size, _ := strconv.ParseFloat(ask.Size, 64)
		if size > 0 {
			depth.Asks = append(depth.Asks, [2]float64{price, size})
		}
	}
	return depth
}

func LoadMarketInfo(ctx context.Context, api *V4API) (
	tickSizes map[string]float64,
	stepSizes map[string]float64,
	minSizes map[string]float64,
	err error,
) {
	markets, err := api.GetPerpetualMarkets(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	tickSizes = make(map[string]float64)
	stepSizes = make(map[string]float64)
	minSizes = make(map[string]float64)
	for ticker, m := range markets {
		ts := m.TickSizeFloat()
		ss := m.StepSizeFloat()
		tickSizes[ticker] = ts
		stepSizes[ticker] = ss
		minSizes[ticker] = ss
		logger.Debugf("market %s tickSize=%v stepSize=%v", ticker, ts, ss)
	}
	return tickSizes, stepSizes, minSizes, nil
}
