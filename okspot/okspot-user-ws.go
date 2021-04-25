package okspot

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/gorilla/websocket"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

type UserWebsocket struct {
	messageCh   chan []byte
	OrdersCh    chan []WSOrder
	BalancesCh  chan []Balance
	RestartCh   chan interface{}
	Key         string
	Secret      string
	Passphrase  string
	writeCh     chan interface{}
	done        chan interface{}
	reconnectCh chan interface{}
	topicCh     chan string
	loginCh     chan bool
	pongCh      chan []byte
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
					logger.Debugf("json.Marshal err %v", err)
					continue
				}
			}
			err = conn.SetWriteDeadline(time.Now().Add(300 * time.Millisecond))
			if err != nil {
				logger.Debugf("conn.SetWriteDeadline %s error %v", string(msgBytes), err)
				w.restart()
				return
			}

			err = conn.WriteMessage(websocket.TextMessage, msgBytes)
			if err != nil {
				logger.Debugf("conn.WriteMessage %s error %v", string(msgBytes), err)
				w.restart()
				return
			}
		}
	}
}

func (w *UserWebsocket) parseBinaryResponse(resp []byte) ([]byte, error) {
	var standardMessage []byte
	var err error
	// Detect GZIP
	if resp[0] == 31 && resp[1] == 139 {
		b := bytes.NewReader(resp)
		var gReader *gzip.Reader
		gReader, err = gzip.NewReader(b)
		if err != nil {
			return standardMessage, err
		}
		standardMessage, err = w.readAll(gReader)
		if err != nil {
			return standardMessage, err
		}
		err = gReader.Close()
		if err != nil {
			return standardMessage, err
		}
	} else {
		reader := flate.NewReader(bytes.NewReader(resp))
		standardMessage, err = w.readAll(reader)
		if err != nil {
			return standardMessage, err
		}
		err = reader.Close()
		if err != nil {
			return standardMessage, err
		}
	}
	return standardMessage, nil
}

