package kucoin_usdtspot

import (
	"github.com/geometrybase/hft-micro/common"
	"time"
)

const (
	CandleType1Min   = "1min"
	CandleType3Min   = "3min"
	CandleType5Min   = "5min"
	CandleType15Min  = "15min"
	CandleType30Min  = "30min"
	CandleType1Hour  = "1hour"
	CandleType4Hour  = "4hour"
	CandleType6Hour  = "6hour"
	CandleType8Hour  = "8hour"
	CandleType12Hour = "12hour"
	CandleType1Day   = "1day"
	CandleType1Week  = "1week"

	OrderStatusOpen  = "open"
	OrderStatusMatch = "match"
	OrderStatusDone  = "done"

	OrderTypeOpen     = "open"
	OrderTypeMatch    = "match"
	OrderTypeFilled   = "filled"
	OrderTypeCanceled = "canceled"
	OrderTypeUpdate   = "update"

	SystemStatusOpen       = "open"
	SystemStatusCancelOnly = "cancelonly"
	SystemStatusClose      = "close"

	TradeSideBuy  = "buy"
	TradeSideSell = "sell"
	ExchangeID    = common.KucoinUsdtSpot
)

var CandleTypeDurations = map[string]time.Duration{
	CandleType1Min:   time.Minute,
	CandleType3Min:   time.Minute * 3,
	CandleType5Min:   time.Minute * 5,
	CandleType15Min:  time.Minute * 15,
	CandleType30Min:  time.Minute * 30,
	CandleType1Hour:  time.Hour,
	CandleType4Hour:  time.Hour * 4,
	CandleType6Hour:  time.Hour * 6,
	CandleType8Hour:  time.Hour * 8,
	CandleType12Hour: time.Hour * 12,
	CandleType1Day:   time.Hour * 24,
	CandleType1Week:  time.Hour * 168,
}


