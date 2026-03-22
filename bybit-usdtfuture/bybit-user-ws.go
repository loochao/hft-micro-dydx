package bybit_usdtfuture

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
	messageCh chan []byte
	OrdersCh  chan []Order
	//ExecutionsCh chan []Execution
	PositionsCh chan []WSPosition
	WalletsCh   chan []Wallet
	pingCh      chan interface{}
	RestartCh   chan interface{}
	writeCh     chan interface{}
	done        chan interface{}
	reconnectCh chan interface{}
	stopped     int32
	topicCh     chan string
	symbols     []string
	api         *API
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

func (w *UserWS) readLoop(conn *websocket.Conn) {
	logger.Debugf("START readLoop")
	defer logger.Debugf("EXIT readLoop")
	//logSilentTime := time.Now()
	for {

		//deadline是一个ping interval
		err := conn.SetReadDeadline(time.Now().Add(time.Minute))
		if err != nil {
			logger.Warnf("SetReadDeadline error %v", err)
			w.restart()
			return
		}
		_, r, err := conn.NextReader()
		if err != nil {
			logger.Warnf("NextReader error %v", err)
			w.restart()
			return
		}
		msg, err := w.readAll(r)
		if err != nil {
			logger.Warnf("readAll error %v", err)
			w.restart()
			return
		}
		select {
		case w.messageCh <- msg:
		default:
			logger.Debugf("w.messageCh <- msg failed, ch len %d", len(w.messageCh))
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
	wsCap := WSCap{}
	logSilentTime := time.Now()
	positions := make([]WSPosition, 0)
	orders := make([]Order, 0)
	//executions := make([]Execution, 0)
	wallets := make([]Wallet, 0)
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case msg := <-w.messageCh:
			//logger.Debugf("%s", msg)
			//if len(msg) < 28 || msg[27] != 'p' {
				//logger.Debugf("%s", msg)
			//}
			wsCap.Topic = ""
			err := json.Unmarshal(msg, &wsCap)
			if err != nil {
				logger.Debugf("json.Unmarshal error %v %s", err, msg)
				continue
			}
			//logger.Debugf("%s", wsCap.Topic)
			switch wsCap.Topic {
			case "position":
				err = json.Unmarshal(wsCap.Data, &positions)
				if err != nil {
					if time.Now().Sub(logSilentTime) > 0 {
						logSilentTime = time.Now().Add(time.Minute)
						logger.Debugf("json.Unmarshal error %v", err)
					}
				} else {
					select {
					case w.PositionsCh <- positions:
					default:
						if time.Now().Sub(logSilentTime) > 0 {
							logSilentTime = time.Now().Add(time.Minute)
							logger.Debugf("w.PositionsCh <- positions failed ch len %d", len(w.PositionsCh))
						}
					}
				}
				select {
				case w.topicCh <- wsCap.Topic:
				default:
					if time.Now().Sub(logSilentTime) > 0 {
						logSilentTime = time.Now().Add(time.Minute)
						logger.Debugf("w.topicCh <- wsCap.Topic failed ch len %d", len(w.topicCh))
					}
				}
				break
			case "order":
				err = json.Unmarshal(wsCap.Data, &orders)
				if err != nil {
					if time.Now().Sub(logSilentTime) > 0 {
						logSilentTime = time.Now().Add(time.Minute)
						logger.Debugf("json.Unmarshal error %v", err)
					}
				} else {
					select {
					case w.OrdersCh <- orders:
					default:
						if time.Now().Sub(logSilentTime) > 0 {
							logSilentTime = time.Now().Add(time.Minute)
							logger.Debugf("w.OrdersCh <- orders failed ch len %d", len(w.OrdersCh))
						}
					}
				}
				select {
				case w.topicCh <- wsCap.Topic:
				default:
					if time.Now().Sub(logSilentTime) > 0 {
						logSilentTime = time.Now().Add(time.Minute)
						logger.Debugf("w.topicCh <- wsCap.Topic failed ch len %d", len(w.topicCh))
					}
				}
				break
			//case "execution":
			//	err = json.Unmarshal(wsCap.Data, &executions)
			//	if err != nil {
			//		if time.Now().Sub(logSilentTime) > 0 {
			//			logSilentTime = time.Now().Add(time.Minute)
			//			logger.Debugf("json.Unmarshal error %v", err)
			//		}
			//	} else {
			//		select {
			//		case w.ExecutionsCh <- executions:
			//		default:
			//			if time.Now().Sub(logSilentTime) > 0 {
			//				logSilentTime = time.Now().Add(time.Minute)
			//				logger.Debugf("w.ExecutionsCh <- executions failed ch len %d", len(w.pingCh))
			//			}
			//		}
			//	}
			//	break
			case "wallet":
				err = json.Unmarshal(wsCap.Data, &wallets)
				if err != nil {
					if time.Now().Sub(logSilentTime) > 0 {
						logSilentTime = time.Now().Add(time.Minute)
						logger.Debugf("json.Unmarshal error %v", err)
					}
				} else {
					select {
					case w.WalletsCh <- wallets:
					default:
						if time.Now().Sub(logSilentTime) > 0 {
							logSilentTime = time.Now().Add(time.Minute)
							logger.Debugf("w.WalletsCh <- wallets failed ch len %d", len(w.WalletsCh))
						}
					}
				}
				select {
				case w.topicCh <- wsCap.Topic:
				default:
					if time.Now().Sub(logSilentTime) > 0 {
						logSilentTime = time.Now().Add(time.Minute)
						logger.Debugf("w.topicCh <- wsCap.Topic failed ch len %d", len(w.topicCh))
					}
				}
				break
			case "":
				//logger.Debugf("%v", wsCap.Request)
				if wsCap.Request != nil {
					if wsCap.Request.Op == "ping" {
						select {
						case w.pingCh <- nil:
						default:
							if time.Now().Sub(logSilentTime) > 0 {
								logSilentTime = time.Now().Add(time.Minute)
								logger.Debugf("w.pingCh <- nil failed ch len %d", len(w.pingCh))
							}
						}
					} else {
						for _, topic := range wsCap.Request.Args {
							select {
							case w.topicCh <- topic:
							default:
								if time.Now().Sub(logSilentTime) > 0 {
									logSilentTime = time.Now().Add(time.Minute)
									logger.Debugf("w.topicCh <- topic failed ch len %d", len(w.topicCh))
								}
							}
						}
					}
				}
				break
			}
		}
	}
}

