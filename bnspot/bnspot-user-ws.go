package bnspot

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/gorilla/websocket"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"
)

type UserWebsocket struct {
	AccountUpdateEventCh chan *AccountUpdateEvent
	OrderUpdateEventCh   chan *OrderUpdateEvent
	messageCh            chan []byte
	done                 chan interface{}
	reconnectCh          chan interface{}
	RestartCh            chan interface{}
	stopped              int32
}

func (w *UserWebsocket) readLoop(conn *websocket.Conn) {
	logger.Debugf("START readLoop")
	defer logger.Debugf("EXIT readLoop")
	logSilentTime := time.Now()
	for {
		err := conn.SetReadDeadline(time.Now().Add(time.Hour * 4))
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
			logger.Warnf("w.readAll error %v", err)
			w.restart()
			return
		}
		select {
		case w.messageCh <- msg:
		default:
			if time.Now().Sub(logSilentTime) > 0 {
				logger.Debugf("w.messageCh <- msg failed, len(w.messageCh) = %d, msg %s", len(w.messageCh), msg)
				logSilentTime = time.Now().Add(time.Minute / 2)
			}
		}
	}
}

func (w *UserWebsocket) readAll(r io.Reader) ([]byte, error) {
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

func (w *UserWebsocket) dataHandleLoop(ctx context.Context, id int) {
	logger.Debugf("START dataHandleLoop %d", id)
	defer logger.Debugf("EXIT dataHandleLoop %d", id)
	logSilentTime := time.Now()
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case msg := <-w.messageCh:
			if msg[0] == '{' && len(msg) > 14 {
				if msg[2] == 'e' && msg[6] == 'o' {
					accountUpdateEvent := AccountUpdateEvent{}
					err := json.Unmarshal(msg, &accountUpdateEvent)
					if err != nil {
						logger.Debugf("json.Unmarshal(msg, &accountUpdateEvent) error %s %s", err, msg)
						continue
					}
					select {
					case w.AccountUpdateEventCh <- &accountUpdateEvent:
					default:
						if time.Now().Sub(logSilentTime) > 0 {
							logger.Debugf("w.AccountUpdateEventCh <- failed, len(w.AccountUpdateEventCh) = %d, msg %s", len(w.AccountUpdateEventCh), msg)
							logSilentTime = time.Now().Add(time.Minute / 2)
						}
					}

				} else if msg[2] == 'e' && msg[6] == 'e' {
					orderUpdateEvent := OrderUpdateEvent{}
					//logger.Debugf("%s", msg)
					err := json.Unmarshal(msg, &orderUpdateEvent)
					if err != nil {
						logger.Debugf("json.Unmarshal(msg, &orderUpdateEvent) error %v %s", err, msg)
						continue
					}
					select {
					case w.OrderUpdateEventCh <- &orderUpdateEvent:
					default:
						if time.Now().Sub(logSilentTime) > 0 {
							logger.Debugf("w.OrderUpdateEventCh <- failed, len(w.OrderUpdateEventCh) = %d, msg %s", len(w.OrderUpdateEventCh), msg)
							logSilentTime = time.Now().Add(time.Minute / 2)
						}
					}
					continue
				} else if msg[2] == 'e' && msg[6] == 'b' {
					continue
				} else {
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("other msg %s", msg)
						logSilentTime = time.Now().Add(time.Minute / 2)
					}
				}
			} else {
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("other msg %s", msg)
					logSilentTime = time.Now().Add(time.Minute / 2)
				}
			}
		}
	}
}

func (w *UserWebsocket) reconnect(ctx context.Context, wsUrl string, proxy string, counter int64) (*websocket.Conn, error) {

	if counter != 0 {
		logger.Debugf("RECONNECT %s, %d RETRIES", wsUrl, counter)
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

func (w *UserWebsocket) mainLoop(ctx context.Context, urlStr string, proxy string) {
	logger.Debugf("START mainLoop")
	ctx, cancel := context.WithCancel(ctx)
	var internalCtx context.Context
	var internalCancel context.CancelFunc

	defer func() {
		cancel()
		w.Stop()
		logger.Debugf("EXIT mainLoop")
	}()
	reconnectTimer := time.NewTimer(time.Hour * 999)
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
				logger.Debugf("w.reconnect error %v", err)
				internalCancel()
				return
			}
			go w.readLoop(conn)
			go w.heartbeatLoop(internalCtx, conn)
		}
	}
}

