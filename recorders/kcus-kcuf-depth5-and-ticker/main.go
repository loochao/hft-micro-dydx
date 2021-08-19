package main

import (
	"context"
	"flag"
	kucoin_usdtfuture "github.com/geometrybase/hft-micro/kucoin-usdtfuture"
	kucoin_usdtspot "github.com/geometrybase/hft-micro/kucoin-usdtspot"
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

	proxyAddress := flag.String("proxy", "", "proxy address")
	savePath := flag.String("path", "/root/kcus-kcuf-depth5-and-ticker", "data save folder")

	//savePath := flag.String("path", "/Users/chenjilin/Downloads", "data save folder")
	//proxyAddress := flag.String("proxy", "socks5://127.0.0.1:1083", "proxy address")


	flag.Parse()

	kcufApi, err := kucoin_usdtfuture.NewAPI("", "", "", *proxyAddress)
	if err != nil {
		logger.Fatal(err)
	}

	symbolsMap := make(map[string]string)
	symbols := []string{"BTC-USDT"}
	symbolsMap["BTC-USDT"] = "XBTUSDTM"
	for key := range kucoin_usdtspot.TickSizes{
		if _, ok := kucoin_usdtfuture.TickSizes[strings.Replace(key, "-USDT", "USDTM", -1)]; ok {
			symbols = append(symbols, key)
			symbolsMap[key] = strings.Replace(key, "-USDT", "USDTM", -1)
		}
	}
	//symbols = symbols[:1]
	sort.Strings(symbols)
	logger.Debugf("SYMBOLS %s", symbols)
	ctx, cancel := context.WithCancel(context.Background())
	fileSavedCh := make(chan string, len(symbols))
	kcufAllChMap := make(map[string]chan *Message)
	for start := 0; start < len(symbols); start += *batchSize {
		end := start + *batchSize
		if end > len(symbols) {
			end = len(symbols)
		}
		kcusChMap := make(map[string]chan *Message)
		kcufChMap := make(map[string]chan *Message)
		for _, xSymbol := range symbols[start:end] {
			ySymbol := symbolsMap[xSymbol]
			kcusChMap[xSymbol] = make(chan *Message, 1024)
			kcufAllChMap[ySymbol] = kcusChMap[xSymbol]
			kcufChMap[ySymbol] = kcusChMap[xSymbol]
			go saveLoop(ctx, cancel, *savePath, xSymbol, ySymbol, kcusChMap[xSymbol], fileSavedCh)
		}
		go func(ctx context.Context, cancel context.CancelFunc, proxy string, outputChMap map[string]chan *Message) {
			ws1 := NewKcusDepth5WS(ctx, proxy, outputChMap)
			ws2 := NewKcusTickerWS(ctx, proxy, outputChMap)
			select {
			case <-ctx.Done():
			case <-ws1.Done():
				cancel()
			case <-ws2.Done():
				cancel()
			}
		}(ctx, cancel, *proxyAddress, kcusChMap)
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
