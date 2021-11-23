package binance_usdtfuture

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

type UserWebsocket struct {
	messageCh                       chan []byte
	BalanceAndPositionUpdateEventCh chan *BalanceAndPositionUpdateEvent
	OrderUpdateEventCh              chan *OrderUpdateEvent
	done                            chan interface{}
	reconnectCh                     chan interface{}
	RestartCh                       chan interface{}
	stopped                         int32
}

func (w *UserWebsocket) readLoop(conn *websocket.Conn) {
	logger.Debugf("START readLoop")
	defer logger.Debugf("EXIT readLoop")
	for {
		err := conn.SetReadDeadline(time.Now().Add(time.Hour * 4))
		if err != nil {
			logger.Warnf("conn.SetReadDeadline error %v", err)
			w.restart()
			return
		}
		_, r, err := conn.NextReader()
		if err != nil {
			logger.Warnf("conn.NextReader error %v", err)
			w.restart()
			return
		}
		msg, err := w.readAll(r)
		if err != nil {
			logger.Warnf("w.readAll error %v", err)
			w.restart()
			return
		}
		select {
		case w.messageCh <- msg:
		default:
			logger.Debugf("w.messageCh <- msg failed len(w.messageCh) = %d %s", len(w.messageCh), msg)
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
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case msg := <-w.messageCh:
			//{"e":"ACCOUNT_UPDATE","T":1616821544492,"E":1616821544496,"a":{"B":[{"a":"BNB","wb":"0.06858897","cw":"0"}],"P":[],"m":"DEPOSIT"}}
			if msg[0] == '{' && len(msg) > 14 {
				if msg[2] == 'e' && msg[6] == 'A' && msg[14] == 'U' {
					balanceAndPositionUpdateEvent := BalanceAndPositionUpdateEvent{}
					//logger.Debugf("%s", msg)
					err := json.Unmarshal(msg, &balanceAndPositionUpdateEvent)
					if err != nil {
						logger.Debugf("json.Unmarshal error %v %s", err, msg)
						break
					}
					select {
					case w.BalanceAndPositionUpdateEventCh <- &balanceAndPositionUpdateEvent:
					default:
						logger.Warnf("w.BalanceAndPositionUpdateEventCh <- &balanceAndPositionUpdateEvent failed, len(w.BalanceAndPositionUpdateEventCh) = %d %s", len(w.BalanceAndPositionUpdateEventCh), msg)
					}

				} else if msg[2] == 'e' && msg[6] == 'O' {
					//{"e":"ORDER_TRADE_UPDATE","T":1616821790804,"E":1616821790808,"o":{"s":"LTCUSDT","c":"web_g5yhWZ53GcE18wViaj4O","S":"SELL","o":"LIMIT","f":"GTC","q":"0.100","p":"200","ap":"0","sp":"0","x":"NEW","X":"NEW","i":11207370007,"l":"0","z":"0","L":"0","T":1616821790804,"t":0,"b":"0","a":"20","m":false,"R":false,"wt":"CONTRACT_PRICE","ot":"LIMIT","ps":"BOTH","cp":false,"rp":"0","pP":false,"si":0,"ss":0}}
					orderUpdateEvent := OrderUpdateEvent{}
					err := json.Unmarshal(msg, &orderUpdateEvent)
					if err != nil {
						logger.Debugf("json.Unmarshal error %v msg %s", err, msg)
						break
					}
					select {
					case w.OrderUpdateEventCh <- &orderUpdateEvent:
					default:
						logger.Debugf("w.OrderUpdateEventCh <- &orderUpdateEvent failed, len(w.OrderUpdateEventCh) = %d msg %s", len(w.OrderUpdateEventCh), msg)
					}
				} else {
					logger.Debugf("other msg %s", msg)
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

func (w *UserWebsocket) mainLoop(ctx context.Context, urlStr string, proxy string) {
	logger.Debugf("START mainLoop")
	var internalCtx context.Context
	var internalCancel context.CancelFunc
	defer func() {
		logger.Debugf("EXIT mainLoop")
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
			reconnectTimer.Reset(time.Second * 15)
		case <-reconnectTimer.C:
			if internalCancel != nil {
				internalCancel()
				internalCancel = nil
			}
			internalCtx, internalCancel = context.WithCancel(ctx)
			conn, err := w.reconnect(internalCtx, urlStr, proxy, 0)
			if err != nil {
				internalCancel()
				internalCancel = nil
				logger.Debugf("w.reconnect error %v, stop ws", err)
				w.Stop()
				return
			}
			go w.readLoop(conn)
			go w.heartbeatLoop(internalCtx, conn)
			reconnectTimer.Reset(time.Hour * 9999)
		}
	}
}

func (w *UserWebsocket) heartbeatLoop(ctx context.Context, conn *websocket.Conn) {
	logger.Debugf("START heartbeatLoop")
	defer func() {
		logger.Debugf("EXIT heartbeatLoop")
		err := conn.Close()
		if err != nil {
			logger.Debugf("conn.Close() ERROR %v", err)
		}
	}()

	trafficCh := make(chan interface{})

	conn.SetPingHandler(func(msg string) error {
		select {
		case trafficCh <- nil:
		default:
		}
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

	timer := time.NewTimer(time.Minute * 15)

	for {
		select {
		case <-timer.C:
			logger.Warnf("no traffic in %v, restart ws", time.Minute*15)
			w.restart()
			return
		case <-trafficCh:
			timer.Reset(time.Minute * 15)
		case <-ctx.Done():
			return
		case <-w.done:
			return
		}
	}

}

func (w *UserWebsocket) Stop() {
	if atomic.CompareAndSwapInt32(&w.stopped, 0, 1) {
		close(w.done)
	}
}

func (w *UserWebsocket) restart() {
	select {
	case w.RestartCh <- nil:
	default:
		logger.Debugf("w.RestartCh <- nil failed")
	}
	select {
	case <-w.done:
	case <-time.After(time.Second):
		logger.Debugf("w.reconnectCh <- nil timeout in 1s, stop ws")
		w.Stop()
	case w.reconnectCh <- nil:
		logger.Debugf("restart")
	}
}

func (w *UserWebsocket) Done() chan interface{} {
	return w.done
}

func NewUserWebsocket(
	ctx context.Context,
	api *API,
	proxy string,
) (*UserWebsocket, error) {
	var listenKey ListenKey
	err := api.SendAuthenticatedHTTPRequest(
		ctx,
		http.MethodPost,
		"/fapi/v1/listenKey",
		nil,
		&listenKey,
	)
	if err != nil {
		return nil, err
	}
	wsUrl := "wss://fstream.binance.com/ws/" + listenKey.ListenKey
	ws := UserWebsocket{
		done:                            make(chan interface{}),
		reconnectCh:                     make(chan interface{}),
		RestartCh:                       make(chan interface{}, common.ChannelSizeLowLoad),
		OrderUpdateEventCh:              make(chan *OrderUpdateEvent, common.ChannelSizeMediumLoad),
		BalanceAndPositionUpdateEventCh: make(chan *BalanceAndPositionUpdateEvent, common.ChannelSizeMediumLoad),
		messageCh:                       make(chan []byte, common.ChannelSizeHighLoad),
		stopped:                         0,
	}
	go func(ctx context.Context, ws *UserWebsocket, listenKey ListenKey) {
		timer := time.NewTimer(time.Minute * 20)
		retryCounter := 0
		for {
			select {
			case <-ctx.Done():
				return
			case <-ws.Done():
				return
			case <-timer.C:
				var resp interface{}
				ctx, _ := context.WithTimeout(ctx, time.Minute)
				err := api.SendAuthenticatedHTTPRequest(
					ctx,
					http.MethodPut,
					"/fapi/v1/listenKey",
					&listenKey,
					&resp,
				)
				if err != nil {
					if strings.Contains(err.Error(), "connection reset by peer") && retryCounter < 10 {
						retryCounter++
						timer.Reset(time.Second * 15)
						continue
					}
					logger.Debugf("UPDATE LISTEN KEY FAILED %v, STOP WS!", err)
					ws.Stop()
					return
				}
				retryCounter = 0
				timer.Reset(time.Minute * 15)
			}
		}
	}(ctx, &ws, listenKey)
	go ws.mainLoop(ctx, wsUrl, proxy)
	go ws.dataHandleLoop(ctx)
	ws.reconnectCh <- nil
	return &ws, nil
}
