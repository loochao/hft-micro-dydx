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

func main() {

	batchSize := flag.Int("batch", 40, "symbols group batch size")

	proxyAddress := flag.String("proxy", "", "symbols group batch size")
	symbolsStr := flag.String("symbols", "1000SHIBUSDT,1INCHUSDT,AAVEUSDT,ADAUSDT,AKROUSDT,ALGOUSDT,ALICEUSDT,ALPHAUSDT,ANKRUSDT,ATOMUSDT,AVAXUSDT,AXSUSDT,BAKEUSDT,BALUSDT,BANDUSDT,BATUSDT,BCHUSDT,BELUSDT,BLZUSDT,BNBUSDT,BTCDOMUSDT,BTCUSDT,BTSUSDT,BTTUSDT,BZRXUSDT,CELRUSDT,CHRUSDT,CHZUSDT,COMPUSDT,COTIUSDT,CRVUSDT,CTKUSDT,CVCUSDT,DASHUSDT,DEFIUSDT,DENTUSDT,DGBUSDT,DODOUSDT,DOGEUSDT,DOTUSDT,EGLDUSDT,ENJUSDT,EOSUSDT,ETCUSDT,ETHUSDT,FILUSDT,FLMUSDT,FTMUSDT,GRTUSDT,GTCUSDT,HBARUSDT,HNTUSDT,HOTUSDT,ICPUSDT,ICXUSDT,IOSTUSDT,IOTAUSDT,KAVAUSDT,KEEPUSDT,KNCUSDT,KSMUSDT,LINAUSDT,LINKUSDT,LITUSDT,LRCUSDT,LTCUSDT,LUNAUSDT,MANAUSDT,MATICUSDT,MKRUSDT,MTLUSDT,NEARUSDT,NEOUSDT,NKNUSDT,OCEANUSDT,OGNUSDT,OMGUSDT,ONEUSDT,ONTUSDT,QTUMUSDT,REEFUSDT,RENUSDT,RLCUSDT,RSRUSDT,RUNEUSDT,RVNUSDT,SANDUSDT,SCUSDT,SFPUSDT,SKLUSDT,SNXUSDT,SOLUSDT,SRMUSDT,STMXUSDT,STORJUSDT,SUSHIUSDT,SXPUSDT,THETAUSDT,TOMOUSDT,TRBUSDT,TRXUSDT,UNFIUSDT,UNIUSDT,VETUSDT,WAVESUSDT,XEMUSDT,XLMUSDT,XMRUSDT,XRPUSDT,XTZUSDT,YFIIUSDT,YFIUSDT,ZECUSDT,ZENUSDT,ZILUSDT,ZRXUSDT", "symbols, separate by comma")
	savePath := flag.String("path", "/root/bnuf-ticker", "data save folder")

	//proxyAddress := flag.String("proxy", "socks5://127.0.0.1:1083", "symbols group batch size")
	//symbolsStr := flag.String("symbols", "DOGEUSDT", "symbols, separate by comma")
	//savePath := flag.String("path", "/Users/chenjilin/Downloads", "data save folder")
	flag.Parse()
	symbols := strings.Split(*symbolsStr, ",")
	ctx, cancel := context.WithCancel(context.Background())
	fileSavedCh := make(chan string, len(symbols))
	for start := 0; start < len(symbols); start += *batchSize {
		end := start + *batchSize
		if end > len(symbols) {
			end = len(symbols)
		}
		messageChMap := make(map[string]chan []byte)
		for _, symbol := range symbols[start:end] {
			messageChMap[strings.ToLower(symbol)] = make(chan []byte, 10000)
			go saveLoop(ctx, cancel, *savePath, symbol, messageChMap[strings.ToLower(symbol)], fileSavedCh)
		}
		go func(ctx context.Context, cancel context.CancelFunc, proxy string, outputChMap map[string]chan []byte) {
			ws := NewBookTickerWS(ctx, proxy, outputChMap)
			select {
			case <-ctx.Done():
			case <-ws.Done():
				cancel()
			}
		}(ctx, cancel, *proxyAddress, messageChMap)
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
