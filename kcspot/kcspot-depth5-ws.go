package kcspot

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/gorilla/websocket"
	"io"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"
)

type Depth5Websocket struct {
	messageCh   chan []byte
	DataCh      chan *Depth5
	RestartCh   chan interface{}
	writeCh     chan interface{}
	done        chan interface{}
	reconnectCh chan interface{}
	symbolCh    chan string
	stopped     int32
}

func (w *Depth5Websocket) writeLoop(ctx context.Context, conn *websocket.Conn) {
	logger.Debugf("START writeLoop")
	defer logger.Debugf("EXIT writeLoop")
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
					logger.Debugf("json.Marshal err %v", err)
					continue
				}
			}
			err = conn.SetWriteDeadline(time.Now().Add(300 * time.Millisecond))
			if err != nil {
				w.restart()
				return
			}

			err = conn.WriteMessage(websocket.TextMessage, bytes)

			if err != nil {
				logger.Debugf("conn.WriteMessage %s error %v", string(bytes), err)
				w.restart()
				return
			}
		}
	}
}

func (w *Depth5Websocket) readLoop(conn *websocket.Conn) {
	logger.Debugf("START readLoop")
	defer logger.Debugf("EXIT readLoop")
	totalCount := 0
	totalLen := 0
	for {
		err := conn.SetReadDeadline(time.Now().Add(time.Minute))
		if err != nil {
			logger.Debugf("conn.SetReadDeadline error %v", err)
			go w.restart()
			return
		}
		_, r, err := conn.NextReader()
		if err != nil {
			logger.Debugf("conn.NextReader error %v", err)
			go w.restart()
			return
		}
		msg, err := w.readAll(r)
		if err != nil {
			logger.Debugf("readAll error %v", err)
			go w.restart()
			return
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
			logger.Debug("KCSPOT DEPTH50 MSG TO MESSAGE CH TIMEOUT IN 1MS")
		case w.messageCh <- msg:
		}
	}

}

func (w *Depth5Websocket) readAll(r io.Reader) ([]byte, error) {
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

func (w *Depth5Websocket) startDataHandler(ctx context.Context, id int) {
	logger.Debugf("START depthParseLoop %d", id)
	defer logger.Debugf("EXIT depthParseLoop %d", id)
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case msg := <-w.messageCh:
			if msg[2] == 'd' && msg[5] == 'a' {
				depth20, err := ParseDepth5(msg)
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
					logger.Debugf("w.DataCh <- depth20 timeout in 1ms")
				case w.DataCh <- depth20:
				}
				select {
				case w.symbolCh <- depth20.Symbol:
				default:
				}
			}
		}
	}
}

