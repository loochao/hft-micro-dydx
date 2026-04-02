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

// V4TickerWS uses the v4_orderbook channel to derive BBO ticker updates
// (same approach as dydx v3 TickerWS)
type V4TickerWS struct {
	writeCh       chan interface{}
	done          chan interface{}
	reconnectCh   chan interface{}
	marketCh      chan string
	marketResetCh chan string
	stopped       int32
}

func (w *V4TickerWS) writeLoop(ctx context.Context, conn *websocket.Conn) {
	logger.Debugf("START V4TickerWS writeLoop")
	defer logger.Debugf("EXIT V4TickerWS writeLoop")
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case msg := <-w.writeCh:
			msgBytes, err := json.Marshal(msg)
			if err != nil {
				logger.Debugf("json.Marshal error %v", err)
				continue
			}
			err = conn.SetWriteDeadline(time.Now().Add(time.Minute))
			if err != nil {
				logger.Debugf("conn.SetWriteDeadline error %v", err)
				w.restart()
				return
			}
			err = conn.WriteMessage(websocket.TextMessage, msgBytes)
			if err != nil {
				logger.Debugf("conn.WriteMessage error %v", err)
				w.restart()
				return
			}
		}
	}
}

func (w *V4TickerWS) readLoop(conn *websocket.Conn, channels map[string]chan []byte) {
	logger.Debugf("START V4TickerWS readLoop")
	defer logger.Debugf("EXIT V4TickerWS readLoop")
	logSilentTime := time.Now()
	for {
		err := conn.SetReadDeadline(time.Now().Add(time.Minute * 2))
		if err != nil {
			logger.Debugf("conn.SetReadDeadline error %v", err)
			w.restart()
			return
		}
		_, msg, err := conn.ReadMessage()
		if err != nil {
			logger.Warnf("V4TickerWS conn.ReadMessage error %v", err)
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
		if envelope.Channel != WsChannelOrderbook {
			continue
		}
		market := envelope.ID
		if ch, ok := channels[market]; ok {
			select {
			case ch <- msg:
			default:
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("ch <- msg %s ch len %d", market, len(ch))
					logSilentTime = time.Now().Add(time.Minute)
				}
			}
		}
	}
}

