package common

type BinaryKline struct {
	Exchange  ExchangeID
	CloseTime int64
	Open      float64
	Close     float64
	High      float64
	Low       float64
	Volume    float64
}

type BinaryTicker struct {
	Exchange  ExchangeID
	EventTime int64
	BidPrice  float64
	BidSize   float64
	AskPrice  float64
	AskSize   float64
}

// Spread = Y - X
type MatchedSpread struct {
	ServerTime           int64
	EventTime            int64
	XBidPrice            float64
	XBidSize             float64
	XAskPrice            float64
	XAskSize             float64
	YBidPrice            float64
	YBidSize             float64
	YAskPrice            float64
	YAskSize             float64
	XFundingRate         float64
	YFundingRate         float64
	ShortLastSpread      float64
	ShortMedianSpread    float64
	LongLastSpread       float64
	LongMedianSpread     float64
	SpreadQuantile995    float64
	SpreadQuantile95     float64
	SpreadQuantile80     float64
	SpreadQuantile50     float64
	SpreadQuantile20     float64
	SpreadQuantile05     float64
	SpreadQuantile005    float64
}

// Spread = Y - X
type MatchedSpread32 struct {
	ServerTime           int64
	EventTime            int64
	XBidPrice            float32
	XBidSize             float32
	XAskPrice            float32
	XAskSize             float32
	YBidPrice            float32
	YBidSize             float32
	YAskPrice            float32
	YAskSize             float32
	XFundingRate         float32
	YFundingRate         float32
	ShortLastSpread      float32
	ShortMedianSpread    float32
	LongLastSpread       float32
	LongMedianSpread     float32
	SpreadQuantile995    float32
	SpreadQuantile95     float32
	SpreadQuantile80     float32
	SpreadQuantile50     float32
	SpreadQuantile20     float32
	SpreadQuantile05     float32
	SpreadQuantile005    float32
}
