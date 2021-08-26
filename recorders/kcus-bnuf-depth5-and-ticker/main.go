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

	batchSize := flag.Int("batch", 30, "symbols group batch size")

	proxyAddress := flag.String("proxy", "", "symbols group batch size")
	symbolsStr := flag.String("symbols", "1INCH-USDT,ADA-USDT,ALGO-USDT,ANKR-USDT,ATOM-USDT,AVAX-USDT,BAT-USDT,BCH-USDT,BNB-USDT,BTC-USDT,BTT-USDT,DASH-USDT,DGB-USDT,DODO-USDT,DOGE-USDT,EOS-USDT,ETC-USDT,ETH-USDT,FIL-USDT,FTM-USDT,GRT-USDT,ICP-USDT,IOST-USDT,LRC-USDT,LTC-USDT,LUNA-USDT,MATIC-USDT,NEAR-USDT,NEO-USDT,OGN-USDT,OMG-USDT,ONE-USDT,ONT-USDT,STMX-USDT,SXP-USDT,TOMO-USDT,TRX-USDT,VET-USDT,XEM-USDT,XLM-USDT,XMR-USDT,XRP-USDT,XTZ-USDT,ZEC-USDT,ZEN-USDT,ZIL-USDT", "symbols, separate by comma")
	savePath := flag.String("path", "/root/kcus-bnuf-depth5-and-ticker", "data save folder")

	//savePath := flag.String("path", "/Users/chenjilin/Downloads", "data save folder")
	//symbolsStr := flag.String("symbols", "BTC-USDT", "symbols, separate by comma")
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
		kcusChMap := make(map[string]chan *Message)
		bnufChMap := make(map[string]chan *Message)
		for _, xSymbol := range symbols[start:end] {
			ySymbol := strings.Replace(xSymbol, "-USDT", "USDT", -1)
			kcusChMap[xSymbol] = make(chan *Message, 1024)
			bnufChMap[strings.ToLower(ySymbol)] = kcusChMap[xSymbol]
			go saveLoop(ctx, cancel, *savePath, xSymbol, ySymbol, kcusChMap[xSymbol], fileSavedCh)
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
			ws1 := NewKcusTickerWS(ctx, proxy, outputChMap)
			ws2 := NewKcusDepth5WS(ctx, proxy, outputChMap)
			select {
			case <-ctx.Done():
			case <-ws1.Done():
				cancel()
			case <-ws2.Done():
				cancel()
			}
		}(ctx, cancel, *proxyAddress, kcusChMap)
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
