package hbspot

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/gorilla/websocket"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"
)

const userWebsocketHost = "api-aws.huobi.pro"

type UserWebsocket struct {
	messageCh   chan []byte
	OrderCh     chan *WSOrder
	BalanceCh   chan *WSBalance
	RestartCh   chan interface{}
	Key         string
	Secret      string
	writeCh     chan interface{}
	done        chan interface{}
	reconnectCh chan interface{}
	topicCh     chan string
	loginCh     chan bool
	pingCh      chan []byte
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
					logger.Debugf("json.Marshal err %v", err)
					continue
				}
			}
			err = conn.SetWriteDeadline(time.Now().Add(300 * time.Millisecond))
			if err != nil {
				logger.Debugf("conn.SetWriteDeadline %s error %v", string(bytes), err)
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

func (w *UserWebsocket) readLoop(conn *websocket.Conn) {
	logger.Debugf("START readLoop")
	defer logger.Debugf("EXIT readLoop")
	totalCount := 0
	totalLen := 0
	logSilentTime := time.Now()
	for {
		err := conn.SetReadDeadline(time.Now().Add(time.Hour * 24))
		if err != nil {
			logger.Debugf("conn.SetReadDeadline error %v", err)
			w.restart()
			return
		}
		_, r, err := conn.NextReader()
		if err != nil {
			logger.Debugf("conn.NextReader error %v", err)
			w.restart()
			return
		}
		msg, err := w.readAll(r)
		if err != nil {
			logger.Debugf("w.readAll error %v", err)
			w.restart()
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
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case msg := <-w.messageCh:
			if len(msg) < 24 {
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("bad msg %s", msg)
					logSilentTime = time.Now().Add(time.Minute)
				}
				break
			}
			switch msg[11] {
			case 'p':
				switch msg[12] {
				case 'i':
					msg[12] = 'o'
					select {
					case w.pingCh <- msg:
					default:
						if time.Now().Sub(logSilentTime) > 0 {
							logger.Debugf("w.pingCh <- msg failed, ch len %d", len(w.pingCh))
							logSilentTime = time.Now().Add(time.Minute)
						}
					}
				case 'u':
					switch msg[23] {
					case 'a':
						balanceEvent := WSBalanceEvent{}
						err = json.Unmarshal(msg, &balanceEvent)
						if err != nil {
							logger.Debugf("json.Unmarshal(msg, &balanceEvent) error %v", err)
							continue
						}
						select {
						case w.BalanceCh <- &balanceEvent.Balance:
						default:
							if time.Now().Sub(logSilentTime) > 0 {
								logger.Debugf("w.BalancesCh <- &balanceEvent.Balance failed, ch len %d", len(w.BalanceCh))
								logSilentTime = time.Now().Add(time.Minute)
							}
						}
						select {
						case w.topicCh <- balanceEvent.Ch:
						default:
							if time.Now().Sub(logSilentTime) > 0 {
								logger.Debugf("w.topicCh <- balanceEvent.Ch failed, ch len %d", len(w.topicCh))
								logSilentTime = time.Now().Add(time.Minute)
							}
						}
					case 'o':
						orderEvent := WSOrderEvent{}
						err = json.Unmarshal(msg, &orderEvent)
						if err != nil {
							logger.Debugf("json.Unmarshal(msg, &orderEvent) error %v %s", err, msg)
							continue
						}
						select {
						case w.OrderCh <- &orderEvent.Order:
						default:
							if time.Now().Sub(logSilentTime) > 0 {
								logger.Debugf("w.OrdersCh <- &orderEvent.Order failed, ch len %d", len(w.OrderCh))
								logSilentTime = time.Now().Add(time.Minute)
							}
						}
						select {
						case w.topicCh <- orderEvent.Ch:
						default:
							if time.Now().Sub(logSilentTime) > 0 {
								logger.Debugf("w.topicCh <- orderEvent.Ch failed, ch len %d", len(w.topicCh))
								logSilentTime = time.Now().Add(time.Minute)
							}
						}
					default:
						if time.Now().Sub(logSilentTime) > 0 {
							logger.Debugf("other msg %s", msg)
							logSilentTime = time.Now().Add(time.Minute)
						}
					}
				}
			case 'r':
				switch msg[33] {
				case 'a':
					select {
					case w.loginCh <- true:
					default:
						if time.Now().Sub(logSilentTime) > 0 {
							logger.Debugf("w.loginCh <- true failed, ch len %d", len(w.loginCh))
							logSilentTime = time.Now().Add(time.Minute)
						}
					}
				default:
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("other msg %s", msg)
						logSilentTime = time.Now().Add(time.Minute)
					}
				}
			case 's':
				subResp := WsCap{}
				err = json.Unmarshal(msg, &subResp)
				if err != nil {
					logger.Debugf("son.Unmarshal(msg, &subResp) error %v %s", err, msg)
					break
				}
				//logger.Debugf("SUB %s", msg)
				select {
				case w.topicCh <- subResp.Ch:
				default:
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("w.topicCh <- subResp.Ch failed, ch len %d", len(w.topicCh))
						logSilentTime = time.Now().Add(time.Minute)
					}
				}
			default:
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
			conn, err := w.reconnect(internalCtx, fmt.Sprintf("wss://%s/ws/v2", userWebsocketHost), proxy, 0)
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

	topics := make([]string, 0)
	topics = append(topics, "accounts.update#2")
	for _, symbol := range symbols {
		topics = append(topics, fmt.Sprintf("orders#%s", symbol))
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
		case <-loginTimer.C:
			if !login {
				timestamp := time.Now().UTC().Format("2006-01-02T15:04:05")
				values := url.Values{}
				values.Set("accessKey", w.Key)
				values.Set("signatureMethod", "HmacSHA256")
				values.Set("signatureVersion", "2.1")
				values.Set("timestamp", timestamp)
				payload := fmt.Sprintf("%s\n%s\n%s\n%s", "GET", userWebsocketHost, "/ws/v2", values.Encode())
				hmac := common.GetHMAC(common.HashSHA256, []byte(payload), []byte(w.Secret))
				select {
				case w.writeCh <- AuthenticationParam{
					Action: "req",
					Ch:     "auth",
					Params: AuthenticationParams{
						AuthType:         "api",
						AccessKey:        w.Key,
						SignatureMethod:  "HmacSHA256",
						SignatureVersion: "2.1",
						Timestamp:        timestamp,
						Signature:        common.Base64Encode(hmac),
					},
				}:
				default:
					logger.Debugf("w.writeCh <- AuthenticationParam failed, ch len %d", len(w.writeCh))
				}
			}
			loginTimer.Reset(time.Minute)
		case topic := <-w.topicCh:
			topic = strings.ToLower(topic)
			if _, ok := topicUpdatedTimes[topic]; ok {
				topicUpdatedTimes[topic] = time.Now().Add(time.Hour * 8)
			}
		case msg := <-w.pingCh:
			select {
			case w.writeCh <- msg:
			default:
				logger.Debugf("w.writeCh <- msg failed, ch len %d", len(w.writeCh))
			}
			break
		case login = <-w.loginCh:
			break
		case <-topicCheckTimer.C:
			if login {
			loop:
				for _, topic := range topics {
					if time.Now().Sub(topicUpdatedTimes[topic]) > topicTimeout {
						logger.Debugf("SUBSCRIBE %s", topic)
						select {
						case w.writeCh <- AccountSubParam{
							Action: "sub",
							Ch:     topic,
						}:
							topicUpdatedTimes[topic] = time.Now().Add(topicCheckInterval * time.Duration(len(symbols)*2))
							break loop
						default:
							logger.Debugf("w.writeCh <- AccountSubParam failed, ch len %d", len(w.writeCh))
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

func NewUserWebsocket(
	ctx context.Context,
	key, secret string,
	symbols []string,
	proxy string,
) *UserWebsocket {
	ws := UserWebsocket{
		Key:         key,
		Secret:      secret,
		done:        make(chan interface{}),
		reconnectCh: make(chan interface{}, 100),
		OrderCh:     make(chan *WSOrder, 100*len(symbols)),
		BalanceCh:   make(chan *WSBalance, 100*len(symbols)),
		RestartCh:   make(chan interface{}, 100),
		messageCh:   make(chan []byte, 100*len(symbols)),
		writeCh:     make(chan interface{}, 100*len(symbols)),
		topicCh:     make(chan string, 100*len(symbols)),
		pingCh:      make(chan []byte, 100),
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
