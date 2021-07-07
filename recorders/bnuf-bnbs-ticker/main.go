package main

import (
	"context"
	"flag"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

var symbolsMap = map[string]string{
	"FILUSDT":   "FILBUSD",
	"NEOUSDT":   "NEOBUSD",
	"OCEANUSDT": "OCEANBUSD",
	"SFPUSDT":   "SFPBUSD",
	"QTUMUSDT":  "QTUMBUSD",
	"STMXUSDT":  "STMXBUSD",
	"BTTUSDT":   "BTTBUSD",
	"BZRXUSDT":  "BZRXBUSD",
	"RSRUSDT":   "RSRBUSD",
	"GTCUSDT":   "GTCBUSD",
	"ZILUSDT":   "ZILBUSD",
	"ATOMUSDT":  "ATOMBUSD",
	"ICXUSDT":   "ICXBUSD",
	"BELUSDT":   "BELBUSD",
	"CHZUSDT":   "CHZBUSD",
	"RLCUSDT":   "RLCBUSD",
	"DODOUSDT":  "DODOBUSD",
	"XRPUSDT":   "XRPBUSD",
	"KNCUSDT":   "KNCBUSD",
	"KAVAUSDT":  "KAVABUSD",
	"TOMOUSDT":  "TOMOBUSD",
	"DGBUSDT":   "DGBBUSD",
	"SRMUSDT":   "SRMBUSD",
	"ICPUSDT":   "ICPBUSD",
	"RVNUSDT":   "RVNBUSD",
	"ENJUSDT":   "ENJBUSD",
	"MANAUSDT":  "MANABUSD",
	"BAKEUSDT":  "BAKEBUSD",
	"THETAUSDT": "THETABUSD",
	"AXSUSDT":   "AXSBUSD",
	"LINAUSDT":  "LINABUSD",
	"FLMUSDT":   "FLMBUSD",
	"ONTUSDT":   "ONTBUSD",
	"LITUSDT":   "LITBUSD",
	"ETHUSDT":   "ETHBUSD",
	"SCUSDT":    "SCBUSD",
	"BALUSDT":   "BALBUSD",
	"LINKUSDT":  "LINKBUSD",
	"WAVESUSDT": "WAVESBUSD",
	"YFIIUSDT":  "YFIIBUSD",
	"VETUSDT":   "VETBUSD",
	"1INCHUSDT": "1INCHBUSD",
	"LRCUSDT":   "LRCBUSD",
	"REEFUSDT":  "REEFBUSD",
	"MKRUSDT":   "MKRBUSD",
	"ZECUSDT":   "ZECBUSD",
	"BCHUSDT":   "BCHBUSD",
	"OMGUSDT":   "OMGBUSD",
	"RUNEUSDT":  "RUNEBUSD",
	"BATUSDT":   "BATBUSD",
	"KEEPUSDT":  "KEEPBUSD",
	"AVAXUSDT":  "AVAXBUSD",
	"XTZUSDT":   "XTZBUSD",
	"FTMUSDT":   "FTMBUSD",
	"ZENUSDT":   "ZENBUSD",
	"EGLDUSDT":  "EGLDBUSD",
	"BTCUSDT":   "BTCBUSD",
	"CHRUSDT":   "CHRBUSD",
	"BANDUSDT":  "BANDBUSD",
	"UNIUSDT":   "UNIBUSD",
	"LTCUSDT":   "LTCBUSD",
	"XLMUSDT":   "XLMBUSD",
	"KSMUSDT":   "KSMBUSD",
	"LUNAUSDT":  "LUNABUSD",
	"SXPUSDT":   "SXPBUSD",
	"DASHUSDT":  "DASHBUSD",
	"DOGEUSDT":  "DOGEBUSD",
	"ALPHAUSDT": "ALPHABUSD",
	"EOSUSDT":   "EOSBUSD",
	"CELRUSDT":  "CELRBUSD",
	"GRTUSDT":   "GRTBUSD",
	"XEMUSDT":   "XEMBUSD",
	"ZRXUSDT":   "ZRXBUSD",
	"ONEUSDT":   "ONEBUSD",
	"YFIUSDT":   "YFIBUSD",
	"COMPUSDT":  "COMPBUSD",
	"HBARUSDT":  "HBARBUSD",
	"TRXUSDT":   "TRXBUSD",
	"SUSHIUSDT": "SUSHIBUSD",
	"SOLUSDT":   "SOLBUSD",
	"IOTAUSDT":  "IOTABUSD",
	"IOSTUSDT":  "IOSTBUSD",
	"BNBUSDT":   "BNBBUSD",
	"ALICEUSDT": "ALICEBUSD",
	"DOTUSDT":   "DOTBUSD",
	"MATICUSDT": "MATICBUSD",
	"UNFIUSDT":  "UNFIBUSD",
	"XMRUSDT":   "XMRBUSD",
	"ALGOUSDT":  "ALGOBUSD",
	"AAVEUSDT":  "AAVEBUSD",
	"ETCUSDT":   "ETCBUSD",
	"SANDUSDT":  "SANDBUSD",
	"HNTUSDT":   "HNTBUSD",
	"SKLUSDT":   "SKLBUSD",
	"HOTUSDT":   "HOTBUSD",
	"SNXUSDT":   "SNXBUSD",
	"COTIUSDT":  "COTIBUSD",
	"NEARUSDT":  "NEARBUSD",
	"TRBUSDT":   "TRBBUSD",
	"CTKUSDT":   "CTKBUSD",
	"CRVUSDT":   "CRVBUSD",
	"ADAUSDT":   "ADABUSD",
}

func main() {

	batchSize := flag.Int("batch", 40, "symbols group batch size")

	proxyAddress := flag.String("proxy", "", "symbols group batch size")
	savePath := flag.String("path", "/root/bnuf-bnbs-ticker", "data save folder")

	//savePath := flag.String("path", "/Users/chenjilin/Downloads", "data save folder")
	//proxyAddress := flag.String("proxy", "socks5://127.0.0.1:1080", "symbols group batch size")
	flag.Parse()
	symbols := make([]string, 0)
	for symbol := range symbolsMap {
		symbols = append(symbols, symbol)
	}
	ctx, cancel := context.WithCancel(context.Background())
	fileSavedCh := make(chan string, len(symbols))
	for start := 0; start < len(symbols); start += *batchSize {
		end := start + *batchSize
		if end > len(symbols) {
			end = len(symbols)
		}
		ufMsgChMap := make(map[string]chan Message)
		bsMsgChMap := make(map[string]chan Message)
		for _, symbol := range symbols[start:end] {
			ufMsgChMap[strings.ToLower(symbol)] = make(chan Message, 10000)
			bsMsgChMap[strings.ToLower(symbolsMap[symbol])] = ufMsgChMap[strings.ToLower(symbol)]
			go saveLoop(ctx, cancel, *savePath, symbol, symbolsMap[symbol], ufMsgChMap[strings.ToLower(symbol)], fileSavedCh)
		}
		go func(ctx context.Context, cancel context.CancelFunc, proxy string, outputChMap map[string]chan Message) {
			ws := NewBnufTicker(ctx, proxy, outputChMap)
			select {
			case <-ctx.Done():
			case <-ws.Done():
				cancel()
			}
		}(ctx, cancel, *proxyAddress, ufMsgChMap)
		go func(ctx context.Context, cancel context.CancelFunc, proxy string, outputChMap map[string]chan Message) {
			ws := NewBnbsTicker(ctx, proxy, outputChMap)
			select {
			case <-ctx.Done():
			case <-ws.Done():
				cancel()
			}
		}(ctx, cancel, *proxyAddress, bsMsgChMap)
	}
	go archiveFiles(context.Background(), *savePath)
	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			sig := <-sigs
			logger.Debugf("catch signal %v", sig)
			cancel()
		}()
	}()
	<-ctx.Done()
	logger.Debugf("waiting 88s to write files")
	counter := 0
	for {
		select {
		case symbol := <-fileSavedCh:
			logger.Debugf("%s saved", symbol)
			counter++
			if counter == len(symbols) {
				logger.Debugf("all symbols' file saved")
				return
			}
		case <-time.After(time.Second * 88):
			logger.Debugf("save timeout in 88s")
		}
	}
}