func (w *UserWebsocket) readLoop(conn *websocket.Conn) {
	logger.Debugf("START readLoop")
	defer logger.Debugf("EXIT readLoop")
	totalCount := 0
	totalLen := 0
	logSilentTime := time.Now()
	var msg []byte
	var mType int
	var resp []byte
	var err error
	for {
		err = conn.SetReadDeadline(time.Now().Add(time.Hour * 24))
		if err != nil {
			logger.Debugf("conn.SetReadDeadline error %v", err)
			w.restart()
			return
		}
		mType, resp, err = conn.ReadMessage()
		if err != nil {
			logger.Debugf("conn.ReadMessage error %v", err)
			w.restart()
			return
		}
		switch mType {
		case websocket.TextMessage:
			msg = resp
		case websocket.BinaryMessage:
			msg, err = w.parseBinaryResponse(resp)
			if err != nil {
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("w.parseBinaryResponse error %v", err)
					logSilentTime = time.Now().Add(time.Minute)
				}
				continue
			}
		}
		totalCount += 1
		totalLen += len(msg)
		if totalCount > 1000000 {
			logger.Debugf("AVERAGE MESSAGE LENGTH %d/%d = %d", totalLen, totalCount, totalLen/totalCount)
			totalLen = 0
			totalCount = 0
		}
		select {
		case w.messageCh <- msg:
		default:
			if time.Now().Sub(logSilentTime) > 0 {
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
	logSilentTime := time.Now()
	var err error
	var commonCap CommonCapture
	var loginEvent LoginEvent
	var subscribeEvent SubscribeEvent
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case msg := <-w.messageCh:
			if len(msg) == 4 && msg[0] == 'p' {
				select {
				case w.pongCh <- msg:
				default:
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("w.pongCh <- msg failed ch len %d", len(w.pongCh))
						logSilentTime = time.Now().Add(time.Minute)
					}
				}
				continue
			}

			err = json.Unmarshal(msg, &commonCap)
			if err != nil {
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("json.Unmarshal(msg, &commonCap) error %v %s", err, msg)
					logSilentTime = time.Now().Add(time.Minute)
				}
				continue
			}

			if commonCap.Table != nil && commonCap.Data != nil {
				switch w.getChannelWithoutOrderType(*commonCap.Table) {
				case okspotWsAccount:
					var balances []Balance
					err = json.Unmarshal(*commonCap.Data, &balances)
					if err != nil {
						if time.Now().Sub(logSilentTime) > 0 {
							logger.Debugf("json.Unmarshal(*commonCap.Data, &balances) error %v %s", err, msg)
							logSilentTime = time.Now().Add(time.Minute)
						}
						continue
					}
					select {
					case w.BalancesCh <- balances:
					default:
						if time.Now().Sub(logSilentTime) > 0 {
							logger.Debugf("w.BalancesCh <- balances failed ch len %d", len(w.BalancesCh))
							logSilentTime = time.Now().Add(time.Minute)
						}
					}
					for _, balance := range balances {
						select {
						case w.topicCh <- *commonCap.Table + ":" + balance.Currency:
						default:
							if time.Now().Sub(logSilentTime) > 0 {
								logger.Debugf("w.topicCh <- %s failed ch len %d", *commonCap.Table+":"+balance.Currency, len(w.topicCh))
								logSilentTime = time.Now().Add(time.Minute)
							}
						}
					}
					continue
				case okspotWsOrder:
					var orders []WSOrder
					err = json.Unmarshal(*commonCap.Data, &orders)
					if err != nil {
						if time.Now().Sub(logSilentTime) > 0 {
							logger.Debugf("json.Unmarshal(*commonCap.Data, &orders) error %v %s", err, msg)
							logSilentTime = time.Now().Add(time.Minute)
						}
						continue
					}
					select {
					case w.OrdersCh <- orders:
					default:
						if time.Now().Sub(logSilentTime) > 0 {
							logger.Debugf("w.OrdersCh <- orders failed ch len %d", len(w.OrdersCh))
							logSilentTime = time.Now().Add(time.Minute)
						}
					}
					for _, order := range orders {
						select {
						case w.topicCh <- *commonCap.Table + ":" + order.Symbol:
						default:
							if time.Now().Sub(logSilentTime) > 0 {
								logger.Debugf("w.topicCh <- %s failed ch len %d", *commonCap.Table+":"+order.Symbol, len(w.topicCh))
								logSilentTime = time.Now().Add(time.Minute)
							}
						}
					}
				default:
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("other table msg %s", msg)
						logSilentTime = time.Now().Add(time.Minute)
					}
				}
			} else if commonCap.Event != nil {
				switch *commonCap.Event {
				case "login":
					err = json.Unmarshal(msg, &loginEvent)
					if err != nil {
						if time.Now().Sub(logSilentTime) > 0 {
							logger.Debugf("json.Unmarshal(msg, &loginEvent) error %v %s", err, msg)
							logSilentTime = time.Now().Add(time.Minute)
						}
					}
					if loginEvent.Success {
						select {
						case w.loginCh <- true:
						default:
							logger.Debugf("send login true failed, stop ws")
							w.Stop()
							return
						}
					} else {
						logger.Debugf("login failed, stop ws")
						w.Stop()
						return
					}
					continue
				case "error":
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("error msg  %s", msg)
						logSilentTime = time.Now().Add(time.Minute)
					}
					continue
				case "subscribe":
					err = json.Unmarshal(msg, &subscribeEvent)
					if err != nil {
						if time.Now().Sub(logSilentTime) > 0 {
							logger.Debugf("json.Unmarshal(msg, &subscribeEvent) error %v %s", err, msg)
							logSilentTime = time.Now().Add(time.Minute)
						}
					}
					select {
					case w.topicCh <- subscribeEvent.Channel:
					default:
						if time.Now().Sub(logSilentTime) > 0 {
							logger.Debugf("w.topicCh <- subscribeEvent.Channel failed ch len %d", len(w.topicCh))
							logSilentTime = time.Now().Add(time.Minute)
						}
					}
					continue
				default:
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("other event msg %s", msg)
						logSilentTime = time.Now().Add(time.Minute)
					}
				}
			} else {
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("other msg %s", msg)
					logSilentTime = time.Now().Add(time.Minute)
				}
			}
		}
	}
}

