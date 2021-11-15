package okexv5_usdtswap

import (
	"context"
	"fmt"
)

func (api *API) GetInstruments(ctx context.Context) ([]Instrument, error) {
	var instruments []Instrument
	return instruments, api.SendHTTPRequest(ctx, "/api/v5/public/instruments?instType=SWAP", &instruments)
}

func (api *API) GetPositionTiers(ctx context.Context, param PositionTierParam) ([]PositionTier, error) {
	var positionTiers []PositionTier
	return positionTiers, api.SendHTTPRequest(
		ctx,
		fmt.Sprintf("/api/v5/public/position-tiers?instType=%s&tdMode=%s&tier=%s&uly=%s", param.InstType, param.TdMode, param.Tier, param.Uly),
		&positionTiers,
	)
}

func (api *API) GetStatus(ctx context.Context) ([]Status, error) {
	stats := make([]Status, 0)
	return stats, api.SendHTTPRequest(ctx, "/api/v5/system/status", &stats)
}
