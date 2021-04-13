package kcperp

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
	"time"
)

type UserWebsocket struct {
	messageCh   chan []byte
	OrderCh     chan *WSOrder
	PositionCh  chan *WSPosition
	BalanceCh   chan *WsBalanceEvent
	writeCh     chan interface{}
	done        chan interface{}
	reconnectCh chan interface{}
	topicCh     chan string
}

func (w *UserWebsocket) startWrite(ctx context.Context, conn *websocket.Conn) {
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
					logger.Warnf("Marshal err %v", err)
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
				logger.Warnf("WriteMessage %s error %v", string(bytes), err)
				w.restart()
				return
			}
		}
	}
}

func (w *UserWebsocket) startRead(conn *websocket.Conn) {
	totalCount := 0
	totalLen := 0
	for {
		err := conn.SetReadDeadline(time.Now().Add(time.Minute))
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
		totalCount += 1
		totalLen += len(msg)
		if totalCount > 1000000 {
			logger.Debugf("AVERAGE MESSAGE LENGTH %d/%d = %d", totalLen, totalCount, totalLen/totalCount)
			totalLen = 0
			totalCount = 0
		}
		select {
		case <-time.After(time.Millisecond):
			logger.Debug("KCPERP DEPTH20 MSG TO MESSAGE CH TIMEOUT IN 1MS")
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

func (w *UserWebsocket) startDataHandler(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case msg := <-w.messageCh:
			//logger.Debugf("PERP %s", msg)
			var wsCap WsCap
			err := json.Unmarshal(msg, &wsCap)
			if err != nil {
				logger.Debugf("Unmarshal error %v %s", err, msg)
				continue
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
				case <-time.After(time.Millisecond):
					logger.Warn("KCPERP WS ORDER TO OUTPUT CH TIME OUT IN 1MS")
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
				case <-time.After(time.Millisecond):
					logger.Warn("KCPERP WS POSITION TO OUTPUT CH TIME OUT IN 1MS")
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
				case <-time.After(time.Millisecond):
					logger.Warn("KCPERP WS BALANCE EVENT TO OUTPUT CH TIME OUT IN 1MS")
				case w.BalanceCh <- &balance:
				}
				select {
				case w.topicCh <- wsCap.Topic:
				default:
				}
				break
			default:
				if wsCap.Type == "welcome" {
					logger.Debugf("WELCOME %s", wsCap.ID)
				} else if wsCap.Type == "pong" {
					logger.Debugf("PONG %s", wsCap.ID)
				} else {
					logger.Debugf("KCPERP OTHER MSG %s", msg)
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

func (w *UserWebsocket) start(ctx context.Context, api *API, symbols []string, proxy string) {

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
			return
		case <-w.reconnectCh:
			reconnectTimer.Reset(time.Second * 15)
		case <-reconnectTimer.C:
			if internalCancel != nil {
				internalCancel()
			}
			internalCtx, internalCancel = context.WithCancel(ctx)
			connectToken, err := api.GetPrivateConnectToken(internalCtx)
			if err != nil {
				logger.Fatalf("GetPublicConnectToken error %v", err)
				return
			}
			if len(connectToken.InstanceServers) == 0 {
				if err != nil {
					logger.Fatalf("No InstanceServers %v", connectToken)
					return
				}
			}
			urlStr := connectToken.InstanceServers[0].Endpoint + "?token=" + connectToken.Token
			//logger.Debugf("KCPERP DEPTH50 WS %s", urlStr)

			conn, err := w.reconnect(internalCtx, urlStr, proxy, 0)
			if err != nil {
				logger.Fatalf("RECONNECT ERROR %v", err)
				return
			}
			go w.startRead(conn)
			go w.startWrite(ctx, conn)
			go w.maintainHeartbeat(internalCtx, conn, topics, time.Duration(connectToken.InstanceServers[0].PingInterval)*time.Millisecond)

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

func (w *UserWebsocket) maintainHeartbeat(ctx context.Context, conn *websocket.Conn, topics []string, pingInterval time.Duration) {

	defer func() {
		err := conn.Close()
		if err != nil {
			logger.Debugf("conn.Close() ERROR %v", err)
		}
	}()

	conn.SetPingHandler(func(msg string) error {
		logger.Debugf("KCPERP DEPTH20 WS PingHandler %s", msg)
		err := conn.WriteControl(websocket.PongMessage, []byte(msg), time.Now().Add(time.Minute))
		if err != nil {
			go w.restart()
			return nil
		}
		return nil
	})

	topicTimeout := time.Minute
	topicCheckInterval := time.Minute * 5
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
			topicUpdatedTimes[topic] = time.Now()
		case <-pingTimer.C:
			pingTimer.Reset(pingInterval)
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Millisecond):
				logger.Debug("SEND PING TO WRITE TIMEOUT IN 1MS")
				break
			case w.writeCh <- Ping{
				ID:   fmt.Sprintf("%d", time.Now().Nanosecond()/1000000),
				Type: "ping",
			}:
				break
			}
			break
		case <-topicCheckTimer.C:
			for topic, updateTime := range topicUpdatedTimes {
				if time.Now().Sub(updateTime) > topicTimeout {
					select {
					case <-ctx.Done():
						return
					case <-time.After(time.Millisecond):
						logger.Debugf("SEND SUBSCRIBE %s TO WRITE TIMEOUT IN 1MS", topic)
						break
					case w.writeCh <- SubscribeMsg{
						ID:             topic,
						Type:           "subscribe",
						Topic:          topic,
						PrivateChannel: true,
						Response:       false,
					}:
						break
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
	if _, ok := <-w.done; ok {
		close(w.done)
		logger.Infof("BNSWAP MARK PRICE WS STOPPED")
	}
}

func (w *UserWebsocket) restart() {
	logger.Infof("BNSWAP WS RESTART")
	select {
	case <-w.done:
		return
	default:
	}
	timer := time.NewTimer(time.Millisecond)
	defer timer.Stop()
	select {
	case <-timer.C:
		logger.Fatal("NIL TO RECONNECT CH TIMEOUT IN 1MS, EXIT")
	case w.reconnectCh <- nil:
		return
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
		OrderCh:     make(chan *WSOrder, 100),
		BalanceCh:   make(chan *WsBalanceEvent, 100),
		PositionCh:  make(chan *WSPosition, 100),
		messageCh:   make(chan []byte, 10000),
		writeCh:     make(chan interface{}, 100),
		topicCh:     make(chan string, 100),
	}
	go ws.start(ctx, api, symbols, proxy)
	ws.reconnectCh <- nil
	return &ws
}