func (w *Depth5Websocket) reconnect(ctx context.Context, wsUrl string, proxy string, counter int64) (*websocket.Conn, error) {

	if counter != 0 {
		logger.Debugf("reconnect %s, %d retries", wsUrl, counter)
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

func (w *Depth5Websocket) mainLoop(ctx context.Context, api *API, symbols []string, proxy string) {
	logger.Debugf("START mainLoop")
	defer logger.Debugf("EXIT mainLoop")

	ctx, cancel := context.WithCancel(ctx)
	var internalCtx context.Context
	var internalCancel context.CancelFunc

	defer func() {
		if internalCancel != nil {
			internalCancel()
		}
		cancel()
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
			reconnectTimer.Reset(time.Second * 15)
		case <-reconnectTimer.C:
			if internalCancel != nil {
				internalCancel()
			}
			internalCtx, internalCancel = context.WithCancel(ctx)
			connectToken, err := api.GetPublicConnectToken(internalCtx)
			if err != nil {
				logger.Debugf("api.GetPublicConnectToken error %v", err)
				internalCancel()
				w.Stop()
				return
			}
			if len(connectToken.InstanceServers) == 0 {
				logger.Debugf("no InstanceServers %v", connectToken)
				internalCancel()
				w.Stop()
				return
			}
			urlStr := connectToken.InstanceServers[0].Endpoint + "?token=" + connectToken.Token
			conn, err := w.reconnect(internalCtx, urlStr, proxy, 0)
			if err != nil {
				internalCancel()
				logger.Debugf("w.reconnect error %v", err)
				return
			}
			go w.readLoop(conn)
			go w.writeLoop(internalCtx, conn)
			go w.heartbeatLoop(internalCtx, conn, symbols)

		}
	}
}

func (w *Depth5Websocket) heartbeatLoop(ctx context.Context, conn *websocket.Conn, symbols []string) {
	logger.Debugf("START heartbeatLoop")
	defer func() {
		logger.Debugf("EXIT heartbeatLoop")
		err := conn.Close()
		if err != nil {
			logger.Debugf("conn.Close() ERROR %v", err)
		}
	}()

	symbolTimeout := time.Minute
	symbolCheckInterval := time.Second
	pingTimer := time.NewTimer(time.Second)
	symbolCheckTimer := time.NewTimer(time.Second)
	defer symbolCheckTimer.Stop()
	symbolUpdatedTimes := make(map[string]time.Time)
	for _, symbol := range symbols {
		symbolUpdatedTimes[symbol] = time.Unix(0, 0)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case symbol := <-w.symbolCh:
			symbolUpdatedTimes[symbol] = time.Now()
			break
		case <-pingTimer.C:
			pingTimer.Reset(time.Second * 15)
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Millisecond):
				logger.Debug("w.writeCh <- Ping timeout in 1ms")
			case w.writeCh <- Ping{
				ID:   fmt.Sprintf("%d", time.Now().Nanosecond()/1000000),
				Type: "ping",
			}:
			}
			break
		case <-symbolCheckTimer.C:
		loop:
			for symbol, updateTime := range symbolUpdatedTimes {
				if time.Now().Sub(updateTime) > symbolTimeout {
					logger.Debugf("SUBSCRIBE %s", fmt.Sprintf("/spotMarket/level2Depth5:%s", symbol))
					select {
					case <-ctx.Done():
					case <-time.After(time.Millisecond):
						logger.Debugf("w.writeCh <- SubscribeMsg timeout in 1ms, %s", fmt.Sprintf("/spotMarket/level2Depth5:%s", symbol))
					case w.writeCh <- SubscribeMsg{
						ID:             fmt.Sprintf("/spotMarket/level2Depth5:%s", symbol),
						Type:           "subscribe",
						Topic:          fmt.Sprintf("/spotMarket/level2Depth5:%s", symbol),
						PrivateChannel: false,
						Response:       false,
					}:
						symbolUpdatedTimes[symbol] = time.Now().Add(symbolCheckInterval * time.Duration(len(symbols)*2))
						break loop
					}
				}
			}
			symbolCheckTimer.Reset(symbolCheckInterval)
			break
		}
	}

}

func (w *Depth5Websocket) Stop() {
	if atomic.LoadInt32(&w.stopped) == 0 {
		atomic.StoreInt32(&w.stopped, 1)
		close(w.done)
		logger.Debugf("stopped")
	}
}

func (w *Depth5Websocket) restart() {
	select {
	case w.RestartCh <- nil:
	default:
		logger.Debugf("w.RestartCh <- nil failed")
	}
	select {
	case <-w.done:
		return
	case <-time.After(time.Millisecond):
		logger.Debugf("w.reconnectCh <- nil time out in 1m, stop ws")
		w.Stop()
	case w.reconnectCh <- nil:
	}
}

func (w *Depth5Websocket) Done() chan interface{} {
	return w.done
}

func NewDepth5Websocket(
	ctx context.Context,
	api *API,
	symbols []string,
	proxy string,
) *Depth5Websocket {
	ws := Depth5Websocket{
		done:        make(chan interface{}),
		reconnectCh: make(chan interface{}),
		DataCh:      make(chan *Depth5, 100*len(symbols)),
		RestartCh:   make(chan interface{}, 100),
		messageCh:   make(chan []byte, 100*len(symbols)),
		writeCh:     make(chan interface{}, 100*len(symbols)),
		symbolCh:    make(chan string, 100*len(symbols)),
		stopped:     0,
	}
	go ws.mainLoop(ctx, api, symbols, proxy)
	go ws.startDataHandler(ctx, 1)
	go ws.startDataHandler(ctx, 2)
	go ws.startDataHandler(ctx, 3)
	go ws.startDataHandler(ctx, 4)
	ws.reconnectCh <- nil
	return &ws
}
