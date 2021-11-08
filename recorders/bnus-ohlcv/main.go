package main

import (
	"bufio"
	"compress/gzip"
	"context"
	"flag"
	"fmt"
	"github.com/geometrybase/hft-micro/binance-usdtfuture"
	binance_usdtspot "github.com/geometrybase/hft-micro/binance-usdtspot"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"os/signal"
	"path"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func main() {

	intervalsStr := flag.String("intervals", "1d,1h,5m,1m", "data save folder")
	startYearStr := flag.String("startYear", "2015", "data save folder")
	batchSize := flag.Int("batch", 100000, "kline save batch size")
	sleepInterval := flag.Duration("sleepInterval", time.Second, "sleepInterval")
	roundInterval := flag.Duration("roundInterval", time.Minute, "roundInterval")

	proxyAddress := flag.String("proxy", "", "proxy address")
	savePath := flag.String("path", "/root/bnus-ohlcv", "data save folder")
	//savePath := flag.String("path", "/Users/chenjilin/Downloads", "data save folder")
	//proxyAddress := flag.String("proxy", "socks5://127.0.0.1:1083", "symbols group batch size")
	flag.Parse()

	intervals := strings.Split(*intervalsStr, ",")

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

	startYear, _ := time.Parse("2006", *startYearStr)

	exchangeInfo, err := api.GetExchangeInfo(ctx)
	if err != nil {
		logger.Fatal(err)
	}

	symbols := make([]string, 0)
	for _, symbol := range exchangeInfo.Symbols {
		if symbol.Status != "TRADING" || (symbol.QuoteAsset != "USDT" &&
			symbol.QuoteAsset != "BUSD" &&
			symbol.QuoteAsset != "USDC" &&
			symbol.QuoteAsset != "USDP" &&
			symbol.QuoteAsset != "TUSD") {
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
			symbol.BaseAsset == "BUSD" ||
			symbol.BaseAsset == "FTT" {
			symbols = append(symbols, symbol.Symbol)
		} else if symbol.QuoteAsset == "USDP" ||
			symbol.QuoteAsset == "USDC" ||
			symbol.QuoteAsset == "TUSD" {
			symbols = append(symbols, symbol.Symbol)
		} else if symbol.QuoteAsset == "USDT" {
			if _, ok := futuresMap[symbol.Symbol]; ok {
				symbols = append(symbols, symbol.Symbol)
			}
		} else if symbol.QuoteAsset == "BUSD" {
			if _, ok := futuresMap[symbol.Symbol]; ok {
				symbols = append(symbols, symbol.Symbol)
			} else if _, ok := futuresMap[strings.Replace(symbol.Symbol, "BUSD", "USDT", -1)]; ok {
				symbols = append(symbols, symbol.Symbol)
			}
		}
	}

	sort.Strings(symbols)

	//intervals = []string{"1d"}
	//symbols = symbols[:1]

	logger.Debugf("SYMBOLS %s\n", symbols)
	logger.Debugf("INTERVALS %s\n", intervals)

	for _, interval := range intervals {
		err = os.MkdirAll(path.Join(*savePath, "/"+interval), 0777)
		if err != nil {
			logger.Debugf("os.MkdirAll error %v", err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			sig := <-sigs
			logger.Debugf("catch signal %v", sig)
			cancel()
		}()
	}()

	go func() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Hour * 24 * 7):
			cancel()
		}
	}()

	//backward loop
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		getAllCount := 0
		for _, interval := range intervals {
			for _, symbol := range symbols {
				startTime, err := getFirstLineTimestamp(*savePath, interval, symbol, startYear)
				if err != nil {
					logger.Debugf("%s %s err %v", interval, symbol, err)
					continue
				}
				if startTime.Sub(startYear) <= 0 {
					logger.Debugf("%s %s ignore backward query", interval, symbol)
					getAllCount++
					continue
				}
				kLines := make([]common.KLine, 0)
				retryCount := 10
				getAll := false
				for {
					select {
					case <-ctx.Done():
						return
					default:
					}
					subCtxt, _ := context.WithTimeout(ctx, time.Second*15)
					o, err := api.GetKlines(subCtxt, binance_usdtspot.KlineParams{
						Symbol:   symbol,
						Interval: interval,
						EndTime:  startTime.Unix() * 1000,
						Limit:    1000,
					})
					logger.Debugf("%s %s backward query to %v", interval, symbol, startTime)
					if err != nil {
						logger.Debugf("%s %s GetKLines error %v", interval, symbol, err)
						select {
						case <-ctx.Done():
							return
						default:
						}
						if retryCount <= 0 {
							break
						} else {
							retryCount--
							continue
						}
					}
					olderKlines := make([]common.KLine, 0)
					for _, l := range o {
						if startTime.Sub(l.Timestamp) > 0 {
							olderKlines = append(olderKlines, l)
						}
					}
					kLines = append(olderKlines, kLines...)
					if len(o) < 1000 || len(kLines) >= *batchSize {
						if len(o) < 1000 {
							getAll = true
						}
						break
					}
					//startTime是include的
					startTime = o[0].Timestamp.Add(-time.Second)
					select {
					case <-ctx.Done():
						return
					case <-time.After(*sleepInterval):
					}
				}
				if getAll {
					getAllCount++
				}
				if len(kLines) > 0 {
					logger.Debugf("%s %s GET ALL %v", interval, symbol, getAll)
					err = prependSave(*savePath, interval, symbol, kLines, getAll)
					if err != nil {
						logger.Debugf("%s %s prependSave error %v", interval, symbol, err)
					}
				}
				select {
				case <-ctx.Done():
					return
				case <-time.After(*sleepInterval):
				}
			}
		}
		if getAllCount == len(intervals)*len(symbols) {
			logger.Debugf("GET ALL HISTORY")
			break
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(*sleepInterval):
			logger.Debugf("\n\n BACKWARD NEXT ROUND \n\n")
		}
	}

	//forward loop
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		for _, interval := range intervals {
			for _, symbol := range symbols {
				startTime, err := getLastLineTimestamp(*savePath, interval, symbol, startYear)
				if err != nil {
					logger.Debugf("%s %s err %v", interval, symbol, err)
					continue
				}
				if startTime.Sub(startYear) <= 0 {
					logger.Debugf("%s %s bad start time %v, ignore", interval, symbol, startTime)
					continue
				}
				if time.Now().Sub(startTime) < time.Hour*8 {
					logger.Debugf("%s %s not long than 8 hours %v %v, ignore", interval, symbol, startTime, time.Now())
					continue
				}
				//logger.Debugf("OVERALL %s %s forward query from %v", interval, symbol, startTime)
				kLines := make([]common.KLine, 0)
				retryCount := 10
				for {
					select {
					case <-ctx.Done():
						return
					default:
					}
					subCtxt, _ := context.WithTimeout(ctx, time.Second*15)
					o, err := api.GetKlines(subCtxt, binance_usdtspot.KlineParams{
						Symbol:    symbol,
						Interval:  interval,
						StartTime: startTime.Unix() * 1000,
						Limit:     1000,
					})
					logger.Debugf("%s %s forward query from %v", interval, symbol, startTime)
					if err != nil {
						logger.Debugf("%s %s GetKLines error %v", interval, symbol, err)
						select {
						case <-ctx.Done():
							return
						default:
						}
						if retryCount <= 0 {
							break
						} else {
							retryCount--
							continue
						}
					}
					for _, l := range o {
						if startTime.Sub(l.Timestamp) < 0 && l.Timestamp.Sub(time.Now()) < 0 {
							kLines = append(kLines, l)
						}
					}
					if len(o) < 1000 || len(kLines) >= *batchSize {
						break
					}
					//startTime是include的
					startTime = o[len(o)-1].Timestamp.Add(time.Second)
					select {
					case <-ctx.Done():
						return
					case <-time.After(*sleepInterval):
					}
				}
				if len(kLines) > 0 {
					logger.Debugf("%s %s GET %d", interval, symbol, len(kLines))
					err = appendSave(*savePath, interval, symbol, kLines)
					if err != nil {
						logger.Debugf("%s %s append save error %v", interval, symbol, err)
					}
				}
				select {
				case <-ctx.Done():
					return
				case <-time.After(*sleepInterval):
				}
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(*roundInterval):
			logger.Debugf("\n\n FORWARD NEXT ROUND \n\n")
		}
	}
}

