package bnswap

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/gorilla/websocket"
	"io"
	"net/http"
	"net/url"
	"time"
)

type UserWebsocket struct {
	messageCh                       chan []byte
	BalanceAndPositionUpdateEventCh chan *BalanceAndPositionUpdateEvent
	OrderUpdateEventCh              chan *OrderUpdateEvent
	done                            chan interface{}
	reconnectCh                     chan interface{}
}

func (w *UserWebsocket) startRead(conn *websocket.Conn, readTimeout time.Duration) {
	totalLen := 0
	totalCount := 0
	for {
		err := conn.SetReadDeadline(time.Now().Add(readTimeout))
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
		msg, err := w.readAll(r)
		if err != nil {
			logger.Warnf("readAll error %v", err)
			go w.restart()
			return
		}
		totalCount += 1
		totalLen += len(msg)
		if totalLen > 1000000 {
			logger.Debugf("BNSWAP USER WS AVERAGE LENGTH %d/%d = %d", totalLen, totalCount, totalLen/totalCount)
			totalLen = 0
			totalCount = 0
		}
		select {
		case <-time.After(time.Millisecond):
			logger.Warnf("BNSWAP USER WS MSG TO MESSAGE CH TIMEOUT IN 1MS")
		case w.messageCh <- msg:
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

func (w *UserWebsocket) startDataHandler(ctx context.Context) {
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
					//logger.Debugf("%s", msg)
					balanceAndPositionUpdateEvent := BalanceAndPositionUpdateEvent{}
					err := json.Unmarshal(msg, &balanceAndPositionUpdateEvent)
					if err != nil {
						logger.Debugf("Unmarshal BalanceAndPositionUpdateEvent %v %s",err, msg)
						break
					}
					select {
					case <-ctx.Done():
						return
					case <-w.done:
						return
					case <-time.After(time.Millisecond):
						logger.Warnf("ACCOUNT_UPDATE TO OUTPUT CH TIME OUT IN 1MS %s", msg)
					case w.BalanceAndPositionUpdateEventCh <- &balanceAndPositionUpdateEvent:
					}

				} else if msg[2] == 'e' && msg[6] == 'O' {
					//{"e":"ORDER_TRADE_UPDATE","T":1616821790804,"E":1616821790808,"o":{"s":"LTCUSDT","c":"web_g5yhWZ53GcE18wViaj4O","S":"SELL","o":"LIMIT","f":"GTC","q":"0.100","p":"200","ap":"0","sp":"0","x":"NEW","X":"NEW","i":11207370007,"l":"0","z":"0","L":"0","T":1616821790804,"t":0,"b":"0","a":"20","m":false,"R":false,"wt":"CONTRACT_PRICE","ot":"LIMIT","ps":"BOTH","cp":false,"rp":"0","pP":false,"si":0,"ss":0}}
					//logger.Debugf("%s", msg)
					orderUpdateEvent := OrderUpdateEvent{}
					err := json.Unmarshal(msg, &orderUpdateEvent)
					if err != nil {
						logger.Debugf("Unmarshal OrderUpdateEvent %s", msg)
						logger.Debugf("Unmarshal OrderUpdateEvent %v", err)
						break
					}
					select {
					case <-ctx.Done():
						return
					case <-w.done:
						return
					case <-time.After(time.Millisecond):
						logger.Warnf("ORDER_TRADE_UPDATE TO OUTPUT CH TIME OUT IN 1MS")
					case w.OrderUpdateEventCh <- &orderUpdateEvent:
					}
				} else {
					logger.Debugf("OTHER MSG %s", msg)
				}
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

func (w *UserWebsocket) start(ctx context.Context, urlStr string, proxy string) {
	logger.Debugf("BNSWAP USER WS %s", urlStr)

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
			conn, err := w.reconnect(internalCtx, urlStr, proxy, 0)
			if err != nil {
				logger.Fatalf("RECONNECT ERROR %v", err)
				return
			}
			go w.startRead(conn, time.Hour*24)
			go w.maintainHeartbeat(internalCtx, conn, time.Minute*10)

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

func (w *UserWebsocket) maintainHeartbeat(ctx context.Context, conn *websocket.Conn, timeout time.Duration) {

	defer func() {
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
		//logger.Debugf("BNSWAP USER WS PingHandler %s", msg)
		err := conn.WriteControl(websocket.PongMessage, []byte(msg), time.Now().Add(timeout))
		if err != nil {
			go w.restart()
			return nil
		}
		return nil
	})

	timer := time.NewTimer(timeout)

	for {
		select {
		case <-timer.C:
			logger.Warnf("USER WS TIMEOUT, NO TRAFFIC IN %v, RESTART", timeout)
			go w.restart()
			return
		case <-trafficCh:
			timer.Reset(timeout)
		case <-ctx.Done():
			return
		case <-w.done:
			return
		}
	}

}

func (w *UserWebsocket) Stop() {
	if _, ok := <-w.done; ok {
		close(w.done)
		logger.Infof("BNSWAP USER WS STOPPED")
	}
}

func (w *UserWebsocket) restart() {
	logger.Infof("BNSWAP USER WS RESTART")
	select {
	case <-w.done:
		return
	default:
	}
	timer := time.NewTimer(time.Millisecond)
	defer timer.Stop()
	select {
	case <-timer.C:
		logger.Fatal("NIL TO RECONNECT CH TIMEOUT IN 1MS, EXIT WS")
	case w.reconnectCh <- nil:
		return
	}
}

func (w *UserWebsocket) Done() chan interface{} {
	return w.done
}

func NewUserWebsocket(
	ctx context.Context,
	api *API,
	proxy string,
) *UserWebsocket {
	var listenKey ListenKey
	err := api.SendAuthenticatedHTTPRequest(
		ctx,
		http.MethodPost,
		"/fapi/v1/listenKey",
		nil,
		&listenKey,
	)
	if err != nil {
		logger.Fatal(err)
	}
	wsUrl := "wss://fstream.binance.com/ws/" + listenKey.ListenKey
	ws := UserWebsocket{
		done:                            make(chan interface{}),
		reconnectCh:                     make(chan interface{}),
		OrderUpdateEventCh:              make(chan *OrderUpdateEvent, 10),
		BalanceAndPositionUpdateEventCh: make(chan *BalanceAndPositionUpdateEvent, 10),
		messageCh:                       make(chan []byte, 10),
	}
	go func(ctx context.Context, ws *UserWebsocket, listenKey ListenKey) {
		timer := time.NewTimer(time.Minute * 20)
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
					logger.Fatal(err)
				}
				timer.Reset(time.Minute * 20)
			}
		}
	}(ctx, &ws, listenKey)
	go ws.start(ctx, wsUrl, proxy)
	ws.reconnectCh <- nil
	return &ws
}
