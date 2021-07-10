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
