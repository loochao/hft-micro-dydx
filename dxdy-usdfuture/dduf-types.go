package dxdy_usdtfuture

type WSOrderBookSubscribe struct {
	Type           string `json:"type"`
	Channel        string `json:"channel"`
	Id             string `json:"id"`
	IncludeOffsets bool   `json:"includeOffsets,omitempty"`
}
