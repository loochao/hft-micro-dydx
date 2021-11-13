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
	FtxUsdSpot
	OkexUsdtSpot
	MexcUsdtFuture
	BybitUsdtFuture
	HoubiUsdtFuture
	BitfinexUsdtFuture
	DydxUsdFuture
	OkexV5UsdtSpot
	OkexV5UsdtFuture
)
