package hbspot

import (
	"compress/gzip"
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

type TradeRoutedWS struct {
	writeCh     chan interface{}
	done        chan interface{}
	reconnectCh chan interface{}
	messageCh   chan []byte
	symbolCh    chan string
	pingCh      chan []byte
	stopped     int32
}

func (w *TradeRoutedWS) writeLoop(ctx context.Context, conn *websocket.Conn) {
	logger.Debugf("START writeLoop")
	defer logger.Debugf("EXIT writeLoop")
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case msg := <-w.writeCh:
			var bytes []byte
			var err error
			switch d := msg.(type) {
			case []byte:
				bytes = d
			case string:
				bytes = ([]byte)(d)
			default:
				bytes, err = json.Marshal(msg)
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

			err = conn.WriteMessage(websocket.TextMessage, bytes)
			if err != nil {
				logger.Debugf("conn.WriteMessage error %v", err)
				w.restart()
				return
			}
		}
	}
}

func (w *TradeRoutedWS) readLoop(conn *websocket.Conn) {
	logger.Debugf("START readLoop")
	defer logger.Debugf("EXIT readLoop")
	logSilentTime := time.Now()
	for {
		err := conn.SetReadDeadline(time.Now().Add(time.Minute))
		if err != nil {
			logger.Debugf("conn.SetReadDeadline error %v", err)
			w.restart()
			return
		}
		_, r, err := conn.NextReader()
		if err != nil {
			logger.Debugf("conn.NextReader error %v", err)
			w.restart()
			return
		}
		gr, err := gzip.NewReader(r)
		if err != nil {
			logger.Debugf("gzip.NewReader error %v", err)
			w.restart()
			return
		}
		msg, err := w.readAll(gr)
		if err != nil {
			logger.Debugf("w.readAll error %v", err)
			w.restart()
			return
		}

		select {
		case w.messageCh <- msg:
		default:
			if time.Now().Sub(logSilentTime) > 0 {
				logSilentTime = time.Now().Add(time.Minute)
				logger.Debugf("w.messageCh <- msg failed ch len %d", len(w.messageCh))
			}
		}
		err = gr.Close()
		if err != nil {
			logger.Debugf("gr.Close() error %v", err)
			go w.restart()
			return
		}
	}
}

func (w *TradeRoutedWS) readAll(r io.Reader) ([]byte, error) {
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

func (w *TradeRoutedWS) reconnect(ctx context.Context, wsUrl string, proxy string, counter int64) (*websocket.Conn, error) {

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

func (w *TradeRoutedWS) mainLoop(ctx context.Context, proxy string, channels map[string]chan common.Trade) {
	logger.Debugf("START mainLoop")
	defer logger.Debugf("EXIT mainLoop")
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
			conn, err := w.reconnect(internalCtx, "wss://api-aws.huobi.pro/ws", proxy, 0)
			if err != nil {
				logger.Debugf("w.reconnect error %v", err)
				internalCancel()
				w.Stop()
				return
			}
			go w.readLoop(conn)
			go w.writeLoop(internalCtx, conn)
			go w.heartbeatLoop(internalCtx, conn, symbols)
		}
	}
}

func (w *TradeRoutedWS) heartbeatLoop(ctx context.Context, conn *websocket.Conn, symbols []string) {
	logger.Debugf("START heartbeatLoop")
	defer func() {
		logger.Debugf("Exit heartbeatLoop")
		err := conn.Close()
		if err != nil {
			logger.Debugf("conn.Close() ERROR %v", err)
		}
	}()

	symbolTimeout := time.Minute
	symbolCheckInterval := time.Second
	symbolCheckTimer := time.NewTimer(time.Second)
	defer symbolCheckTimer.Stop()
	symbolUpdatedTimes := make(map[string]time.Time)
	for _, symbol := range symbols {
		symbolUpdatedTimes[symbol] = time.Unix(0, 0)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case symbol := <-w.symbolCh:
			symbolUpdatedTimes[symbol] = time.Now()
			break
		case msg := <-w.pingCh:
			select {
			case w.writeCh <- msg:
				break
			default:
				logger.Debugf("w.writeCh <- msg failed, ch len %d", len(w.writeCh))
			}
			break
		case <-symbolCheckTimer.C:
		loop:
			for symbol, updateTime := range symbolUpdatedTimes {
				if time.Now().Sub(updateTime) > symbolTimeout {
					logger.Debugf("SUBSCRIBE %s", fmt.Sprintf("market.%s.trade.detail", symbol))
					select {
					case w.writeCh <- SubParam{
						ID:  fmt.Sprintf("market.%s.trade.detail", symbol),
						Sub: fmt.Sprintf("market.%s.trade.detail", symbol),
					}:
						symbolUpdatedTimes[symbol] = time.Now().Add(symbolCheckInterval * time.Duration(len(symbols)*2))
						break loop
					default:
						logger.Debugf("w.writeCh <- SubParam failed, ch len %d", len(w.writeCh))
					}
				}
			}
			symbolCheckTimer.Reset(symbolCheckInterval)
			break
		}
	}

}

