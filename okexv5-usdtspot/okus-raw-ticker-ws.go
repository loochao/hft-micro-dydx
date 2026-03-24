package okexv5_usdtspot

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/gorilla/websocket"
	"io"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"
)

type RawTickerWS struct {
	writeCh     chan interface{}
	done        chan interface{}
	reconnectCh chan interface{}
	symbolCh    chan string
	pingCh      chan []byte
	prefix      []byte
	stopped     int32
}

func (w *RawTickerWS) writeLoop(ctx context.Context, conn *websocket.Conn) {
	logger.Debugf("START writeLoop")
	defer logger.Debugf("EXIT writeLoop")
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case msg := <-w.writeCh:
			var msgBytes []byte
			var err error
			switch d := msg.(type) {
			case []byte:
				msgBytes = d
			case string:
				msgBytes = ([]byte)(d)
			default:
				msgBytes, err = json.Marshal(msg)
				if err != nil {
					logger.Debugf("json.Marshal(msg) err %v", err)
					continue
				}
			}
			err = conn.SetWriteDeadline(time.Now().Add(time.Minute))
			if err != nil {
				logger.Debugf("conn.SetWriteDeadline error %v", err)
				w.restart()
				return
			}
			err = conn.WriteMessage(websocket.TextMessage, msgBytes)
			if err != nil {
				logger.Debugf("conn.WriteMessage error %v", err)
				w.restart()
				return
			}
		}
	}
}

func (w *RawTickerWS) readLoop(conn *websocket.Conn, channels map[string]chan *common.RawMessage) {
	logger.Debugf("START readLoop")
	defer logger.Debugf("EXIT readLoop")
	logSilentTime := time.Now()

	var symbol string
	var msg []byte
	var err error
	var ch chan *common.RawMessage
	var ok bool
	var message *common.RawMessage
	var r io.Reader
	const bufferSize = 8192
	index := -1
	pool := [bufferSize]*common.RawMessage{}
	for i := 0; i < bufferSize; i++ {
		pool[i] = &common.RawMessage{
			Prefix: w.prefix,
		}
	}
	var msgLen int
	for {
		err = conn.SetReadDeadline(time.Now().Add(time.Minute))
		if err != nil {
			logger.Debugf("conn.SetReadDeadline error %v", err)
			w.restart()
			return
		}
		_, r, err = conn.NextReader()
		if err != nil {
			logger.Warnf("conn.NextReader error %v", err)
			w.restart()
			return
		}
		msg, err = w.readAll(r)
		if err != nil {
			logger.Warnf("w.readAll error %v", err)
			w.restart()
			return
		}
		msgLen = len(msg)
		if msgLen > 128 && msg[2] == 'a' && msg[37] == '"' {
			//{"arg":{"channel":"tickers","instId":"BTC-USDT"},"data":[{"instType":"SPOT","instId":"BTC-USDT","last":"63059.6","lastSz":"0.00001","askPx":"63069.9","askSz":"1.21105406","bidPx":"63069.8","bidSz":"0.21630249","open24h":"65050.2","high24h":"65459.9","low24h":"62258","sodUtc0":"64774.5","sodUtc8":"63150.1","volCcy24h":"912956387.50099638","vol24h":"14315.06206226","ts":"1636737707175"}]}
			if msg[46] == '"' {
				symbol = common.UnsafeBytesToString(msg[38:46])
			} else if msg[45] == '"' {
				symbol = common.UnsafeBytesToString(msg[38:45])
			} else if msg[47] == '"' {
				symbol = common.UnsafeBytesToString(msg[38:47])
			} else if msg[48] == '"' {
				symbol = common.UnsafeBytesToString(msg[38:48])
			} else if msg[49] == '"' {
				symbol = common.UnsafeBytesToString(msg[38:48])
			} else if msg[50] == '"' {
				symbol = common.UnsafeBytesToString(msg[38:50])
			} else if msg[51] == '"' {
				symbol = common.UnsafeBytesToString(msg[38:51])
			} else {
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("symbol not found for %s", msg)
					logSilentTime = time.Now().Add(time.Minute)
				}
				continue
			}
		} else if msgLen == 4 && msg[0] == 'p' {
			select {
			case w.pingCh <- msg:
			default:
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("w.pongCh <- msg failed %s ch len %d", symbol, len(w.pingCh))
					logSilentTime = time.Now().Add(time.Minute)
				}
			}
			continue
		} else {
			if time.Now().Sub(logSilentTime) > 0 {
				logger.Debugf("MSG %s", msg)
				logSilentTime = time.Now().Add(time.Minute)
			}
			continue
		}
		if ch, ok = channels[symbol]; ok {
			index++
			if index == bufferSize {
				index = 0
			}
			message = pool[index]
			message.Time = time.Now().UnixNano()
			message.Data = msg
			select {
			case ch <- message:
			default:
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf(" ch <- message %s ch len %d", symbol, len(ch))
					logSilentTime = time.Now().Add(time.Minute)
				}
			}
			select {
			case w.symbolCh <- symbol:
			default:
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("w.symbolCh <- symbol failed %s ch len %d", symbol, len(w.symbolCh))
					logSilentTime = time.Now().Add(time.Minute)
				}
			}
		}
	}
}

