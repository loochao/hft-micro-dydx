package okspot

import (
	"context"
	"net/http"
)

func (api *API) GetAccounts(ctx context.Context, credentials Credentials) (accounts []Balance, _ error) {
	return accounts, api.SendAuthenticatedHTTPRequest(ctx, credentials, http.MethodGet, "/api/spot/v3/accounts", nil, &accounts)
}

func (api *API) SubmitOrder(ctx context.Context, credentials Credentials, request NewOrderParams) (resp NewOrderResponse, _ error) {
	return resp, api.SendAuthenticatedHTTPRequest(ctx, credentials, http.MethodPost, "/api/spot/v3/orders", request, &resp)
}

func (api *API) GetFundBalances(ctx context.Context, credentials Credentials) (accounts []Balance, _ error) {
	return accounts, api.SendAuthenticatedHTTPRequest(ctx, credentials, http.MethodGet, "/api/account/v3/wallet", nil, &accounts)
}
