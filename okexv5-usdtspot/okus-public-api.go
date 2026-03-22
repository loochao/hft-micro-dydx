package okexv5_usdtspot

import (
	"context"
)

func (api *API) GetInstruments(ctx context.Context) ([]Instrument, error) {
	var instruments []Instrument
	return instruments, api.SendHTTPRequest(ctx, "/api/v5/public/instruments?instType=SPOT", &instruments)
}

func (api *API) GetStatus(ctx context.Context) ([]Status, error) {
	stats := make([]Status, 0)
	return stats, api.SendHTTPRequest(ctx, "/api/v5/system/status", &stats)
}
