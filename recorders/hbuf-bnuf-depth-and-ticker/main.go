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
	symbolsStr := flag.String("symbols", "IOST-USDT,LUNA-USDT,XRP-USDT,AAVE-USDT,BNB-USDT,YFII-USDT,ATOM-USDT,ICP-USDT,XTZ-USDT,SUSHI-USDT,ZEN-USDT,ALGO-USDT,ETC-USDT,MATIC-USDT,REEF-USDT,STORJ-USDT,CRV-USDT,DOGE-USDT,FIL-USDT,SNX-USDT,NEAR-USDT,DASH-USDT,KSM-USDT,QTUM-USDT,ONE-USDT,XEM-USDT,DOT-USDT,IOTA-USDT,THETA-USDT,WAVES-USDT,ZEC-USDT,XMR-USDT,CHR-USDT,CHZ-USDT,LTC-USDT,UNI-USDT,VET-USDT,BCH-USDT,KAVA-USDT,ONT-USDT,RSR-USDT,OMG-USDT,XLM-USDT,1INCH-USDT,BTC-USDT,COMP-USDT,ENJ-USDT,NEO-USDT,ADA-USDT,BTT-USDT,ZIL-USDT,YFI-USDT,GRT-USDT,LINK-USDT,SOL-USDT,TRX-USDT,MKR-USDT,AVAX-USDT,EOS-USDT,ETH-USDT,MANA-USDT", "symbols, separate by comma")
	savePath := flag.String("path", "/root/hbuf-bnuf-depth-and-ticker", "data save folder")

	//savePath := flag.String("path", "/home/clu/Downloads", "data save folder")
	//symbolsStr := flag.String("symbols", "BTC-USDT", "symbols, separate by comma")
	//proxyAddress := flag.String("proxy", "socks5://127.0.0.1:1083", "symbols group batch size")

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
			ws1 := NewHbufDepth20WS(ctx, proxy, outputChMap)
			ws2 := NewHbufTickerWS(ctx, proxy, outputChMap)
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
