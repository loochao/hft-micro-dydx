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

	batchSize := flag.Int("batch", 10, "symbols group batch size")

	proxyAddress := flag.String("proxy", "", "symbols group batch size")
	symbolsStr := flag.String("symbols", "BZRXUSDT,SOLUSDT,SKLUSDT,XLMUSDT,STORJUSDT,KEEPUSDT,XMRUSDT,LINKUSDT,THETAUSDT,TRBUSDT,RUNEUSDT,LUNAUSDT,SANDUSDT,GTCUSDT,MANAUSDT,SFPUSDT,HOTUSDT,UNIUSDT,ENJUSDT,ZENUSDT,ETCUSDT,BAKEUSDT,ALGOUSDT,BTCUSDT,IOTAUSDT,DODOUSDT,RVNUSDT,CTKUSDT,EGLDUSDT,REEFUSDT,BCHUSDT,OMGUSDT,QTUMUSDT,AKROUSDT,ZILUSDT,LITUSDT,ADAUSDT,DENTUSDT,ONTUSDT,FILUSDT,UNFIUSDT,NEARUSDT,HNTUSDT,TOMOUSDT,MTLUSDT,DGBUSDT,AXSUSDT,BANDUSDT,CVCUSDT,SXPUSDT,AAVEUSDT,KNCUSDT,CRVUSDT,KAVAUSDT,AVAXUSDT,BLZUSDT,CHZUSDT,BALUSDT,RENUSDT,BNBUSDT,ALPHAUSDT,DASHUSDT,LRCUSDT,ETHUSDT,SUSHIUSDT,IOSTUSDT,XEMUSDT,ALICEUSDT,SNXUSDT,BTTUSDT,FLMUSDT,ONEUSDT,OGNUSDT,MKRUSDT,XTZUSDT,YFIIUSDT,MATICUSDT,ICXUSDT,BATUSDT,BTSUSDT,CELRUSDT,LTCUSDT,COTIUSDT,1INCHUSDT,OCEANUSDT,RLCUSDT,STMXUSDT,GRTUSDT,KSMUSDT,FTMUSDT,SCUSDT,WAVESUSDT,LINAUSDT,NKNUSDT,ZECUSDT,DOTUSDT,ATOMUSDT,ANKRUSDT,VETUSDT,TRXUSDT,RSRUSDT,NEOUSDT,DOGEUSDT,ICPUSDT,YFIUSDT,BELUSDT,XRPUSDT,COMPUSDT,EOSUSDT,ZRXUSDT,HBARUSDT,SRMUSDT,CHRUSDT", "symbols, separate by comma")
	savePath := flag.String("path", "/root/bnus-bnuf-depth5-and-ticker", "data save folder")

	//savePath := flag.String("path", "/Users/chenjilin/Downloads", "data save folder")
	//symbolsStr := flag.String("symbols", "BTCUSDT", "symbols, separate by comma")
	//proxyAddress := flag.String("proxy", "socks5://127.0.0.1:1080", "symbols group batch size")
	flag.Parse()
	symbols := strings.Split(*symbolsStr, ",")
	ctx, cancel := context.WithCancel(context.Background())
	fileSavedCh := make(chan string, len(symbols))
	for start := 0; start < len(symbols); start += *batchSize {
		end := start + *batchSize
		if end > len(symbols) {
			end = len(symbols)
		}
		bnusChMap := make(map[string]chan *Message)
		bnufChMap := make(map[string]chan *Message)
		for _, xSymbol := range symbols[start:end] {
			ySymbol := strings.Replace(xSymbol, "USDT", "USDT", -1)
			bnusChMap[strings.ToLower(xSymbol)] = make(chan *Message, 1024)
			bnufChMap[strings.ToLower(ySymbol)] = bnusChMap[strings.ToLower(xSymbol)]
			go saveLoop(ctx, cancel, *savePath, xSymbol, ySymbol, bnusChMap[strings.ToLower(xSymbol)], fileSavedCh)
		}
		go func(ctx context.Context, cancel context.CancelFunc, proxy string, outputChMap map[string]chan *Message) {
			ws1 := NewBnufDepth5WS(ctx, proxy, outputChMap)
			ws2 := NewBnufBookTickerWS(ctx, proxy, outputChMap)
			select {
			case <-ctx.Done():
			case <-ws1.Done():
				cancel()
			case <-ws2.Done():
				cancel()
			}
		}(ctx, cancel, *proxyAddress, bnufChMap)
		go func(ctx context.Context, cancel context.CancelFunc, proxy string, outputChMap map[string]chan *Message) {
			ws1 := NewBnusDepth5WS(ctx, proxy, outputChMap)
			ws2 := NewBnusBookTickerWS(ctx, proxy, outputChMap)
			select {
			case <-ctx.Done():
			case <-ws1.Done():
				cancel()
			case <-ws2.Done():
				cancel()
			}
		}(ctx, cancel, *proxyAddress, bnusChMap)
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
