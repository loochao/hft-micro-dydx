package dydx_usdfuture

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
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
	messageCh chan []byte
	//OrderCh     chan *WSOrder
	//PositionCh  chan *WSPosition
	//BalanceCh   chan *WsBalanceEvent
	AccountCh   chan Account
	PositionsCh chan []Position
	OrdersCh    chan []Order
	credentials *Credentials
	RestartCh   chan interface{}
	writeCh     chan interface{}
	done        chan interface{}
	reconnectCh chan interface{}
	trafficCh   chan interface{}
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

func (w *UserWebsocket) readLoop(conn *websocket.Conn) {
	logger.Debugf("START readLoop")
	defer logger.Debugf("EXIT readLoop")

	var readPool = [userReadPoolSize][]byte{}
	var readIndex = -1
	var msg []byte
	var n int
	var r io.Reader
	var err error
	for i := range readPool {
		readPool[i] = make([]byte, userReadMsgSize)
	}
	//readCounter := 0
	//partialReadCounter := 0
	//allocateCounter := 0
mainLoop:
	for {
		err = conn.SetReadDeadline(time.Now().Add(time.Hour))
		if err != nil {
			logger.Debugf("conn.SetReadDeadline error %v", err)
			w.restart()
			return
		}
		_, r, err = conn.NextReader()
		if err != nil {
			logger.Warnf("conn.NextReader error %v", err)
			w.restart()
			return
		}
		readIndex += 1
		if readIndex == userReadPoolSize {
			readIndex = 0
		}
		msg = readPool[readIndex]
		n, err = r.Read(msg)
		if err == nil {
			//readCounter++
			msg = msg[:n]
			if n < 2 || msg[n-1] != '}' || msg[n-2] != '}'{
				//partialReadCounter++
			readLoop:
				for {
					if len(msg) == cap(msg) {
						// Add more capacity (let append pick how much).
						msg = append(msg, 0)[:len(msg)]
						//allocateCounter++
						//logger.Debugf("BAD BUFFER SIZE CAN'T READ INTO %d, MSG: %s", userReadMsgSize, msg)
					}
					n, err = r.Read(msg[len(msg):cap(msg)])
					msg = msg[:len(msg)+n]
					if err != nil {
						if err == io.EOF {
							break readLoop
						} else {
							logger.Debugf("r.Read error %v", err)
							continue mainLoop
						}
					}
				}
			}
		} else {
			logger.Debugf("r.Read error %v", err)
			continue mainLoop
		}

		//if readCounter%100 == 0 {
		//	logger.Debugf("DYDX USER WS READ SIZE %d TOTAL READ %d PARTIAL READ %d EXPAND ALLOCATE %d", userReadMsgSize,readCounter, partialReadCounter, allocateCounter)
		//}

		select {
		case w.messageCh <- msg:
		default:
			logger.Debugf("w.messageCh <- msg failed, ch len %d", len(w.messageCh))
		}
	}

}

//func (w *UserWebsocket) readAll(r io.Reader) ([]byte, error) {
//	b := make([]byte, 0, 1024)
//	for {
//		if len(b) == cap(b) {
//			// Add more capacity (let append pick how much).
//			b = append(b, 0)[:len(b)]
//		}
//		n, err := r.Read(b[len(b):cap(b)])
//		b = b[:len(b)+n]
//		if err != nil {
//			if err == io.EOF {
//				err = nil
//			}
//			return b, err
//		}
//	}
//}

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
			wsCap := &WSUserCap{}
			err := json.Unmarshal(msg, wsCap)
			if err != nil {
				logger.Debugf("%v", err)
				break
			}
			switch wsCap.Type {
			case "subscribed":
				wsUserSubscribed := &WSUserSubscribed{}
				err = json.Unmarshal(wsCap.Contents, wsUserSubscribed)
				if err != nil {
					logger.Debugf("%v", err)
					break
				}

				if len(wsUserSubscribed.Orders) > 0 {
					select {
					case w.OrdersCh <- wsUserSubscribed.Orders:
					default:
						logger.Debugf("w.OrdersCh <- wsUserSubscribed.Orders failed, len %d", len(w.OrdersCh))
					}
				}
				if len(wsUserSubscribed.Account.OpenPositions) > 0 {
					ps := make([]Position, 0)
					for _, pos := range wsUserSubscribed.Account.OpenPositions {
						pos := pos
						ps = append(ps, pos)
					}
					select {
					case w.PositionsCh <- ps:
					default:
						logger.Debugf("w.PositionsCh <- ps failed, len %d", len(w.PositionsCh))
					}
				}
				select {
				case w.AccountCh <- wsUserSubscribed.Account:
				default:
					logger.Debugf("w.AccountCh <- wsUserSubscribed.Account failed, len %d", len(w.AccountCh))
				}

				select {
				case w.trafficCh <- nil:
				default:
					logger.Debugf("w.trafficCh <- nil: failed, len %d", len(w.trafficCh))
				}
			case "channel_data":
				//logger.Debugf("%s", msg)
				wsUserChannelData := &WSUserChannelData{}
				err = json.Unmarshal(wsCap.Contents, wsUserChannelData)
				if err != nil {
					logger.Debugf("%v", err)
					break
				}
				if len(wsUserChannelData.Orders) > 0 {
					//logger.Debugf("%s", wsCap.Contents)
					select {
					case w.OrdersCh <- wsUserChannelData.Orders:
						//for _, o := range wsUserChannelData.Orders {
						//	if o.CancelReason != nil {
						//		logger.Debugf("%s %s %s %s", o.Market, o.Side,o.ClientID, *o.CancelReason)
						//	}
						//}
					default:
						logger.Debugf("w.OrdersCh <- wsUserChannelData.Orders failed, len %d", len(w.OrdersCh))
					}
				}
				if len(wsUserChannelData.Positions) > 0 {
					select {
					case w.PositionsCh <- wsUserChannelData.Positions:
					default:
						logger.Debugf("w.PositionsCh <- wsUserChannelData.Positions failed, len %d", len(w.PositionsCh))
					}
				}
				for _, account := range wsUserChannelData.Accounts {
					select {
					case w.AccountCh <- account:
					default:
						logger.Debugf("w.AccountCh <- account failed, len %d", len(w.AccountCh))
					}
				}
				select {
				case w.trafficCh <- nil:
				default:
					logger.Debugf("w.trafficCh <- nil: failed, len %d", len(w.trafficCh))
				}
				break
			case "error":
				if strings.Contains(wsCap.Message, "Invalid subscribe message: already subscribed") {
					select {
					case w.trafficCh <- nil:
					default:
						logger.Debugf("w.trafficCh <- nil: failed, len %d", len(w.trafficCh))
					}
				}
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

func (w *UserWebsocket) mainLoop(ctx context.Context, proxy string) {
	logger.Debugf("START mainLoop")
	defer logger.Debugf("EXIT mainLoop")
	ctx, cancel := context.WithCancel(ctx)
	var internalCtx context.Context
	var internalCancel context.CancelFunc
	defer func() {
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
			conn, err := w.reconnect(internalCtx, "wss://api.dydx.exchange/v3/ws", proxy, 0)
			if err != nil {
				logger.Debugf("w.reconnect error %v", err)
				internalCancel()
				w.Stop()
				return
			}
			go w.readLoop(conn)
			go w.writeLoop(internalCtx, conn)
			go w.heartbeatLoop(internalCtx, conn)
		}
	}
}

