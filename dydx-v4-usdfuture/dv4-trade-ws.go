package dydx_v4_usdfuture

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/gorilla/websocket"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"sync/atomic"
	"time"
)

type V4TradeWS struct {
	writeCh       chan interface{}
	done          chan interface{}
	reconnectCh   chan interface{}
	marketCh      chan string
	marketResetCh chan string
	stopped       int32
}

func (w *V4TradeWS) writeLoop(ctx context.Context, conn *websocket.Conn) {
	logger.Debugf("START V4TradeWS writeLoop")
	defer logger.Debugf("EXIT V4TradeWS writeLoop")
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case msg := <-w.writeCh:
			msgBytes, err := json.Marshal(msg)
			if err != nil {
				continue
			}
			err = conn.SetWriteDeadline(time.Now().Add(time.Minute))
			if err != nil {
				w.restart()
				return
			}
			err = conn.WriteMessage(websocket.TextMessage, msgBytes)
			if err != nil {
				w.restart()
				return
			}
		}
	}
}

func (w *V4TradeWS) readLoop(conn *websocket.Conn, channels map[string]chan []byte) {
	logger.Debugf("START V4TradeWS readLoop")
	defer logger.Debugf("EXIT V4TradeWS readLoop")
	logSilentTime := time.Now()
	for {
		err := conn.SetReadDeadline(time.Now().Add(time.Minute * 2))
		if err != nil {
			w.restart()
			return
		}
		_, msg, err := conn.ReadMessage()
		if err != nil {
			logger.Warnf("V4TradeWS ReadMessage error %v", err)
			w.restart()
			return
		}
		if len(msg) < 50 {
			continue
		}
		var envelope WSMessage
		err = json.Unmarshal(msg, &envelope)
		if err != nil {
			continue
		}
		if envelope.Channel != WsChannelTrades {
			continue
		}
		market := envelope.ID
		if ch, ok := channels[market]; ok {
			select {
			case ch <- msg:
			default:
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("trade ch <- msg %s ch len %d", market, len(ch))
					logSilentTime = time.Now().Add(time.Minute)
				}
			}
		}
	}
}

