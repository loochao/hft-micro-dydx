package bnspot

import (
	"context"
	"fmt"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/gorilla/websocket"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Depth20Websocket struct {
	messageCh   chan []byte
	DataCh      chan *Depth
	done        chan interface{}
	reconnectCh chan interface{}
}

func (w *Depth20Websocket) startRead(conn *websocket.Conn, readTimeout time.Duration) {
	totalLen := 0
	totalCount := 0
	for {
		err := conn.SetReadDeadline(time.Now().Add(readTimeout))
		if err != nil {
			logger.Warnf("SetReadDeadline error %v", err)
			go w.restart()
			return
		}
		_, r, err := conn.NextReader()
		if err != nil {
			logger.Warnf("NextReader error %v", err)
			go w.restart()
			return
		}
		msg, err := w.readAll(r)
		if err != nil {
			logger.Warnf("readAll error %v", err)
			go w.restart()
			return
		}
		if len(msg) < 128 {
			logger.Debugf("PING PONG %s", msg)
			continue
		}
		totalCount += 1
		totalLen += len(msg)
		if totalCount > 1000000 {
			logger.Debugf("AVERAGE MESSAGE LENGTH %d/%d = %d", totalLen, totalCount, totalLen/totalCount)
			totalLen = 0
			totalCount = 0
		}
		select {
		case <-time.After(time.Millisecond):
			logger.Debug("BNSPOT DEPTH20 MSG TO MESSAGE CH TIMEOUT IN 1MS")
		case w.messageCh <- msg:
		}
	}

}

func (w *Depth20Websocket) readAll(r io.Reader) ([]byte, error) {
	b := make([]byte, 0, 2048)
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

func (w *Depth20Websocket) startDataHandler(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case msg := <-w.messageCh:
			depth20, err := ParseDepth20(msg)
			if err != nil {
				logger.Debugf("ParseDepth20 error %v %s", err, msg)
				continue
			}
			select {
			case <-ctx.Done():
				return
			case <-w.done:
				return
			case <-time.After(time.Millisecond):
				logger.Warn("DEPTH20 TO OUTPUT CH TIME OUT IN 1MS")
			case w.DataCh <- depth20:
			}
		}
	}
}

func (w *Depth20Websocket) reconnect(ctx context.Context, wsUrl string, proxy string, counter int64) (*websocket.Conn, error) {

	if counter != 0 {
		logger.Debugf("RECONNECT %s, %d RETRIES", wsUrl, counter)
	}

	var dialer *websocket.Dialer

	if proxy != "" {
		proxyUrl, err := url.Parse(proxy)
		if err != nil {
			logger.Fatalf("PARSE PROXY %v", err)
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

func (w *Depth20Websocket) start(ctx context.Context, symbols []string, timeout time.Duration, proxy string) {
	urlStr := "wss://stream.binance.com:9443/stream?streams="
	for _, symbol := range symbols {
		urlStr += fmt.Sprintf(
			"%s@depth20@100ms/",
			strings.ToLower(symbol),
		)
	}
	urlStr = urlStr[:len(urlStr)-1]
	logger.Debugf("BNSPOT DEPTH20 WS %s", urlStr)

	ctx, cancel := context.WithCancel(ctx)
	var internalCtx context.Context
	var internalCancel context.CancelFunc

	defer func() {
		cancel()
		w.Stop()
		if internalCancel != nil {
			internalCancel()
		}
	}()
	reconnectTimer := time.NewTimer(time.Hour * 9999)
	defer reconnectTimer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.reconnectCh:
			reconnectTimer.Reset(time.Second * 15)
		case <-reconnectTimer.C:
			if internalCancel != nil {
				internalCancel()
			}
			internalCtx, internalCancel = context.WithCancel(ctx)
			conn, err := w.reconnect(internalCtx, urlStr, proxy, 0)
			if err != nil {
				logger.Fatalf("RECONNECT ERROR %v", err)
				return
			}
			go w.startRead(conn, timeout)
			go w.maintainHeartbeat(internalCtx, conn, timeout)

			go w.startDataHandler(internalCtx)
			go w.startDataHandler(internalCtx)
			go w.startDataHandler(internalCtx)
			go w.startDataHandler(internalCtx)
		}
	}
}

func (w *Depth20Websocket) maintainHeartbeat(ctx context.Context, conn *websocket.Conn, timeout time.Duration) {

	defer func() {
		err := conn.Close()
		if err != nil {
			logger.Debugf("conn.Close() ERROR %v", err)
		}
	}()

	conn.SetPingHandler(func(msg string) error {
		//logger.Debugf("BNSPOT DEPTH20 WS PingHandler %s", msg)
		err := conn.WriteControl(websocket.PongMessage, []byte(msg), time.Now().Add(timeout))
		if err != nil {
			go w.restart()
			return nil
		}
		return nil
	})

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		}
	}

}

func (w *Depth20Websocket) Stop() {
	if _, ok := <-w.done; ok {
		close(w.done)
		logger.Infof("BNSPOT DEPTH20 WS STOPPED")
	}
}

func (w *Depth20Websocket) restart() {
	logger.Debugf("BNSPOT DEPTH20 WS RESTART")
	select {
	case <-w.done:
		return
	default:
	}
	timer := time.NewTimer(time.Millisecond)
	defer timer.Stop()
	select {
	case <-timer.C:
		logger.Fatal("NIL TO RECONNECT CH TIMEOUT IN 1MS, EXIT WS")
	case w.reconnectCh <- nil:
		return
	}
}

func (w *Depth20Websocket) Done() chan interface{} {
	return w.done
}

func NewDepth20Websocket(
	ctx context.Context,
	symbols []string,
	timeout time.Duration,
	proxy string,
) *Depth20Websocket {
	ws := Depth20Websocket{
		done:        make(chan interface{}),
		reconnectCh: make(chan interface{}),
		DataCh:      make(chan *Depth, len(symbols)),
		messageCh:   make(chan []byte, 10*len(symbols)),
	}
	go ws.start(ctx, symbols, timeout, proxy)
	ws.reconnectCh <- nil
	return &ws
}
