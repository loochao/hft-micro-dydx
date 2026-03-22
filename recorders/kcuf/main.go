package main

import (
	"context"
	"flag"
	"github.com/geometrybase/hft-micro/common"
	kucoin_usdtfuture "github.com/geometrybase/hft-micro/kucoin-usdtfuture"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"os/signal"
	"path"
	"sort"
	"syscall"
	"time"
)

func main() {

	batchSize := flag.Int("batch", 30, "symbols group batch size")

	proxyAddress := flag.String("proxy", "", "proxy address")
	savePath := flag.String("path", "/root/kcuf", "data save folder")

	//savePath := flag.String("path", "/home/clu/Downloads", "data save folder")
	//proxyAddress := flag.String("proxy", "socks5://127.0.0.1:1083", "symbols group batch size")
	flag.Parse()

	api, err := kucoin_usdtfuture.NewAPI("", "", "", *proxyAddress)
	if err != nil {
		logger.Fatal(err)
	}

	ctx := context.Background()
	contracts, err := api.GetContracts(ctx)
	if err != nil {
		logger.Fatal(err)
	}

	symbols := make([]string, 0)
	for _, symbol := range contracts {
		if symbol.QuoteCurrency == "USDT" && symbol.Status == "Open" && symbol.FairMethod == "FundingRate" {
			symbols = append(symbols, symbol.Symbol)
		}
	}
	sort.Strings(symbols)
	logger.Debugf("SYMBOLS %s", symbols)
	logger.Debugf("SYMBOLS LEN %d", len(symbols))
	//symbols = symbols[:1]

	err = os.MkdirAll(path.Join(*savePath, "/archive"), 0777)
	if err != nil {
		logger.Debugf("os.MkdirAll error %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	fileSavedCh := make(chan string, len(symbols))
	kcufAllChMap := make(map[string]chan *common.RawMessage)
	for start := 0; start < len(symbols); start += *batchSize {
		end := start + *batchSize
		if end > len(symbols) {
			end = len(symbols)
		}
		kcufChMap := make(map[string]chan *common.RawMessage)
		for _, xSymbol := range symbols[start:end] {
			kcufChMap[xSymbol] = make(chan *common.RawMessage, 1024)
			kcufAllChMap[xSymbol] = kcufChMap[xSymbol]
			go common.RawWSMessageSaveLoopForSingleSymbol(ctx, cancel, *savePath, xSymbol, kcufChMap[xSymbol], fileSavedCh)
		}
		go func(ctx context.Context, cancel context.CancelFunc, proxy string, outputChMap map[string]chan *common.RawMessage) {
			ws1 := kucoin_usdtfuture.NewRawDepth5WS(ctx, proxy, []byte{'D'}, outputChMap)
			ws2 := kucoin_usdtfuture.NewRawTickerWS(ctx, proxy, []byte{'B'}, outputChMap)
			ws3 := kucoin_usdtfuture.NewRawTradeWS(ctx, proxy, []byte{'T'}, outputChMap)
			select {
			case <-ctx.Done():
			case <-ws1.Done():
				cancel()
			case <-ws2.Done():
				cancel()
			case <-ws3.Done():
				cancel()
			}
		}(ctx, cancel, *proxyAddress, kcufChMap)
	}
	go common.ArchiveDailyJlGzFiles(ctx, *savePath)
	go kucoin_usdtfuture.StreamRawFundingRate(ctx, *proxyAddress, []byte{'F'}, kcufAllChMap)
	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			sig := <-sigs
			logger.Debugf("catch signal %v", sig)
			cancel()
		}()
	}()
	select {
	case <-ctx.Done():
	case <-time.After(time.Hour * 48):
		logger.Debugf("48H Restart...")
		cancel()
	}
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
