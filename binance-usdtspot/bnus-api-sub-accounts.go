package binance_usdtspot

import (
	"context"
	"net/http"
)

func (api *API) QuerySubAccountList(ctx context.Context) ([]SubAccount, map[string]int64, error) {
	var subAccounts = &SubAccounts{}
	limits, err := api.SendAuthenticatedHTTPRequest(
		ctx,
		http.MethodGet,
		"/sapi/v1/sub-account/list",
		nil,
		subAccounts,
	)
	return subAccounts.SubAccounts, limits, err
}


func (api *API) QuerySubAccountAssets(ctx context.Context, params SubAccountParams) ([]SubAccountBalance, map[string]int64, error) {
	var subAccounts = &SubAccountBalances{}
	limits, err := api.SendAuthenticatedHTTPRequest(
		ctx,
		http.MethodGet,
		"/sapi/v3/sub-account/assets",
		&params,
		subAccounts,
	)
	return subAccounts.Balances, limits, err
}

func (api *API) QuerySubAccountFuturesAccount(ctx context.Context, params SubAccountParams) (*SubAccountFutureAccount, map[string]int64, error) {
	var subAccounts = &SubAccountFutureAccount{}
	limits, err := api.SendAuthenticatedHTTPRequest(
		ctx,
		http.MethodGet,
		"/sapi/v1/sub-account/futures/account",
		&params,
		subAccounts,
	)
	return subAccounts, limits, err
}

func (api *API) QuerySubAccountFuturePositionRisk(ctx context.Context, params SubAccountParams) ([]PositionRisk, map[string]int64, error) {
	risks := make([]PositionRisk, 0)
	limits, err := api.SendAuthenticatedHTTPRequest(
		ctx,
		http.MethodGet,
		"/sapi/v1/sub-account/futures/positionRisk",
		&params,
		&risks,
	)
	return risks, limits, err
}

func (api *API) SubAccountUniversalTransfer(ctx context.Context, params SubAccountUniversalTransferParams) (*UniversalTransferResp, map[string]int64, error) {
	var resp = &UniversalTransferResp{}
	limits, err := api.SendAuthenticatedHTTPRequest(
		ctx,
		http.MethodPost,
		"/sapi/v1/sub-account/universalTransfer",
		&params,
		resp,
	)
	return resp, limits, err
}
