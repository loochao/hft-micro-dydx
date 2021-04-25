package okspot

import (
	"context"
	"net/http"
)

func (api *API) GetAccounts(ctx context.Context) (accounts []Balance, _ error) {
	return accounts, api.SendAuthenticatedHTTPRequest(ctx, http.MethodGet, "/api/spot/v3/accounts", nil, &accounts)
}

func (api *API) SubmitOrder(ctx context.Context, request NewOrderParam) (resp OrderResponse, _ error) {
	return resp, api.SendAuthenticatedHTTPRequest(ctx, http.MethodPost, "/api/spot/v3/orders", request, &resp)
}

func (api *API) CancelBatchOrders(ctx context.Context, request CancelBatchOrders) (resp map[string][]OrderResponse, _ error) {
	return resp, api.SendAuthenticatedHTTPRequest(ctx, http.MethodPost, "/api/spot/v3/cancel_batch_orders", request, &resp)
}

func (api *API) GetFundBalances(ctx context.Context) (accounts []Balance, _ error) {
	return accounts, api.SendAuthenticatedHTTPRequest(ctx, http.MethodGet, "/api/account/v3/wallet", nil, &accounts)
}