func prependSave(rootPath, interval, symbol string, klines []common.KLine, getAll bool) error {
	oldContents := make([]string, 0)
	dataPath := path.Join(rootPath, interval, fmt.Sprintf("%s.gz", symbol))
	logger.Debugf("prepend save %s\n\n", dataPath)
	f, err := os.OpenFile(dataPath, os.O_RDONLY, 0600)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err == nil {
		g, err := gzip.NewReader(f)
		if err != nil {
			_ = f.Close()
			logger.Debugf("gzip.NewReader %v", err)
			return err
		}
		scanner := bufio.NewScanner(g)
		for scanner.Scan() {
			if tmp := scanner.Text(); len(tmp) > 0 {
				oldContents = append(oldContents, tmp)
			}
		}
		_ = g.Close()
		_ = f.Close()
	}
	f, err = os.OpenFile(dataPath, os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		logger.Debugf("os.OpenFile %v", err)
		return err
	}
	g, err := gzip.NewWriterLevel(f, gzip.BestCompression)
	if err != nil {
		_ = f.Close()
		logger.Debugf("zip.NewWriterLevel %v", err)
		return err
	}
	writer := bufio.NewWriter(g)
	defer func() {
		_ = writer.Flush()
		_ = g.Close()
		_ = f.Close()
	}()
	if getAll {
		_, err = writer.WriteString("timestamp,open,high,low,close,volume\n")
		if err != nil {
			logger.Debugf("writer.WriteString %v", err)
			return err
		}
	}
	for _, k := range klines {
		_, err = writer.WriteString(fmt.Sprintf(
			"%s,%s,%s,%s,%s,%s\n",
			k.Timestamp.UTC().Format(time.RFC3339),
			strconv.FormatFloat(k.Open, 'f', -1, 64),
			strconv.FormatFloat(k.High, 'f', -1, 64),
			strconv.FormatFloat(k.Low, 'f', -1, 64),
			strconv.FormatFloat(k.Close, 'f', -1, 64),
			strconv.FormatFloat(k.Volume, 'f', -1, 64),
		))
		if err != nil {
			logger.Debugf("writer.WriteString %v", err)
			return err
		}
	}
	for _, c := range oldContents {
		_, err = writer.WriteString(fmt.Sprintf("%s\n", c))
		if err != nil {
			logger.Debugf("writer.WriteString %v", err)
			return err
		}
	}
	return nil
}

