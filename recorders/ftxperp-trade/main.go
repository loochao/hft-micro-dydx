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
	marketsStr := flag.String("markets", "KIN-PERP,ETC-PERP,KAVA-PERP,ORBS-PERP,YFII-PERP,DMG-PERP,ENJ-PERP,LEO-PERP,LTC-PERP,BNT-PERP,EOS-PERP,RSR-PERP,SECO-PERP,BAL-PERP,CRO-PERP,DOT-PERP,HOLY-PERP,LINA-PERP,RUNE-PERP,HT-PERP,SKL-PERP,SXP-PERP,TRX-PERP,DAWN-PERP,LUNA-PERP,MTL-PERP,YFI-PERP,AAVE-PERP,HUM-PERP,STMX-PERP,THETA-PERP,BTC-PERP,VET-PERP,WAVES-PERP,XEM-PERP,AVAX-PERP,DODO-PERP,ETH-PERP,EXCH-PERP,NEO-PERP,REN-PERP,SHIT-PERP,KNC-PERP,OXY-PERP,PRIV-PERP,PROM-PERP,SOL-PERP,AR-PERP,COMP-PERP,FIDA-PERP,HOT-PERP,KSM-PERP,REEF-PERP,BRZ-PERP,BTT-PERP,NEAR-PERP,OKB-PERP,BSV-PERP,CRV-PERP,ALT-PERP,BAO-PERP,CREAM-PERP,CUSDT-PERP,EGLD-PERP,MEDIA-PERP,ATOM-PERP,LINK-PERP,DEFI-PERP,FTT-PERP,MATIC-PERP,STX-PERP,XAUT-PERP,OMG-PERP,BTMX-PERP,ROOK-PERP,SNX-PERP,MID-PERP,MTA-PERP,QTUM-PERP,SAND-PERP,ZEC-PERP,CONV-PERP,ICP-PERP,SC-PERP,USDT-PERP,XMR-PERP,ALGO-PERP,RAMP-PERP,TRU-PERP,ZRX-PERP,DASH-PERP,FLOW-PERP,HNT-PERP,SRN-PERP,STEP-PERP,SUSHI-PERP,ALCX-PERP,BADGER-PERP,CAKE-PERP,GRT-PERP,MKR-PERP,PUNDIX-PERP,TOMO-PERP,BAND-PERP,BNB-PERP,DENT-PERP,ZIL-PERP,DRGN-PERP,PAXG-PERP,RAY-PERP,UNI-PERP,XLM-PERP,AMPL-PERP,BAT-PERP,BCH-PERP,ONT-PERP,XRP-PERP,FIL-PERP,FLM-PERP,HBAR-PERP,LRC-PERP,TRYB-PERP,UNISWAP-PERP,ADA-PERP,CHZ-PERP,DOGE-PERP,FTM-PERP,SRM-PERP,AXS-PERP,IOTA-PERP,MAPS-PERP,1INCH-PERP,ALPHA-PERP,AUDIO-PERP,PERP-PERP,STORJ-PERP,XTZ-PERP", "markets, separate by comma")
	savePath := flag.String("path", "/mnt/d1/ftx-usdfuture-trade", "data save folder")
	batchSize := flag.Int("batch", 20, "markets group batch size")
	proxyAddress := flag.String("proxy", "", "markets group batch size")
	flag.Parse()
	markets := strings.Split(*marketsStr, ",")
	ctx, cancel := context.WithCancel(context.Background())
	fileSavedCh := make(chan string, len(markets))
	for start := 0; start < len(markets); start += *batchSize {
		end := start + *batchSize
		if end > len(markets) {
			end = len(markets)
		}
		go func(ctx context.Context, cancel context.CancelFunc, proxy, savePath string, symbols []string, fileSavedCh chan string) {
			ws := NewTradeWS(ctx, proxy, savePath, symbols, fileSavedCh)
			select {
			case <-ctx.Done():
			case <-ws.Done():
				cancel()
			}
		}(ctx, cancel, *proxyAddress, *savePath, markets[start:end], fileSavedCh)
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
			if counter == len(markets) {
				logger.Debugf("all markets' file saved")
				return
			}
		case <-time.After(time.Second * 88):
			logger.Debugf("save timeout in 88s")
		}
	}
}
