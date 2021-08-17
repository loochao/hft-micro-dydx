package main

import (
	"context"
	"flag"
	binance_usdtspot "github.com/geometrybase/hft-micro/binance-usdtspot"
	kucoin_usdtfuture "github.com/geometrybase/hft-micro/kucoin-usdtfuture"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"
)



func main() {

	batchSize := flag.Int("batch", 30, "symbols group batch size")

	proxyAddress := flag.String("proxy", "", "symbols group batch size")
	savePath := flag.String("path", "/root/kcuf-bnus-depth5-and-ticker", "data save folder")

	//savePath := flag.String("path", "/Users/chenjilin/Downloads", "data save folder")
	//proxyAddress := flag.String("proxy", "socks5://127.0.0.1:1083", "symbols group batch size")


	flag.Parse()

	kcufApi, err := kucoin_usdtfuture.NewAPI("", "", "", *proxyAddress)
	if err != nil {
		logger.Fatal(err)
	}

	symbolsMap := make(map[string]string)
	symbols := []string{"XBTUSDTM"}
	symbolsMap["XBTUSDTM"] = "BTCUSDT"
	for key := range kucoin_usdtfuture.TickSizes{
		if _, ok := binance_usdtspot.TickSizes[strings.Replace(key, "USDTM", "USDT", -1)]; ok {
			symbols = append(symbols, key)
			symbolsMap[key] = strings.Replace(key, "USDTM", "USDT", -1)
		}
	}
	//symbols = symbols[:1]
	sort.Strings(symbols)
	ctx, cancel := context.WithCancel(context.Background())
	fileSavedCh := make(chan string, len(symbols))
	kcufAllChMap := make(map[string]chan *Message)
	for start := 0; start < len(symbols); start += *batchSize {
		end := start + *batchSize
		if end > len(symbols) {
			end = len(symbols)
		}
		kcufChMap := make(map[string]chan *Message)
		bnusChMap := make(map[string]chan *Message)
		for _, xSymbol := range symbols[start:end] {
			ySymbol := symbolsMap[xSymbol]
			kcufChMap[xSymbol] = make(chan *Message, 1024)
			kcufAllChMap[xSymbol] = kcufChMap[xSymbol]
			bnusChMap[strings.ToLower(ySymbol)] = kcufChMap[xSymbol]
			go saveLoop(ctx, cancel, *savePath, xSymbol, ySymbol, kcufChMap[xSymbol], fileSavedCh)
		}
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
	go streamFundingRate(ctx, kcufApi, kcufAllChMap)
	go archiveFiles(ctx, *savePath)
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