func (w *UserWebsocket) reconnect(ctx context.Context, wsUrl string, proxy string, counter int64) (*websocket.Conn, error) {

	if counter != 0 {
		logger.Debugf("reconnect %s, %d retires", wsUrl, counter)
	}

	var dialer *websocket.Dialer

	if proxy != "" {
		proxyUrl, err := url.Parse(proxy)
		if err != nil {
			return nil, fmt.Errorf("url.Parse(proxy) %v", err)
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

func (w *UserWebsocket) mainLoop(ctx context.Context, symbols []string, proxy string) {
	logger.Debugf("START mainLoop")

	ctx, cancel := context.WithCancel(ctx)
	var internalCtx context.Context
	var internalCancel context.CancelFunc

	defer func() {
		logger.Debugf("EXIT mainLoop")
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
			conn, err := w.reconnect(internalCtx, "wss://real.okex.com:8443/ws/v3", proxy, 0)
			if err != nil {
				logger.Debugf("w.reconnect error %v, stop ws", err)
				if internalCancel != nil {
					internalCancel()
				}
				w.Stop()
				return
			}
			go w.readLoop(conn)
			go w.writeLoop(internalCtx, conn)
			go w.heartbeatLoop(internalCtx, conn, symbols)
		}
	}
}

func (w *UserWebsocket) heartbeatLoop(ctx context.Context, conn *websocket.Conn, symbols []string) {
	logger.Debugf("START heartbeatLoop")
	defer func() {
		logger.Debugf("EXIT heartbeatLoop")
		err := conn.Close()
		if err != nil {
			logger.Debugf("conn.Close() ERROR %v", err)
		}
	}()

	topicTimeout := time.Hour * 24
	topicCheckInterval := time.Second
	topicCheckTimer := time.NewTimer(time.Second)
	defer topicCheckTimer.Stop()

	login := false
	loginTimer := time.NewTimer(time.Second)
	defer loginTimer.Stop()

	trafficTimeout := time.NewTimer(time.Minute * 5)
	pingTimer := time.NewTimer(time.Second * 15)
	defer trafficTimeout.Stop()
	defer pingTimer.Stop()

	topics := make([]string, 0)
	for _, symbol := range symbols {
		topics = append(topics, fmt.Sprintf("spot/account:%s", symbol[:len(symbol)-5]))
		topics = append(topics, fmt.Sprintf("spot/order:%s", symbol))
	}

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
		case <-trafficTimeout.C:
			logger.Debugf("traffic timeout in 30s, restart ws")
			w.restart()
			return
		case <-pingTimer.C:
			pingTimer.Reset(time.Second * 15)
			select {
			case w.writeCh <- []byte("ping"):
				break
			default:
				logger.Debugf("w.writeCh <- ping failed, ch len %d", len(w.writeCh))
			}
			break
		case <-loginTimer.C:
			if !login {
				unixTime := time.Now().UTC().Unix()
				signPath := "/users/self/verify"
				hmac := common.GetHMAC(common.HashSHA256,
					[]byte(strconv.FormatInt(unixTime, 10)+http.MethodGet+signPath),
					[]byte(w.Secret),
				)
				base64 := common.Base64Encode(hmac)
				select {
				case w.writeCh <- Subscription{
					Op: "login",
					Args: []string{
						w.Key,
						w.Passphrase,
						strconv.FormatInt(unixTime, 10),
						base64,
					},
				}:
					loginTimer.Reset(time.Minute)
				default:
					logger.Debugf("w.writeCh <- Subscription for login failed, ch len %d", len(w.writeCh))
					loginTimer.Reset(time.Second * 15)

				}
			} else {
				loginTimer.Reset(time.Minute * 10)
			}
			loginTimer.Reset(time.Minute)
		case topic := <-w.topicCh:
			pingTimer.Reset(time.Second * 15)
			trafficTimeout.Reset(time.Second * 30)
			if _, ok := topicUpdatedTimes[topic]; ok {
				topicUpdatedTimes[topic] = time.Now().Add(time.Hour * 8)
			}
		case <-w.pongCh:
			pingTimer.Reset(time.Second * 15)
			trafficTimeout.Reset(time.Second * 30)
			break
		case login = <-w.loginCh:
			break
		case <-topicCheckTimer.C:
			if login {
				args := make([]string, 0)
				for _, topic := range topics {
					if time.Now().Sub(topicUpdatedTimes[topic]) > topicTimeout {
						args = append(args, topic)
						topicUpdatedTimes[topic] = time.Now().Add(topicTimeout)
					}
				}
				if len(args) > 0 {
					for start := 0; start < len(args); start += 50 {
						end := start + 50
						if end > len(args) {
							end = len(args)
						}
						logger.Debugf("SUBSCRIBE %s", args[start:end])
						select {
						case w.writeCh <- Subscription{
							Op:   "subscribe",
							Args: args[start:end],
						}:
						default:
							logger.Debugf("w.writeCh <- Subscription failed, ch len %d", len(w.writeCh))
						}
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
		logger.Debugf("stopped")
	}
}

func (w *UserWebsocket) restart() {
	select {
	case w.RestartCh <- nil:
	default:
		logger.Debugf("w.RestartCh <- nil failed, ch len %d", len(w.RestartCh))
	}
	select {
	case w.reconnectCh <- nil:
		logger.Debugf("restart ws")
	default:
		logger.Debugf("w.reconnectCh <- nil failed, ch len %d", len(w.reconnectCh))
	}
}

func (w *UserWebsocket) Done() chan interface{} {
	return w.done
}

func (w *UserWebsocket) getChannelWithoutOrderType(table string) string {
	index := strings.Index(table, ":")
	// Some events do not contain a currency
	if index == -1 {
		return table
	}
	return table[:index]
}

func NewUserWebsocket(
	ctx context.Context,
	key, secret, passphrase string,
	symbols []string,
	proxy string,
) *UserWebsocket {
	ws := UserWebsocket{
		Key:         key,
		Secret:      secret,
		Passphrase:  passphrase,
		done:        make(chan interface{}),
		reconnectCh: make(chan interface{}, 100),
		OrdersCh:    make(chan []WSOrder, 100*len(symbols)),
		BalancesCh:  make(chan []Balance, 100*len(symbols)),
		RestartCh:   make(chan interface{}, 100),
		messageCh:   make(chan []byte, 100*len(symbols)),
		writeCh:     make(chan interface{}, 100*len(symbols)),
		topicCh:     make(chan string, 100*len(symbols)),
		pongCh:      make(chan []byte, 100),
		loginCh:     make(chan bool, 100),
		stopped:     0,
	}
	go ws.mainLoop(ctx, symbols, proxy)
	go ws.dataHandleLoop(ctx)
	go ws.dataHandleLoop(ctx)
	go ws.dataHandleLoop(ctx)
	go ws.dataHandleLoop(ctx)
	ws.reconnectCh <- nil
	return &ws
}
