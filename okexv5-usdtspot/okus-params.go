package okexv5_usdtspot

type NewOrderParam struct {
	InstId     string  `json:"instId"`
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