func (w *UserWS) reconnect(ctx context.Context, wsUrl string, proxy string, counter int64) (*websocket.Conn, error) {

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

func (w *UserWS) mainLoop(ctx context.Context, key, secret string, proxy string) {
	logger.Debugf("START mainLoop")
	defer logger.Debugf("EXIT mainLoop")

	var internalCtx context.Context
	var internalCancel context.CancelFunc

	topics := []string{"position", "order", "wallet"}
	defer func() {
		w.Stop()
		if internalCancel != nil {
			internalCancel()
		}
	}()
	reconnectTimer := time.NewTimer(time.Hour * 9999)
	defer reconnectTimer.Stop()
	for {
		select {
		case <-w.done:
			if internalCancel != nil {
				internalCancel()
				internalCancel = nil
			}
			return
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

			expires := time.Now().UnixNano()/1000000 + 1000
			signature := common.HexEncodeToString(common.GetHMAC(
				common.HashSHA256,
				[]byte(fmt.Sprintf("GET/realtime%d", expires)),
				[]byte(secret),
			))
			urlStr := fmt.Sprintf(
				"%s?api_key=%s&expires=%d&signature=%s",
				"wss://stream.bybit.com/realtime_private",
				key,
				expires,
				signature,
			)
			logger.Debugf("%s", urlStr)
			conn, err := w.reconnect(internalCtx, urlStr, proxy, 0)
			if err != nil {
				logger.Debugf("w.reconnect error %v", err)
				internalCancel()
				w.Stop()
				return
			}
			go w.readLoop(conn)
			go w.writeLoop(internalCtx, conn)
			go w.heartbeatLoop(internalCtx, conn, topics)
			reconnectTimer.Reset(time.Hour * 9999)
		}
	}
}

func (w *UserWS) heartbeatLoop(ctx context.Context, conn *websocket.Conn, topics []string) {
	logger.Debugf("START heartbeatLoop")
	defer func() {
		logger.Debugf("EXIT heartbeatLoop")
		err := conn.Close()
		if err != nil {
			logger.Debugf("conn.Close() error %v", err)
		}
	}()

	pingInterval := time.Second * 30
	topicTimeout := time.Minute
	topicCheckInterval := time.Second
	pingTimer := time.NewTimer(time.Second)
	topicCheckTimer := time.NewTimer(time.Second)
	defer topicCheckTimer.Stop()
	topicUpdatedTimes := make(map[string]time.Time)
	for _, topic := range topics {
		topicUpdatedTimes[topic] = time.Unix(0, 0)
	}
	trafficTimeoutTimer := time.NewTimer(time.Minute)

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case <-trafficTimeoutTimer.C:
			logger.Debugf("traffic timeout in 1m")
			w.restart()
			break
		case topic := <-w.topicCh:
			if _, ok := topicUpdatedTimes[topic]; ok {
				topicUpdatedTimes[topic] = time.Now().Add(time.Hour)
			}
			trafficTimeoutTimer.Reset(time.Minute)
			break
		case <-w.pingCh:
			//logger.Debugf("<-w.pingCh")
			trafficTimeoutTimer.Reset(time.Minute)
			break
		case <-pingTimer.C:
			select {
			case w.writeCh <- WSRequest{
				Op: "ping",
			}:
				pingTimer.Reset(pingInterval)
			default:
				logger.Debugf("w.writeCh <- WSRequest failed, ch len %d", len(w.writeCh))
				pingTimer.Reset(pingInterval / 4)
			}
			break
		case <-topicCheckTimer.C:
			for topic, updateTime := range topicUpdatedTimes {
				if time.Now().Sub(updateTime) > topicTimeout {
					logger.Debugf("SUBSCRIBE %s", topic)
					select {
					case w.writeCh <- WSRequest{
						Op:   "subscribe",
						Args: []string{topic},
					}:
						topicUpdatedTimes[topic] = time.Now().Add(topicCheckInterval * time.Duration(len(topics)*2))
					default:
						logger.Debugf("w.writeCh <- WSRequest failed, topic %s ch len %d", topic, len(w.writeCh))
					}
				}
			}
			topicCheckTimer.Reset(topicCheckInterval)
			break
		}
	}

}

