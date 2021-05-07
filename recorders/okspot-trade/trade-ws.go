package main

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/geometrybase/hft-micro/okspot"
	"github.com/gorilla/websocket"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"sync/atomic"
	"time"
	"unsafe"
)

type TradeWS struct {
	writeCh     chan interface{}
	done        chan interface{}
	reconnectCh chan interface{}
	symbolCh    chan string
	pongCh      chan []byte

	stopped int32
}

func (w *TradeWS) writeLoop(ctx context.Context, conn *websocket.Conn) {
	logger.Debugf("START writeLoop")
	defer func() {
		logger.Debugf("EXIT writeLoop")
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case msg := <-w.writeCh:
			var bytes []byte
			var err error
			switch d := msg.(type) {
			case []byte:
				bytes = d
			case string:
				bytes = ([]byte)(d)
			default:
				bytes, err = json.Marshal(msg)
				if err != nil {
					logger.Debugf("Marshal err %v", err)
					continue
				}
			}
			err = conn.SetWriteDeadline(time.Now().Add(time.Minute))
			if err != nil {
				logger.Debugf("conn.SetWriteDeadline error %v", err)
				w.restart()
				return
			}

			err = conn.WriteMessage(websocket.TextMessage, bytes)
			if err != nil {
				logger.Debugf("conn.WriteMessage error %v, %s", err, string(bytes))
				w.restart()
				return
			}
		}
	}
}

func (w *TradeWS) readLoop(conn *websocket.Conn, channels map[string]chan []byte) {
	logger.Debugf("START readLoop")
	defer logger.Debugf("EXIT readLoop")
	logSilentTime := time.Now()
	var msg []byte
	var mType int
	var resp []byte
	var err error
	var msgLen int
	var symbolBytes []byte
	var symbol string
	var ok bool
	var ch chan []byte
	for {
		err = conn.SetReadDeadline(time.Now().Add(time.Minute))
		if err != nil {
			logger.Debugf("conn.SetReadDeadline error %v", err)
			w.restart()
			return
		}

		mType, resp, err = conn.ReadMessage()
		if err != nil {
			logger.Debugf("conn.NextReader error %v", err)
			w.restart()
			return
		}

		switch mType {
		case websocket.TextMessage:
			msg = resp
		case websocket.BinaryMessage:
			msg, err = w.parseBinaryResponse(resp)
			if err != nil {
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("w.parseBinaryResponse error %v", err)
					logSilentTime = time.Now().Add(time.Minute)
				}
				continue
			}
		}

		msgLen = len(msg)
		if msgLen > 128 {
			if msg[msgLen-53] == ':' {
				symbolBytes = msg[msgLen-51 : msgLen-43]
				symbol = *(*string)(unsafe.Pointer(&symbolBytes))
			} else if msg[msgLen-54] == ':' {
				symbolBytes = msg[msgLen-52 : msgLen-43]
				symbol = *(*string)(unsafe.Pointer(&symbolBytes))
			} else if msg[msgLen-55] == ':' {
				symbolBytes = msg[msgLen-53 : msgLen-43]
				symbol = *(*string)(unsafe.Pointer(&symbolBytes))
			} else if msg[msgLen-56] == ':' {
				symbolBytes = msg[msgLen-54 : msgLen-43]
				symbol = *(*string)(unsafe.Pointer(&symbolBytes))
			} else if msg[msgLen-57] == ':' {
				symbolBytes = msg[msgLen-55 : msgLen-43]
				symbol = *(*string)(unsafe.Pointer(&symbolBytes))
			} else {
				if time.Now().Sub(logSilentTime) > 0 {
					logSilentTime = time.Now().Add(time.Minute)
					logger.Debugf("other msg %s", msg)
				}
				continue
			}
		} else {
			if time.Now().Sub(logSilentTime) > 0 {
				logSilentTime = time.Now().Add(time.Minute)
				logger.Debugf("other msg %s", msg)
			}
			continue
		}

		if ch, ok = channels[symbol]; ok {
			select {
			case ch <- msg:
				select {
				case w.symbolCh <- symbol:
				default:
					if time.Now().Sub(logSilentTime) > 0 {
						logSilentTime = time.Now().Add(time.Minute)
						logger.Debugf("w.symbolCh <- symbol failed ch len %d", len(w.symbolCh))
					}
				}
			default:
				if time.Now().Sub(logSilentTime) > 0 {
					logSilentTime = time.Now().Add(time.Minute)
					logger.Debugf("ch <- msg failed ch len %d", len(ch))
				}
			}
		}

	}
}

