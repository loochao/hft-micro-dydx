package okspot

import (
	"context"
	"fmt"
	"net/http"
)

func (api *API) GetAccounts(ctx context.Context) (accounts []Balance, _ error) {
	return accounts, api.SendAuthenticatedHTTPRequest(ctx, http.MethodGet, "/api/spot/v3/accounts", nil, &accounts)
}

func (api *API) SubmitOrder(ctx context.Context, request NewOrderParam) (resp OrderResponse, _ error) {
	return resp, api.SendAuthenticatedHTTPRequest(ctx, http.MethodPost, "/api/spot/v3/orders", request, &resp)
}

func (api *API) CancelBatchOrders(ctx context.Context, request []CancelBatchOrders) (resp map[string][]OrderResponse, _ error) {
	return resp, api.SendAuthenticatedHTTPRequest(ctx, http.MethodPost, "/api/spot/v3/cancel_batch_orders", request, &resp)
}

func (api *API) CancelOrders(ctx context.Context, request CancelOrderParam) (resp OrderResponse, _ error) {
	if request.ClientOid != "" {
		path := "/api/spot/v3/cancel_orders/"+request.ClientOid
		request.ClientOid = ""
		return resp, api.SendAuthenticatedHTTPRequest(ctx, http.MethodPost, path, request, &resp)
	}else if request.OrderId != "" {
		path := "/api/spot/v3/cancel_orders/"+request.OrderId
		request.ClientOid = ""
		return resp, api.SendAuthenticatedHTTPRequest(ctx, http.MethodPost, path, request, &resp)
	}else {
		return resp, fmt.Errorf("need client oid or order id")
	}
}

func (api *API) GetFundBalances(ctx context.Context) (accounts []Balance, _ error) {
	return accounts, api.SendAuthenticatedHTTPRequest(ctx, http.MethodGet, "/api/account/v3/wallet", nil, &accounts)
}
