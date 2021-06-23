package tests

import (
	binance_busdspot "github.com/geometrybase/hft-micro/binance-busdspot"
	"github.com/geometrybase/hft-micro/logger"
	"strings"
	"testing"
)

func TestMatchBusdUsdt(t *testing.T) {

	symbols := []string{"1INCHUSDT", "AAVEUSDT", "ADAUSDT", "AKROUSDT", "ALGOUSDT", "ALICEUSDT", "ALPHAUSDT", "ANKRUSDT", "ATOMUSDT", "AVAXUSDT", "AXSUSDT", "BALUSDT", "BANDUSDT", "BATUSDT", "BCHUSDT", "BELUSDT", "BLZUSDT", "BTCUSDT", "BTSUSDT", "BZRXUSDT", "CELRUSDT", "CHRUSDT", "CHZUSDT", "COMPUSDT", "COTIUSDT", "CRVUSDT", "CTKUSDT", "CVCUSDT", "DASHUSDT", "DENTUSDT", "DGBUSDT", "DODOUSDT", "DOGEUSDT", "DOTUSDT", "EGLDUSDT", "ENJUSDT", "EOSUSDT", "ETCUSDT", "ETHUSDT", "FILUSDT", "FLMUSDT", "FTMUSDT", "GRTUSDT", "HBARUSDT", "HNTUSDT", "HOTUSDT", "ICPUSDT", "ICXUSDT", "IOSTUSDT", "IOTAUSDT", "KAVAUSDT", "KNCUSDT", "KSMUSDT", "LINAUSDT", "LINKUSDT", "LITUSDT", "LRCUSDT", "LTCUSDT", "LUNAUSDT", "MANAUSDT", "MATICUSDT", "MKRUSDT", "MTLUSDT", "NEARUSDT", "NEOUSDT", "NKNUSDT", "OCEANUSDT", "OGNUSDT", "OMGUSDT", "ONEUSDT", "ONTUSDT", "QTUMUSDT", "REEFUSDT", "RENUSDT", "RLCUSDT", "RSRUSDT", "RUNEUSDT", "RVNUSDT", "SANDUSDT", "SFPUSDT", "SKLUSDT", "SNXUSDT", "SOLUSDT", "SRMUSDT", "STMXUSDT", "STORJUSDT", "SUSHIUSDT", "SXPUSDT", "THETAUSDT", "TOMOUSDT", "TRBUSDT", "TRXUSDT", "UNFIUSDT", "UNIUSDT", "VETUSDT", "WAVESUSDT", "XEMUSDT", "XLMUSDT", "XMRUSDT", "XRPUSDT", "XTZUSDT", "YFIIUSDT", "YFIUSDT", "ZECUSDT", "ZENUSDT", "ZILUSDT", "ZRXUSDT"}
	for _, uSymbol := range symbols {
		bSymbol := strings.Replace(uSymbol, "USDT", "BUSD", -1)
		if _, ok := binance_busdspot.TickSizes[bSymbol]; !ok {
			logger.Debugf("%s %s", uSymbol, bSymbol)
		}
	}
	//for uSymbol := range binance_usdtfuture.TickSizes {
	//
	//	logger.Debugf("%s", uSymbol)
	//}
}
