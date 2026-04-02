package dydx_v4_usdfuture

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/gorilla/websocket"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"
)

type V4AccountWS struct {
	messageCh   chan []byte
	AccountCh   chan Subaccount
	PositionsCh chan []V4Position
	OrdersCh    chan []V4Order
	RestartCh   chan interface{}
	writeCh     chan interface{}
	done        chan interface{}
	reconnectCh chan interface{}
	trafficCh   chan interface{}
	stopped     int32
	address     string
	subaccountNumber int
	proxy       string
}

func (w *V4AccountWS) writeLoop(ctx context.Context, conn *websocket.Conn) {
	logger.Debugf("START V4AccountWS writeLoop")
	defer logger.Debugf("EXIT V4AccountWS writeLoop")
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case msg := <-w.writeCh:
			msgBytes, err := json.Marshal(msg)
			if err != nil {
				logger.Warnf("json.Marshal error %v", err)
				continue
			}
			err = conn.SetWriteDeadline(time.Now().Add(time.Second * 5))
			if err != nil {
				w.restart()
				return
			}
			err = conn.WriteMessage(websocket.TextMessage, msgBytes)
			if err != nil {
				logger.Warnf("conn.WriteMessage error %v", err)
				w.restart()
				return
			}
		}
	}
}

func (w *V4AccountWS) readLoop(conn *websocket.Conn) {
	logger.Debugf("START V4AccountWS readLoop")
	defer logger.Debugf("EXIT V4AccountWS readLoop")
	for {
		err := conn.SetReadDeadline(time.Now().Add(time.Hour))
		if err != nil {
			w.restart()
			return
		}
		_, msg, err := conn.ReadMessage()
		if err != nil {
			logger.Warnf("V4AccountWS ReadMessage error %v", err)
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

func (w *V4AccountWS) dataHandleLoop(ctx context.Context) {
	logger.Debugf("START V4AccountWS dataHandleLoop")
	defer logger.Debugf("EXIT V4AccountWS dataHandleLoop")
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case msg := <-w.messageCh:
			var envelope WSMessage
			err := json.Unmarshal(msg, &envelope)
			if err != nil {
				logger.Debugf("unmarshal error %v", err)
				continue
			}
			if envelope.Channel != WsChannelSubaccounts {
				continue
			}
			switch envelope.Type {
			case "subscribed":
				var subscribed WSSubaccountSubscribed
				err = json.Unmarshal(envelope.Contents, &subscribed)
				if err != nil {
					logger.Debugf("parse subscribed error %v", err)
					continue
				}
				if len(subscribed.Orders) > 0 {
					select {
					case w.OrdersCh <- subscribed.Orders:
					default:
						logger.Debugf("w.OrdersCh <- orders failed, len %d", len(w.OrdersCh))
					}
				}
				if len(subscribed.Subaccount.OpenPerpetualPositions) > 0 {
					ps := make([]V4Position, 0)
					for _, pos := range subscribed.Subaccount.OpenPerpetualPositions {
						ps = append(ps, pos)
					}
					select {
					case w.PositionsCh <- ps:
					default:
						logger.Debugf("w.PositionsCh <- ps failed, len %d", len(w.PositionsCh))
					}
				}
				select {
				case w.AccountCh <- subscribed.Subaccount:
				default:
					logger.Debugf("w.AccountCh <- account failed, len %d", len(w.AccountCh))
				}
				select {
				case w.trafficCh <- nil:
				default:
				}

			case "channel_data":
				var update WSSubaccountUpdate
				err = json.Unmarshal(envelope.Contents, &update)
				if err != nil {
					logger.Debugf("parse channel_data error %v", err)
					continue
				}
				if len(update.Orders) > 0 {
					select {
					case w.OrdersCh <- update.Orders:
					default:
						logger.Debugf("w.OrdersCh <- orders failed, len %d", len(w.OrdersCh))
					}
				}
				if len(update.Positions) > 0 {
					select {
					case w.PositionsCh <- update.Positions:
					default:
						logger.Debugf("w.PositionsCh <- positions failed, len %d", len(w.PositionsCh))
					}
				}
				if update.Subaccount != nil {
					select {
					case w.AccountCh <- *update.Subaccount:
					default:
						logger.Debugf("w.AccountCh <- account failed, len %d", len(w.AccountCh))
					}
				}
				select {
				case w.trafficCh <- nil:
				default:
				}

			case "error":
				logger.Warnf("V4AccountWS error: %s", envelope.Message)
			}
		}
	}
}

func (w *V4AccountWS) reconnect(ctx context.Context, counter int64) (*websocket.Conn, error) {
	for {
		if counter != 0 {
			logger.Debugf("V4AccountWS reconnect %d retries", counter)
		}
		var dialer *websocket.Dialer
		if w.proxy != "" {
			proxyUrl, err := url.Parse(w.proxy)
			if err != nil {
				return nil, fmt.Errorf("url.Parse error %v", err)
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
		conn, _, err := dialer.DialContext(ctx, IndexerWsURL, http.Header{
			"User-Agent": []string{"Mozilla/5.0"},
		})
		if err != nil {
			logger.Warnf("V4AccountWS dial error %v", err)
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("context done")
			case <-w.done:
				return nil, fmt.Errorf("ws done")
			case <-time.After(time.Second * 10):
				counter++
				continue
			}
		}
		return conn, nil
	}
}

func (w *V4AccountWS) mainLoop(ctx context.Context) {
	logger.Debugf("START V4AccountWS mainLoop")
	defer logger.Debugf("EXIT V4AccountWS mainLoop")
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
			conn, err := w.reconnect(internalCtx, 0)
			if err != nil {
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

func (w *V4AccountWS) heartbeatLoop(ctx context.Context, conn *websocket.Conn) {
	logger.Debugf("START V4AccountWS heartbeatLoop")
	defer func() {
		logger.Debugf("EXIT V4AccountWS heartbeatLoop")
		conn.Close()
	}()
	subscribeInterval := time.Minute * 15
	subscribeTime := time.Now()
	checkTimer := time.NewTimer(time.Second)
	defer checkTimer.Stop()
	trafficTimeout := time.NewTimer(time.Minute * 5)
	defer trafficTimeout.Stop()

	conn.SetPingHandler(func(msg string) error {
		trafficTimeout.Reset(time.Minute)
		conn.WriteControl(websocket.PongMessage, []byte(msg), time.Now().Add(time.Minute))
		return nil
	})

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case <-trafficTimeout.C:
			logger.Debugf("V4AccountWS traffic timeout, restart")
			w.restart()
			return
		case <-w.trafficCh:
			subscribeTime = time.Now().Add(subscribeInterval)
			trafficTimeout.Reset(time.Minute)
		case <-checkTimer.C:
			checkTimer.Reset(time.Second)
			if time.Now().Sub(subscribeTime) > 0 {
				subscribeTime = time.Now().Add(time.Minute)
				// v4 subaccounts channel: no auth needed, just address/subaccountNumber
				subID := fmt.Sprintf("%s/%d", w.address, w.subaccountNumber)
				select {
				case w.writeCh <- WSSubscribe{
					Type:    "subscribe",
					Channel: WsChannelSubaccounts,
					ID:      subID,
				}:
				default:
					logger.Debugf("w.writeCh <- subscribe failed, ch len %d", len(w.writeCh))
				}
			}
		}
	}
}

func (w *V4AccountWS) Stop() {
	if atomic.CompareAndSwapInt32(&w.stopped, 0, 1) {
		close(w.done)
		logger.Infof("V4AccountWS stopped")
	}
}

func (w *V4AccountWS) restart() {
	select {
	case w.RestartCh <- nil:
	default:
	}
	select {
	case <-w.done:
	case <-time.After(time.Millisecond):
		w.Stop()
	case w.reconnectCh <- nil:
		logger.Debugf("V4AccountWS restart")
	}
}

func (w *V4AccountWS) Done() chan interface{} {
	return w.done
}

func NewV4AccountWS(
	ctx context.Context,
	address string,
	subaccountNumber int,
	proxy string,
) *V4AccountWS {
	ws := V4AccountWS{
		done:             make(chan interface{}),
		reconnectCh:      make(chan interface{}),
		AccountCh:        make(chan Subaccount, 16),
		PositionsCh:      make(chan []V4Position, 16),
		OrdersCh:         make(chan []V4Order, 16),
		RestartCh:        make(chan interface{}, 16),
		messageCh:        make(chan []byte, 128),
		writeCh:          make(chan interface{}, 128),
		trafficCh:        make(chan interface{}, 128),
		stopped:          0,
		address:          address,
		subaccountNumber: subaccountNumber,
		proxy:            proxy,
	}
	go ws.mainLoop(ctx)
	go ws.dataHandleLoop(ctx)
	ws.reconnectCh <- nil
	return &ws
}