func (w *TradeWS) parseBinaryResponse(resp []byte) ([]byte, error) {
	var standardMessage []byte
	var err error
	// Detect GZIP
	if resp[0] == 31 && resp[1] == 139 {
		b := bytes.NewReader(resp)
		var gReader *gzip.Reader
		gReader, err = gzip.NewReader(b)
		if err != nil {
			return standardMessage, err
		}
		standardMessage, err = w.readAll(gReader)
		if err != nil {
			return standardMessage, err
		}
		err = gReader.Close()
		if err != nil {
			return standardMessage, err
		}
	} else {
		reader := flate.NewReader(bytes.NewReader(resp))
		standardMessage, err = w.readAll(reader)
		if err != nil {
			return standardMessage, err
		}
		err = reader.Close()
		if err != nil {
			return standardMessage, err
		}
	}
	return standardMessage, nil
}

func (w *TradeWS) readAll(r io.Reader) ([]byte, error) {
	b := make([]byte, 0, 512)
	for {
		if len(b) == cap(b) {
			// Add more capacity (let append pick how much).
			b = append(b, 0)[:len(b)]
		}
		n, err := r.Read(b[len(b):cap(b)])
		b = b[:len(b)+n]
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return b, err
		}
	}
}

func (w *TradeWS) reconnect(ctx context.Context, wsUrl string, proxy string, counter int64) (*websocket.Conn, error) {

	if counter != 0 {
		logger.Debugf("reconnect %d %s", counter, wsUrl)
	}

	var dialer *websocket.Dialer

	if proxy != "" {
		proxyUrl, err := url.Parse(proxy)
		if err != nil {
			return nil, fmt.Errorf("url.Parse error %v", err)
		}
		dialer = &websocket.Dialer{
			Proxy:            http.ProxyURL(proxyUrl),
			HandshakeTimeout: 60 * time.Second,
		}
	} else {
		dialer = &websocket.Dialer{
			HandshakeTimeout: 10 * time.Second,
		}
	}
	conn, _, err := dialer.DialContext(
		ctx,
		wsUrl,
		http.Header{
			"User-Agent":      []string{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/76.0.3809.132 Safari/537.36"},
			"Accept-Encoding": []string{"gzip, deflate, br"},
			"Accept-Language": []string{"en-US,en;q=0.9,zh-CN;q=0.8,zh;q=0.7,fr;q=0.6,nl;q=0.5,zh-TW;q=0.4,vi;q=0.3"},
		},
	)
	if err != nil {
		logger.Debugf("dialer.DialContext ERROR %v", err)
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("reconnect error: context is done")
		case <-w.done:
			return nil, fmt.Errorf("reconnect error: ws is done")
		case <-time.After(time.Second * 10):
			return w.reconnect(ctx, wsUrl, proxy, counter+1)
		}
	}
	return conn, nil
}

