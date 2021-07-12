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
	symbolsStr := flag.String("symbols", "LINA-PERP,FLM-PERP,SOL-PERP,XLM-PERP,MATIC-PERP,BAND-PERP,SXP-PERP,DASH-PERP,SC-PERP,STORJ-PERP,EGLD-PERP,NEO-PERP,KSM-PERP,XTZ-PERP,ALGO-PERP,WAVES-PERP,1INCH-PERP,BAL-PERP,BNB-PERP,ATOM-PERP,CHZ-PERP,ETC-PERP,LRC-PERP,SNX-PERP,YFII-PERP,ADA-PERP,BCH-PERP,IOTA-PERP,OMG-PERP,ONT-PERP,ICP-PERP,HOT-PERP,RSR-PERP,QTUM-PERP,ZRX-PERP,KAVA-PERP,ZEC-PERP,DOGE-PERP,HNT-PERP,BTC-PERP,AVAX-PERP,NEAR-PERP,VET-PERP,THETA-PERP,GRT-PERP,XEM-PERP,FIL-PERP,DOT-PERP,LINK-PERP,DEFI-PERP,BAT-PERP,TRX-PERP,ALPHA-PERP,REEF-PERP,HBAR-PERP,UNI-PERP,SRM-PERP,AXS-PERP,CRV-PERP,SAND-PERP,DENT-PERP,YFI-PERP,MKR-PERP,ENJ-PERP,DODO-PERP,SUSHI-PERP,ZIL-PERP,FTM-PERP,COMP-PERP,AAVE-PERP,TOMO-PERP,EOS-PERP,REN-PERP,KNC-PERP,SKL-PERP,BTT-PERP,MTL-PERP,LTC-PERP,XRP-PERP,RUNE-PERP,ETH-PERP,XMR-PERP,LUNA-PERP,STMX-PERP", "symbols, separate by comma")
	savePath := flag.String("path", "/root/ftxuf-bnuf-depth5-and-ticker", "data save folder")

	//savePath := flag.String("path", "/Users/chenjilin/Downloads", "data save folder")
	//symbolsStr := flag.String("symbols", "BTC-PERP", "symbols, separate by comma")
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
		ftxufChMap := make(map[string]chan *Message)
		bnufChMap := make(map[string]chan *Message)
		for _, xSymbol := range symbols[start:end] {
			ySymbol := strings.Replace(xSymbol, "-PERP", "USDT", -1)
			ftxufChMap[xSymbol] = make(chan *Message, 1024)
			bnufChMap[strings.ToLower(ySymbol)] = ftxufChMap[xSymbol]
			go saveLoop(ctx, cancel, *savePath, xSymbol, ySymbol, ftxufChMap[xSymbol], fileSavedCh)
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
			ws1 := NewFtxufTickerWS(ctx, proxy, outputChMap)
			select {
			case <-ctx.Done():
			case <-ws1.Done():
				cancel()
			}
		}(ctx, cancel, *proxyAddress, ftxufChMap)
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
