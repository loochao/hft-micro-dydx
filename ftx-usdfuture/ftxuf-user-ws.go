package ftx_usdfuture

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
	"sync/atomic"
	"time"
)

type UserWS struct {
	writeCh     chan interface{}
	done        chan interface{}
	reconnectCh chan interface{}
	RestartCh   chan interface{}
	messageCh   chan []byte
	OrderCh     chan Order
	FillCh      chan Fill
	loginCh     chan bool
	stopped     int32
	trafficCh   chan string
	key         string
	secret      string
	subAccount  string
	proxy       string
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
					logger.Debugf("json.Marshal(msg) err %v", err)
					continue
				}
			}
			err = conn.SetWriteDeadline(time.Now().Add(time.Minute))
			if err != nil {
				logger.Debugf("conn.SetWriteDeadline error %v", err)
				w.restart()
				return
			}

			//logger.Debugf("%s", msgBytes)
			err = conn.WriteMessage(websocket.TextMessage, msgBytes)
			if err != nil {
				logger.Debugf("conn.WriteMessage error %v", err)
				w.restart()
				return
			}
		}
	}
}

func (w *UserWS) readLoop(conn *websocket.Conn) {
	logger.Debugf("START readLoop")
	defer logger.Debugf("EXIT readLoop")
	var logSilentTime = time.Time{}
	var readPool = [userReadPoolSize][]byte{}
	var readIndex = -1
	var msg []byte
	var n int
	var r io.Reader
	var err error
	for i := range readPool {
		readPool[i] = make([]byte, userReadMsgSize)
	}
	readCounter := 0
	partialReadCounter := 0
	allocateCounter := 0
mainLoop:
	for {
		err = conn.SetReadDeadline(time.Now().Add(time.Minute))
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
			readCounter++
			msg = msg[:n]
			if n < 1 || msg[n-1] != '}' {
				partialReadCounter++
				for {
					if len(msg) == cap(msg) {
						msg = append(msg, 0)[:len(msg)]
						logger.Debugf("BAD BUFFER SIZE CAN'T READ INTO %d, MSG: %s", userReadMsgSize, msg)
						allocateCounter++
					}
					n, err = r.Read(msg[len(msg):cap(msg)])
					msg = msg[:len(msg)+n]
					if err != nil {
						if err == io.EOF {
							break
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
		if readCounter%10000 == 0 {
			logger.Debugf("FTXUF USER READ SIZE %d TOTAL %d PARTIAL %d ALLOCATE %d", userReadMsgSize, readCounter, partialReadCounter, allocateCounter)
		}
		select {
		case w.messageCh <- msg:
		default:
			if time.Now().Sub(logSilentTime) > 0 {
				logSilentTime = time.Now().Add(time.Minute)
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
			return nil, fmt.Errorf("url.Parse(proxy) error %v", err)
		}
		dialer = &websocket.Dialer{
			Proxy:            http.ProxyURL(proxyUrl),
			HandshakeTimeout: 60 * time.Second,
			//EnableCompression: true,
		}
	} else {
		dialer = &websocket.Dialer{
			HandshakeTimeout: 10 * time.Second,
			//EnableCompression: true,
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

func (w *UserWS) mainLoop(ctx context.Context, key, secret, proxy string) {
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
			conn, err := w.reconnect(internalCtx, "wss://ftx.com/ws/", proxy, 0)
			if err != nil {
				logger.Debugf("w.reconnect error %v", err)
				internalCancel()
				w.Stop()
				return
			}
			go w.readLoop(conn)
			go w.writeLoop(internalCtx, conn)
			go w.heartbeatLoop(internalCtx, key, secret, conn)
		}
	}
}

func (w *UserWS) heartbeatLoop(ctx context.Context, key, secret string, conn *websocket.Conn) {
	logger.Debugf("START heartbeatLoop")
	defer func() {
		logger.Debugf("Exit heartbeatLoop")
		err := conn.Close()
		if err != nil {
			logger.Debugf("conn.Close() ERROR %v", err)
		}
	}()
	dataTimeout := time.Minute
	dataCheckInterval := time.Second
	dataCheckTimer := time.NewTimer(time.Second)
	defer dataCheckTimer.Stop()
	fillsUpdateTime := time.Unix(0, 0)
	ordersUpdateTime := time.Unix(0, 0)
	trafficTimeout := time.NewTimer(time.Minute * 5)
	pingTimer := time.NewTimer(time.Second * 15)
	defer trafficTimeout.Stop()
	defer pingTimer.Stop()
	loginTimer := time.NewTimer(time.Second)
	defer loginTimer.Stop()
	login := false
	loginSuccessTimer := time.NewTimer(time.Hour * 9999)
	defer loginSuccessTimer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case <-loginSuccessTimer.C:
			login = true
			dataCheckTimer.Reset(time.Nanosecond)
			loginSuccessTimer.Reset(time.Hour * 9999)
		case <-loginTimer.C:
			if !login {
				param := LoginParam{}
				param.Args.Key = key
				param.Args.Time = time.Now().UnixNano() / 1000000
				signature := fmt.Sprintf("%dwebsocket_login", param.Args.Time)
				param.Args.SubAccount = w.subAccount
				param.Args.Sign = common.HexEncodeToString(common.GetHMAC(common.HashSHA256, []byte(signature), []byte(secret)))
				param.Op = "login"
				select {
				case w.writeCh <- param:
					loginSuccessTimer.Reset(time.Second * 3)
					break
				default:
					logger.Debugf("w.writeCh <- param failed, ch len %d", len(w.writeCh))
				}
			}
			loginTimer.Reset(time.Second * 15)
			break
		case <-trafficTimeout.C:
			logger.Debugf("traffic timeout in 60s, restart ws")
			w.restart()
			return
		case login = <-w.loginCh:
			if !login {
				logger.Debugf("login failed, stop ws")
				w.Stop()
				return
			}
			dataCheckTimer.Reset(time.Nanosecond)
			break
		case <-pingTimer.C:
			pingTimer.Reset(time.Second * 15)
			select {
			case w.writeCh <- []byte("{\"op\": \"ping\"}"):
				break
			default:
				logger.Debugf("w.writeCh <- ping failed, ch len %d", len(w.writeCh))
			}
			break
		case traffic := <-w.trafficCh:
			switch traffic {
			case "pong":
				break
			case "fills":
				fillsUpdateTime = time.Now().Add(time.Hour * 24)
				break
			case "orders":
				ordersUpdateTime = time.Now().Add(time.Hour * 24)
				break
			}
			trafficTimeout.Reset(time.Minute)
			break
		case <-dataCheckTimer.C:
			if login {
				if time.Now().Sub(fillsUpdateTime) > dataTimeout {
					select {
					case w.writeCh <- SubscribeParam{
						Operation: "subscribe",
						Channel:   "fills",
					}:
						fillsUpdateTime = time.Now().Add(time.Minute)
						break
					default:
						logger.Debugf("w.writeCh <- SubscribePara failed, ch len %d", len(w.writeCh))
					}
				}
				if time.Now().Sub(ordersUpdateTime) > dataTimeout {
					select {
					case w.writeCh <- SubscribeParam{
						Operation: "subscribe",
						Channel:   "orders",
					}:
						ordersUpdateTime = time.Now().Add(time.Minute)
						break
					default:
						logger.Debugf("w.writeCh <- SubscribePara failed, ch len %d", len(w.writeCh))
					}
				}
			}
			dataCheckTimer.Reset(dataCheckInterval)
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
	case w.reconnectCh <- nil:
		select {
		case w.RestartCh <- nil:
		default:
			logger.Debugf("w.RestartCh <- nil failed, ch len %d", len(w.RestartCh))
		}
	default:
		logger.Debugf("w.reconnectCh <- nil failed, ch len %d", len(w.reconnectCh))
	}
}

func (w *UserWS) Done() chan interface{} {
	return w.done
}

func (w *UserWS) dataHandleLoop(ctx context.Context) {
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
			dataCap := UserDataCap{}
			err = json.Unmarshal(msg, &dataCap)
			if err != nil {
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("json.Unmarshal(msg, &dataCap) error %v", err)
					logSilentTime = time.Now().Add(time.Minute)
				}
				continue
			}
			if dataCap.Type == "error" {
				if dataCap.Msg == "Already logged in" {
					select {
					case w.loginCh <- true:
						logger.Debugf("LOGIN success")
					default:
						logger.Debugf("w.loginCh <- true failed, ch len %d", len(w.loginCh))
					}
				} else if dataCap.Msg == "Invalid login credentials" {
					select {
					case w.loginCh <- false:
						logger.Debugf("login failed")
					default:
						logger.Debugf("w.loginCh <- true failed, ch len %d", len(w.loginCh))
					}
				} else {
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("other error msg %s", msg)
						logSilentTime = time.Now().Add(time.Minute)
					}
				}
				continue
			} else if dataCap.Type == "pong" {
				select {
				case w.trafficCh <- "pong":
				default:
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("w.trafficCh <- \"pong\" failed, ch len %d", len(w.trafficCh))
						logSilentTime = time.Now().Add(time.Minute)
					}
				}
			} else if dataCap.Type == "subscribed" {
				select {
				case w.trafficCh <- dataCap.Channel:
				default:
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("w.trafficCh <- dataCap.Channel failed, ch len %d", len(w.trafficCh))
						logSilentTime = time.Now().Add(time.Minute)
					}
				}
			} else if dataCap.Type == "update" {
				switch dataCap.Channel {
				case "fills":
					fill := Fill{}
					err = json.Unmarshal(dataCap.Data, &fill)
					if err != nil {
						logger.Debugf("err = json.Unmarshal(dataCap.Data, &order) error %v", err)
					} else {
						select {
						case w.FillCh <- fill:
						default:
							logger.Debugf("w.FillCh <- fill failed, ch len %d %s", len(w.FillCh), msg)
						}
					}
					continue
				case "orders":
					order := Order{}
					err = json.Unmarshal(dataCap.Data, &order)
					if err != nil {
						logger.Debugf("err = json.Unmarshal(dataCap.Data, &order) error %v", err)
					} else {
						select {
						case w.OrderCh <- order:
						default:
							logger.Debugf("w.Order <- order failed, ch len %d %s", len(w.OrderCh), msg)
						}
					}
					continue
				}
			}
		}
	}
}

func (w *UserWS) Start(ctx context.Context) {
	go w.dataHandleLoop(ctx)
	go w.mainLoop(ctx, w.key, w.secret, w.proxy)
	w.reconnectCh <- nil
}

func NewUserWS(
	key, secret, subAccount,
	proxy string,
) *UserWS {
	ws := UserWS{
		done:        make(chan interface{}),
		reconnectCh: make(chan interface{}, common.ChannelSizeLowLoad),
		writeCh:     make(chan interface{}, common.ChannelSizeLowDropRatio),
		messageCh:   make(chan []byte, common.ChannelSizeLowDropRatio),
		RestartCh:   make(chan interface{}, common.ChannelSizeLowLoad),
		trafficCh:   make(chan string, common.ChannelSizeLowDropRatio),
		loginCh:     make(chan bool, common.ChannelSizeLowLoad),
		stopped:     0,
		key:         key,
		secret:      secret,
		subAccount:  subAccount,
		proxy:       proxy,
		OrderCh:     make(chan Order, common.ChannelSizeLowDropRatio),
		FillCh:      make(chan Fill, common.ChannelSizeLowDropRatio),
	}
	return &ws
}
