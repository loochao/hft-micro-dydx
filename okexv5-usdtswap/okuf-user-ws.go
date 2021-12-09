package okexv5_usdtswap

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
	"strconv"
	"sync/atomic"
	"time"
)

type UserWS struct {
	messageCh   chan []byte
	OrdersCh    chan []Order
	AccountsCh  chan []Account
	PositionsCh chan []Position
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

func (w *UserWS) writeLoop(ctx context.Context, conn *websocket.Conn) {
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

			//logger.Debugf("%s", msgBytes)

			err = conn.WriteMessage(websocket.TextMessage, msgBytes)
			if err != nil {
				logger.Debugf("conn.WriteMessage %s error %v", string(msgBytes), err)
				w.restart()
				return
			}
		}
	}
}

func (w *UserWS) readLoop(conn *websocket.Conn) {
	logger.Debugf("START readLoop")
	defer logger.Debugf("EXIT readLoop")
	logSilentTime := time.Now()
	var msg []byte
	var err error
	var r io.Reader
	for {
		err = conn.SetReadDeadline(time.Now().Add(time.Hour * 24))
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
		msg, err = w.readAll(r)
		if err != nil {
			logger.Warnf("w.readAll error %v", err)
			w.restart()
			return
		}
		//logger.Debugf("%s", msg)
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

func (w *UserWS) readAll(r io.Reader) ([]byte, error) {
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

func (w *UserWS) dataHandleLoop(ctx context.Context) {
	logger.Debugf("START dataHandleLoop")
	defer logger.Debugf("EXIT dataHandleLoop")
	logSilentTime := time.Now()
	var err error
	var commonCap CommonCapture
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case msg := <-w.messageCh:
			if len(msg) == 4 && msg[0] == 'p' {
				//logger.Debugf("%s", msg)
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
			commonCap.Event = ""
			commonCap.Data = nil
			commonCap.Arg.Channel = ""
			err = json.Unmarshal(msg, &commonCap)
			if err != nil {
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("json.Unmarshal(msg, &commonCap) error %v %s", err, msg)
					logSilentTime = time.Now().Add(time.Minute)
				}
				continue
			}

			if commonCap.Data != nil {
				switch commonCap.Arg.Channel {

				case "account":
					bds := make([]AccountData, 0)
					err = json.Unmarshal(commonCap.Data, &bds)
					if err != nil {
						if time.Now().Sub(logSilentTime) > 0 {
							logger.Debugf("json.Unmarshal(commonCap.Data, &bd) error %v %s", err, msg)
							logSilentTime = time.Now().Add(time.Minute)
						}
						continue
					}
					for _, bd := range bds {
						select {
						case w.AccountsCh <- bd.Details:
						default:
							if time.Now().Sub(logSilentTime) > 0 {
								logger.Debugf("w.BalancesCh <- balances failed ch len %d", len(w.AccountsCh))
								logSilentTime = time.Now().Add(time.Minute)
							}
						}
					}
					select {
					case w.topicCh <- "account":
					default:
						if time.Now().Sub(logSilentTime) > 0 {
							logger.Debugf("w.topicCh <- account failed ch len %d", len(w.topicCh))
							logSilentTime = time.Now().Add(time.Minute)
						}
					}
					continue

				case "balance_and_position":
					bps := make([]BalanceAndPosition, 0)
					err = json.Unmarshal(commonCap.Data, &bps)
					if err != nil {
						if time.Now().Sub(logSilentTime) > 0 {
							logger.Debugf("json.Unmarshal(commonCap.Data, &bd) error %v %s", err, msg)
							logSilentTime = time.Now().Add(time.Minute)
						}
						continue
					}
					for _, bd := range bps {
						select {
						case w.PositionsCh <- bd.PosData:
						default:
							if time.Now().Sub(logSilentTime) > 0 {
								logger.Debugf("w.PositionsCh <- balances failed ch len %d", len(w.AccountsCh))
								logSilentTime = time.Now().Add(time.Minute)
							}
						}
					}
					select {
					case w.topicCh <- "balance_and_position":
					default:
						if time.Now().Sub(logSilentTime) > 0 {
							logger.Debugf("w.topicCh <- balance_and_position failed ch len %d", len(w.topicCh))
							logSilentTime = time.Now().Add(time.Minute)
						}
					}
					continue

				case "orders", "orders-algo", "algo-advance":
					//logger.Debugf("ORDER-WS %s", commonCap.Data)
					orders := make([]Order, 0)
					err = json.Unmarshal(commonCap.Data, &orders)
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
					select {
					case w.topicCh <- "orders":
					default:
						if time.Now().Sub(logSilentTime) > 0 {
							logger.Debugf("w.topicCh <- orders failed ch len %d", len(w.topicCh))
							logSilentTime = time.Now().Add(time.Minute)
						}
					}
				default:
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("other table msg %s", msg)
						logSilentTime = time.Now().Add(time.Minute)
					}
				}
			} else if commonCap.Event != "" {
				switch commonCap.Event {
				case "login":
					if commonCap.Code == "0" {
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
					if commonCap.Arg.Channel != "" {
						select {
						case w.topicCh <- commonCap.Arg.Channel:
						default:
							if time.Now().Sub(logSilentTime) > 0 {
								logger.Debugf("w.topicCh <- commonCap.Arg.Channel failed ch len %d", len(w.topicCh))
								logSilentTime = time.Now().Add(time.Minute)
							}
						}
					}
					continue
					//default:
					//if time.Now().Sub(logSilentTime) > 0 {
					//	logger.Debugf("MSG %s", msg)
					//	logSilentTime = time.Now().Add(time.Minute)
					//}
				}
				//} else {
				//	if time.Now().Sub(logSilentTime) > 0 {
				//		logger.Debugf("MSG %s", msg)
				//		logSilentTime = time.Now().Add(time.Minute)
				//	}
			}
		}
	}
}

func (w *UserWS) reconnect(ctx context.Context, wsUrl string, proxy string, counter int64) (*websocket.Conn, error) {

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
			Proxy:             http.ProxyURL(proxyUrl),
			HandshakeTimeout:  60 * time.Second,
			EnableCompression: true,
		}
	} else {
		dialer = &websocket.Dialer{
			HandshakeTimeout:  10 * time.Second,
			EnableCompression: true,
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

func (w *UserWS) mainLoop(ctx context.Context, proxy string) {
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
			reconnectTimer.Reset(time.Second * 5)
		case <-reconnectTimer.C:
			if internalCancel != nil {
				internalCancel()
			}
			internalCtx, internalCancel = context.WithCancel(ctx)
			conn, err := w.reconnect(internalCtx, "wss://ws.okex.com:8443/ws/v5/private", proxy, 0)
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
			go w.heartbeatLoop(internalCtx, conn)
		}
	}
}

