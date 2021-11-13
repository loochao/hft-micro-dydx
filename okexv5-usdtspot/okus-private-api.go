package okexv5_usdtspot

import (
	"context"
	"fmt"
	"net/http"
)

func (api *API) GetAccount(ctx context.Context) (*Balance, error) {
	data := make([]BalancesData, 0)
	err := api.SendAuthenticatedHTTPRequest(ctx, http.MethodGet, "/api/v5/account/balance?ccy=USDT", nil, &data)
	if err != nil {
		return nil, err
	}
	if len(data) > 0 {
		for _, d := range data[0].Details {
			if d.Ccy == "USDT" {
				return &d, nil
			}
		}
	}
	return nil, fmt.Errorf("no usdt balance found")
}

func (api *API) GetBalances(ctx context.Context) ([]Balance, error) {
	data := make([]BalancesData, 0)
	err := api.SendAuthenticatedHTTPRequest(ctx, http.MethodGet, "/api/v5/account/balance", nil, &data)
	if err != nil {
		return nil, err
	}
	balances := make([]Balance, 0)
	if len(data) > 0 {
		for _, d := range data {
			for _, dd := range d.Details {
				if dd.Ccy != "USDT" {
					balances = append(balances, dd)
				}
			}
		}
	}
	return balances, nil
}

func (api *API) SubmitOrder(ctx context.Context, param NewOrderParam) (*OrderResponse,  error) {
	param.TdMode = TdModeCash
	ors := make([]OrderResponse, 0)
	err := api.SendAuthenticatedHTTPRequest(ctx, http.MethodPost, "/api/v5/trade/order", param, &ors)
	if err != nil {
		return nil, err
	}
	if len(ors) > 0 {
		r := ors[0]
		return &r, nil
	}else{
		return nil, fmt.Errorf("no response from request %v", param)
	}
}


func (api *API) CancelOrders(ctx context.Context,  param CancelOrderParam) (*OrderResponse,error) {
	ors := make([]OrderResponse, 0)
	err := api.SendAuthenticatedHTTPRequest(ctx, http.MethodPost, "/api/v5/trade/cancel-order", param, &ors)
	if err != nil {
		return nil, err
	}
	if len(ors) > 0 {
		r := ors[0]
		return &r, nil
	}else{
		return nil, fmt.Errorf("no response from request %v", param)
	}
}