func getFirstLineTimestamp(rootPath, interval, symbol string, startYear time.Time) (time.Time, error) {
	t := time.Now()
	dataPath := path.Join(rootPath, interval, fmt.Sprintf("%s.gz", symbol))
	f, err := os.OpenFile(dataPath, os.O_RDONLY, 0600)
	if os.IsNotExist(err) {
		return t, nil
	}
	if err != nil {
		return t, err
	}
	g, err := gzip.NewReader(f)
	if err != nil {
		_ = f.Close()
		logger.Debugf("gzip.NewReader error %v %s", err, dataPath)
		return t, err
	}
	defer func() {
		_ = g.Close()
		_ = f.Close()
	}()

	scanner := bufio.NewScanner(g)
	for scanner.Scan() {
		if tmp := scanner.Bytes(); len(tmp) > 15 {
			logger.Debugf("%s", tmp)
			if tmp[0] == 't' {
				return startYear.Add(-time.Second), nil
			} else {
				tt, err := time.Parse(time.RFC3339, strings.Split(string(tmp), ",")[0])
				if err != nil {
					return time.Time{}, err
				} else {
					return tt, nil
				}
			}
		}
	}
	return t, nil
}

func getLastLineTimestamp(rootPath, interval, symbol string, startYear time.Time) (time.Time, error) {
	dataPath := path.Join(rootPath, interval, fmt.Sprintf("%s.gz", symbol))
	f, err := os.OpenFile(dataPath, os.O_RDONLY, 0600)
	if err != nil {
		return time.Time{}, err
	}
	g, err := gzip.NewReader(f)
	if err != nil {
		_ = f.Close()
		logger.Debugf("gzip.NewReader error %v %s", err, dataPath)
		return time.Time{}, err
	}
	defer func() {
		_ = g.Close()
		_ = f.Close()
	}()
	scanner := bufio.NewScanner(g)
	lastMsg := make([]byte, 0)
	for scanner.Scan() {
		if tmp := scanner.Bytes(); len(tmp) > 0 {
			lastMsg = tmp
		}
	}
	if len(lastMsg) > 10 {
		tt, err := time.Parse(time.RFC3339, strings.Split(string(lastMsg), ",")[0])
		if err != nil {
			return time.Time{}, err
		} else {
			return tt, nil
		}
	} else {
		return time.Time{}, fmt.Errorf("can't get timestamp from %s", dataPath)
	}
}

func appendSave(rootPath, interval, symbol string, klines []common.KLine) error {
	dataPath := path.Join(rootPath, interval, fmt.Sprintf("%s.gz", symbol))
	logger.Debugf("append save %s\n\n", dataPath)
	f, err := os.OpenFile(dataPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		logger.Debugf("os.OpenFile %v", err)
		return err
	}
	g, err := gzip.NewWriterLevel(f, gzip.BestCompression)
	if err != nil {
		_ = f.Close()
		logger.Debugf("zip.NewWriterLevel %v", err)
		return err
	}
	writer := bufio.NewWriter(g)
	defer func() {
		_ = writer.Flush()
		_ = g.Close()
		_ = f.Close()
	}()
	for _, k := range klines {
		_, err = writer.WriteString(fmt.Sprintf(
			"%s,%s,%s,%s,%s,%s\n",
			k.Timestamp.UTC().Format(time.RFC3339),
			strconv.FormatFloat(k.Open, 'f', -1, 64),
			strconv.FormatFloat(k.High, 'f', -1, 64),
			strconv.FormatFloat(k.Low, 'f', -1, 64),
			strconv.FormatFloat(k.Close, 'f', -1, 64),
			strconv.FormatFloat(k.Volume, 'f', -1, 64),
		))
		if err != nil {
			logger.Debugf("writer.WriteString %v", err)
			return err
		}
	}
	return nil
}
