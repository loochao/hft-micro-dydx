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
	symbolsStr := flag.String("symbols", "1INCH-USDT,AAVE-USDT,ADA-USDT,ALGO-USDT,ALPHA-USDT,ATOM-USDT,AVAX-USDT,BAL-USDT,BAND-USDT,BAT-USDT,BCH-USDT,BTC-USDT,BTT-USDT,CELR-USDT,CHZ-USDT,COMP-USDT,CRV-USDT,CVC-USDT,DASH-USDT,DGB-USDT,DOGE-USDT,DOT-USDT,EGLD-USDT,ENJ-USDT,EOS-USDT,ETC-USDT,ETH-USDT,FIL-USDT,FLM-USDT,FTM-USDT,GRT-USDT,HBAR-USDT,ICP-USDT,ICX-USDT,IOST-USDT,IOTA-USDT,KNC-USDT,KSM-USDT,LINK-USDT,LRC-USDT,LTC-USDT,LUNA-USDT,MANA-USDT,MATIC-USDT,MKR-USDT,NEAR-USDT,NEO-USDT,OMG-USDT,ONT-USDT,QTUM-USDT,REN-USDT,RSR-USDT,RVN-USDT,SAND-USDT,SC-USDT,SKL-USDT,SNX-USDT,SOL-USDT,SRM-USDT,STORJ-USDT,SUSHI-USDT,THETA-USDT,TRB-USDT,TRX-USDT,UNI-USDT,WAVES-USDT,XEM-USDT,XLM-USDT,XMR-USDT,XRP-USDT,XTZ-USDT,YFI-USDT,YFII-USDT,ZEC-USDT,ZEN-USDT,ZIL-USDT,ZRX-USDT", "symbols, separate by comma")
	savePath := flag.String("path", "/root/okus-ticker", "data save folder")

	//proxyAddress := flag.String("proxy", "socks5://127.0.0.1:1083", "symbols group batch size")
	//symbolsStr := flag.String("symbols", "BTC-USDT,AAVE-USDT,WAVES-USDT", "symbols, separate by comma")
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
			messageChMap[symbol] = make(chan []byte, 10000)
			go saveLoop(ctx, cancel, *savePath, symbol, messageChMap[symbol], fileSavedCh)
		}
		go func(ctx context.Context, cancel context.CancelFunc, proxy string, outputChMap map[string]chan []byte) {
			ws := NewTickerWS(ctx, proxy, outputChMap)
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