func (w *V4TickerWS) reconnect(ctx context.Context, proxy string, counter int64) (*websocket.Conn, error) {
	for {
		if counter != 0 {
			logger.Debugf("V4TickerWS reconnect %d retries", counter)
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
		conn, _, err := dialer.DialContext(ctx, IndexerWsURL, http.Header{
			"User-Agent": []string{"Mozilla/5.0"},
		})
		if err != nil {
			logger.Debugf("dialer.DialContext ERROR %v", err)
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

func (w *V4TickerWS) mainLoop(ctx context.Context, proxy string, channels map[string]chan []byte) {
	logger.Debugf("START V4TickerWS mainLoop")
	defer logger.Debugf("EXIT V4TickerWS mainLoop")
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
				logger.Debugf("w.reconnect error %v", err)
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

func (w *V4TickerWS) heartbeatLoop(ctx context.Context, conn *websocket.Conn, markets []string) {
	logger.Debugf("START V4TickerWS heartbeatLoop")
	defer func() {
		logger.Debugf("EXIT V4TickerWS heartbeatLoop")
		conn.Close()
	}()
	marketTimeout := time.Minute
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
		err := conn.WriteControl(websocket.PongMessage, []byte(msg), time.Now().Add(time.Minute))
		if err != nil {
			select {
			case <-ctx.Done():
			default:
				go w.restart()
			}
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
			logger.Debugf("V4TickerWS traffic timeout, restart")
			w.restart()
			return
		case market := <-w.marketCh:
			trafficTimeout.Reset(time.Second * 30)
			marketUpdateTimes[market] = time.Now().Add(marketTimeout)
		case market := <-w.marketResetCh:
			marketResetTimes[market] = time.Now()
		case <-marketCheckTimer.C:
			for market := range marketUpdateTimes {
				if time.Now().Sub(marketUpdateTimes[market]) > 0 ||
					time.Now().Sub(marketResetTimes[market]) > 0 {
					select {
					case w.writeCh <- WSSubscribe{Type: "unsubscribe", Channel: WsChannelOrderbook, ID: market}:
					default:
					}
					select {
					case w.writeCh <- WSSubscribe{Type: "subscribe", Channel: WsChannelOrderbook, ID: market}:
					default:
					}
					marketUpdateTimes[market] = time.Now().Add(marketTimeout)
					marketResetTimes[market] = time.Now().Add(time.Duration(rand.Intn(int(marketCheckInterval/time.Second)))*time.Second + marketResetInterval)
				}
			}
			marketCheckTimer.Reset(marketCheckInterval)
		}
	}
}

func (w *V4TickerWS) Stop() {
	if atomic.CompareAndSwapInt32(&w.stopped, 0, 1) {
		close(w.done)
		logger.Debugf("V4TickerWS stopped")
	}
}

func (w *V4TickerWS) restart() {
	select {
	case w.reconnectCh <- nil:
	default:
	}
}

func (w *V4TickerWS) Done() chan interface{} {
	return w.done
}

func (w *V4TickerWS) dataHandleLoop(ctx context.Context, market string, inputCh chan []byte, outputCh chan common.Ticker) {
	logger.Debugf("START V4TickerWS dataHandleLoop %s", market)
	defer logger.Debugf("EXIT V4TickerWS dataHandleLoop %s", market)
	logSilentTime := time.Now()
	hour999 := time.Hour * 999
	resubDelay := time.Second * 15
	resubTimer := time.NewTimer(hour999)
	defer resubTimer.Stop()

	depth := &V4Depth{Market: market}
	msgCounter := 0

	index := -1
	pool := [common.BufferSizeForRealTimeData]*V4Depth{}
	for i := 0; i < common.BufferSizeForRealTimeData; i++ {
		pool[i] = &V4Depth{}
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case <-resubTimer.C:
			select {
			case w.marketResetCh <- market:
			default:
			}
			logger.Debugf("%s RESUB SENT", market)
			resubTimer.Reset(hour999)
		case msg := <-inputCh:
			var envelope WSMessage
			err := json.Unmarshal(msg, &envelope)
			if err != nil {
				continue
			}
			switch envelope.Type {
			case "subscribed":
				var snapshot WSOrderbookSnapshot
				err = json.Unmarshal(envelope.Contents, &snapshot)
				if err != nil {
					logger.Debugf("parse snapshot error %v", err)
					continue
				}
				depth.Bids = make(common.Bids, 0, len(snapshot.Bids))
				for _, b := range snapshot.Bids {
					price, _ := strconv.ParseFloat(b.Price, 64)
					size, _ := strconv.ParseFloat(b.Size, 64)
					if size > 0 {
						depth.Bids = append(depth.Bids, [2]float64{price, size})
					}
				}
				depth.Asks = make(common.Asks, 0, len(snapshot.Asks))
				for _, a := range snapshot.Asks {
					price, _ := strconv.ParseFloat(a.Price, 64)
					size, _ := strconv.ParseFloat(a.Size, 64)
					if size > 0 {
						depth.Asks = append(depth.Asks, [2]float64{price, size})
					}
				}
				depth.WithSnapshotData = true
				depth.ParseTime = time.Now()
				msgCounter = 0
			case "channel_data":
				if !depth.WithSnapshotData {
					continue
				}
				var update WSOrderbookUpdate
				err = json.Unmarshal(envelope.Contents, &update)
				if err != nil {
					continue
				}
				for _, b := range update.Bids {
					price, _ := strconv.ParseFloat(b.Price, 64)
					size, _ := strconv.ParseFloat(b.Size, 64)
					depth.Bids = depth.Bids.Update([2]float64{price, size})
				}
				for _, a := range update.Asks {
					price, _ := strconv.ParseFloat(a.Price, 64)
					size, _ := strconv.ParseFloat(a.Size, 64)
					depth.Asks = depth.Asks.Update([2]float64{price, size})
				}
				depth.ParseTime = time.Now()
			default:
				continue
			}

			if !depth.IsValid() {
				if msgCounter > 0 {
					resubTimer.Reset(resubDelay)
					msgCounter = -4
				} else {
					msgCounter -= 4
				}
				if len(depth.Bids) > 0 {
					depth.Bids = depth.Bids.Update([2]float64{depth.Bids[0][0], 0.0})
				}
				if len(depth.Asks) > 0 {
					depth.Asks = depth.Asks.Update([2]float64{depth.Asks[0][0], 0.0})
				}
			} else if depth.Asks[0][0] > depth.Bids[0][0] {
				msgCounter++
				if msgCounter >= 0 {
					index++
					if index == common.BufferSizeForRealTimeData {
						index = 0
					}
					outputDepth := pool[index]
					*outputDepth = *depth
					select {
					case outputCh <- outputDepth:
					default:
						if time.Now().Sub(logSilentTime) > 0 {
							logger.Debugf("outputCh <- ticker failed, ch len %d", len(outputCh))
							logSilentTime = time.Now().Add(time.Minute)
						}
					}
				}
				select {
				case w.marketCh <- market:
				default:
				}
				resubTimer.Reset(hour999)
			}
		}
	}
}

func NewV4TickerWS(
	ctx context.Context,
	proxy string,
	channels map[string]chan common.Ticker,
) *V4TickerWS {
	ws := V4TickerWS{
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
