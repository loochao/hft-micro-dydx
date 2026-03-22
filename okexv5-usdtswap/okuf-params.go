package okexv5_usdtswap

type NewOrderParam struct {
	InstId     string  `json:"instId"`
	Ccy        string  `json:"ccy"`
	TdMode     string  `json:"tdMode"`
	ClOrdId    string  `json:"clOrdId,omitempty"`
	Side       string  `json:"side"`
	OrderType  string  `json:"ordType"`
	ReduceOnly bool    `json:"reduceOnly,omitempty"`
	Size       string  `json:"sz"`
	Price      *string `json:"px,omitempty"`
}

type CancelOrderParam struct {
	InstId  string `json:"instId"`
	OrdId   string `json:"ordId,omitempty"`
	ClOrdId string `json:"clOrdId,omitempty"`
}

type PositionMode struct {
	PosMode string `json:"posMode"`
}

type Leverage struct {
	InstId  string `json:"instId"`
	Lever   int    `json:"lever,string"`
	MgnMode string `json:"mgnMode"`
	PosSide string `json:"posSide,omitempty"`
}
