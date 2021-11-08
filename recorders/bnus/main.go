package main

import (
	"context"
	"flag"
	binance_usdtfuture "github.com/geometrybase/hft-micro/binance-usdtfuture"
	binance_usdtspot "github.com/geometrybase/hft-micro/binance-usdtspot"
	"github.com/geometrybase/hft-micro/common"
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
	savePath := flag.String("path", "/root/bnus", "data save folder")

	//savePath := flag.String("path", "/Users/chenjilin/Downloads", "data save folder")
	//proxyAddress := flag.String("proxy", "socks5://127.0.0.1:1083", "symbols group batch size")
	flag.Parse()

	fapi, err := binance_usdtfuture.NewAPI(&common.Credentials{}, *proxyAddress)
	if err != nil {
		logger.Fatal(err)
	}

	ctx := context.Background()
	fExchangeInfo, err := fapi.GetExchangeInfo(ctx)
	if err != nil {
		logger.Fatal(err)
	}

	futuresMap := make(map[string]string)
	for _, symbol := range fExchangeInfo.Symbols {
		if symbol.Status != "TRADING" {
			continue
		}
		futuresMap[symbol.Symbol] = symbol.Symbol
	}

	api, err := binance_usdtspot.NewAPI(&common.Credentials{}, *proxyAddress)
	if err != nil {
		logger.Fatal(err)
	}

	exchangeInfo, err := api.GetExchangeInfo(ctx)
	if err != nil {
		logger.Fatal(err)
	}

	symbols := make([]string, 0)
	for _, symbol := range exchangeInfo.Symbols {
		if symbol.Status != "TRADING" ||
			(symbol.QuoteAsset != "USDT" &&
				symbol.QuoteAsset != "TUSD" &&
				symbol.QuoteAsset != "USDP" &&
				symbol.QuoteAsset != "USDC" &&
				symbol.QuoteAsset != "BUSD") {
			continue
		}
		if strings.Contains(symbol.Symbol, "DOWNUSDT") {
			continue
		}
		if strings.Contains(symbol.Symbol, "UPUSDT") {
			continue
		}
		if symbol.BaseAsset == "USDC" ||
			symbol.BaseAsset == "TUSD" ||
			symbol.BaseAsset == "USDP" ||
			symbol.BaseAsset == "FTT" {
			symbols = append(symbols, symbol.Symbol)
		} else if symbol.QuoteAsset == "USDT" {
			if _, ok := futuresMap[symbol.Symbol]; ok {
				symbols = append(symbols, symbol.Symbol)
			}
		} else if symbol.QuoteAsset == "TUSD" ||
			symbol.QuoteAsset == "USDP" ||
			symbol.QuoteAsset == "USDC" {
			symbols = append(symbols, symbol.Symbol)
		} else if symbol.QuoteAsset == "BUSD" {
			if _, ok := futuresMap[symbol.Symbol]; ok {
				symbols = append(symbols, symbol.Symbol)
			} else if _, ok := futuresMap[strings.Replace(symbol.Symbol, "BUSD", "USDT", -1)]; ok {
				symbols = append(symbols, symbol.Symbol)
			}
		}
	}

	sort.Strings(symbols)
	logger.Debugf("SYMBOLS %s", symbols)
	logger.Debugf("SYMBOLS LEN %d", len(symbols))

	err = os.MkdirAll(path.Join(*savePath, "/archive"), 0777)
	if err != nil {
		logger.Debugf("os.MkdirAll error %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	fileSavedCh := make(chan string, len(symbols))
	bnusAllChMap := make(map[string]chan *common.RawMessage)
	for start := 0; start < len(symbols); start += *batchSize {
		end := start + *batchSize
		if end > len(symbols) {
			end = len(symbols)
		}
		bnusChMap := make(map[string]chan *common.RawMessage)
		for _, xSymbol := range symbols[start:end] {
			bnusChMap[xSymbol] = make(chan *common.RawMessage, 1024)
			bnusAllChMap[xSymbol] = bnusChMap[xSymbol]
			go common.RawWSMessageSaveLoopForSingleSymbol(ctx, cancel, *savePath, xSymbol, bnusChMap[xSymbol], fileSavedCh)
		}
		go func(ctx context.Context, cancel context.CancelFunc, proxy string, outputChMap map[string]chan *common.RawMessage) {
			ws1 := binance_usdtspot.NewRawDepth20WS(ctx, proxy, []byte{'D'}, outputChMap)
			ws2 := binance_usdtspot.NewRawBookTickerWS(ctx, proxy, []byte{'B'}, outputChMap)
			ws3 := binance_usdtspot.NewRawTradeWS(ctx, proxy, []byte{'T'}, outputChMap)
			select {
			case <-ctx.Done():
			case <-ws1.Done():
				cancel()
			case <-ws2.Done():
				cancel()
			case <-ws3.Done():
				cancel()
			}
		}(ctx, cancel, *proxyAddress, bnusChMap)
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
	case <-time.After(time.Hour * 120):
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
