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
	symbolsStr := flag.String("symbols", "1INCH-PERP,AAVE-PERP,ADA-PERP,ALGO-PERP,ALPHA-PERP,ATOM-PERP,AVAX-PERP,AXS-PERP,BAL-PERP,BAND-PERP,BAT-PERP,BCH-PERP,BNB-PERP,BTC-PERP,BTT-PERP,CHZ-PERP,COMP-PERP,CRV-PERP,DASH-PERP,DEFI-PERP,DENT-PERP,DODO-PERP,DOGE-PERP,DOT-PERP,EGLD-PERP,ENJ-PERP,EOS-PERP,ETC-PERP,ETH-PERP,FIL-PERP,FLM-PERP,FTM-PERP,GRT-PERP,HBAR-PERP,HNT-PERP,HOT-PERP,ICP-PERP,IOTA-PERP,KAVA-PERP,KNC-PERP,KSM-PERP,LINA-PERP,LINK-PERP,LRC-PERP,LTC-PERP,LUNA-PERP,MATIC-PERP,MKR-PERP,MTL-PERP,NEAR-PERP,NEO-PERP,OMG-PERP,ONT-PERP,QTUM-PERP,REEF-PERP,REN-PERP,RSR-PERP,RUNE-PERP,SAND-PERP,SC-PERP,SKL-PERP,SNX-PERP,SOL-PERP,SRM-PERP,STMX-PERP,STORJ-PERP,SUSHI-PERP,SXP-PERP,THETA-PERP,TOMO-PERP,TRX-PERP,UNI-PERP,VET-PERP,WAVES-PERP,XEM-PERP,XLM-PERP,XMR-PERP,XRP-PERP,XTZ-PERP,YFI-PERP,YFII-PERP,ZEC-PERP,ZIL-PERP,ZRX-PERP", "symbols, separate by comma")
	savePath := flag.String("path", "/root/ftxuf-ticker", "data save folder")

	//proxyAddress := flag.String("proxy", "socks5://127.0.0.1:1083", "symbols group batch size")
	//symbolsStr := flag.String("symbols", "AAVE-PERP", "symbols, separate by comma")
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
