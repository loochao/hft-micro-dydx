package hbcrossswap

import (
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
	"strings"
	"sync"
	"time"
)

type UserWebsocket struct {
	messageCh   chan []byte
	PositionCh  chan *WSPositions
	OrderCh     chan *WSOrder
	AccountCh   chan *WSAccounts
	RestartCh   chan interface{}
	Key         string
	Secret      string
	writeCh     chan interface{}
	done        chan interface{}
	reconnectCh chan interface{}
	topicCh     chan string
	loginCh     chan bool
	pingCh      chan []byte
	mu          sync.Mutex
	stopped     bool
}

func (w *UserWebsocket) startWrite(ctx context.Context, conn *websocket.Conn) {
	defer func() {
		logger.Debugf("EXIT startWrite")
	}()
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
				select {
				case <-ctx.Done():
					break
				default:
					w.restart()
				}
				return
			}

			err = conn.WriteMessage(websocket.TextMessage, bytes)

			if err != nil {
				logger.Warnf("WriteMessage %s error %v", string(bytes), err)
				select {
				case <-ctx.Done():
					break
				default:
					w.restart()
				}
				return
			}
		}
	}
}

func (w *UserWebsocket) startRead(ctx context.Context, conn *websocket.Conn) {
	defer func() {
		logger.Debugf("EXIT startRead")
	}()
	totalCount := 0
	totalLen := 0
	for {
		err := conn.SetReadDeadline(time.Now().Add(time.Minute))
		if err != nil {
			logger.Warnf("SetReadDeadline error %v", err)
			select {
			case <-ctx.Done():
				break
			default:
				w.restart()
			}
			return
		}
		_, r, err := conn.NextReader()
		if err != nil {
			logger.Warnf("NextReader error %v", err)
			select {
			case <-ctx.Done():
				break
			default:
				w.restart()
			}
			return
		}
		gr, err := gzip.NewReader(r)
		if err != nil {
			logger.Warnf("NewReader error %v", err)
			select {
			case <-ctx.Done():
				break
			default:
				w.restart()
			}
			return
		}
		msg, err := w.readAll(gr)
		if err != nil {
			logger.Warnf("readAll error %v", err)
			select {
			case <-ctx.Done():
				break
			default:
				w.restart()
			}
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
			logger.Debug("HBSWAP DEPTH20 MSG TO MESSAGE CH TIMEOUT IN 1MS")
		case w.messageCh <- msg:
		}
		err = gr.Close()
		if err != nil {
			logger.Warnf("gr.Close() error %v", err)
			select {
			case <-ctx.Done():
				break
			default:
				w.restart()
			}
			return
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
	var err error
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case msg := <-w.messageCh:
			if len(msg) < 24 {
				logger.Debugf("bad msg %s", msg)
				break
			}
			switch msg[7] {
			case 's':
				subResp := SubResp{}
				err = json.Unmarshal(msg, &subResp)
				if err != nil {
					logger.Debugf("Unmarshal subResp error %v %s", err, msg)
					break
				}
				if subResp.ErrCode == 0 || subResp.ErrCode == 2014 {
					select {
					case w.topicCh <- subResp.Topic:
					default:
					}
					//logger.Debugf("SUB SUCCESS %s", msg)
				} else {
					logger.Debugf("SUB FAILURE %s", msg)
				}
			case 'p':
				msg[8] = 'o'
				select {
				case w.pingCh <- msg:
				default:
				}
			case 'a':
				wsUser := WsUser{}
				err = json.Unmarshal(msg, &wsUser)
				if err != nil {
					logger.Debugf("Unmarshal wsUser error %v", err)
					logger.Debugf("msg %s", msg)
					break
				}
				if wsUser.ErrCode == 0 {
					select {
					case <-ctx.Done():
						return
					case <-w.done:
						return
					case <-time.After(time.Millisecond):
						logger.Warn("HBSWAP TO LOGIN CH TIME OUT IN 1MS")
					case w.loginCh <- true:
					}
				} else {
					logger.Debugf("HBSWAP LOGIN FAILED %s, STOP WS", msg)
					w.Stop()
				}
			case 'n':
				switch msg[24] {
				case 'a':
					wsAccounts := WSAccounts{}
					err = json.Unmarshal(msg, &wsAccounts)
					if err != nil {
						logger.Debugf("Unmarshal wsAccounts error %v", err)
						logger.Debugf("msg %s", msg)
						break
					}
					select {
					case w.topicCh <- wsAccounts.Topic:
					default:
					}
					select {
					case <-ctx.Done():
						return
					case <-w.done:
						return
					case <-time.After(time.Millisecond):
						logger.Warn("HBSWAP WS ACCOUNTS TO OUTPUT CH TIME OUT IN 1MS")
					case w.AccountCh <- &wsAccounts:
					}
				case 'p':
					wsPositions := WSPositions{}
					err = json.Unmarshal(msg, &wsPositions)
					if err != nil {
						logger.Debugf("Unmarshal wsPositions error %v", err)
						logger.Debugf("msg %s", msg)
						break
					}
					for _, p := range wsPositions.Positions {
						select {
						case w.topicCh <- fmt.Sprintf("positions_cross.%s", p.Symbol):
						default:
						}
					}
					select {
					case <-ctx.Done():
						return
					case <-w.done:
						return
					case <-time.After(time.Millisecond):
						logger.Warn("HBSWAP WS POSITIONS TO OUTPUT CH TIME OUT IN 1MS")
					case w.PositionCh <- &wsPositions:
					}
				case 'o':
					wsOrder := WSOrder{}
					err = json.Unmarshal(msg, &wsOrder)
					if err != nil {
						logger.Debugf("Unmarshal wsOrder error %v", err)
						logger.Debugf("msg %s", msg)
						break
					}
					select {
					case w.topicCh <- wsOrder.Topic:
					default:
					}
					select {
					case <-ctx.Done():
						return
					case <-w.done:
						return
					case <-time.After(time.Millisecond):
						logger.Warn("HBSWAP WS ORDEr TO OUTPUT CH TIME OUT IN 1MS")
					case w.OrderCh <- &wsOrder:
					}
				default:
					logger.Debugf("OTHER NOTIFY %s", msg)
				}
			default:
				logger.Debugf("OTHER MSG %s", msg)
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
			logger.Debugf("PARSE PROXY %v", err)
			return nil, err
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

func (w *UserWebsocket) start(ctx context.Context, symbols []string, proxy string) {

	ctx, cancel := context.WithCancel(ctx)
	var internalCtx context.Context
	var internalCancel context.CancelFunc

	defer func() {
		logger.Debugf("EXIT start")
		cancel()
		if internalCancel != nil {
			internalCancel()
		}
		w.Stop()
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
			conn, err := w.reconnect(internalCtx, "wss://api.hbdm.vn/linear-swap-notification", proxy, 0)
			if err != nil {
				logger.Debugf("HBSWAP RECONNECT ERROR %v, STOP WS", err)
				internalCancel()
				w.Stop()
				return
			}
			go w.startRead(internalCtx, conn)
			go w.startWrite(internalCtx, conn)
			go w.maintainHeartbeat(internalCtx, conn, symbols)

			go w.startDataHandler(internalCtx, 0)
			go w.startDataHandler(internalCtx, 1)
			go w.startDataHandler(internalCtx, 2)
			go w.startDataHandler(internalCtx, 3)

			go w.startDataHandler(internalCtx, 4)
			go w.startDataHandler(internalCtx, 5)
			go w.startDataHandler(internalCtx, 6)
			go w.startDataHandler(internalCtx, 7)
		}
	}
}

func (w *UserWebsocket) maintainHeartbeat(ctx context.Context, conn *websocket.Conn, symbols []string) {

	defer func() {
		logger.Debugf("EXIT maintainHeartbeat")
		err := conn.Close()
		if err != nil {
			logger.Debugf("conn.Close() ERROR %v", err)
		}
	}()

	topicTimeout := time.Minute
	topicCheckInterval := time.Second
	topicCheckTimer := time.NewTimer(time.Second)
	defer topicCheckTimer.Stop()

	login := false
	loginTimer := time.NewTimer(time.Second)
	defer loginTimer.Stop()

	topics := make([]string, 0)
	topics = append(topics, "accounts_cross.USDT")
	for _, symbol := range symbols {
		topics = append(topics, strings.ToLower(fmt.Sprintf("positions_cross.%s", symbol)))
		topics = append(topics, strings.ToLower(fmt.Sprintf("orders_cross.%s", symbol)))
	}

	topicUpdatedTimes := make(map[string]time.Time)
	for _, topic := range topics {
		topicUpdatedTimes[topic] = time.Unix(0, 0)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-loginTimer.C:
			if !login {
				timestamp := time.Now().UTC().Format("2006-01-02T15:04:05")
				values := url.Values{}
				values.Set("AccessKeyId", w.Key)
				values.Set("SignatureMethod", "HmacSHA256")
				values.Set("SignatureVersion", "2")
				values.Set("Timestamp", timestamp)
				payload := fmt.Sprintf("%s\napi.hbdm.vn\n%s\n%s", "GET", "/linear-swap-notification", values.Encode())
				hmac := common.GetHMAC(common.HashSHA256, []byte(payload), []byte(w.Secret))
				//logger.Debugf("LOGIN")
				select {
				case <-ctx.Done():
					return
				case <-time.After(time.Millisecond):
					logger.Debug("SEND AUTH TO WRITE TIMEOUT IN 1MS")
					break
				case w.writeCh <- AuthenticationParam{
					Op:               "auth",
					Type:             "api",
					AccessKeyId:      w.Key,
					SignatureMethod:  "HmacSHA256",
					SignatureVersion: "2",
					Timestamp:        timestamp,
					Signature:        common.Base64Encode(hmac),
				}:
					break
				}
			}
			loginTimer.Reset(time.Minute)
		case topic := <-w.topicCh:
			topic = strings.ToLower(topic)
			if topic == "accounts_cross" {
				topic = "accounts_cross.usdt"
			}
			if _, ok := topicUpdatedTimes[topic]; ok {
				topicUpdatedTimes[topic] = time.Now().Add(time.Hour * 8)
				//logger.Debugf("topic update %s %v", topic, time.Now().Add(time.Hour*8))
			}
		case msg := <-w.pingCh:
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Millisecond):
				logger.Debug("SEND PONG TO WRITE TIMEOUT IN 1MS")
				break
			case w.writeCh <- msg:
				break
			}
			break
		case login = <-w.loginCh:
			break
		case <-topicCheckTimer.C:
			if login {
			loop:
				for _, topic := range topics {
					updateTime := topicUpdatedTimes[strings.ToLower(topic)]
					if time.Now().Sub(updateTime) > topicTimeout {
						//logger.Debugf("HBSWAP CROSS SUBSCRIBE %s", topic)
						select {
						case <-ctx.Done():
							return
						case <-time.After(time.Millisecond):
							logger.Debugf("SEND SUBSCRIBE %s TO WRITE TIMEOUT IN 1MS", topic)
							break
						case w.writeCh <- AccountSubParam{
							Op:    "sub",
							Topic: topic,
						}:
							topicUpdatedTimes[strings.ToLower(topic)] = time.Now().Add(topicCheckInterval * time.Duration(len(symbols)*2))
							break loop
						}
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
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.stopped {
		w.stopped = true
		close(w.done)
	}
}

func (w *UserWebsocket) restart() {
	logger.Infof("HBSWAP WS RESTART")
	select {
	case <-w.done:
		return
	default:
	}
	select {
	case <-time.After(time.Second):
		logger.Debugf("HBSWAP NIL TO RESTART CH TIMEOUT IN 1S, STOP WS")
		w.Stop()
		return
	case w.RestartCh <- nil:
	}
	select {
	case <-time.After(time.Second):
		logger.Debugf("HBSWAP NIL TO RECONNECT CH TIMEOUT IN 1MS, STOP WS")
		w.Stop()
		return
	case w.reconnectCh <- nil:
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
		reconnectCh: make(chan interface{}),
		PositionCh:  make(chan *WSPositions, 100*len(symbols)),
		RestartCh:   make(chan interface{}, 100),
		OrderCh:     make(chan *WSOrder, 100*len(symbols)),
		AccountCh:   make(chan *WSAccounts, 100*len(symbols)),
		messageCh:   make(chan []byte, 100*len(symbols)),
		writeCh:     make(chan interface{}, 100*len(symbols)),
		topicCh:     make(chan string, 100*len(symbols)),
		pingCh:      make(chan []byte, 100),
		loginCh:     make(chan bool, 100),
		mu:          sync.Mutex{},
		stopped:     false,
	}
	go ws.start(ctx, symbols, proxy)
	ws.reconnectCh <- nil
	return &ws
}
