package kucoin_usdtspot

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
	messageCh   chan []byte
	OrderCh     chan *WSOrder
	BalanceCh   chan *WsBalance
	RestartCh   chan interface{}
	writeCh     chan interface{}
	done        chan interface{}
	reconnectCh chan interface{}
	topicCh     chan string
	stopped     int32
}

func (w *UserWebsocket) writeLoop(ctx context.Context, conn *websocket.Conn) {
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
					logger.Debugf("Marshal err %v", err)
					continue
				}
			}
			err = conn.SetWriteDeadline(time.Now().Add(time.Minute))
			if err != nil {
				logger.Debugf("conn.SetWriteDeadline error %v, restart ws", err)
				w.restart()
				return
			}
			err = conn.WriteMessage(websocket.TextMessage, bytes)
			if err != nil {
				logger.Debugf("conn.WriteMessage %s error %v", bytes, err)
				w.restart()
				return
			}
		}
	}
}

func (w *UserWebsocket) readLoop(conn *websocket.Conn, pingInterval time.Duration) {
	logger.Debugf("START readLoop")
	defer logger.Debugf("EXIT readLoop")
	logSilentTime := time.Time{}
	pingInterval *= 10
	for {
		err := conn.SetReadDeadline(time.Now().Add(pingInterval))
		if err != nil {
			logger.Debugf("conn.SetReadDeadline error %v, restart", err)
			w.restart()
			return
		}
		_, r, err := conn.NextReader()
		if err != nil {
			logger.Debugf("conn.NextReader error %v, restart", err)
			w.restart()
			return
		}
		msg, err := w.readAll(r)
		if err != nil {
			logger.Debugf("w.readAll error %v, restart", err)
			w.restart()
			return
		}
		select {
		case w.messageCh <- msg:
		default:
			if time.Now().Sub(logSilentTime) > 0{
				logSilentTime = time.Now().Add(time.Minute)
				logger.Debugf("w.messageCh <- msg failed, ch len %d", len(w.messageCh))
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

func (w *UserWebsocket) dataHandleLoop(ctx context.Context) {
	logger.Debugf("START dataHandleLoop")
	defer logger.Debugf("EXIT dataHandleLoop")
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case msg := <-w.messageCh:
			var wsCap WsCap
			err := json.Unmarshal(msg, &wsCap)
			if err != nil {
				logger.Debugf("json.Unmarshal(msg, &wsCap) error %v %s", err, msg)
				continue
			}
			if len(msg) < 128 {
				if wsCap.Type == "ack" {
					select {
					case w.topicCh <- wsCap.ID:
					default:
					}
				} else if wsCap.Type == "pong" || wsCap.Type == "welcome"{
				} else if wsCap.Topic == "" {
					logger.Debugf("other msg %s", msg)
				}
				continue
			}
			splits := strings.Split(wsCap.Topic, ":")
			if len(splits) == 0 {
				continue
			}
			switch splits[0] {
			case "/spotMarket/tradeOrders":
				order := WSOrder{}
				err = json.Unmarshal(wsCap.Data, &order)
				if err != nil {
					logger.Debugf("Unmarshal wsOrder error %v %s", err, msg)
					continue
				}
				select {
				case <-ctx.Done():
					return
				case <-w.done:
					return
				case <-time.After(time.Second):
					logger.Debugf("w.OrdersCh <- &order timeout in 1s")
				case w.OrderCh <- &order:
				}
				select {
				case w.topicCh <- strings.Split(wsCap.Topic, ":")[0]:
				default:
				}
				break
			case "/account/balance":
				//logger.Debugf("%s", wsCap.Data)
				balance := WsBalance{}
				err = json.Unmarshal(wsCap.Data, &balance)
				if err != nil {
					logger.Debugf("Unmarshal wsOrder error %v %s", err, msg)
					continue
				}
				select {
				case <-ctx.Done():
					return
				case <-w.done:
					return
				case <-time.After(time.Second):
					logger.Debugf("w.BalancesCh <- &balance timeout in 1s")
				case w.BalanceCh <- &balance:
				}
				select {
				case w.topicCh <- wsCap.Topic:
				default:
				}
				break
			default:
				logger.Debugf("other msg %s", msg)
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
		logger.Debugf("dialer.DialContext error %v", err)
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

func (w *UserWebsocket) mainLoop(ctx context.Context, api *API, topics []string, proxy string) {
	logger.Debugf("START mainLoop")
	defer logger.Debugf("EXIT mainLoop")

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
			connectToken, err := api.GetPrivateConnectToken(internalCtx)
			if err != nil {
				logger.Debugf("api.GetPrivateConnectToken error %v, stop ws", err)
				internalCancel()
				w.Stop()
				return
			}
			if len(connectToken.InstanceServers) == 0 {
				logger.Debugf("no instanceServers %v, stop ws", connectToken)
				internalCancel()
				w.Stop()
				return
			}
			urlStr := connectToken.InstanceServers[0].Endpoint + "?token=" + connectToken.Token

			conn, err := w.reconnect(internalCtx, urlStr, proxy, 0)
			if err != nil {
				logger.Debugf("w.reconnect error %v", err)
				internalCancel()
				w.Stop()
				return
			}
			go w.readLoop(conn, time.Duration(connectToken.InstanceServers[0].PingInterval)*time.Millisecond)
			go w.writeLoop(internalCtx, conn)
			go w.heartbeatLoop(internalCtx, conn, topics, time.Duration(connectToken.InstanceServers[0].PingInterval)*time.Millisecond)
		}
	}
}

func (w *UserWebsocket) heartbeatLoop(ctx context.Context, conn *websocket.Conn, topics []string, pingInterval time.Duration) {

	logger.Debugf("START heartbeatLoop")
	defer func() {
		logger.Debugf("EXIT heartbeatLoop")
		err := conn.Close()
		if err != nil {
			logger.Debugf("conn.Close() error %v", err)
		}
	}()

	topicTimeout := time.Minute
	topicCheckInterval := time.Second
	pingTimer := time.NewTimer(time.Second)
	topicCheckTimer := time.NewTimer(time.Second)
	defer topicCheckTimer.Stop()
	topicUpdatedTimes := make(map[string]time.Time)
	for _, topic := range topics {
		topicUpdatedTimes[topic] = time.Unix(0, 0)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case topic := <-w.topicCh:
			if _, ok := topicUpdatedTimes[topic]; ok {
				//logger.Debugf("TOPIC %s add 4 hour", topic)
				topicUpdatedTimes[topic] = time.Now().Add(time.Hour * 4)
			}
		case <-pingTimer.C:
			pingTimer.Reset(pingInterval/2)
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Millisecond):
				logger.Debugf("w.writeCh <- Ping timeout in 1ms")
			case w.writeCh <- Ping{
				ID:   fmt.Sprintf("%d", time.Now().Nanosecond()/1000000),
				Type: "ping",
			}:
			}
			break
		case <-topicCheckTimer.C:
		loop:
			for topic, updateTime := range topicUpdatedTimes {
				if time.Now().Sub(updateTime) > topicTimeout {
					logger.Debugf("SUBSCRIBE %s", topic)
					select {
					case <-ctx.Done():
						return
					case <-time.After(time.Millisecond):
						logger.Debugf("w.writeCh <- SubscribeMsg %s timeout in 1ms", topic)
					case w.writeCh <- SubscribeMsg{
						ID:             topic,
						Type:           "subscribe",
						Topic:          topic,
						PrivateChannel: true,
						Response:       true,
					}:
						topicUpdatedTimes[topic] = time.Now().Add(topicCheckInterval * time.Duration(len(topics)*2))
						break loop
					}
				}
			}
			topicCheckTimer.Reset(topicCheckInterval)
			break
		case <-w.done:
			return
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
	logger.Debugf("BNSWAP WS RESTART")
	select {
	case w.RestartCh <- nil:
	default:
		logger.Debugf("w.RestartCh <- nil failed")
	}
	select {
	case <-w.done:
		return
	case <-time.After(time.Second):
		logger.Debugf("w.reconnectCh <- nil timeout in 1s, stop ws")
		w.Stop()
	case w.reconnectCh <- nil:
	}
}

func (w *UserWebsocket) Done() chan interface{} {
	return w.done
}

func NewUserWebsocket(
	ctx context.Context,
	api *API,
	proxy string,
) *UserWebsocket {
	ws := UserWebsocket{
		done:        make(chan interface{}),
		reconnectCh: make(chan interface{}),
		OrderCh:     make(chan *WSOrder, 100),
		BalanceCh:   make(chan *WsBalance, 100),
		messageCh:   make(chan []byte, 10000),
		RestartCh:   make(chan interface{}, 100),
		writeCh:     make(chan interface{}, 100),
		topicCh:     make(chan string, 100),
		stopped:     0,
	}
	go ws.mainLoop(ctx, api, []string{"/account/balance", "/spotMarket/tradeOrders"}, proxy)
	go ws.dataHandleLoop(ctx)
	ws.reconnectCh <- nil
	return &ws
}
