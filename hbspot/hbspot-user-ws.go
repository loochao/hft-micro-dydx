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
	"time"
)

const userWebsocketHost = "api-aws.huobi.pro"

type UserWebsocket struct {
	messageCh   chan []byte
	OrderCh     chan *WSOrder
	BalanceCh   chan *WSBalance
	Key         string
	Secret      string
	writeCh     chan interface{}
	done        chan interface{}
	reconnectCh chan interface{}
	topicCh     chan string
	loginCh     chan bool
	pingCh      chan []byte
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
		err := conn.SetReadDeadline(time.Now().Add(time.Hour * 24))
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
		//gr, err := gzip.NewReader(r)
		//if err != nil {
		//	logger.Warnf("NewReader error %v", err)
		//	go w.restart()
		//	return
		//}
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
			logger.Debug("HBSPOT DEPTH20 MSG TO MESSAGE CH TIMEOUT IN 1MS")
		case w.messageCh <- msg:
		}
		//err = gr.Close()
		//if err != nil {
		//	logger.Warnf("gr.Close() error %v", err)
		//	go w.restart()
		//	return
		//}
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
			switch msg[11] {
			case 'p':
				switch msg[12] {
				case 'i':
					msg[12] = 'o'
					select {
					case w.pingCh <- msg:
					default:
					}
				case 'u':
					switch msg[23] {
					case 'a':
						balanceEvent := WSBalanceEvent{}
						err := json.Unmarshal(msg, &balanceEvent)
						if err != nil {
							logger.Debugf("Unmarshal WSBalanceEvent error %v", err)
							continue
						}
						select {
						case <-ctx.Done():
							return
						case <-w.done:
							return
						case <-time.After(time.Millisecond):
							logger.Warn("HBSPOT BALANCE TO OUTPUT CH TIME OUT IN 1MS")
						case w.BalanceCh <- &balanceEvent.Balance:
						}
						select {
						case w.topicCh <- balanceEvent.Ch:
						default:
						}
					case 'o':
						orderEvent := WSOrderEvent{}
						err := json.Unmarshal(msg, &orderEvent)
						if err != nil {
							logger.Debugf("Unmarshal WSOrderEvent error %v %s", err, msg)
							continue
						}
						select {
						case <-ctx.Done():
							return
						case <-w.done:
							return
						case <-time.After(time.Millisecond):
							logger.Warn("HBSPOT BALANCE TO OUTPUT CH TIME OUT IN 1MS")
						case w.OrderCh <- &orderEvent.Order:
						}
						select {
						case w.topicCh <- orderEvent.Ch:
						default:
						}
					default:
						logger.Debugf("OTHER PUSH %s", msg)
					}
				}
			case 'r':
				switch msg[33] {
				case 'a':
					select {
					case <-ctx.Done():
						return
					case <-w.done:
						return
					case <-time.After(time.Millisecond):
						logger.Warn("HBSPOT TO LOGIN CH TIME OUT IN 1MS")
					case w.loginCh <- true:
					}
				default:
					logger.Debugf("OTHER REQ MSG %s", msg)
				}
			case 's':
				subResp := WsCap{}
				err = json.Unmarshal(msg, &subResp)
				if err != nil {
					logger.Debugf("Unmarshal subResp error %v", err)
					logger.Debugf("msg %s", msg)
					break
				}
				logger.Debugf("SUB %s", msg)
				select {
				case w.topicCh <- subResp.Ch:
				default:
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

func (w *UserWebsocket) start(ctx context.Context, symbols []string, proxy string) {

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
			conn, err := w.reconnect(internalCtx, fmt.Sprintf("wss://%s/ws/v2", userWebsocketHost), proxy, 0)
			if err != nil {
				logger.Fatalf("RECONNECT ERROR %v", err)
				return
			}
			go w.startRead(conn)
			go w.startWrite(ctx, conn)
			go w.maintainHeartbeat(internalCtx, conn, symbols)

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

func (w *UserWebsocket) maintainHeartbeat(ctx context.Context, conn *websocket.Conn, symbols []string) {

	defer func() {
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
				case <-ctx.Done():
					return
				case <-time.After(time.Millisecond):
					logger.Debug("SEND AUTH TO WRITE TIMEOUT IN 1MS")
					break
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
					break
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
				for _, topic := range topics {
					if time.Now().Sub(topicUpdatedTimes[topic]) > topicTimeout {
						logger.Debugf("HBSPOT SUBSCRIBE %s", topic)
						select {
						case <-ctx.Done():
							return
						case <-time.After(time.Millisecond):
							logger.Debugf("SEND SUBSCRIBE %s TO WRITE TIMEOUT IN 1MS", topic)
							break
						case w.writeCh <- AccountSubParam{
							Action: "sub",
							Ch:     topic,
						}:
							topicUpdatedTimes[topic] = time.Now().Add(topicCheckInterval * time.Duration(len(symbols)*2))
							continue
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
	if _, ok := <-w.done; ok {
		close(w.done)
		logger.Infof("HBSPOT MARK PRICE WS STOPPED")
	}
}

func (w *UserWebsocket) restart() {
	logger.Infof("HBSPOT WS RESTART")
	select {
	case <-w.done:
		return
	default:
	}
	select {
	case <-time.After(time.Second):
		logger.Fatal("NIL TO RECONNECT CH TIMEOUT IN 1S, EXIT")
	case w.reconnectCh <- nil:
		return
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
		messageCh:   make(chan []byte, 100*len(symbols)),
		writeCh:     make(chan interface{}, 100*len(symbols)),
		topicCh:     make(chan string, 100*len(symbols)),
		pingCh:      make(chan []byte, 100),
		loginCh:     make(chan bool, 100),
	}
	go ws.start(ctx, symbols, proxy)
	ws.reconnectCh <- nil
	return &ws
}