func (w *TradeWS) mainLoop(
	ctx context.Context,
	proxy string,
	channels map[string]chan []byte,
) {

	logger.Debugf("START mainLoop")

	symbols := make([]string, 0)
	for symbol := range channels {
		symbols = append(symbols, symbol)
	}
	ctx, cancel := context.WithCancel(ctx)
	var internalCtx context.Context
	var internalCancel context.CancelFunc

	defer func() {
		cancel()
		if internalCancel != nil {
			internalCancel()
		}
		w.Stop()
		logger.Debugf("EXIT mainLoop")
	}()
	reconnectTimer := time.NewTimer(time.Hour * 9999)
	defer reconnectTimer.Stop()

	for {
		select {
		case <-ctx.Done():
			if internalCancel != nil {
				internalCancel()
				internalCancel = nil
			}
			return
		case <-w.reconnectCh:
			if internalCancel != nil {
				internalCancel()
				internalCancel = nil
			}
			reconnectTimer.Reset(time.Second * 15)
		case <-reconnectTimer.C:
			if internalCancel != nil {
				internalCancel()
			}
			internalCtx, internalCancel = context.WithCancel(ctx)
			conn, err := w.reconnect(internalCtx, "wss://real.okex.com:8443/ws/v3", proxy, 0)
			if err != nil {
				logger.Debugf("w.reconnect error %v", err)
				internalCancel()
				w.Stop()
				return
			}
			go w.readLoop(conn, channels)
			go w.writeLoop(internalCtx, conn)
			go w.heartbeatLoop(internalCtx, conn, symbols)
		}
	}
}

func (w *TradeWS) heartbeatLoop(ctx context.Context, conn *websocket.Conn, symbols []string) {
	logger.Debugf("START heartbeatLoop")
	defer func() {
		logger.Debugf("Exit heartbeatLoop")
		err := conn.Close()
		if err != nil {
			logger.Debugf("conn.Close() ERROR %v", err)
		}
	}()
	symbolTimeout := time.Minute
	symbolCheckInterval := time.Second
	symbolCheckTimer := time.NewTimer(time.Second)
	defer symbolCheckTimer.Stop()
	symbolUpdatedTimes := make(map[string]time.Time)
	for _, symbol := range symbols {
		symbolUpdatedTimes[symbol] = time.Unix(0, 0)
	}
	trafficTimeout := time.NewTimer(time.Minute * 5)
	pingTimer := time.NewTimer(time.Second * 15)
	defer trafficTimeout.Stop()
	defer pingTimer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case <-trafficTimeout.C:
			logger.Debugf("traffic timeout in 30s, restart ws")
			w.restart()
			return
		case <-pingTimer.C:
			pingTimer.Reset(time.Second * 15)
			select {
			case w.writeCh <- []byte("ping"):
				break
			default:
				logger.Debugf("w.writeCh <- ping failed, ch len %d", len(w.writeCh))
			}
			break
		case symbol := <-w.symbolCh:
			pingTimer.Reset(time.Second * 15)
			trafficTimeout.Reset(time.Second * 30)
			symbolUpdatedTimes[symbol] = time.Now()
			break
		case <-w.pongCh:
			pingTimer.Reset(time.Second * 15)
			trafficTimeout.Reset(time.Second * 30)
			break
		case <-symbolCheckTimer.C:
			args := make([]string, 0)
			for symbol, updateTime := range symbolUpdatedTimes {
				if time.Now().Sub(updateTime) > symbolTimeout {
					args = append(args, fmt.Sprintf("spot/trade:%s", symbol))
					symbolUpdatedTimes[symbol] = time.Now().Add(symbolTimeout)
				}
			}
			if len(args) > 0 {
				logger.Debugf("SUBSCRIBE %s", args)
				for start := 0; start < len(args); start += 50 {
					end := start + 50
					if end > len(args) {
						end = len(args)
					}
					select {
					case w.writeCh <- okspot.Subscription{
						Op:   "subscribe",
						Args: args[start:end],
					}:
					default:
						logger.Debugf("w.writeCh <- Subscription failed, ch len %d", len(w.writeCh))
					}
				}
			}
			symbolCheckTimer.Reset(symbolCheckInterval)
			break
		}
	}

}

func (w *TradeWS) Stop() {
	if atomic.LoadInt32(&w.stopped) == 0 {
		atomic.StoreInt32(&w.stopped, 1)
		close(w.done)
		logger.Infof("KCPERP DEPTH5 WS STOPPED")
	}
}

func (w *TradeWS) restart() {
	select {
	case <-w.done:
	case <-time.After(time.Millisecond):
		w.Stop()
		logger.Debugf("KCPERP NIL TO RECONNECT CH TIMEOUT IN 1MS, STOP WS!")
	case w.reconnectCh <- nil:
		logger.Infof("KCPERP WS RESTART")
	}
}

