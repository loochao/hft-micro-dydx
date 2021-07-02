package common

type ExchangeID uint8

const (
	UnknownExchange uint8 = iota
	BinanceUsdtFuture
	BinanceUsdtSpot
)
