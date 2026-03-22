package main

import (
	"context"
	"flag"
	binance_usdtfuture "github.com/geometrybase/hft-micro/binance-usdtfuture"
	"github.com/geometrybase/hft-micro/common"
	kucoin_usdtfuture "github.com/geometrybase/hft-micro/kucoin-usdtfuture"
	kucoin_usdtspot "github.com/geometrybase/hft-micro/kucoin-usdtspot"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"os/signal"
	"path"
	"sort"
	"strings"
	"syscall"
	"time"
)

func main() {

	batchSize := flag.Int("batch", 30, "symbols group batch size")

	proxyAddress := flag.String("proxy", "", "proxy address")
	savePath := flag.String("path", "/root/kcus", "data save folder")

	//savePath := flag.String("path", "/home/clu/Downloads", "data save folder")
	//proxyAddress := flag.String("proxy", "socks5://127.0.0.1:1083", "symbols group batch size")
	flag.Parse()

	ctx := context.Background()

	filterMap := make(map[string]string)
	fApi, err := kucoin_usdtfuture.NewAPI("", "", "", *proxyAddress)
	if err != nil {
		logger.Fatal(err)
	}
	contracts, err := fApi.GetContracts(ctx)
	if err != nil {
		logger.Fatal(err)
	}
	for _, symbol := range contracts {
		if symbol.QuoteCurrency == "USDT" && symbol.Status == "Open" && symbol.FairMethod == "FundingRate" {
			filterMap[symbol.Symbol] = symbol.Symbol
		}
	}
	bApi, err := binance_usdtfuture.NewAPI(&common.Credentials{}, *proxyAddress)
	if err != nil {
		logger.Fatal(err)
	}
	exchangeInfo, err := bApi.GetExchangeInfo(ctx)
	if err != nil {
		logger.Fatal(err)
	}
	for _, symbol := range exchangeInfo.Symbols {
		if symbol.Status != "TRADING" {
			continue
		}
		filterMap[symbol.Symbol] = symbol.Symbol
	}


	api, err := kucoin_usdtspot.NewAPI("", "", "", *proxyAddress)
	if err != nil {
		logger.Fatal(err)
	}
	markets, err := api.GetSymbols(ctx)
	if err != nil {
		logger.Fatal(err)
	}
	symbols := make([]string, 0)
	for _, s := range markets {
		if s.QuoteCurrency == "USDT" && s.Market == "USDS" && s.EnableTrading {
			_, ok1 := filterMap[strings.Replace(s.Symbol, "-USDT", "USDT", -1)]
			_, ok2 := filterMap[strings.Replace(s.Symbol, "-USDT", "USDTM", -1)]
			if ok1 || ok2 {
				symbols = append(symbols, s.Symbol)
			}
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
	kcusAllChMap := make(map[string]chan *common.RawMessage)
	for start := 0; start < len(symbols); start += *batchSize {
		end := start + *batchSize
		if end > len(symbols) {
			end = len(symbols)
		}
		kcusChMap := make(map[string]chan *common.RawMessage)
		for _, xSymbol := range symbols[start:end] {
			kcusChMap[xSymbol] = make(chan *common.RawMessage, 1024)
			kcusAllChMap[xSymbol] = kcusChMap[xSymbol]
			go common.RawWSMessageSaveLoopForSingleSymbol(ctx, cancel, *savePath, xSymbol, kcusChMap[xSymbol], fileSavedCh)
		}
		go func(ctx context.Context, cancel context.CancelFunc, proxy string, outputChMap map[string]chan *common.RawMessage) {
			ws1 := kucoin_usdtspot.NewRawDepth5WS(ctx, proxy, []byte{'D'}, outputChMap)
			ws2 := kucoin_usdtspot.NewRawTickerWS(ctx, proxy, []byte{'B'}, outputChMap)
			ws3 := kucoin_usdtspot.NewRawTradeWS(ctx, proxy, []byte{'T'}, outputChMap)
			select {
			case <-ctx.Done():
			case <-ws1.Done():
				cancel()
			case <-ws2.Done():
				cancel()
			case <-ws3.Done():
				cancel()
			}
		}(ctx, cancel, *proxyAddress, kcusChMap)
	}
	go common.ArchiveDailyJlGzFiles(ctx, *savePath)
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
