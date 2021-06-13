package kucoin_coinfuture

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
	PositionCh  chan *WSPosition
	BalanceCh   chan *WsBalanceEvent
	RestartCh   chan interface{}
	writeCh     chan interface{}
	done        chan interface{}
	reconnectCh chan interface{}
	stopped     int32
	topicCh     chan string
	symbols     []string
	api         *API
	proxy       string
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
					logger.Warnf("json.Marshal error %v", err)
					continue
				}
			}
			err = conn.SetWriteDeadline(time.Now().Add(300 * time.Millisecond))
			if err != nil {
				logger.Debugf("conn.SetWriteDeadline error %v", err)
				w.restart()
				return
			}
			//logger.Debugf("%s", msgBytes)
			err = conn.WriteMessage(websocket.TextMessage, msgBytes)
			if err != nil {
				logger.Warnf("conn.WriteMessage %s error %v", string(msgBytes), err)
				w.restart()
				return
			}
		}
	}
}

func (w *UserWebsocket) readLoop(conn *websocket.Conn, pingInterval time.Duration) {
	logger.Debugf("START readLoop")
	defer logger.Debugf("EXIT readLoop")
	//logSilentTime := time.Now()
	pingInterval *= 10
	for {
		err := conn.SetReadDeadline(time.Now().Add(pingInterval))
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
		//logger.Debugf("%s", msg)
		select {
		case w.messageCh <- msg:
		default:
			logger.Debugf("w.messageCh <- msg failed, ch len %d", len(w.messageCh))
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
				logger.Debugf("json.Unmarshal error %v %s", err, msg)
				continue
			}
			if len(msg) < 128 {
				if wsCap.Type == "ack" {
					logger.Debugf("SUB SUCCESS %s", wsCap.ID)
					select {
					case w.topicCh <- wsCap.ID:
					default:
						logger.Debugf("w.topicCh <- wsCap.ID failed, ch len %d", len(w.topicCh))
					}
				} else if wsCap.Type == "pong" || wsCap.Type == "welcome" {
				} else if wsCap.Topic == "" {
					logger.Debugf("other msg %s", msg)
				}
			}
			splits := strings.Split(wsCap.Topic, ":")
			if len(splits) == 0 || splits[0] == "" {
				continue
			}
			switch splits[0] {
			case "/contractMarket/tradeOrders":
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
				case w.OrderCh <- &order:
				}
				select {
				case w.topicCh <- wsCap.Topic:
				default:
				}
				break
			case "/contract/position":
				if len(splits) != 2 {
					continue
				}
				position := WSPosition{}
				err = json.Unmarshal(wsCap.Data, &position)
				if err != nil {
					logger.Debugf("Unmarshal WS Position error %v %s", err, msg)
					continue
				}
				position.Symbol = splits[1]
				select {
				case <-ctx.Done():
					return
				case <-w.done:
					return
				case w.PositionCh <- &position:
				}
				select {
				case w.topicCh <- wsCap.Topic:
				default:
				}
				break
			case "/contractAccount/wallet":
				balance := WsBalanceEvent{}
				err = json.Unmarshal(wsCap.Data, &balance)
				if err != nil {
					logger.Debugf("Unmarshal Balance event error %v %s", err, msg)
					continue
				}
				select {
				case <-ctx.Done():
					return
				case <-w.done:
					return
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
		logger.Debugf("reconnect %s, %d retries", wsUrl, counter)
	}

	var dialer *websocket.Dialer

	if proxy != "" {
		proxyUrl, err := url.Parse(proxy)
		if err != nil {
			return nil, fmt.Errorf("url.Parse %s error %v", proxy, err)
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

func (w *UserWebsocket) mainLoop(ctx context.Context, api *API, symbols []string, proxy string) {
	logger.Debugf("START mainLoop")
	defer logger.Debugf("EXIT mainLoop")

	ctx, cancel := context.WithCancel(ctx)
	var internalCtx context.Context
	var internalCancel context.CancelFunc

	topics := []string{"/contractMarket/tradeOrders", "/contractAccount/wallet"}
	for _, symbol := range symbols {
		topics = append(topics, fmt.Sprintf("/contract/position:%s", symbol))
	}

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
				logger.Debugf("api.GetPrivateConnectToken error %v", err)
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

			logger.Debugf("%s", urlStr)

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
		case <-w.done:
			return
		case topic := <-w.topicCh:
			if _, ok := topicUpdatedTimes[topic]; ok {
				//logger.Debugf("TOPIC %s add 1 hour", topic)
				topicUpdatedTimes[topic] = time.Now().Add(time.Hour)
			}
			break
		case <-pingTimer.C:
			pingTimer.Reset(pingInterval / 2)
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Millisecond):
				logger.Debug("w.writeCh <- Ping timeout in 1ms")
			case w.writeCh <- Ping{
				ID:   fmt.Sprintf("%d", time.Now().UnixNano()/1000000),
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
						logger.Debugf("w.writeCh <- SubscribeMsg timeout in 1ms %s", topic)
						break
					case w.writeCh <- SubscribeMsg{
						ID:             topic,
						Type:           "subscribe",
						Topic:          topic,
						PrivateChannel: true,
						Response:       true,
					}:
						topicUpdatedTimes[topic] = time.Now().Add(topicCheckInterval * time.Duration(len(topics)*2))
						break loop
					default:
						logger.Debugf("w.writeCh <- SubscribeMsg failed, ch len %d", len(w.writeCh))
					}
				}
			}
			topicCheckTimer.Reset(topicCheckInterval)
			break
		}
	}

}

func (w *UserWebsocket) Stop() {
	if atomic.LoadInt32(&w.stopped) == 0 {
		atomic.StoreInt32(&w.stopped, 1)
		close(w.done)
		logger.Infof("stopped")
	}
}

func (w *UserWebsocket) restart() {
	select {
	case w.RestartCh <- nil:
	default:
		logger.Debugf("nil to RestartCh failed")
	}
	select {
	case <-w.done:
	case <-time.After(time.Millisecond):
		logger.Debugf("nil to reconnectCh timeout in 1ms, stop ws")
		w.Stop()
	case w.reconnectCh <- nil:
		logger.Debugf("ws restart")
	}
}

func (w *UserWebsocket) Done() chan interface{} {
	return w.done
}

func NewUserWebsocket(
	ctx context.Context,
	api *API,
	symbols []string,
	proxy string,
) *UserWebsocket {
	ws := UserWebsocket{
		done:        make(chan interface{}),
		reconnectCh: make(chan interface{}),
		OrderCh:     make(chan *WSOrder, 10000),
		BalanceCh:   make(chan *WsBalanceEvent, 10000),
		PositionCh:  make(chan *WSPosition, 100),
		RestartCh:   make(chan interface{}, 100),
		messageCh:   make(chan []byte, 10000),
		writeCh:     make(chan interface{}, 100),
		topicCh:     make(chan string, 100),
		stopped:     0,
	}
	go ws.mainLoop(ctx, api, symbols, proxy)
	go ws.dataHandleLoop(ctx)
	ws.reconnectCh <- nil
	return &ws
}
