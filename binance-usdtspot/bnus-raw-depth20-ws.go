package binance_usdtspot

import (
	"context"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/gorilla/websocket"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"
)

type RawDepth20WS struct {
	done        chan interface{}
	reconnectCh chan interface{}
	stopped     int32
	source      []byte
}

func (w *RawDepth20WS) readLoop(conn *websocket.Conn, channels map[string]chan *common.RawMessage) {
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
	index := -1
	const bufferLen = 8192
	pool := [bufferLen]*common.RawMessage{}
	for i := 0; i < bufferLen; i++ {
		pool[i] = &common.RawMessage{
			Prefix: w.source,
		}
	}

	for {
		err = conn.SetReadDeadline(time.Now().Add(time.Minute))
		if err != nil {
			logger.Warnf("conn.SetReadDeadline error %v", err)
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
		if len(msg) > 128 {
			if msg[18] == '@' {
				symbol = common.UnsafeBytesToString(msg[11:18])
			} else if msg[19] == '@' {
				symbol = common.UnsafeBytesToString(msg[11:19])
			} else if msg[20] == '@' {
				symbol = common.UnsafeBytesToString(msg[11:20])
			} else if msg[21] == '@' {
				symbol = common.UnsafeBytesToString(msg[11:21])
			} else if msg[17] == '@' {
				symbol = common.UnsafeBytesToString(msg[11:17])
			}else{
				if time.Now().Sub(logSilentTime)> 0 {
					logger.Debugf("other msg %s", msg)
					logSilentTime = time.Now().Add(time.Minute)
				}
				continue
			}
			if ch, ok = channels[symbol]; ok {
				index++
				if index == bufferLen {
					index = 0
				}
				message = pool[index]
				message.Time = time.Now().UnixNano()
				message.Data = msg
				select {
				case ch <- message:
				default:
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf(" ch <- msg %s ch len %d", symbol, len(ch))
						logSilentTime = time.Now().Add(time.Minute)
					}
				}
			}
		}
	}
}

func (w *RawDepth20WS) readAll(r io.Reader) ([]byte, error) {
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

func (w *RawDepth20WS) reconnect(ctx context.Context, wsUrl string, proxy string, counter int64) (*websocket.Conn, error) {

	if counter != 0 {
		logger.Debugf("reconnect %s, %d retries", wsUrl, counter)
	}

	var dialer *websocket.Dialer

	if proxy != "" {
		proxyUrl, err := url.Parse(proxy)
		if err != nil {
			return nil, fmt.Errorf("url.Parse(proxy) error %v", err)
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
		logger.Warnf("dialer.DialContext error %v", err)
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

func (w *RawDepth20WS) mainLoop(ctx context.Context, channels map[string]chan *common.RawMessage, proxy string) {
	logger.Debugf("START mainLoop")
	urlStr := "wss://stream.binance.com:9443/stream?streams="
	for symbol := range channels {
		urlStr += fmt.Sprintf(
			"%s@depth20@100ms/",
			strings.ToLower(symbol),
		)
	}
	urlStr = urlStr[:len(urlStr)-1]

	ctx, cancel := context.WithCancel(ctx)
	var internalCtx context.Context
	var internalCancel context.CancelFunc

	defer func() {
		cancel()
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
		case <-w.done:
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
				internalCancel = nil
			}
			internalCtx, internalCancel = context.WithCancel(ctx)
			conn, err := w.reconnect(internalCtx, urlStr, proxy, 0)
			if err != nil {
				logger.Debugf("w.reconnect error %v, stop ws", err)
				if internalCancel != nil {
					internalCancel()
					internalCancel = nil
				}
				return
			}
			go w.readLoop(conn, channels)
			go w.heartbeatLoop(internalCtx, conn)
			reconnectTimer.Reset(time.Hour * 9999)
		}
	}
}

func (w *RawDepth20WS) heartbeatLoop(ctx context.Context, conn *websocket.Conn) {
	logger.Debugf("START heartbeatLoop")
	defer func() {
		logger.Debugf("EXIT heartbeatLoop")
		err := conn.Close()
		if err != nil {
			logger.Debugf("conn.Close() ERROR %v", err)
		}
	}()

	trafficCh := make(chan interface{}, 100)
	conn.SetPingHandler(func(msg string) error {
		select {
		case trafficCh <- nil:
		default:
		}
		err := conn.WriteControl(websocket.PongMessage, []byte(msg), time.Now().Add(time.Minute))
		if err != nil {
			w.restart()
			return err
		}
		return nil
	})

	timer := time.NewTimer(time.Minute * 15)
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case <-timer.C:
			logger.Warnf("no traffic in %v, restart ws", time.Minute*15)
			w.restart()
			return
		case <-trafficCh:
			timer.Reset(time.Minute * 15)
		}
	}

}

func (w *RawDepth20WS) Stop() {
	if atomic.CompareAndSwapInt32(&w.stopped, 0, 1) {
		close(w.done)
		logger.Debugf("stopped")
	}
}

func (w *RawDepth20WS) restart() {
	select {
	case <-w.done:
	case w.reconnectCh <- nil:
		logger.Debugf("restart")
	default:
		logger.Debugf("w.reconnectCh <- nil failed, stop ws")
		w.Stop()
	}
}

func (w *RawDepth20WS) Done() chan interface{} {
	return w.done
}

func NewRawDepth20WS(
	ctx context.Context,
	proxy string,
	source []byte,
	channels map[string]chan *common.RawMessage,
) *RawDepth20WS {
	ws := RawDepth20WS{
		done:        make(chan interface{}),
		reconnectCh: make(chan interface{}, 4),
		stopped: 0,
		source:  source,
	}
	lowerCaseChs := make(map[string]chan *common.RawMessage)
	for symbol, ch := range channels {
		lowerCaseChs[strings.ToLower(symbol)] = ch
	}
	go ws.mainLoop(ctx, lowerCaseChs, proxy)
	ws.reconnectCh <- nil
	return &ws
}
