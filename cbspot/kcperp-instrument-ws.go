package cbspot

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

type InstrumentWebsocket struct {
	messageCh     chan []byte
	MarkPriceCh   chan *MarkPrice
	writeCh       chan interface{}
	done          chan interface{}
	reconnectCh   chan interface{}
	topicCh       chan string
}

func (w *InstrumentWebsocket) startWrite(ctx context.Context, conn *websocket.Conn) {
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

func (w *InstrumentWebsocket) startRead(conn *websocket.Conn) {
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

func (w *InstrumentWebsocket) readAll(r io.Reader) ([]byte, error) {
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

func (w *InstrumentWebsocket) startDataHandler(ctx context.Context) {
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
				logger.Debugf("Unmarshal error %v %s", err, msg)
				continue
			}
			splits := strings.Split(wsCap.Topic, ":")
			if len(splits) < 2 || splits[1] == "" {
				//logger.Debugf("OTHER MSG %s", msg)
				continue
			}
			switch wsCap.Subject {
			case "mark.index.price":
				mp := MarkPrice{}
				err = json.Unmarshal(wsCap.Data, &mp)
				if err != nil {
					logger.Debugf("Unmarshal wsOrder error %v %s", err, msg)
					continue
				}
				mp.Symbol = splits[1]
				select {
				case <-ctx.Done():
					return
				case <-w.done:
					return
				case <-time.After(time.Millisecond):
					logger.Warn("KCPERP WS MARK PRICE TO OUTPUT CH TIME OUT IN 1MS")
				case w.MarkPriceCh <- &mp:
				}
				select {
				case w.topicCh <- wsCap.Topic:
				default:
				}
				break
			default:
			}
		}
	}
}

func (w *InstrumentWebsocket) reconnect(ctx context.Context, wsUrl string, proxy string, counter int64) (*websocket.Conn, error) {

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

func (w *InstrumentWebsocket) start(ctx context.Context, api *API, symbols []string, proxy string) {

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
			connectToken, err := api.GetPublicConnectToken(internalCtx)
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
			go w.maintainHeartbeat(internalCtx, conn, symbols, time.Millisecond*time.Duration(connectToken.InstanceServers[0].PingInterval))

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

func (w *InstrumentWebsocket) maintainHeartbeat(ctx context.Context, conn *websocket.Conn, symbols []string, pingInterval time.Duration) {

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
	symbolCheckInterval := time.Second * 15
	pingTimer := time.NewTimer(time.Second)
	symbolCheckTimer := time.NewTimer(time.Second)
	defer symbolCheckTimer.Stop()
	topicsUpdatedTimes := make(map[string]time.Time)
	for _, symbol := range symbols {
		topicsUpdatedTimes[fmt.Sprintf("/contract/instrument:%s", symbol)] = time.Unix(0, 0)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case symbol := <-w.topicCh:
			topicsUpdatedTimes[symbol] = time.Now()
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
		case <-symbolCheckTimer.C:
			for topic, updateTime := range topicsUpdatedTimes {
				if time.Now().Sub(updateTime) > topicTimeout {
					logger.Debugf("KCPERP SUBSCRIBE %s", topic)
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
						PrivateChannel: false,
						Response:       false,
					}:
						break
					}
				}
			}
			symbolCheckTimer.Reset(symbolCheckInterval)
			break
		case <-w.done:
			return
		}
	}

}

func (w *InstrumentWebsocket) Stop() {
	if _, ok := <-w.done; ok {
		close(w.done)
		logger.Infof("KCPERP MARK PRICE WS STOPPED")
	}
}

func (w *InstrumentWebsocket) restart() {
	logger.Infof("KCPERP WS RESTART")
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

func (w *InstrumentWebsocket) Done() chan interface{} {
	return w.done
}

func NewInstrumentWebsocket(
	ctx context.Context,
	api *API,
	symbols []string,
	proxy string,
	markPriceCh chan *MarkPrice,
) *InstrumentWebsocket {
	if markPriceCh == nil {
		markPriceCh = make(chan *MarkPrice, 100*len(symbols))
	}
	ws := InstrumentWebsocket{
		done:          make(chan interface{}),
		reconnectCh:   make(chan interface{}),
		MarkPriceCh:   markPriceCh,
		messageCh:     make(chan []byte, 100*len(symbols)),
		writeCh:       make(chan interface{}, 100*len(symbols)),
		topicCh:       make(chan string, 100*len(symbols)),
	}
	go ws.start(ctx, api, symbols, proxy)
	ws.reconnectCh <- nil
	return &ws
}