func (w *TradeWS) Done() chan interface{} {
	return w.done
}

func (w *TradeWS) saveLoop(ctx context.Context, savePath, symbol string, inputCh chan []byte, fileSavedCh chan string) {
	logger.Debugf("START saveLoop %s", symbol)
	hourUpdateTimer := time.NewTimer(time.Second)
	var dayTime time.Time
	var outPath string
	var file *os.File
	var gw *gzip.Writer
	var msg []byte
	var err error
	var nextLine = []byte("\n")
	defer func() {
		if gw != nil {
			logger.Debugf("close gzip writer for %s", symbol)
			err = gw.Close()
			if err != nil {
				logger.Debugf("close gzip writer %s error %v, stop ws", outPath, err)
			}
		}
		if file != nil {
			logger.Debugf("close file %s", symbol)
			err = file.Close()
			if err != nil {
				logger.Debugf("close file %s error %v, stop ws", outPath, err)
			}
		}
		fileSavedCh <- symbol
		logger.Debugf("EXIT saveLoop %s", symbol)
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case <-hourUpdateTimer.C:
			if file != nil {
				err = file.Close()
				if err != nil {
					logger.Debugf("close file %s error %v, stop ws", outPath, err)
					w.Stop()
					return
				}
			}
			if gw != nil {
				err = gw.Close()
				if err != nil {
					logger.Debugf("close gzip writer %s error %v, stop ws", outPath, err)
					w.Stop()
					return
				}
			}
			dayTime = time.Now().Truncate(time.Hour * 24)
			outPath = fmt.Sprintf("%s/%s-%s.okspot.trade.jl.gz", savePath, dayTime.Format("20060102"), symbol)
			file, err = os.OpenFile(outPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0755)
			if err != nil {
				w.Stop()
				logger.Debugf("os.OpenFile error %v, stop ws", err)
				return
			}
			gw, err = gzip.NewWriterLevel(file, gzip.BestCompression)
			if err != nil {
				w.Stop()
				logger.Debugf("gzip.NewWriterLevel error %v, stop ws", err)
				return
			}
			gw.Name = fmt.Sprintf("%s-%s.depth20.jl", dayTime.Format("20060102"), symbol)
			gw.ModTime = time.Now()
			gw.Comment = fmt.Sprintf("depth20 raw json line for %s@%s", symbol, dayTime.Format("20060102"))
			hourUpdateTimer.Reset(
				time.Now().Truncate(
					time.Hour * 24,
				).Add(
					time.Hour * 24,
				).Add(
					time.Duration(rand.Intn(60)) * time.Second,
				).Sub(time.Now()),
			)
		case msg = <-inputCh:
			if gw != nil {
				_, err = gw.Write(msg)
				if err != nil {
					w.Stop()
					logger.Debugf("gw.Write error %v, stop ws", err)
					return
				}
				_, err = gw.Write(nextLine)
				if err != nil {
					w.Stop()
					logger.Debugf("gw.Write error %v, stop ws", err)
					return
				}
			}
		}
	}
}

func NewTradeWS(
	ctx context.Context,
	proxy string,
	savePath string,
	symbols []string,
	fileSavedCh chan string,
) *TradeWS {
	ws := TradeWS{
		done:        make(chan interface{}),
		reconnectCh: make(chan interface{}),
		writeCh:     make(chan interface{}, 100*len(symbols)),
		symbolCh:    make(chan string, 100*len(symbols)),
		pongCh:      make(chan []byte, 100),
		stopped:     0,
	}
	channels := make(map[string]chan []byte)
	for _, symbol := range symbols {
		channels[symbol] = make(chan []byte, 10000)
		go ws.saveLoop(ctx, savePath, symbol, channels[symbol], fileSavedCh)
	}
	go ws.mainLoop(ctx, proxy, channels)
	ws.reconnectCh <- nil
	return &ws
}