func (w *UserWebsocket) heartbeatLoop(ctx context.Context, conn *websocket.Conn) {
	logger.Debugf("START heartbeatLoop")
	defer func() {
		logger.Debugf("Exit heartbeatLoop")
		err := conn.Close()
		if err != nil {
			logger.Debugf("conn.Close() ERROR %v", err)
		}
	}()
	subscribeInterval := time.Minute * 15
	subscribeTime := time.Now()

	checkInterval := time.Second
	checkTimer := time.NewTimer(time.Second)
	defer checkTimer.Stop()

	trafficTimeout := time.NewTimer(time.Minute * 5)
	defer trafficTimeout.Stop()

	conn.SetPingHandler(func(msg string) error {
		trafficTimeout.Reset(time.Minute)
		err := conn.WriteControl(websocket.PongMessage, []byte(msg), time.Now().Add(time.Minute))
		if err != nil {
			select {
			case <-ctx.Done():
				break
			default:
				go w.restart()
			}
			return nil
		}
		return nil
	})

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
		case <-w.trafficCh:
			subscribeTime = time.Now().Add(subscribeInterval)
			trafficTimeout.Reset(time.Minute)
			break
		case <-checkTimer.C:
			checkTimer.Reset(checkInterval)
			if time.Now().Sub(subscribeTime) > 0 {
				subscribeTime = time.Now().Add(time.Minute)
				isoTimestamp := time.Now().UTC().Format(TimeLayout)
				signature := fmt.Sprintf(
					"%s%s%s",
					isoTimestamp,
					"GET",
					"/ws/accounts",
				)
				secret, err := base64.URLEncoding.DecodeString(w.credentials.ApiSecret)
				if err != nil {
					logger.Debugf("%v", err)
					break
				}
				h := hmac.New(sha256.New, secret)
				h.Write([]byte(signature))
				hmacSigned := h.Sum(nil)
				signStr := base64.URLEncoding.EncodeToString(hmacSigned)
				select {
				case w.writeCh <- WSAccountSubscribe{
					Type:          "subscribe",
					Channel:       "v3_accounts",
					AccountNumber: w.credentials.AccountNumber,
					ApiKey:        w.credentials.ApiKey,
					Signature:     signStr,
					Timestamp:     isoTimestamp,
					Passphrase:    w.credentials.ApiPassphrase,
				}:
				default:
					logger.Debugf("w.writeCh <- Subscription failed, ch len %d", len(w.writeCh))
				}
			}
			break
		}
	}
}

func (w *UserWebsocket) Stop() {
	if atomic.CompareAndSwapInt32(&w.stopped, 0, 1) {
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
	credentials *Credentials,
	proxy string,
) *UserWebsocket {
	ws := UserWebsocket{
		done:        make(chan interface{}),
		reconnectCh: make(chan interface{}),
		//OrderCh:     make(chan *WSOrder, 16),
		//BalanceCh:   make(chan *WsBalanceEvent, 16),
		//PositionCh:  make(chan *WSPosition, 16),
		AccountCh:   make(chan Account, 16),
		PositionsCh: make(chan []Position, 16),
		OrdersCh:    make(chan []Order, 16),
		RestartCh:   make(chan interface{}, 16),
		messageCh:   make(chan []byte, 128),
		writeCh:     make(chan interface{}, 128),
		topicCh:     make(chan string, 128),
		trafficCh:   make(chan interface{}, 128),
		credentials: credentials,
		stopped:     0,
	}
	go ws.mainLoop(ctx, proxy)
	go ws.dataHandleLoop(ctx)
	ws.reconnectCh <- nil
	return &ws
}