func (w *UserWebsocket) heartbeatLoop(ctx context.Context, conn *websocket.Conn) {
	logger.Debugf("START heartbeatLoop")
	defer func() {
		logger.Debugf("EXIT heartbeatLoop")
		err := conn.Close()
		if err != nil {
			logger.Debugf("conn.Close() ERROR %v", err)
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
			go w.restart()
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
			w.restart()
			logger.Debugf("no traffic in %v, restart ws", time.Minute*15)
			return
		case <-trafficCh:
			timer.Reset(time.Minute * 15)
		}
	}

}

func (w *UserWebsocket) Stop() {
	if atomic.LoadInt32(&w.stopped) == 0 {
		atomic.StoreInt32(&w.stopped, 1)
		close(w.done)
		logger.Debugf("stopped")
	}
}

func (w *UserWebsocket) restart() {
	select {
	case w.RestartCh <- nil:
	default:
		logger.Debugf("w.RestartCh <- nil failed, len(w.RestartCh) = %d", len(w.RestartCh))
	}
	select {
	case <-w.done:
	case w.reconnectCh <- nil:
	default:
		logger.Debugf("w.reconnectCh <- nil failed, len(w.reconnectCh) = %d, restart ws", len(w.reconnectCh))
	}
}

func (w *UserWebsocket) Done() chan interface{} {
	return w.done
}

func NewUserWebsocket(
	ctx context.Context,
	api *API,
	proxy string,
) (*UserWebsocket, error) {
	var listenKey ListenKey
	_, err := api.SendAuthenticatedHTTPRequest(
		ctx,
		http.MethodPost,
		"/api/v3/userDataStream",
		nil,
		&listenKey,
	)
	if err != nil {
		return nil, err
	}
	wsUrl := "wss://stream.binance.com:9443/ws/" + listenKey.ListenKey
	ws := UserWebsocket{
		done:                 make(chan interface{}),
		reconnectCh:          make(chan interface{}),
		RestartCh:            make(chan interface{}, 100),
		OrderUpdateEventCh:   make(chan *OrderUpdateEvent, 100),
		AccountUpdateEventCh: make(chan *AccountUpdateEvent, 100),
		messageCh:            make(chan []byte, 10000),
		stopped:              0,
	}
	go func(ctx context.Context, ws *UserWebsocket, listenKey ListenKey) {
		timer := time.NewTimer(time.Minute * 20)
		retryCounter := 0
		for {
			select {
			case <-ctx.Done():
				return
			case <-ws.Done():
				return
			case <-timer.C:
				var resp interface{}
				ctx, _ := context.WithTimeout(ctx, time.Minute)
				_, err := api.SendAuthenticatedHTTPRequest(
					ctx,
					http.MethodPut,
					"/api/v3/userDataStream",
					&listenKey,
					&resp,
				)
				if err != nil {
					if strings.Contains(err.Error(), "connection reset by peer") && retryCounter < 10 {
						retryCounter++
						timer.Reset(time.Second * 15)
						continue
					}
					logger.Debugf("api.SendAuthenticatedHTTPRequest error %v", err)
					ws.Stop()
					return
				}
				retryCounter = 0
				timer.Reset(time.Minute * 20)
			}
		}
	}(ctx, &ws, listenKey)
	go ws.mainLoop(ctx, wsUrl, proxy)
	go ws.dataHandleLoop(ctx, 1)
	//go ws.dataHandleLoop(ctx, 2)
	//go ws.dataHandleLoop(ctx, 3)
	//go ws.dataHandleLoop(ctx, 4)
	ws.reconnectCh <- nil
	return &ws, nil
}