func (w *UserWS) Stop() {
	if atomic.CompareAndSwapInt32(&w.stopped, 0, 1) {
		close(w.done)
		logger.Infof("stopped")
	}
}

func (w *UserWS) restart() {
	select {
	case w.RestartCh <- nil:
	default:
		logger.Debugf("nil to RestartCh failed")
	}
	select {
	case w.reconnectCh <- nil:
		logger.Debugf("ws restart")
		break
	default:
		logger.Debugf("w.reconnectCh <- nil failed ch len %d", len(w.reconnectCh))
	}
}

func (w *UserWS) Done() chan interface{} {
	return w.done
}

func NewUserWS(
	ctx context.Context,
	key, secret string,
	proxy string,
) *UserWS {
	ws := UserWS{
		done:        make(chan interface{}),
		reconnectCh: make(chan interface{}),
		OrdersCh:    make(chan []Order, 16),
		//ExecutionsCh: make(chan []Execution, 16),
		PositionsCh: make(chan []WSPosition, 16),
		WalletsCh:   make(chan []Wallet, 16),
		RestartCh:   make(chan interface{}, 16),
		pingCh:      make(chan interface{}, 16),
		messageCh:   make(chan []byte, 128),
		writeCh:     make(chan interface{}, 128),
		topicCh:     make(chan string, 128),
		stopped:     0,
	}
	go ws.mainLoop(ctx, key, secret, proxy)
	go ws.dataHandleLoop(ctx)
	ws.reconnectCh <- nil
	return &ws
}
