package bncoinfuture

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

type MarkPriceWebsocket struct {
	messageCh   chan []byte
	DataCh      chan *MarkPrice
	reconnectCh chan interface{}
	done        chan interface{}
}

func (w *MarkPriceWebsocket) startRead(conn *websocket.Conn, readTimeout time.Duration) {
	for {
		err := conn.SetReadDeadline(time.Now().Add(readTimeout))
		if err != nil {
			logger.Warnf("SetReadDeadline error %v", err)
			w.restart()
			return
		}
		_, r, err := conn.NextReader()
		if err != nil {
			logger.Warnf("NextReader error %v", err)
			w.restart()
			return
		}
		msg, err := w.readAll(r)
		if err != nil {
			logger.Warnf("readAll error %v", err)
			w.restart()
			return
		}
		select {
		case <-time.After(time.Millisecond):
			logger.Debug("BNSWAP MARK PRICE MSG TO MESSAGE CH TIMEOUT IN 1MS")
		case w.messageCh <- msg:
		}
	}

}

func (w *MarkPriceWebsocket) readAll(r io.Reader) ([]byte, error) {
	b := make([]byte, 0, 256)
	for {
		if len(b) == cap(b) {
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

func (w *MarkPriceWebsocket) startDataHandler(ctx context.Context) {
	restarted := false
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case msg := <-w.messageCh:
			if restarted {
				break
			}
			markPrice, err := ParseMarkPrice(msg)
			if err != nil {
				logger.Debugf("ParseMarkPrice error %v", err)
				logger.Debugf("ParseMarkPrice %s", msg)
				go w.restart()
				restarted = true
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-w.done:
				return
			case <-time.After(time.Millisecond):
				logger.Warn("MARK PRICE TO OUTPUT CH TIME OUT IN 1MS")
			case w.DataCh <- markPrice:
			}
		}
	}
}

func (w *MarkPriceWebsocket) reconnect(ctx context.Context, wsUrl string, proxy string, counter int64) (*websocket.Conn, error) {

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

func (w *MarkPriceWebsocket) start(ctx context.Context, symbols []string, proxy string) {
	urlStr := "wss://fstream.binance.com/stream?streams="
	for _, symbol := range symbols {
		urlStr += fmt.Sprintf(
			"%s@markPrice@1s/",
			strings.ToLower(symbol),
		)
	}
	urlStr = urlStr[:len(urlStr)-1]
	logger.Debugf("BNSWAP MARK PRICE WS %s", urlStr)

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
			go w.startRead(conn, time.Minute)
			go w.maintainHeartbeat(internalCtx, conn, time.Minute)

			go w.startDataHandler(internalCtx)
			go w.startDataHandler(internalCtx)
			go w.startDataHandler(internalCtx)
			go w.startDataHandler(internalCtx)

			go w.startDataHandler(internalCtx)
			go w.startDataHandler(internalCtx)
			go w.startDataHandler(internalCtx)
			go w.startDataHandler(internalCtx)
		}
	}
}

func (w *MarkPriceWebsocket) maintainHeartbeat(ctx context.Context, conn *websocket.Conn, timeout time.Duration) {

	defer func() {
		err := conn.Close()
		if err != nil {
			logger.Debugf("conn.Close() ERROR %v", err)
		}
	}()

	conn.SetPingHandler(func(msg string) error {
		//logger.Debugf("BNSWAP MARK PRICE WS PingHandler %s", msg)
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

func (w *MarkPriceWebsocket) Stop() {
	if _, ok := <-w.done; ok {
		close(w.done)
		logger.Infof("BNSWAP MARK PRICE WS STOPPED")
	}
}

func (w *MarkPriceWebsocket) restart() {
	logger.Debugf("BNSWAP MARK PRICE WS RESTART")
	select {
	case <-w.done:
		return
	default:
	}
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for {
		select {
		case <-w.done:
			return
		case <-timer.C:
			logger.Warn("NIL TO RECONNECT CH TIMEOUT IN 1S, EXIT")
		case w.reconnectCh <- nil:
			return
		}
		timer.Reset(time.Second)
	}
}

func (w *MarkPriceWebsocket) Done() chan interface{} {
	return w.done
}

func NewMarkPriceWebsocket(
	ctx context.Context,
	symbols []string,
	proxy string,
) *MarkPriceWebsocket {
	ws := MarkPriceWebsocket{
		done:        make(chan interface{}),
		reconnectCh: make(chan interface{}),
		DataCh:      make(chan *MarkPrice, len(symbols)),
		messageCh:   make(chan []byte, 10*len(symbols)),
	}
	go ws.start(ctx, symbols, proxy)
	ws.reconnectCh <- nil
	return &ws
}
