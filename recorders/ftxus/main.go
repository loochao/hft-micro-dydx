package main

import (
	"context"
	"flag"
	binance_usdtfuture "github.com/geometrybase/hft-micro/binance-usdtfuture"
	"github.com/geometrybase/hft-micro/common"
	ftx_usdfuture "github.com/geometrybase/hft-micro/ftx-usdfuture"
	ftx_usdspot "github.com/geometrybase/hft-micro/ftx-usdspot"
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
	savePath := flag.String("path", "/root/ftxus", "data save folder")

	//savePath := flag.String("path", "/home/clu/Downloads", "data save folder")
	//proxyAddress := flag.String("proxy", "socks5://127.0.0.1:1083", "symbols group batch size")
	flag.Parse()

	filterMap := make(map[string]string)
	ctx := context.Background()
	fapi, err := ftx_usdfuture.NewAPI("", "", "", *proxyAddress)
	if err != nil {
		logger.Fatal(err)
	}
	futures, err := fapi.GetFutures(ctx)
	if err != nil {
		logger.Fatal(err)
	}
	for _, future := range futures {
		if future.Type == "perpetual" && future.Enabled{
			filterMap[future.Name] = future.Name
		}
	}
	bapi, err := binance_usdtfuture.NewAPI(&common.Credentials{}, *proxyAddress)
	if err != nil {
		logger.Fatal(err)
	}
	exchangeInfo, err := bapi.GetExchangeInfo(ctx)
	if err != nil {
		logger.Fatal(err)
	}
	for _, symbol := range exchangeInfo.Symbols {
		if symbol.Status != "TRADING" {
			continue
		}
		filterMap[symbol.Symbol] = symbol.Symbol
	}

	filterMap["USDT/USD"] = "USDT/USD"

	api, err := ftx_usdspot.NewAPI("", "", "", *proxyAddress)
	if err != nil {
		logger.Fatal(err)
	}
	markets, err := api.GetMarkets(ctx)
	if err != nil {
		logger.Fatal(err)
	}
	symbols := make([]string, 0)
	for _, market := range markets {
		if market.Type == "spot" &&
			market.Enabled &&
			(market.QuoteCurrency == "USD" || market.QuoteCurrency == "USDT") &&
			!strings.Contains(market.Name, "BULL") &&
			!strings.Contains(market.Name, "BEAR") &&
			!strings.Contains(market.Name, "HALF") &&
			!strings.Contains(market.Name, "HEDGE") {

			_, ok1 := filterMap[market.Name]
			_, ok2 := filterMap[strings.Replace(market.Name, "/USD", "-PERP", -1)]
			_, ok3 := filterMap[strings.Replace(market.Name, "/USD", "USDT", -1)]
			_, ok4 := filterMap[strings.Replace(market.Name, "/USDT", "-PERP", -1)]
			_, ok5 := filterMap[strings.Replace(market.Name, "/USDT", "USDT", -1)]
			if ok1 || ok2 || ok3 || ok4 || ok5{
				symbols = append(symbols, market.Name)
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
	ftxusAllChMap := make(map[string]chan *common.RawMessage)
	for start := 0; start < len(symbols); start += *batchSize {
		end := start + *batchSize
		if end > len(symbols) {
			end = len(symbols)
		}
		ftxusChMap := make(map[string]chan *common.RawMessage)
		for _, xSymbol := range symbols[start:end] {
			ftxusChMap[xSymbol] = make(chan *common.RawMessage, 1024)
			ftxusAllChMap[xSymbol] = ftxusChMap[xSymbol]
			go common.RawWSMessageSaveLoopForSingleSymbol(ctx, cancel, *savePath, xSymbol, ftxusChMap[xSymbol], fileSavedCh)
		}
		go func(ctx context.Context, cancel context.CancelFunc, proxy string, outputChMap map[string]chan *common.RawMessage) {
			ws1 := ftx_usdspot.NewRawOrderBookWS(ctx, proxy, []byte{'D'}, outputChMap)
			ws2 := ftx_usdspot.NewRawTickerWS(ctx, proxy, []byte{'B'}, outputChMap)
			ws3 := ftx_usdspot.NewRawTradeWS(ctx, proxy, []byte{'T'}, outputChMap)
			select {
			case <-ctx.Done():
			case <-ws1.Done():
				cancel()
			case <-ws2.Done():
				cancel()
			case <-ws3.Done():
				cancel()
			}
		}(ctx, cancel, *proxyAddress, ftxusChMap)
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
