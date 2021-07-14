package ftx_usdspot

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
	logSilentTime := time.Now()
	//var symbolBytes []byte
	//var symbol string
	var msg []byte
	var err error
	//var ch chan []byte
	//var ok bool
	for {
		err = conn.SetReadDeadline(time.Now().Add(time.Minute))
		if err != nil {
			logger.Debugf("conn.SetReadDeadline error %v", err)
			w.restart()
			return
		}

		_, msg, err = conn.ReadMessage()
		if err != nil {
			logger.Debugf("conn.NextReader error %v", err)
			w.restart()
			return
		}
		//logger.Debugf("%s", msg)
		select {
		case w.messageCh <- msg:
		default:
			if time.Now().Sub(logSilentTime) > 0 {
				logSilentTime = time.Now().Add(time.Minute)
			}
		}
		//msgLen := len(msg)
		//if msgLen > 128 && msg[13] == 'o' {
		//	if msg[45] == ',' {
		//		symbolBytes = msg[36:44]
		//		symbol = *(*string)(unsafe.Pointer(&symbolBytes))
		//	} else if msg[46] == ',' {
		//		symbolBytes = msg[36:45]
		//		symbol = *(*string)(unsafe.Pointer(&symbolBytes))
		//	} else if msg[47] == ',' {
		//		symbolBytes = msg[36:46]
		//		symbol = *(*string)(unsafe.Pointer(&symbolBytes))
		//	} else if msg[48] == ',' {
		//		symbolBytes = msg[36:47]
		//		symbol = *(*string)(unsafe.Pointer(&symbolBytes))
		//	} else if msg[49] == ',' {
		//		symbolBytes = msg[36:48]
		//		symbol = *(*string)(unsafe.Pointer(&symbolBytes))
		//	} else if msg[50] == ',' {
		//		symbolBytes = msg[36:49]
		//		symbol = *(*string)(unsafe.Pointer(&symbolBytes))
		//	} else {
		//		if time.Now().Sub(logSilentTime) > 0 {
		//			logger.Debugf("other msg %s", msg)
		//			logSilentTime = time.Now().Add(time.Minute)
		//		}
		//		continue
		//	}
		//} else {
		//	if time.Now().Sub(logSilentTime) > 0 && msgLen > 128 {
		//		logger.Debugf("other msg %s", msg)
		//		logSilentTime = time.Now().Add(time.Minute)
		//	}
		//	continue
		//}
		//if ch, ok = channels[symbol]; ok {
		//	select {
		//	case ch <- msg:
		//	default:
		//		if time.Now().Sub(logSilentTime) > 0 {
		//			logger.Debugf(" ch <- msg %s ch len %d", symbol, len(ch))
		//			logSilentTime = time.Now().Add(time.Minute)
		//		}
		//	}
		//}
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
			//logger.Debugf("%s", msg)
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
							logger.Debugf("w.FillCh <- fill failed, ch len %d", len(w.FillCh))
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
							logger.Debugf("w.Order <- order failed, ch len %d", len(w.OrderCh))
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
	key, secret,
	proxy string,
) *UserWS {
	ws := UserWS{
		done:        make(chan interface{}),
		reconnectCh: make(chan interface{}, 100),
		writeCh:     make(chan interface{}, 100),
		messageCh:   make(chan []byte, 10000),
		RestartCh:   make(chan interface{}, 100),
		trafficCh:   make(chan string, 100),
		loginCh:     make(chan bool, 100),
		stopped:     0,
		key:         key,
		secret:      secret,
		proxy:       proxy,
		OrderCh:     make(chan Order, 1000),
		FillCh:      make(chan Fill, 1000),
	}
	return &ws
}
