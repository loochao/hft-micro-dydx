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
	"sync"
	"time"
)

type UserWebsocket struct {
	AccountUpdateEventCh chan *AccountUpdateEvent
	OrderUpdateEventCh   chan *OrderUpdateEvent
	messageCh            chan []byte
	done                 chan interface{}
	reconnectCh          chan interface{}
	mu                   sync.Mutex
	stopped              bool
}

func (w *UserWebsocket) startRead(ctx context.Context, conn *websocket.Conn) {
	defer func() {
		logger.Debugf("EXIT startRead")
	}()
	for {
		err := conn.SetReadDeadline(time.Now().Add(time.Hour * 4))
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
		//帐号相关的消息不能丢
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case w.messageCh <- msg:
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

func (w *UserWebsocket) startDataHandler(ctx context.Context, id int) {
	defer func() {
		logger.Debugf("EXIT startDataHandler %d", id)
	}()
	totalLen := 0
	totalCount := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case msg := <-w.messageCh:
			totalCount += 1
			totalLen += len(msg)
			if totalLen > 1000000 {
				logger.Debugf("BNSPOT %d AVERAGE LENGTH %d", id, totalLen/totalCount)
				totalLen = 0
				totalCount = 0
			}
			if msg[0] == '{' && len(msg) > 14 {
				if msg[2] == 'e' && msg[6] == 'o' {
					accountUpdateEvent := AccountUpdateEvent{}
					err := json.Unmarshal(msg, &accountUpdateEvent)
					if err != nil {
						logger.Debugf("Unmarshal AccountUpdateEvent error %s %s", err, msg)
						continue
					}
					select {
					case <-ctx.Done():
						return
					case <-w.done:
						return
					//case <-time.After(time.Millisecond):
					//	logger.Warn("outboundAccountPosition TO OUTPUT CH TIME OUT IN 1MS")
					case w.AccountUpdateEventCh <- &accountUpdateEvent:
					}

				} else if msg[2] == 'e' && msg[6] == 'e' {
					orderUpdateEvent := OrderUpdateEvent{}
					err := json.Unmarshal(msg, &orderUpdateEvent)
					if err != nil {
						logger.Debugf("Unmarshal OrderUpdateEvent error %v %s", err, msg)
						continue
					}
					select {
					case <-ctx.Done():
						return
					case <-w.done:
						return
					//case <-time.After(time.Millisecond):
					//	logger.Warn("executionReport TO OUTPUT CH TIME OUT IN 1MS")
					case w.OrderUpdateEventCh <- &orderUpdateEvent:
					}
				} else if msg[2] == 'e' && msg[6] == 'b' {
					continue
				} else {
					logger.Debugf("OTHER MSG %s", msg)
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
			return nil, fmt.Errorf("PARSE PROXY %v", err)
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

func (w *UserWebsocket) start(ctx context.Context, urlStr string, proxy string) {
	logger.Debugf("BNSPOT USER WS %s", urlStr)

	ctx, cancel := context.WithCancel(ctx)
	var internalCtx context.Context
	var internalCancel context.CancelFunc

	defer func() {
		w.Stop()
		cancel()
		logger.Debugf("EXIT start")
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
				logger.Debugf("BNSPOT RECONNECT ERROR %v", err)
				internalCancel()
				return
			}
			go w.startRead(ctx, conn)
			go w.maintainHeartbeat(internalCtx, conn)
		}
	}
}

func (w *UserWebsocket) maintainHeartbeat(ctx context.Context, conn *websocket.Conn) {

	defer func() {
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
		case <-timer.C:
			logger.Warnf("BNSPOT USER WS TIMEOUT, NO TRAFFIC IN 15M, RESTART")
			go w.restart()
			return
		case <-trafficCh:
			timer.Reset(time.Minute * 15)
		case <-ctx.Done():
			return
		case <-w.done:
			return
		}
	}

}

func (w *UserWebsocket) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.stopped {
		w.stopped = true
		close(w.done)
		logger.Infof("BNSPOT USER WS STOPPED")
	}
}

func (w *UserWebsocket) restart() {
	logger.Debugf("BNSPOT USER WS RESTART")
	select {
	case <-w.done:
		return
	default:
	}
	select {
	case <-time.After(time.Second):
		logger.Debugf("BNSPOT NIL TO RECONNECT CH TIMEOUT IN 1S, EXIT WS")
		w.Stop()
		break
	case w.reconnectCh <- nil:
		break
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
		OrderUpdateEventCh:   make(chan *OrderUpdateEvent, 10),
		AccountUpdateEventCh: make(chan *AccountUpdateEvent, 10),
		messageCh:            make(chan []byte, 10),
		stopped:              false,
		mu:                   sync.Mutex{},
	}
	go func(ctx context.Context, ws *UserWebsocket, listenKey ListenKey) {
		timer := time.NewTimer(time.Minute * 20)
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
					logger.Debugf("api.SendAuthenticatedHTTPRequest error %v", err)
					ws.Stop()
					return
				}
				timer.Reset(time.Minute * 20)
			}
		}
	}(ctx, &ws, listenKey)
	go ws.start(ctx, wsUrl, proxy)
	go ws.startDataHandler(ctx, 1)
	go ws.startDataHandler(ctx, 2)
	go ws.startDataHandler(ctx, 3)
	go ws.startDataHandler(ctx, 4)
	ws.reconnectCh <- nil
	return &ws, nil
}
