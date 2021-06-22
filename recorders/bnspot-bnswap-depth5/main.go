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

	//proxyAddress := flag.String("proxy", "", "symbols group batch size")
	//symbolsStr := flag.String("symbols", "TRXUSDT,LRCUSDT,ATOMUSDT,ADAUSDT,TRBUSDT,STMXUSDT,BTSUSDT,SKLUSDT,LUNAUSDT,AKROUSDT,XTZUSDT,XRPUSDT,EGLDUSDT,DENTUSDT,STORJUSDT,REEFUSDT,BTTUSDT,XLMUSDT,CRVUSDT,AXSUSDT,RUNEUSDT,IOSTUSDT,BNBUSDT,COTIUSDT,MATICUSDT,ZRXUSDT,TOMOUSDT,BALUSDT,CHZUSDT,ALGOUSDT,SANDUSDT,SUSHIUSDT,KNCUSDT,QTUMUSDT,ETCUSDT,DASHUSDT,BZRXUSDT,MTLUSDT,ALICEUSDT,OMGUSDT,BTCUSDT,BATUSDT,RENUSDT,BANDUSDT,SNXUSDT,LINAUSDT,HBARUSDT,ZECUSDT,BCHUSDT,CVCUSDT,WAVESUSDT,XMRUSDT,IOTAUSDT,XEMUSDT,1INCHUSDT,LINKUSDT,RVNUSDT,AAVEUSDT,ZILUSDT,ICPUSDT,SFPUSDT,KSMUSDT,ONEUSDT,FLMUSDT,KAVAUSDT,ENJUSDT,UNFIUSDT,ZENUSDT,BELUSDT,YFIUSDT,DGBUSDT,CHRUSDT,THETAUSDT,ICXUSDT,1000SHIBUSDT,GTCUSDT,VETUSDT,YFIIUSDT,SXPUSDT,MKRUSDT,RLCUSDT,LTCUSDT,BLZUSDT,NEOUSDT,LITUSDT,UNIUSDT,HNTUSDT,CTKUSDT,EOSUSDT,DOGEUSDT,MANAUSDT,GRTUSDT,FILUSDT,ETHUSDT,OCEANUSDT,HOTUSDT,NKNUSDT,DOTUSDT,RSRUSDT,ALPHAUSDT,NEARUSDT,AVAXUSDT,SRMUSDT,OGNUSDT,SCUSDT,ONTUSDT,DODOUSDT,SOLUSDT,CELRUSDT,FTMUSDT,DEFIUSDT,BAKEUSDT,ANKRUSDT,COMPUSDT", "symbols, separate by comma")
	//savePath := flag.String("path", "/root/bnspot-bnswap-depth5", "data save folder")

	savePath := flag.String("path", "/Users/chenjilin/Downloads", "data save folder")
	symbolsStr := flag.String("symbols", "BTCUSDT", "symbols, separate by comma")
	proxyAddress := flag.String("proxy", "socks5://127.0.0.1:1080", "symbols group batch size")
	flag.Parse()
	symbols := strings.Split(*symbolsStr, ",")
	ctx, cancel := context.WithCancel(context.Background())
	fileSavedCh := make(chan string, len(symbols))
	for start := 0; start < len(symbols); start += *batchSize {
		end := start + *batchSize
		if end > len(symbols) {
			end = len(symbols)
		}
		futureChMap := make(map[string]chan []byte)
		spotChMap := make(map[string]chan []byte)
		for _, symbol := range symbols {
			futureChMap[strings.ToLower(symbol)] = make(chan []byte, 1024)
			spotChMap[strings.ToLower(symbol)] = make(chan []byte, 1024)
			go saveLoop(ctx, cancel, *savePath, symbol, futureChMap[strings.ToLower(symbol)], spotChMap[strings.ToLower(symbol)], fileSavedCh)
		}
		go func(ctx context.Context, cancel context.CancelFunc, proxy string, outputChMap map[string]chan []byte) {
			ws := NewFutureDepth5WS(ctx, proxy, outputChMap)
			select {
			case <-ctx.Done():
			case <-ws.Done():
				cancel()
			}
		}(ctx, cancel, *proxyAddress, futureChMap)
		go func(ctx context.Context, cancel context.CancelFunc, proxy string, outputChMap map[string]chan []byte) {
			ws := NewSpotDepth5WS(ctx, proxy, outputChMap)
			select {
			case <-ctx.Done():
			case <-ws.Done():
				cancel()
			}
		}(ctx, cancel, *proxyAddress, spotChMap)
	}
	//go archiveFiles(context.Background(), *savePath)
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