func (w *RawTickerWS) readAll(r io.Reader) ([]byte, error) {
	b := make([]byte, 0, 1024)
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

func (w *RawTickerWS) reconnect(ctx context.Context, wsUrl string, proxy string, counter int64) (*websocket.Conn, error) {

	if counter != 0 {
		logger.Debugf("reconnect %s, %d retires", wsUrl, counter)
	}

	var dialer *websocket.Dialer

	if proxy != "" {
		proxyUrl, err := url.Parse(proxy)
		if err != nil {
			return nil, fmt.Errorf("url.Parse(proxy) error %v", err)
		}
		dialer = &websocket.Dialer{
			Proxy:             http.ProxyURL(proxyUrl),
			HandshakeTimeout:  60 * time.Second,
			EnableCompression: true,
		}
	} else {
		dialer = &websocket.Dialer{
			HandshakeTimeout:  10 * time.Second,
			EnableCompression: true,
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

func (w *RawTickerWS) mainLoop(ctx context.Context, proxy string, channels map[string]chan *common.RawMessage) {
	logger.Debugf("START mainLoop")
	defer logger.Debugf("EXIT mainLoop")
	ctx, cancel := context.WithCancel(ctx)
	var internalCtx context.Context
	var internalCancel context.CancelFunc
	symbols := make([]string, 0)
	for symbol := range channels {
		symbols = append(symbols, symbol)
	}

	defer func() {
		cancel()
		if internalCancel != nil {
			internalCancel()
		}
		w.Stop()
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
			reconnectTimer.Reset(time.Second * 5)
		case <-reconnectTimer.C:
			if internalCancel != nil {
				internalCancel()
			}
			internalCtx, internalCancel = context.WithCancel(ctx)
			conn, err := w.reconnect(internalCtx, "wss://ws.okx.com:8443/ws/v5/public", proxy, 0)
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

func (w *RawTickerWS) heartbeatLoop(ctx context.Context, conn *websocket.Conn, symbols []string) {
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
		case <-w.pingCh:
			logger.Debugf("PING MSG")
			pingTimer.Reset(time.Second * 15)
			trafficTimeout.Reset(time.Second * 30)
			break
		case <-symbolCheckTimer.C:
			args := make([]WsArgs, 0)
			for symbol, updateTime := range symbolUpdatedTimes {
				if time.Now().Sub(updateTime) > symbolTimeout {
					args = append(args, WsArgs{
						Channel: "tickers",
						InstId:  symbol,
					})
					symbolUpdatedTimes[symbol] = time.Now().Add(symbolTimeout)
				}
			}
			if len(args) > 0 {
				logger.Debugf("SUB %s", args)
				for start := 0; start < len(args); start += 50 {
					end := start + 50
					if end > len(args) {
						end = len(args)
					}
					select {
					case w.writeCh <- WsSubUnsub{
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

func (w *RawTickerWS) Stop() {
	if atomic.CompareAndSwapInt32(&w.stopped, 0, 1) {
		close(w.done)
		logger.Debugf("stopped")
	}
}

func (w *RawTickerWS) restart() {
	select {
	case w.reconnectCh <- nil:
	default:
		logger.Debugf("w.reconnectCh <- nil failed, ch len %d", len(w.reconnectCh))
	}
}

func (w *RawTickerWS) Done() chan interface{} {
	return w.done
}


func NewRawTickerWS(
	ctx context.Context,
	proxy string,
	prefix []byte,
	channels map[string]chan *common.RawMessage,
) *RawTickerWS {
	ws := RawTickerWS{
		done:        make(chan interface{}),
		reconnectCh: make(chan interface{}, 16),
		writeCh:     make(chan interface{}, len(channels)*4),
		symbolCh:    make(chan string, len(channels)*16),
		pingCh:      make(chan []byte, 16),
		stopped:     0,
		prefix:      prefix,
	}
	go ws.mainLoop(ctx, proxy, channels)
	ws.reconnectCh <- nil
	return &ws
}
