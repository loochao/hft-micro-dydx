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
	"XBTUSDTM":   "BTCUSDT",
	"IOSTUSDTM":  "IOSTUSDT",
	"UNIUSDTM":   "UNIUSDT",
	"ICPUSDTM":   "ICPUSDT",
	"THETAUSDTM": "THETAUSDT",
	"YFIUSDTM":   "YFIUSDT",
	"OCEANUSDTM": "OCEANUSDT",
	"XMRUSDTM":   "XMRUSDT",
	"SXPUSDTM":   "SXPUSDT",
	"BCHUSDTM":   "BCHUSDT",
	"TRXUSDTM":   "TRXUSDT",
	"XEMUSDTM":   "XEMUSDT",
	"ETHUSDTM":   "ETHUSDT",
	"MKRUSDTM":   "MKRUSDT",
	"FTMUSDTM":   "FTMUSDT",
	"ATOMUSDTM":  "ATOMUSDT",
	"BANDUSDTM":  "BANDUSDT",
	"DOTUSDTM":   "DOTUSDT",
	"FILUSDTM":   "FILUSDT",
	"AVAXUSDTM":  "AVAXUSDT",
	"QTUMUSDTM":  "QTUMUSDT",
	"COMPUSDTM":  "COMPUSDT",
	"ZECUSDTM":   "ZECUSDT",
	"ADAUSDTM":   "ADAUSDT",
	"DOGEUSDTM":  "DOGEUSDT",
	"XLMUSDTM":   "XLMUSDT",
	"EOSUSDTM":   "EOSUSDT",
	"LTCUSDTM":   "LTCUSDT",
	"VETUSDTM":   "VETUSDT",
	"ONTUSDTM":   "ONTUSDT",
	"RVNUSDTM":   "RVNUSDT",
	"MATICUSDTM": "MATICUSDT",
	"1INCHUSDTM": "1INCHUSDT",
	"XRPUSDTM":   "XRPUSDT",
	"NEOUSDTM":   "NEOUSDT",
	"ALGOUSDTM":  "ALGOUSDT",
	"MANAUSDTM":  "MANAUSDT",
	"WAVESUSDTM": "WAVESUSDT",
	"KSMUSDTM":   "KSMUSDT",
	"AAVEUSDTM":  "AAVEUSDT",
	"LINKUSDTM":  "LINKUSDT",
	"BATUSDTM":   "BATUSDT",
	"DENTUSDTM":  "DENTUSDT",
	"LUNAUSDTM":  "LUNAUSDT",
	"ETCUSDTM":   "ETCUSDT",
	"CHZUSDTM":   "CHZUSDT",
	"CRVUSDTM":   "CRVUSDT",
	"DASHUSDTM":  "DASHUSDT",
	"SNXUSDTM":   "SNXUSDT",
	"GRTUSDTM":   "GRTUSDT",
	"BTTUSDTM":   "BTTUSDT",
	"SUSHIUSDTM": "SUSHIUSDT",
	"ENJUSDTM":   "ENJUSDT",
	"XTZUSDTM":   "XTZUSDT",
	"DGBUSDTM":   "DGBUSDT",
	"SOLUSDTM":   "SOLUSDT",
	"BNBUSDTM":   "BNBUSDT",
}

func main() {

	batchSize := flag.Int("batch", 30, "symbols group batch size")

	//proxyAddress := flag.String("proxy", "", "symbols group batch size")
	//savePath := flag.String("path", "/root/kcuf-bnuf-depth5-and-ticker", "data save folder")

	savePath := flag.String("path", "/Users/chenjilin/Downloads", "data save folder")
	proxyAddress := flag.String("proxy", "socks5://127.0.0.1:1080", "symbols group batch size")

	flag.Parse()
	symbols := make([]string, 0)
	for xSymbol := range symbolsMap {
		symbols = append(symbols, xSymbol)
	}
	symbols = symbols[:1]
	ctx, cancel := context.WithCancel(context.Background())
	fileSavedCh := make(chan string, len(symbols))
	for start := 0; start < len(symbols); start += *batchSize {
		end := start + *batchSize
		if end > len(symbols) {
			end = len(symbols)
		}
		kcufChMap := make(map[string]chan *Message)
		bnufChMap := make(map[string]chan *Message)
		for xSymbol, ySymbol := range symbolsMap {
			kcufChMap[xSymbol] = make(chan *Message, 1024)
			bnufChMap[strings.ToLower(ySymbol)] = kcufChMap[xSymbol]
			go saveLoop(ctx, cancel, *savePath, xSymbol, ySymbol, kcufChMap[xSymbol], fileSavedCh)
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
			ws1 := NewKcufTickerWS(ctx, proxy, outputChMap)
			ws2 := NewKcufDepth5WS(ctx, proxy, outputChMap)
			select {
			case <-ctx.Done():
			case <-ws1.Done():
				cancel()
			case <-ws2.Done():
				cancel()
			}
		}(ctx, cancel, *proxyAddress, kcufChMap)
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