var TickerTestLines = `{"data":{"sequence":"1618200193283","bestAsk":"32704.5","size":"0.0017491","bestBidSize":"0.06093004","price":"32704.4","time":1626290933704,"bestAskSize":"0.02047287","bestBid":"32704.4"},"subject":"trade.ticker","topic":"/market/ticker:BTC-USDT","type":"message"}
{"data":{"sequence":"1612914399696","bestAsk":"13.3792","size":"8","bestBidSize":"167.5597","price":"13.3365","time":1626290933890,"bestAskSize":"2.4461","bestBid":"13.3339"},"subject":"trade.ticker","topic":"/market/ticker:WAVES-USDT","type":"message"}
{"data":{"sequence":"1618200193317","bestAsk":"32704.5","size":"0.0017491","bestBidSize":"0.06093004","price":"32704.4","time":1626290933804,"bestAskSize":"0.02047287","bestBid":"32704.4"},"subject":"trade.ticker","topic":"/market/ticker:BTC-USDT","type":"message"}
{"data":{"sequence":"1618200193358","bestAsk":"32704.5","size":"0.0017491","bestBidSize":"0.06093004","price":"32704.4","time":1626290933903,"bestAskSize":"0.02047287","bestBid":"32704.4"},"subject":"trade.ticker","topic":"/market/ticker:BTC-USDT","type":"message"}
{"data":{"sequence":"1618200193387","bestAsk":"32704.5","size":"0.0017491","bestBidSize":"0.06793004","price":"32704.4","time":1626290934003,"bestAskSize":"0.02047287","bestBid":"32704.4"},"subject":"trade.ticker","topic":"/market/ticker:BTC-USDT","type":"message"}
{"data":{"sequence":"1612914399699","bestAsk":"13.3792","size":"8","bestBidSize":"167.5597","price":"13.3365","time":1626290934090,"bestAskSize":"2.4461","bestBid":"13.3339"},"subject":"trade.ticker","topic":"/market/ticker:WAVES-USDT","type":"message"}
{"data":{"sequence":"1618200193483","bestAsk":"32704.5","size":"0.0017491","bestBidSize":"0.06793004","price":"32704.4","time":1626290934103,"bestAskSize":"0.02047287","bestBid":"32704.4"},"subject":"trade.ticker","topic":"/market/ticker:BTC-USDT","type":"message"}
{"data":{"sequence":"1612914399701","bestAsk":"13.3792","size":"8","bestBidSize":"167.5597","price":"13.3365","time":1626290934190,"bestAskSize":"2.4461","bestBid":"13.3339"},"subject":"trade.ticker","topic":"/market/ticker:WAVES-USDT","type":"message"}
{"data":{"sequence":"1618200193498","bestAsk":"32704.5","size":"0.0017491","bestBidSize":"0.06793004","price":"32704.4","time":1626290934203,"bestAskSize":"0.02047287","bestBid":"32704.4"},"subject":"trade.ticker","topic":"/market/ticker:BTC-USDT","type":"message"}
{"data":{"sequence":"1618200193522","bestAsk":"32704.5","size":"0.0017491","bestBidSize":"0.06793004","price":"32704.4","time":1626290934303,"bestAskSize":"0.02047287","bestBid":"32704.4"},"subject":"trade.ticker","topic":"/market/ticker:BTC-USDT","type":"message"}
{"data":{"sequence":"1618200193538","bestAsk":"32704.5","size":"0.0017491","bestBidSize":"0.06793004","price":"32704.4","time":1626290934403,"bestAskSize":"0.02047287","bestBid":"32704.4"},"subject":"trade.ticker","topic":"/market/ticker:BTC-USDT","type":"message"}
{"data":{"sequence":"1618200193570","bestAsk":"32704.5","size":"0.0017491","bestBidSize":"0.06793004","price":"32704.4","time":1626290934503,"bestAskSize":"0.02047287","bestBid":"32704.4"},"subject":"trade.ticker","topic":"/market/ticker:BTC-USDT","type":"message"}
{"data":{"sequence":"1618200193593","bestAsk":"32704.5","size":"0.0017491","bestBidSize":"0.06793004","price":"32704.4","time":1626290934603,"bestAskSize":"0.02047287","bestBid":"32704.4"},"subject":"trade.ticker","topic":"/market/ticker:BTC-USDT","type":"message"}
{"data":{"sequence":"1612914399704","bestAsk":"13.3792","size":"8","bestBidSize":"167.5597","price":"13.3365","time":1626290934688,"bestAskSize":"2.4461","bestBid":"13.3339"},"subject":"trade.ticker","topic":"/market/ticker:WAVES-USDT","type":"message"}
{"data":{"sequence":"1618200193613","bestAsk":"32704.5","size":"0.0017491","bestBidSize":"0.06793004","price":"32704.4","time":1626290934703,"bestAskSize":"0.02047287","bestBid":"32704.4"},"subject":"trade.ticker","topic":"/market/ticker:BTC-USDT","type":"message"}
{"data":{"sequence":"1612914399705","bestAsk":"13.3792","size":"8","bestBidSize":"167.5597","price":"13.3365","time":1626290934789,"bestAskSize":"2.4461","bestBid":"13.3339"},"subject":"trade.ticker","topic":"/market/ticker:WAVES-USDT","type":"message"}
{"data":{"sequence":"1618200193633","bestAsk":"32704.5","size":"0.0017491","bestBidSize":"0.06793004","price":"32704.4","time":1626290934803,"bestAskSize":"0.02055972","bestBid":"32704.4"},"subject":"trade.ticker","topic":"/market/ticker:BTC-USDT","type":"message"}
{"data":{"sequence":"1612914399707","bestAsk":"13.3792","size":"8","bestBidSize":"167.5597","price":"13.3365","time":1626290934889,"bestAskSize":"2.4461","bestBid":"13.3339"},"subject":"trade.ticker","topic":"/market/ticker:WAVES-USDT","type":"message"}
{"data":{"sequence":"1618200193650","bestAsk":"32704.5","size":"0.0017491","bestBidSize":"0.06793004","price":"32704.4","time":1626290934903,"bestAskSize":"0.02055972","bestBid":"32704.4"},"subject":"trade.ticker","topic":"/market/ticker:BTC-USDT","type":"message"}
{"data":{"sequence":"1612914399710","bestAsk":"13.3792","size":"8","bestBidSize":"167.5597","price":"13.3365","time":1626290934989,"bestAskSize":"2.4461","bestBid":"13.3339"},"subject":"trade.ticker","topic":"/market/ticker:WAVES-USDT","type":"message"}
{"data":{"sequence":"1618200193672","bestAsk":"32704.5","size":"0.0017491","bestBidSize":"0.06793004","price":"32704.4","time":1626290935003,"bestAskSize":"0.02055972","bestBid":"32704.4"},"subject":"trade.ticker","topic":"/market/ticker:BTC-USDT","type":"message"}
{"data":{"sequence":"1618200193755","bestAsk":"32704.5","size":"0.0017491","bestBidSize":"0.06793004","price":"32704.4","time":1626290935103,"bestAskSize":"0.02055972","bestBid":"32704.4"},"subject":"trade.ticker","topic":"/market/ticker:BTC-USDT","type":"message"}
{"data":{"sequence":"1612914399714","bestAsk":"13.3744","size":"8","bestBidSize":"167.5597","price":"13.3365","time":1626290935189,"bestAskSize":"37.826","bestBid":"13.3339"},"subject":"trade.ticker","topic":"/market/ticker:WAVES-USDT","type":"message"}
{"data":{"sequence":"1618200193780","bestAsk":"32704.5","size":"0.0017491","bestBidSize":"0.06793004","price":"32704.4","time":1626290935203,"bestAskSize":"0.02055972","bestBid":"32704.4"},"subject":"trade.ticker","topic":"/market/ticker:BTC-USDT","type":"message"}
{"data":{"sequence":"1612914399724","bestAsk":"13.3733","size":"8","bestBidSize":"167.5597","price":"13.3365","time":1626290935290,"bestAskSize":"26.1762","bestBid":"13.3339"},"subject":"trade.ticker","topic":"/market/ticker:WAVES-USDT","type":"message"}
{"data":{"sequence":"1618200193822","bestAsk":"32704.5","size":"0.0017491","bestBidSize":"0.06793004","price":"32704.4","time":1626290935303,"bestAskSize":"0.02055972","bestBid":"32704.4"},"subject":"trade.ticker","topic":"/market/ticker:BTC-USDT","type":"message"}
{"data":{"sequence":"1618200193842","bestAsk":"32704.5","size":"0.0017491","bestBidSize":"0.06793004","price":"32704.4","time":1626290935404,"bestAskSize":"0.02055972","bestBid":"32704.4"},"subject":"trade.ticker","topic":"/market/ticker:BTC-USDT","type":"message"}
{"data":{"sequence":"1612914399726","bestAsk":"13.3733","size":"8","bestBidSize":"167.5597","price":"13.3365","time":1626290935489,"bestAskSize":"26.1762","bestBid":"13.3339"},"subject":"trade.ticker","topic":"/market/ticker:WAVES-USDT","type":"message"}
{"data":{"sequence":"1618200193858","bestAsk":"32704.5","size":"0.0017491","bestBidSize":"0.06793004","price":"32704.4","time":1626290935503,"bestAskSize":"0.02055972","bestBid":"32704.4"},"subject":"trade.ticker","topic":"/market/ticker:BTC-USDT","type":"message"}
{"data":{"sequence":"1618200193882","bestAsk":"32704.5","size":"0.0017491","bestBidSize":"0.06793004","price":"32704.4","time":1626290935603,"bestAskSize":"0.02055972","bestBid":"32704.4"},"subject":"trade.ticker","topic":"/market/ticker:BTC-USDT","type":"message"}
{"data":{"sequence":"1618200193896","bestAsk":"32704.5","size":"0.0017491","bestBidSize":"0.06793004","price":"32704.4","time":1626290935703,"bestAskSize":"0.02055972","bestBid":"32704.4"},"subject":"trade.ticker","topic":"/market/ticker:BTC-USDT","type":"message"}
{"data":{"sequence":"1618200193922","bestAsk":"32704.5","size":"0.0017491","bestBidSize":"0.06793004","price":"32704.4","time":1626290935803,"bestAskSize":"0.02055972","bestBid":"32704.4"},"subject":"trade.ticker","topic":"/market/ticker:BTC-USDT","type":"message"}
{"data":{"sequence":"1618200193934","bestAsk":"32704.5","size":"0.0017491","bestBidSize":"0.06793004","price":"32704.4","time":1626290935905,"bestAskSize":"0.02055972","bestBid":"32704.4"},"subject":"trade.ticker","topic":"/market/ticker:BTC-USDT","type":"message"}
{"data":{"sequence":"1618200193952","bestAsk":"32704.5","size":"0.0017491","bestBidSize":"0.06793004","price":"32704.4","time":1626290936004,"bestAskSize":"0.02055972","bestBid":"32704.4"},"subject":"trade.ticker","topic":"/market/ticker:BTC-USDT","type":"message"}
{"data":{"sequence":"1618200194031","bestAsk":"32704.5","size":"0.0017491","bestBidSize":"0.06793004","price":"32704.4","time":1626290936105,"bestAskSize":"0.02055972","bestBid":"32704.4"},"subject":"trade.ticker","topic":"/market/ticker:BTC-USDT","type":"message"}
{"data":{"sequence":"1618200194053","bestAsk":"32704.5","size":"0.0017491","bestBidSize":"0.06793004","price":"32704.4","time":1626290936205,"bestAskSize":"0.02055972","bestBid":"32704.4"},"subject":"trade.ticker","topic":"/market/ticker:BTC-USDT","type":"message"}
{"data":{"sequence":"1612914399728","bestAsk":"13.3733","size":"8","bestBidSize":"167.5597","price":"13.3365","time":1626290936289,"bestAskSize":"26.1762","bestBid":"13.3339"},"subject":"trade.ticker","topic":"/market/ticker:WAVES-USDT","type":"message"}
{"data":{"sequence":"1618200194073","bestAsk":"32704.5","size":"0.0017491","bestBidSize":"0.06793004","price":"32704.4","time":1626290936303,"bestAskSize":"0.02055972","bestBid":"32704.4"},"subject":"trade.ticker","topic":"/market/ticker:BTC-USDT","type":"message"}
{"data":{"sequence":"1618200194082","bestAsk":"32704.5","size":"0.0017491","bestBidSize":"0.06793004","price":"32704.4","time":1626290936403,"bestAskSize":"0.02055972","bestBid":"32704.4"},"subject":"trade.ticker","topic":"/market/ticker:BTC-USDT","type":"message"}
{"data":{"sequence":"1618200194101","bestAsk":"32704.5","size":"0.0017491","bestBidSize":"0.06793004","price":"32704.4","time":1626290936503,"bestAskSize":"0.02055972","bestBid":"32704.4"},"subject":"trade.ticker","topic":"/market/ticker:BTC-USDT","type":"message"}
{"data":{"sequence":"1618200194114","bestAsk":"32704.5","size":"0.0017491","bestBidSize":"0.06793004","price":"32704.4","time":1626290936603,"bestAskSize":"0.02055972","bestBid":"32704.4"},"subject":"trade.ticker","topic":"/market/ticker:BTC-USDT","type":"message"}
{"data":{"sequence":"1618200194132","bestAsk":"32704.5","size":"0.0017491","bestBidSize":"0.06793004","price":"32704.4","time":1626290936703,"bestAskSize":"0.02055972","bestBid":"32704.4"},"subject":"trade.ticker","topic":"/market/ticker:BTC-USDT","type":"message"}
{"data":{"sequence":"1618200194148","bestAsk":"32704.5","size":"0.0017491","bestBidSize":"0.06793004","price":"32704.4","time":1626290936803,"bestAskSize":"0.02055972","bestBid":"32704.4"},"subject":"trade.ticker","topic":"/market/ticker:BTC-USDT","type":"message"}
{"data":{"sequence":"1618200194266","bestAsk":"32704.5","size":"0.0017491","bestBidSize":"0.06793004","price":"32704.4","time":1626290936904,"bestAskSize":"0.02055972","bestBid":"32704.4"},"subject":"trade.ticker","topic":"/market/ticker:BTC-USDT","type":"message"}
{"data":{"sequence":"1618200194280","bestAsk":"32704.5","size":"0.0017491","bestBidSize":"0.06793004","price":"32704.4","time":1626290937003,"bestAskSize":"0.02055972","bestBid":"32704.4"},"subject":"trade.ticker","topic":"/market/ticker:BTC-USDT","type":"message"}
{"data":{"sequence":"1612914399730","bestAsk":"13.3733","size":"8","bestBidSize":"167.5597","price":"13.3365","time":1626290937089,"bestAskSize":"26.1762","bestBid":"13.3339"},"subject":"trade.ticker","topic":"/market/ticker:WAVES-USDT","type":"message"}
{"data":{"sequence":"1618200194358","bestAsk":"32704.5","size":"0.0017491","bestBidSize":"0.06793004","price":"32704.4","time":1626290937103,"bestAskSize":"0.02055972","bestBid":"32704.4"},"subject":"trade.ticker","topic":"/market/ticker:BTC-USDT","type":"message"}
{"data":{"sequence":"1618200194373","bestAsk":"32704.5","size":"0.0017491","bestBidSize":"0.06793004","price":"32704.4","time":1626290937204,"bestAskSize":"0.02055972","bestBid":"32704.4"},"subject":"trade.ticker","topic":"/market/ticker:BTC-USDT","type":"message"}
{"data":{"sequence":"1612914399731","bestAsk":"13.3733","size":"8","bestBidSize":"167.5597","price":"13.3365","time":1626290937289,"bestAskSize":"26.1762","bestBid":"13.3339"},"subject":"trade.ticker","topic":"/market/ticker:WAVES-USDT","type":"message"}
{"data":{"sequence":"1618200194392","bestAsk":"32704.5","size":"0.0017491","bestBidSize":"0.06793004","price":"32704.4","time":1626290937303,"bestAskSize":"0.02055972","bestBid":"32704.4"},"subject":"trade.ticker","topic":"/market/ticker:BTC-USDT","type":"message"}
{"data":{"sequence":"1612914399732","bestAsk":"13.3733","size":"8","bestBidSize":"167.5597","price":"13.3365","time":1626290937389,"bestAskSize":"26.1762","bestBid":"13.3339"},"subject":"trade.ticker","topic":"/market/ticker:WAVES-USDT","type":"message"}
{"data":{"sequence":"1618200194406","bestAsk":"32704.5","size":"0.0017491","bestBidSize":"0.06793004","price":"32704.4","time":1626290937422,"bestAskSize":"0.02055972","bestBid":"32704.4"},"subject":"trade.ticker","topic":"/market/ticker:BTC-USDT","type":"message"}
{"data":{"sequence":"1618200194437","bestAsk":"32704.5","size":"0.00058862","bestBidSize":"0.06851866","price":"32704.5","time":1626290937503,"bestAskSize":"0.01955972","bestBid":"32704.4"},"subject":"trade.ticker","topic":"/market/ticker:BTC-USDT","type":"message"}
{"data":{"sequence":"1618200194453","bestAsk":"32704.5","size":"0.00058862","bestBidSize":"0.06704767","price":"32704.5","time":1626290937603,"bestAskSize":"0.01955972","bestBid":"32704.4"},"subject":"trade.ticker","topic":"/market/ticker:BTC-USDT","type":"message"}
{"data":{"sequence":"1612914399736","bestAsk":"13.3733","size":"8","bestBidSize":"167.5597","price":"13.3365","time":1626290937689,"bestAskSize":"26.1762","bestBid":"13.3339"},"subject":"trade.ticker","topic":"/market/ticker:WAVES-USDT","type":"message"}
{"data":{"sequence":"1618200194475","bestAsk":"32704.5","size":"0.00058862","bestBidSize":"0.06951866","price":"32704.5","time":1626290937703,"bestAskSize":"0.01955972","bestBid":"32704.4"},"subject":"trade.ticker","topic":"/market/ticker:BTC-USDT","type":"message"}
{"data":{"sequence":"1618200194499","bestAsk":"32704.5","size":"0.00058862","bestBidSize":"0.06951866","price":"32704.5","time":1626290937804,"bestAskSize":"0.01955972","bestBid":"32704.4"},"subject":"trade.ticker","topic":"/market/ticker:BTC-USDT","type":"message"}
{"type":"message","topic":"/market/ticker:BTC-USDT","subject":"trade.ticker","data":{"bestAsk":"41217.7","bestAskSize":"0.21545096","bestBid":"41217.6","bestBidSize":"0.0265","price":"41217.7","sequence":"1618607525224","size":"0.00043659","time":1627752855836}}`
