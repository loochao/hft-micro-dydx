package okexv5_usdtswap

import (
	"context"
	"fmt"
	"github.com/geometrybase/hft-micro/logger"
	"net/http"
)

func (api *API) GetAccount(ctx context.Context) (*Account, error) {
	data := make([]AccountData, 0)
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

func (api *API) GetPositions(ctx context.Context) ([]Position, error) {
	positions := make([]Position, 0)
	return positions, api.SendAuthenticatedHTTPRequest(ctx, http.MethodGet, "/api/v5/account/positions?instType=SWAP", nil, &positions)
}
func (api *API) GetAccountConfig(ctx context.Context) (*AccountConfig, error) {
	configs := make([]AccountConfig, 0)
	err := api.SendAuthenticatedHTTPRequest(ctx, http.MethodGet, "/api/v5/account/config", nil, &configs)
	if err != nil {
		return nil, err
	}
	if len(configs) == 0 {
		return nil, fmt.Errorf("no account config found")
	}
	config := configs[0]
	return &config, nil
}

func (api *API) UpdatePositionMode(ctx context.Context) error {
	config, err := api.GetAccountConfig(ctx)
	if err != nil {
		return err
	}
	if config.AutoLoan {
		logger.Warnf("AUTO LOAN IS ONNNNNNNNNNNNNNNNNN......")
	}
	if config.PosMode == "net_mode" {
		return nil
	}
	inputPm := &PositionMode{
		PosMode: "net_mode",
	}
	outputPm := &PositionMode{}
	err = api.SendAuthenticatedHTTPRequest(ctx, http.MethodPost, "/api/v5/account/set-position-mode", inputPm, outputPm)
	if err != nil {
		return  err
	}
	if outputPm.PosMode != inputPm.PosMode {
		return fmt.Errorf("change pos mode to net_mode failed current %s", outputPm.PosMode)
	}
	return nil
}


func (api *API) UpdateLeverage(ctx context.Context, param Leverage) error {
	outputLevers := make([]Leverage, 0)
	param.PosSide = "net"
	param.MgnMode = "cross"
	err := api.SendAuthenticatedHTTPRequest(ctx, http.MethodPost, "/api/v5/account/set-leverage", &param, &outputLevers)
	if err != nil {
		return  err
	}
	if len(outputLevers) == 0 {
		return fmt.Errorf("set lever, no result return")
	}
	return nil
}

func (api *API) SubmitOrder(ctx context.Context, param NewOrderParam) (*OrderResponse, error) {
	param.TdMode = TdModeCross
	param.Ccy = "USDT"
	ors := make([]OrderResponse, 0)
	err := api.SendAuthenticatedHTTPRequest(ctx, http.MethodPost, "/api/v5/trade/order", param, &ors)
	if err != nil {
		return nil, err
	}
	if len(ors) > 0 {
		r := ors[0]
		return &r, nil
	} else {
		return nil, fmt.Errorf("no response from request %v", param)
	}
}

func (api *API) CancelOrders(ctx context.Context, param CancelOrderParam) (*OrderResponse, error) {
	ors := make([]OrderResponse, 0)
	err := api.SendAuthenticatedHTTPRequest(ctx, http.MethodPost, "/api/v5/trade/cancel-order", param, &ors)
	if err != nil {
		return nil, err
	}
	if len(ors) > 0 {
		r := ors[0]
		return &r, nil
	} else {
		return nil, fmt.Errorf("no response from request %v", param)
	}
}
