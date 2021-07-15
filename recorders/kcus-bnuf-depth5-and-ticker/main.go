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

var symbolsMap = map[string]string{
	"BTCUSDT":   "XBTUSDTM",
	"IOSTUSDT":  "IOSTUSDTM",
	"UNIUSDT":   "UNIUSDTM",
	"ICPUSDT":   "ICPUSDTM",
	"THETAUSDT": "THETAUSDTM",
	"YFIUSDT":   "YFIUSDTM",
	"OCEANUSDT": "OCEANUSDTM",
	"XMRUSDT":   "XMRUSDTM",
	"SXPUSDT":   "SXPUSDTM",
	"BCHUSDT":   "BCHUSDTM",
	"TRXUSDT":   "TRXUSDTM",
	"XEMUSDT":   "XEMUSDTM",
	"ETHUSDT":   "ETHUSDTM",
	"MKRUSDT":   "MKRUSDTM",
	"FTMUSDT":   "FTMUSDTM",
	"ATOMUSDT":  "ATOMUSDTM",
	"BANDUSDT":  "BANDUSDTM",
	"DOTUSDT":   "DOTUSDTM",
	"FILUSDT":   "FILUSDTM",
	"AVAXUSDT":  "AVAXUSDTM",
	"QTUMUSDT":  "QTUMUSDTM",
	"COMPUSDT":  "COMPUSDTM",
	"ZECUSDT":   "ZECUSDTM",
	"ADAUSDT":   "ADAUSDTM",
	"DOGEUSDT":  "DOGEUSDTM",
	"XLMUSDT":   "XLMUSDTM",
	"EOSUSDT":   "EOSUSDTM",
	"LTCUSDT":   "LTCUSDTM",
	"VETUSDT":   "VETUSDTM",
	"ONTUSDT":   "ONTUSDTM",
	"RVNUSDT":   "RVNUSDTM",
	"MATICUSDT": "MATICUSDTM",
	"1INCHUSDT": "1INCHUSDTM",
	"XRPUSDT":   "XRPUSDTM",
	"NEOUSDT":   "NEOUSDTM",
	"ALGOUSDT":  "ALGOUSDTM",
	"MANAUSDT":  "MANAUSDTM",
	"WAVESUSDT": "WAVESUSDTM",
	"KSMUSDT":   "KSMUSDTM",
	"AAVEUSDT":  "AAVEUSDTM",
	"LINKUSDT":  "LINKUSDTM",
	"BATUSDT":   "BATUSDTM",
	"DENTUSDT":  "DENTUSDTM",
	"LUNAUSDT":  "LUNAUSDTM",
	"ETCUSDT":   "ETCUSDTM",
	"CHZUSDT":   "CHZUSDTM",
	"CRVUSDT":   "CRVUSDTM",
	"DASHUSDT":  "DASHUSDTM",
	"SNXUSDT":   "SNXUSDTM",
	"GRTUSDT":   "GRTUSDTM",
	"BTTUSDT":   "BTTUSDTM",
	"SUSHIUSDT": "SUSHIUSDTM",
	"ENJUSDT":   "ENJUSDTM",
	"XTZUSDT":   "XTZUSDTM",
	"DGBUSDT":   "DGBUSDTM",
	"SOLUSDT":   "SOLUSDTM",
	"BNBUSDT":   "BNBUSDTM",
}

func main() {

	batchSize := flag.Int("batch", 30, "symbols group batch size")

	proxyAddress := flag.String("proxy", "", "symbols group batch size")
	symbolsStr := flag.String("symbols", "XMR-USDT,LUNA-USDT,SXP-USDT,NEO-USDT,ETC-USDT,BNB-USDT,IOST-USDT,TRX-USDT,VET-USDT,EOS-USDT,TOMO-USDT,ZIL-USDT,BAT-USDT,DASH-USDT,ICP-USDT,ADA-USDT,ATOM-USDT,ZEC-USDT,FTM-USDT,DODO-USDT,LTC-USDT,LRC-USDT,MATIC-USDT,BTT-USDT,ONE-USDT,BTC-USDT,STMX-USDT,XEM-USDT,ZEN-USDT,ANKR-USDT,ETH-USDT,XLM-USDT,ONT-USDT,ALGO-USDT,AVAX-USDT,DGB-USDT,DOGE-USDT,FIL-USDT,OMG-USDT,1INCH-USDT,XRP-USDT,BCH-USDT,NEAR-USDT,XTZ-USDT,GRT-USDT,OGN-USDT", "symbols, separate by comma")
	savePath := flag.String("path", "/root/kcus-bnuf-depth5-and-ticker", "data save folder")
	//savePath := flag.String("path", "/Users/chenjilin/Downloads", "data save folder")
	//symbolsStr := flag.String("symbols", "BTC-USDT", "symbols, separate by comma")
	//proxyAddress := flag.String("proxy", "socks5://127.0.0.1:1080", "symbols group batch size")

	flag.Parse()
	symbols := strings.Split(*symbolsStr, ",")
	ctx, cancel := context.WithCancel(context.Background())
	fileSavedCh := make(chan string, len(symbols))
	for start := 0; start < len(symbols); start += *batchSize {
		end := start + *batchSize
		if end > len(symbols) {
			end = len(symbols)
		}
		kcusChMap := make(map[string]chan *Message)
		bnufChMap := make(map[string]chan *Message)
		for _, xSymbol := range symbols {
			ySymbol := strings.Replace(xSymbol, "-USDT", "USDT", -1)
			kcusChMap[xSymbol] = make(chan *Message, 1024)
			bnufChMap[strings.ToLower(ySymbol)] = kcusChMap[xSymbol]
			go saveLoop(ctx, cancel, *savePath, xSymbol, ySymbol, kcusChMap[xSymbol], fileSavedCh)
		}
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
		go func(ctx context.Context, cancel context.CancelFunc, proxy string, outputChMap map[string]chan *Message) {
			ws1 := NewKcusTickerWS(ctx, proxy, outputChMap)
			ws2 := NewKcusDepth5WS(ctx, proxy, outputChMap)
			select {
			case <-ctx.Done():
			case <-ws1.Done():
				cancel()
			case <-ws2.Done():
				cancel()
			}
		}(ctx, cancel, *proxyAddress, kcusChMap)
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
