package common

type ExchangeID uint8

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
	OkexUsdtSpot
	MexcUsdtFuture
	BybitUsdtFuture
	HoubiUsdtFuture
)
