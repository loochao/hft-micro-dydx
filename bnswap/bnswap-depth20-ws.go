package bnswap

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
	"sync"
	"time"
)

type Depth20WS struct {
	done        chan interface{}
	reconnectCh chan interface{}
	messageCh   chan []byte
	stopped     bool
	mu          sync.Mutex
}

func (w *Depth20WS) readLoop(conn *websocket.Conn) {
	logger.Debugf("START readLoop")
	defer logger.Debugf("EXIT readLoop")
	logSilentTime := time.Now()
	for {
		err := conn.SetReadDeadline(time.Now().Add(time.Minute))
		if err != nil {
			logger.Warnf("conn.SetReadDeadline error %v", err)
			w.restart()
			return
		}
		_, r, err := conn.NextReader()
		if err != nil {
			logger.Warnf("conn.NextReader error %v", err)
			w.restart()
			return
		}
		msg, err := w.readAll(r)
		if err != nil {
			logger.Warnf("w.readAl error %v", err)
			w.restart()
			return
		}
		select {
		case w.messageCh <- msg:
		default:
			if time.Now().Sub(logSilentTime) > 0 {
				logger.Debugf("w.messageCh <- msg failed ch len %d", len(w.messageCh))
				logSilentTime = time.Now().Add(time.Minute)
			}
		}

	}

}

func (w *Depth20WS) readAll(r io.Reader) ([]byte, error) {
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

func (w *Depth20WS) reconnect(ctx context.Context, wsUrl string, proxy string, counter int64) (*websocket.Conn, error) {

	if counter != 0 {
		logger.Debugf("RECONNECT %s, %d RETRIES", wsUrl, counter)
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
		logger.Warnf("dialer.DialContext ERROR %v", err)
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

func (w *Depth20WS) mainLoop(ctx context.Context, proxy string, channels map[string]chan common.Depth) {
	urlStr := "wss://fstream.binance.com/stream?streams="
	symbols := make([]string, 0)
	for symbol := range channels {
		symbols = append(symbols, symbol)
		urlStr += fmt.Sprintf(
			"%s@depth20@100ms/",
			strings.ToLower(symbol),
		)
	}
	urlStr = urlStr[:len(urlStr)-1]
	logger.Debugf("START mainLoop %s", symbols)

	ctx, cancel := context.WithCancel(ctx)
	var internalCtx context.Context
	var internalCancel context.CancelFunc

	defer func() {
		w.Stop()
		cancel()
		logger.Debugf("EXIT mainLoop %s", symbols)
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
				w.Stop()
				return
			}
			go w.readLoop(conn)
			go w.heartbeatLoop(internalCtx, conn, symbols)

		}
	}
}

func (w *Depth20WS) heartbeatLoop(ctx context.Context, conn *websocket.Conn, symbols []string) {
	logger.Debugf("START heartbeatLoop %s", symbols)
	defer func() {
		logger.Debugf("EXIT heartbeatLoop %s", symbols)
		err := conn.Close()
		if err != nil {
			logger.Debugf("conn.Close() error %v", err)
		}
	}()

	trafficCh := make(chan interface{})

	conn.SetPingHandler(func(msg string) error {
		select {
		case trafficCh <- nil:
		default:
		}
		err := conn.WriteControl(websocket.PongMessage, []byte(msg), time.Now().Add(time.Minute))
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				w.restart()
			}
			return nil
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

func (w *Depth20WS) Stop() {
	w.mu.Lock()
	if !w.stopped {
		w.stopped = true
		close(w.done)
	}
	w.mu.Unlock()
}

func (w *Depth20WS) restart() {
	select {
	case <-w.done:
		return
	default:
	}
	select {
	case <-time.After(time.Second):
		w.Stop()
		logger.Debugf("w.reconnectCh <- nil timeout in 1s, stop ws")
	case w.reconnectCh <- nil:
		logger.Debugf("restart ws")
		return
	}
}

func (w *Depth20WS) Done() chan interface{} {
	return w.done
}

func (w *Depth20WS) dataHandleLoop(ctx context.Context, id int, channels map[string]chan common.Depth) {
	logger.Debugf("START dataHandleLoop")
	defer logger.Debugf("EXIT dataHandleLoop %d", id)
	logSilentTime := time.Now()
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case msg := <-w.messageCh:
			if len(msg) > 128 && msg[2] == 's' {
				depth20, err := ParseDepth20(msg)
				if err != nil {
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("ParseDepth20(msg) error %s %v", msg, err)
						logSilentTime = time.Now().Add(time.Minute)
						continue
					}
				}
				if ch, ok := channels[depth20.Symbol]; ok {
					select {
					case ch <- depth20:
					default:
						if time.Now().Sub(logSilentTime) > 0 {
							logger.Debugf("ch <- depth20 failed ch len %d", len(ch))
							logSilentTime = time.Now().Add(time.Minute)
							continue
						}
					}
				}
			}
		}
	}
}

func NewDepth20WS(
	ctx context.Context,
	proxy string,
	channels map[string]chan common.Depth,
) *Depth20WS {
	ws := Depth20WS{
		done:        make(chan interface{}),
		reconnectCh: make(chan interface{}),
		messageCh:   make(chan []byte, len(channels)*100),
		stopped:     false,
		mu:          sync.Mutex{},
	}
	go ws.mainLoop(ctx, proxy, channels)
	for i := 0; i < 4; i++ {
		go ws.dataHandleLoop(ctx, i, channels)
	}
	ws.reconnectCh <- nil
	return &ws
}
