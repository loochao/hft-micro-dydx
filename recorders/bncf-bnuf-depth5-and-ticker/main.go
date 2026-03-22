package main

import (
	"context"
	"flag"
	binance_coinfuture "github.com/geometrybase/hft-micro/binance-coinfuture"
	binance_usdtfuture "github.com/geometrybase/hft-micro/binance-usdtfuture"
	"github.com/geometrybase/hft-micro/common"
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
	savePath := flag.String("path", "/root/bncf-bnuf-depth5-and-ticker", "data save folder")

	//savePath := flag.String("path", "/home/clu/Downloads", "data save folder")
	//proxyAddress := flag.String("proxy", "socks5://127.0.0.1:1083", "proxy address")
	flag.Parse()

	symbols := make([]string, 0)
	for xSymbol := range binance_coinfuture.TickSizes {
		if _, ok := binance_usdtfuture.TickSizes[strings.Replace(xSymbol, "USD_PERP", "USDT", -1)]; ok {
			symbols = append(symbols, xSymbol)
		}
	}
	sort.Strings(symbols)
	logger.Debugf("SYMBOLS %s", symbols)
	//symbols = symbols[:1]

	bnufApi, err := binance_usdtfuture.NewAPI(&common.Credentials{}, *proxyAddress)
	if err != nil {
		logger.Fatal(err)
	}
	bncfApi, err := binance_coinfuture.NewAPI(&common.Credentials{}, *proxyAddress)
	if err != nil {
		logger.Fatal(err)
	}

	bnufAllChMap := make(map[string]chan *Message)
	bncfAllChMap := make(map[string]chan *Message)

	ctx, cancel := context.WithCancel(context.Background())
	fileSavedCh := make(chan string, len(symbols))
	for start := 0; start < len(symbols); start += *batchSize {
		end := start + *batchSize
		if end > len(symbols) {
			end = len(symbols)
		}
		bncfChMap := make(map[string]chan *Message)
		bnufChMap := make(map[string]chan *Message)
		for _, xSymbol := range symbols[start:end] {
			ySymbol := strings.Replace(xSymbol, "USD_PERP", "USDT", -1)
			bncfChMap[strings.ToLower(xSymbol)] = make(chan *Message, 1024)
			bnufChMap[strings.ToLower(ySymbol)] = bncfChMap[strings.ToLower(xSymbol)]
			bncfAllChMap[strings.ToLower(xSymbol)] = bncfChMap[strings.ToLower(xSymbol)]
			bnufAllChMap[strings.ToLower(ySymbol)] = bncfChMap[strings.ToLower(xSymbol)]
			go saveLoop(ctx, cancel, *savePath, xSymbol, ySymbol, bncfChMap[strings.ToLower(xSymbol)], fileSavedCh)
		}
		go func(ctx context.Context, cancel context.CancelFunc, proxy string, outputChMap map[string]chan *Message) {
			ws1 := NewBncfDepth5WS(ctx, proxy, outputChMap)
			ws2 := NewBncfBookTickerWS(ctx, proxy, outputChMap)
			select {
			case <-ctx.Done():
			case <-ws1.Done():
				cancel()
			case <-ws2.Done():
				cancel()
			}
		}(ctx, cancel, *proxyAddress, bncfChMap)
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
	}
	go archiveFiles(ctx, *savePath)
	go streamBncfFundingRate(ctx, bncfApi, bncfAllChMap)
	go streamBnufFundingRate(ctx, bnufApi, bnufAllChMap)
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