func (w *V4TradeWS) reconnect(ctx context.Context, proxy string, counter int64) (*websocket.Conn, error) {
	for {
		if counter != 0 {
			logger.Debugf("V4TradeWS reconnect %d retries", counter)
		}
		var dialer *websocket.Dialer
		if proxy != "" {
			proxyUrl, err := url.Parse(proxy)
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

func (w *V4TradeWS) mainLoop(ctx context.Context, proxy string, channels map[string]chan []byte) {
	logger.Debugf("START V4TradeWS mainLoop")
	defer logger.Debugf("EXIT V4TradeWS mainLoop")
	ctx, cancel := context.WithCancel(ctx)
	var internalCtx context.Context
	var internalCancel context.CancelFunc
	symbols := make([]string, 0)
	for symbol := range channels {
		symbols = append(symbols, symbol)
	}
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
			conn, err := w.reconnect(internalCtx, proxy, 0)
			if err != nil {
				internalCancel()
				w.Stop()
				return
			}
			go w.readLoop(conn, channels)
			go w.writeLoop(internalCtx, conn)
			go w.heartbeatLoop(internalCtx, conn, symbols)
		}
	}
}

func (w *V4TradeWS) heartbeatLoop(ctx context.Context, conn *websocket.Conn, markets []string) {
	logger.Debugf("START V4TradeWS heartbeatLoop")
	defer func() {
		logger.Debugf("EXIT V4TradeWS heartbeatLoop")
		conn.Close()
	}()
	marketCheckInterval := time.Second * 5
	marketResetInterval := time.Minute * 30
	marketCheckTimer := time.NewTimer(time.Second)
	defer marketCheckTimer.Stop()

	marketResetTimes := make(map[string]time.Time)
	marketUpdateTimes := make(map[string]time.Time)
	for _, market := range markets {
		marketResetTimes[market] = time.Now().Add(time.Duration(rand.Intn(int(marketResetInterval/time.Second)))*time.Second + marketCheckInterval)
		marketUpdateTimes[market] = time.Unix(0, 0)
	}
	trafficTimeout := time.NewTimer(time.Minute * 5)
	defer trafficTimeout.Stop()

	conn.SetPingHandler(func(msg string) error {
		trafficTimeout.Reset(time.Second * 30)
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
			logger.Debugf("V4TradeWS traffic timeout, restart")
			w.restart()
			return
		case market := <-w.marketCh:
			trafficTimeout.Reset(time.Minute)
			marketUpdateTimes[market] = time.Now().Add(time.Minute * 2)
		case market := <-w.marketResetCh:
			marketResetTimes[market] = time.Now()
		case <-marketCheckTimer.C:
			for market := range marketUpdateTimes {
				if time.Now().Sub(marketUpdateTimes[market]) > 0 ||
					time.Now().Sub(marketResetTimes[market]) > 0 {
					w.writeCh <- WSSubscribe{Type: "unsubscribe", Channel: WsChannelTrades, ID: market}
					w.writeCh <- WSSubscribe{Type: "subscribe", Channel: WsChannelTrades, ID: market}
					marketUpdateTimes[market] = time.Now().Add(time.Minute * 2)
					marketResetTimes[market] = time.Now().Add(time.Duration(rand.Intn(int(marketCheckInterval/time.Second)))*time.Second + marketResetInterval)
				}
			}
			marketCheckTimer.Reset(marketCheckInterval)
		}
	}
}

func (w *V4TradeWS) Stop() {
	if atomic.CompareAndSwapInt32(&w.stopped, 0, 1) {
		close(w.done)
		logger.Debugf("V4TradeWS stopped")
	}
}

func (w *V4TradeWS) restart() {
	select {
	case w.reconnectCh <- nil:
	default:
	}
}

func (w *V4TradeWS) Done() chan interface{} {
	return w.done
}

func (w *V4TradeWS) dataHandleLoop(ctx context.Context, market string, inputCh chan []byte, outputCh chan common.Trade) {
	logger.Debugf("START V4TradeWS dataHandleLoop %s", market)
	defer logger.Debugf("EXIT V4TradeWS dataHandleLoop %s", market)
	logSilentTime := time.Now()
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case msg := <-inputCh:
			var envelope WSMessage
			err := json.Unmarshal(msg, &envelope)
			if err != nil {
				continue
			}
			if envelope.Type != "channel_data" && envelope.Type != "subscribed" {
				continue
			}
			var tradeUpdate WSTradeUpdate
			err = json.Unmarshal(envelope.Contents, &tradeUpdate)
			if err != nil {
				continue
			}
			select {
			case w.marketCh <- market:
			default:
			}
			for _, entry := range tradeUpdate.Trades {
				price, _ := strconv.ParseFloat(entry.Price, 64)
				size, _ := strconv.ParseFloat(entry.Size, 64)
				createdAt, _ := time.Parse(time.RFC3339Nano, entry.CreatedAt)
				trade := &V4Trade{
					Symbol:    market,
					Price:     price,
					Size:      size,
					Side:      entry.Side,
					CreatedAt: createdAt,
				}
				select {
				case outputCh <- trade:
				default:
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("outputCh <- trade failed %s ch len %d", market, len(outputCh))
						logSilentTime = time.Now().Add(time.Minute)
					}
				}
			}
		}
	}
}

func NewV4TradeWS(
	ctx context.Context,
	proxy string,
	channels map[string]chan common.Trade,
) *V4TradeWS {
	ws := V4TradeWS{
		done:          make(chan interface{}),
		reconnectCh:   make(chan interface{}, common.ChannelSizeLowLoad),
		writeCh:       make(chan interface{}, len(channels)*common.ChannelSizeLowLoad),
		marketCh:      make(chan string, len(channels)*common.ChannelSizeLowLoad),
		marketResetCh: make(chan string, len(channels)*common.ChannelSizeLowLoad),
		stopped:       0,
	}
	messagesCh := make(map[string]chan []byte)
	for market, ch := range channels {
		messagesCh[market] = make(chan []byte, common.ChannelSizeLowDropRatio)
		go ws.dataHandleLoop(ctx, market, messagesCh[market], ch)
	}
	go ws.mainLoop(ctx, proxy, messagesCh)
	ws.reconnectCh <- nil
	return &ws
}