func (w *TradeRoutedWS) Stop() {
	if atomic.LoadInt32(&w.stopped) == 0 {
		atomic.StoreInt32(&w.stopped, 1)
		close(w.done)
		logger.Debugf("stopped")
	}
}

func (w *TradeRoutedWS) restart() {
	select {
	case w.reconnectCh <- nil:
	default:
		logger.Debugf("w.reconnectCh <- nil failed, ch len %d", len(w.reconnectCh))
	}
}

func (w *TradeRoutedWS) Done() chan interface{} {
	return w.done
}

func (w *TradeRoutedWS) dataHandleLoop(ctx context.Context, id int, channels map[string]chan common.Trade) {
	logger.Debugf("START dataHandleLoop %d", id)
	defer logger.Debugf("EXIT dataHandleLoop %d", id)
	logSilentTime := time.Now()
	var ch chan common.Trade
	var err error
	var ok bool
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case msg := <-w.messageCh:
			if msg[2] == 'c' {
				var trade WsTrade
				err = json.Unmarshal(msg, &trade)
				if err != nil {
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("json.Unmarshal(msg, &trade) error %v %s", err, msg)
						logSilentTime = time.Now().Add(time.Minute)
					}
					continue
				}
				for _, t := range trade.Tick.Data {
					if ch, ok = channels[t.Symbol]; ok {
						select {
						case ch <- t:
						default:
							if time.Now().Sub(logSilentTime) > 0 {
								logger.Debugf("ch <- &trade failed ch len %d", len(ch))
								logSilentTime = time.Now().Add(time.Minute)
							}
						}
						select {
						case w.symbolCh <- t.Symbol:
						default:
							if time.Now().Sub(logSilentTime) > 0 {
								logger.Debugf("w.symbolCh <- t.Market failed ch len %d", len(w.symbolCh))
								logSilentTime = time.Now().Add(time.Minute)
							}
						}
					}
				}
			} else if msg[2] == 'p' {
				msg[3] = 'o'
				select {
				case w.pingCh <- msg:
				default:
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("w.pingCh <- msg failed ch len %d", len(w.pingCh))
						logSilentTime = time.Now().Add(time.Minute)
					}
				}
			} else if msg[2] == 'i' {
			} else {
				logger.Debugf("other msg %s", msg)
			}

		}
	}
}

func NewTradeRoutedWS(
	ctx context.Context,
	proxy string,
	channels map[string]chan common.Trade,
) *TradeRoutedWS {
	ws := TradeRoutedWS{
		done:        make(chan interface{}),
		reconnectCh: make(chan interface{}, 100),
		writeCh:     make(chan interface{}, 100*len(channels)),
		symbolCh:    make(chan string, 100*len(channels)),
		messageCh:   make(chan []byte, 100*len(channels)),
		pingCh:      make(chan []byte, 100),
		stopped:     0,
	}
	go ws.mainLoop(ctx, proxy, channels)
	for i := 0; i < 4; i++ {
		cs := make(map[string]chan common.Trade)
		for symbol, ch := range channels {
			cs[symbol] = ch
		}
		go ws.dataHandleLoop(ctx, i, cs)
	}
	ws.reconnectCh <- nil
	return &ws
}

//   {
//                "amount": 0.0099,
//                "ts": 1533265950234, //trade time
//                "id": 146507451359183894799,
//                "tradeId": 102043495674,
//                "price": 401.74,
//                "direction": "buy"
//            }

type TradeDetail struct {
	Amount    float64   `json:"-"`
	EventTime time.Time `json:"-"`
	Price     float64   `json:"-"`
	Direction string    `json:"direction"`
	TradeID   int64     `json:"-"`
	Symbol    string    `json:"-"`
}

var TradeSideBuy = "buy"
var TradeSideSell = "sell"

func (trade *TradeDetail) UnmarshalJSON(data []byte) error {
	type Alias TradeDetail
	aux := struct {
		EventTime json.RawMessage `json:"ts"`
		Amount    json.RawMessage `json:"amount"`
		Price     json.RawMessage `json:"price"`
		TradeID   json.RawMessage `json:"tradeId"`
		*Alias
	}{Alias: (*Alias)(trade)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	} else {
		eventTime, err := common.ParseInt(aux.EventTime)
		if err != nil {
			return err
		}
		trade.EventTime = time.Unix(0, eventTime*1000000)
		trade.TradeID, err = common.ParseInt(aux.TradeID)
		if err != nil {
			return err
		}
		trade.Price, err = common.ParseFloat(aux.Price)
		if err != nil {
			return err
		}
		trade.Amount, err = common.ParseFloat(aux.Amount)
		if err != nil {
			return err
		}
		return nil
	}
}

func (trade *TradeDetail) GetSymbol() string  { return trade.Symbol }
func (trade *TradeDetail) GetSize() float64   { return trade.Amount }
func (trade *TradeDetail) GetPrice() float64  { return trade.Price }
func (trade *TradeDetail) GetTime() time.Time { return trade.EventTime }
func (trade *TradeDetail) IsUpTick() bool        { return trade.Direction == TradeSideBuy }
