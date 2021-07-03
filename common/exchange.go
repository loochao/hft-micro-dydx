package common

type ExchangeID int

const (
	UnknownExchange ExchangeID = iota
	BinanceUsdtFuture
	BinanceUsdtSpot
	BinanceBusdFuture
	BinanceBusdSpot
	BinanceUsdcSpot
	BinanceTusdSpot
	BinanceCoinFuture
	KucoinUsdtFuture
	KucoinUsdtSpot
	FtxUsdFuture
)
