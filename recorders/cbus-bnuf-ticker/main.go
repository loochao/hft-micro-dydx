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
	symbolsStr := flag.String("symbols", "MANA-USD,RLC-USD,GRT-USD,UNI-USD,ENJ-USD,ALGO-USD,BCH-USD,MATIC-USD,KEEP-USD,LTC-USD,FIL-USD,BTC-USD,XTZ-USD,DOGE-USD,OMG-USD,LRC-USD,ETC-USD,REN-USD,ZRX-USD,SUSHI-USD,BAT-USD,BAND-USD,LINK-USD,ANKR-USD,MKR-USD,ATOM-USD,SOL-USD,CRV-USD,CHZ-USD,NKN-USD,KNC-USD,DOT-USD,OGN-USD,EOS-USD,ICP-USD,GTC-USD,ZEC-USD,SNX-USD,BAL-USD,AAVE-USD,STORJ-USD,DASH-USD,XLM-USD,TRB-USD,YFI-USD,COMP-USD,ETH-USD,ADA-USD,1INCH-USD,SKL-USD", "symbols, separate by comma")
	savePath := flag.String("path", "/root/cbus-bnuf-ticker", "data save folder")

	//savePath := flag.String("path", "/home/clu/Downloads", "data save folder")
	//symbolsStr := flag.String("symbols", "MATIC-USD", "symbols, separate by comma")
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
		cbusChMap := make(map[string]chan *Message)
		bnufChMap := make(map[string]chan *Message)
		for _, xSymbol := range symbols[start:end] {
			ySymbol := strings.Replace(xSymbol, "-USD", "USDT", -1)
			cbusChMap[xSymbol] = make(chan *Message, 1024)
			bnufChMap[strings.ToLower(ySymbol)] = cbusChMap[xSymbol]
			go saveLoop(ctx, cancel, *savePath, xSymbol, ySymbol, cbusChMap[xSymbol], fileSavedCh)
		}
		go func(ctx context.Context, cancel context.CancelFunc, proxy string, outputChMap map[string]chan *Message) {
			ws2 := NewBnufBookTickerWS(ctx, proxy, outputChMap)
			select {
			case <-ctx.Done():
			case <-ws2.Done():
				cancel()
			}
		}(ctx, cancel, *proxyAddress, bnufChMap)
		go func(ctx context.Context, cancel context.CancelFunc, proxy string, outputChMap map[string]chan *Message) {
			ws1 := NewCbusTickerWS(ctx, proxy, outputChMap)
			select {
			case <-ctx.Done():
			case <-ws1.Done():
				cancel()
			}
		}(ctx, cancel, *proxyAddress, cbusChMap)
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