func (w *UserWS) heartbeatLoop(ctx context.Context, conn *websocket.Conn) {
	logger.Debugf("START heartbeatLoop")
	defer func() {
		logger.Debugf("EXIT heartbeatLoop")
		err := conn.Close()
		if err != nil {
			logger.Debugf("conn.Close() ERROR %v", err)
		}
	}()

	channelTimeout := time.Hour * 24
	channelCheckInterval := time.Second
	channelCheckTimer := time.NewTimer(time.Second)
	defer channelCheckTimer.Stop()

	login := false
	loginTimer := time.NewTimer(time.Second)
	defer loginTimer.Stop()

	trafficTimeout := time.NewTimer(time.Minute * 5)
	pingInterval := time.Second*15
	pingTimer := time.NewTimer(pingInterval)
	defer trafficTimeout.Stop()
	defer pingTimer.Stop()

	channelUpdatedTimes := make(map[string]time.Time)
	for _, topic := range []string{"account", "balance_and_position", "orders", "algo-advance", "orders-algo"} {
		channelUpdatedTimes[topic] = time.Unix(0, 0)
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
			select {
			case w.writeCh <- []byte("ping"):
			default:
				logger.Debugf("w.writeCh <- ping failed, ch len %d", len(w.writeCh))
			}
			pingTimer.Reset(pingInterval)
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
				case w.writeCh <- WsLogin{
					Op: "login",
					Args: []WsLoginArgs{
						{
							ApiKey:     w.Key,
							Passphrase: w.Passphrase,
							Timestamp:  strconv.FormatInt(unixTime, 10),
							Sign:       base64,
						},
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
			break
		case topic := <-w.topicCh:
			//pingTimer.Reset(time.Second * 15)
			trafficTimeout.Reset(time.Second * 30)
			if _, ok := channelUpdatedTimes[topic]; ok {
				channelUpdatedTimes[topic] = time.Now().Add(time.Hour * 8)
			}
			break
		case <-w.pongCh:
			pingTimer.Reset(pingInterval)
			trafficTimeout.Reset(time.Second * 30)
			break
		case login = <-w.loginCh:
			logger.Debugf("login %v", login)
			break
		case <-channelCheckTimer.C:
			if login {
				args := make([]WsArgs, 0)
				for channel, t := range channelUpdatedTimes {
					if time.Now().Sub(t) > channelTimeout {
						if channel == "algo-advance" {
							args = append(args, WsArgs{
								Channel:  channel,
								InstType: "SWAP",
							})
						} else if channel == "orders" {
							args = append(args, WsArgs{
								Channel:  channel,
								InstType: "SWAP",
							})
						} else if channel == "orders-algo" {
							args = append(args, WsArgs{
								Channel:  channel,
								InstType: "SWAP",
							})
						} else {
							args = append(args, WsArgs{
								Channel: channel,
							})
						}
						channelUpdatedTimes[channel] = time.Now().Add(channelTimeout)
					}
				}
				if len(args) > 0 {
					select {
					case w.writeCh <- WsSubUnsub{
						Op:   "subscribe",
						Args: args,
					}:
					default:
						logger.Debugf("w.writeCh <- WsSubUnsub failed, ch len %d", len(w.writeCh))
					}
				}
			}
			channelCheckTimer.Reset(channelCheckInterval)
			break
		}
	}

}

func (w *UserWS) Stop() {
	if atomic.CompareAndSwapInt32(&w.stopped, 0, 1) {
		close(w.done)
		logger.Debugf("stopped")
	}
}

func (w *UserWS) restart() {
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

func (w *UserWS) Done() chan interface{} {
	return w.done
}

func NewUserWebsocket(
	ctx context.Context,
	key, secret, passphrase string,
	proxy string,
) *UserWS {
	ws := UserWS{
		Key:         key,
		Secret:      secret,
		Passphrase:  passphrase,
		done:        make(chan interface{}),
		reconnectCh: make(chan interface{}, 64),
		OrdersCh:    make(chan []Order, 64),
		AccountsCh:  make(chan []Account, 64),
		PositionsCh: make(chan []Position, 64),
		RestartCh:   make(chan interface{}, 64),
		messageCh:   make(chan []byte, 64),
		writeCh:     make(chan interface{}, 64),
		topicCh:     make(chan string, 128),
		pongCh:      make(chan []byte, 64),
		loginCh:     make(chan bool, 64),
		stopped:     0,
	}
	go ws.mainLoop(ctx, proxy)
	go ws.dataHandleLoop(ctx)
	ws.reconnectCh <- nil
	return &ws
}
