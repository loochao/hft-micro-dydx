package main

import (
	"context"
	"flag"
	"github.com/geometrybase/hft-micro/binance-usdtfuture"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/dydx-usdfuture"
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
	savePath := flag.String("path", "/root/dduf-bnuf-depth-and-ticker", "data save folder")

	//savePath := flag.String("path", "/Users/chenjilin/Downloads", "data save folder")
	//proxyAddress := flag.String("proxy", "socks5://127.0.0.1:1083", "symbols group batch size")
	//flag.Parse()

	symbols := make([]string, 0)
	for key := range dydx_usdfuture.TickSizes {
		if _, ok := binance_usdtfuture.TickSizes[strings.Replace(key, "-USD", "USDT", -1)]; ok {
			symbols = append(symbols, key)
		}
	}
	sort.Strings(symbols)
	symbols = symbols[:1]
	logger.Debugf("SYMBOLS %s", symbols)
	//return

	ctx, cancel := context.WithCancel(context.Background())
	fileSavedCh := make(chan string, len(symbols))
	dxufAllChMap := make(map[string]chan *common.RawMessage)
	bnufAllChMap := make(map[string]chan *common.RawMessage)
	for start := 0; start < len(symbols); start += *batchSize {
		end := start + *batchSize
		if end > len(symbols) {
			end = len(symbols)
		}
		dxufChMap := make(map[string]chan *common.RawMessage)
		bnufChMap := make(map[string]chan *common.RawMessage)
		for _, xSymbol := range symbols[start:end] {
			ySymbol := strings.Replace(xSymbol, "-USD", "USDT", -1)
			dxufChMap[xSymbol] = make(chan *common.RawMessage, 1024)
			bnufChMap[ySymbol] = dxufChMap[xSymbol]
			dxufAllChMap[xSymbol] = dxufChMap[xSymbol]
			bnufAllChMap[ySymbol] = dxufChMap[xSymbol]
			go common.RawWSMessageSaveLoop(ctx, cancel, *savePath, xSymbol, ySymbol, dxufChMap[xSymbol], fileSavedCh)
		}
		go func(ctx context.Context, cancel context.CancelFunc, proxy string, outputChMap map[string]chan *common.RawMessage) {
			ws2 := dydx_usdfuture.NewRawDepthWS(ctx, proxy, []byte{'X', 'D'}, outputChMap)
			ws1 := dydx_usdfuture.NewRawTradeWS(ctx, proxy, []byte{'X', 'R'}, outputChMap)
			select {
			case <-ctx.Done():
			case <-ws2.Done():
				cancel()
			case <-ws1.Done():
				cancel()
			}
		}(ctx, cancel, *proxyAddress, dxufChMap)
		go func(ctx context.Context, cancel context.CancelFunc, proxy string, outputChMap map[string]chan *common.RawMessage) {
			ws1 := binance_usdtfuture.NewRawDepth5WS(ctx, proxy, []byte{'Y', 'D'}, outputChMap)
			ws2 := binance_usdtfuture.NewRawBookTickerWS(ctx, proxy, []byte{'Y', 'T'}, outputChMap)
			ws3 := binance_usdtfuture.NewRawTradeWS(ctx, proxy, []byte{'Y', 'R'}, outputChMap)
			select {
			case <-ctx.Done():
			case <-ws1.Done():
				cancel()
			case <-ws2.Done():
				cancel()
			case <-ws3.Done():
				cancel()
			}
		}(ctx, cancel, *proxyAddress, bnufChMap)
	}
	go common.ArchiveDailyJlGzFiles(ctx, *savePath)
	go dydx_usdfuture.StreamRawFundingRate(ctx, *proxyAddress, []byte{'X', 'F'}, dxufAllChMap)
	go binance_usdtfuture.StreamRawFundingRate(ctx, *proxyAddress, []byte{'Y', 'F'}, bnufAllChMap)
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
