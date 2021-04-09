package okspot

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/gorilla/websocket"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const ()

type Websocket struct {
	cancel         context.CancelFunc
	channelAlertCh chan string
	done           chan interface{}
	WriteCh        chan interface{}
	DataCh         chan interface{}
	reconnectCh    chan interface{}
	Url            string
}

func (w *Websocket) handleData(msg []byte) interface{} {
	var commonCap CommonCapture
	err := common.JSONDecode(msg, &commonCap)
	if err != nil {
		return fmt.Errorf(
			"could not load websocket data response: okspot %s",
			string(msg),
		)
	}
	if commonCap.Table != nil && commonCap.Data != nil {
		return w.handleTable(commonCap)
	}
	if commonCap.Event != nil {
		switch *commonCap.Event {
		case "login":
			var loginEvent LoginEvent
			err = common.JSONDecode(msg, &loginEvent)
			if err != nil {
				return err
			}
			if loginEvent.Success {
				w.channelAlertCh <- "login_success"
			} else {
				w.channelAlertCh <- "login_failed"
			}
			return loginEvent
		case "error":
			var errorEvent ErrorEvent
			err = common.JSONDecode(msg, &errorEvent)
			if err != nil {
				return err
			}
			return errorEvent
		case "subscribe":
			var subscribeEvent SubscribeEvent
			err = common.JSONDecode(msg, &subscribeEvent)
			if err != nil {
				return err
			}
			w.channelAlertCh <- subscribeEvent.Channel
			return subscribeEvent
		}
	}
	return fmt.Errorf(
		"websocket error: Unknow data %s", string(msg),
	)
}

// getChannelWithoutOrderType takes WSDataResponse.Table and returns
// The base channel name eg receive "spot/depth5:BTC-USDT" return "depth5"
func (w *Websocket) getChannelWithoutOrderType(table string) string {
	index := strings.Index(table, ":")
	// Some events do not contain a currency
	if index == -1 {
		return table
	}
	return table[:index]
}

func (w *Websocket) handleTable(commonCap CommonCapture) interface{} {
	switch w.getChannelWithoutOrderType(*commonCap.Table) {
	case okspotWsCandle60s, okspotWsCandle180s, okspotWsCandle300s,
		okspotWsCandle900s, okspotWsCandle1800s, okspotWsCandle3600s,
		okspotWsCandle7200s, okspotWsCandle14400s, okspotWsCandle21600s,
		okspotWsCandle43200s, okspotWsCandle86400s, okspotWsCandle604900s:
		return w.processCandles(commonCap)
	case okspotWsTicker:
		var tickers []WSTicker
		err := common.JSONDecode(*commonCap.Data, &tickers)
		if err != nil {
			return err
		}
		for _, data := range tickers {
			select {
			case w.channelAlertCh <- *commonCap.Table + ":" + data.InstrumentID:
			default:
			}
		}
		return tickers
	case okspotWsTrade:
		var trades []WSTrade
		err := common.JSONDecode(*commonCap.Data, &trades)
		if err != nil {
			return err
		}
		for _, data := range trades {
			select {
			case w.channelAlertCh <- *commonCap.Table + ":" + data.InstrumentID:
			default:
			}
		}
		return trades
	case okspotWsDepth5:
		var depths []WSDepth5
		err := common.JSONDecode(*commonCap.Data, &depths)
		if err != nil {
			return err
		}
		for _, data := range depths {
			select {
			case w.channelAlertCh <- *commonCap.Table + ":" + data.InstrumentID:
			default:
			}
		}
		return depths
	case okspotWsAccount:
		var balances []Balance
		err := common.JSONDecode(*commonCap.Data, &balances)
		//logger.Debugf("WS BALANCES %s", *commonCap.Data)
		if err != nil {
			return err
		}
		for _, balance := range balances {
			select {
			case w.channelAlertCh <- *commonCap.Table + ":" + balance.Currency:
			default:
			}
		}
		return balances
	case okspotWsOrder:
		var orders []WSOrder
		err := common.JSONDecode(*commonCap.Data, &orders)
		//logger.Debugf("WS ORDERS %s", *commonCap.Data)
		if err != nil {
			return err
		}
		for _, order := range orders {
			select {
			case w.channelAlertCh <- *commonCap.Table + ":" + order.InstrumentId:
			default:
			}
		}
		return orders
	}
	return fmt.Errorf(
		"websocket error: Unknow data %s", string(*commonCap.Data),
	)
}

// processCandles converts candle data and sends it to the data handler
func (w *Websocket) processCandles(commonCap CommonCapture) interface{} {
	var candles []Candle
	err := common.JSONDecode(*commonCap.Data, &candles)
	if err != nil {
		return err
	}
	candleIndex := strings.LastIndex(*commonCap.Table, okspotWsCandle)
	secondIndex := strings.LastIndex(*commonCap.Table, "0s")
	candleIntervalStr := ""
	if candleIndex > 0 || secondIndex > 0 {
		candleIntervalStr = (*commonCap.Table)[candleIndex+len(okspotWsCandle) : secondIndex]
	}
	if candleIntervalStr == "" {
		return fmt.Errorf("can't parse interval from \"%s\"", *commonCap.Table)
	}
	interval, err := strconv.ParseFloat(candleIntervalStr, 64)
	if err != nil {
		return err
	}
	candleInterval := time.Second * time.Duration(interval)
	ohlcvsMap := make(map[string]common.OHLCVS, 0)
	for _, candle := range candles {
		w.channelAlertCh <- *commonCap.Table + ":" + candle.InstrumentID
		timestamp, err := time.Parse(okspotTimeLayout, candle.Candle[0])
		if err != nil {
			logger.Errorf(
				"could not be parsed candle time: %v", candle.Candle[0],
			)
			continue
		}
		//timestamp	string is start time
		timestamp = timestamp.Add(candleInterval)
		if _, ok := ohlcvsMap[candle.InstrumentID]; !ok {
			ohlcvsMap[candle.InstrumentID] = common.OHLCVS{
				Symbol: candle.InstrumentID,
				OHLCVS: make([]common.OHLCV, 0),
			}
		}
		ohlcvs := ohlcvsMap[candle.InstrumentID]
		ohlcv := common.OHLCV{
			Timestamp: timestamp,
		}
		ohlcv.Open, err = strconv.ParseFloat(candle.Candle[1], 64)
		if err != nil {
			logger.Errorf("parse open %s error %v", candle.Candle[1], err)
			continue
		}
		ohlcv.High, err = strconv.ParseFloat(candle.Candle[2], 64)
		if err != nil {
			logger.Errorf("parse high %s error %v", candle.Candle[2], err)
			continue
		}
		ohlcv.Low, err = strconv.ParseFloat(candle.Candle[3], 64)
		if err != nil {
			logger.Errorf("parse low %s error %v", candle.Candle[3], err)
			continue
		}
		ohlcv.Close, err = strconv.ParseFloat(candle.Candle[4], 64)
		if err != nil {
			logger.Errorf("parse close %s error %v", candle.Candle[4], err)
			continue
		}
		ohlcv.Volume, err = strconv.ParseFloat(candle.Candle[5], 64)
		if err != nil {
			logger.Errorf("parse volume %s error %v", candle.Candle[5], err)
			continue
		}
		ohlcvs.OHLCVS = append(ohlcvs.OHLCVS, ohlcv)
		ohlcvsMap[candle.InstrumentID] = ohlcvs
	}
	return ohlcvsMap
}

func (w *Websocket) startRead(ctx context.Context, conn *websocket.Conn) {

	for {

		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		default:
		}

		err := conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		if err != nil {
			logger.Warnf("SetReadDeadline error, %v", err)
			select {
			case <-ctx.Done():
				return
			case <-w.done:
				return
			default:
				w.Restart()
				return
			}
		}
		mType, resp, err := conn.ReadMessage()

		if err != nil {
			logger.Warnf("read message error, %v", err)
			select {
			case <-ctx.Done():
				return
			case <-w.done:
				return
			default:
				w.Restart()
				return
			}
		}

		var msg []byte
		switch mType {
		case websocket.TextMessage:
			msg = resp
		case websocket.BinaryMessage:
			msg, err = w.parseBinaryResponse(resp)
			if err != nil {
				logger.Debugf("parseBinaryResponse err %v", err)
			}
		}

		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case w.channelAlertCh <- "":
		}

		if strings.Contains(string(msg), "pong") {
			continue
		}

		if strings.Contains(string(msg), "ping") {
			continue
		}

		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case <-time.After(time.Millisecond * 300):
			logger.Warnf("send msg to read timeout in 300ms")
			logger.Debugf(string(msg))
		case w.DataCh <- w.handleData(msg):
		}
	}
}

/// parseBinaryResponse parses a websocket binary response into a usable byte array
func (w *Websocket) parseBinaryResponse(resp []byte) ([]byte, error) {
	var standardMessage []byte
	var err error
	// Detect GZIP
	if resp[0] == 31 && resp[1] == 139 {
		b := bytes.NewReader(resp)
		var gReader *gzip.Reader
		gReader, err = gzip.NewReader(b)
		if err != nil {
			return standardMessage, err
		}
		standardMessage, err = ioutil.ReadAll(gReader)
		if err != nil {
			return standardMessage, err
		}
		err = gReader.Close()
		if err != nil {
			return standardMessage, err
		}
	} else {
		reader := flate.NewReader(bytes.NewReader(resp))
		standardMessage, err = ioutil.ReadAll(reader)
		if err != nil {
			return standardMessage, err
		}
		err = reader.Close()
		if err != nil {
			return standardMessage, err
		}
	}
	return standardMessage, nil
}

func (w *Websocket) startWrite(ctx context.Context, conn *websocket.Conn) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case msg := <-w.WriteCh:
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
					logger.Warnf("Marshal err %v", err)
					continue
				}
			}

			//logger.Debugf("write %v", string(bytes))

			err = conn.SetWriteDeadline(time.Now().Add(300 * time.Millisecond))
			if err != nil {
				logger.Warnf("SetWriteDeadline error %v", err)
				select {
				case <-ctx.Done():
					return
				case <-w.done:
					return
				default:
					w.Restart()
					return
				}
			}

			err = conn.WriteMessage(websocket.TextMessage, bytes)

			if err != nil {
				logger.Warnf("WriteMessage %s error %v", string(bytes), err)
				select {
				case <-ctx.Done():
					return
				case <-w.done:
					return
				default:
					w.Restart()
					return
				}
			}

			select {
			case <-ctx.Done():
				return
			case <-w.done:
				return
			case w.channelAlertCh <- "":
			}

		}
	}
}

func (w *Websocket) startTrafficMonitor(ctx context.Context, conn *websocket.Conn, credentials *Credentials, channels []string) {
	defer func() {
		err := conn.Close()
		if err != nil {
			logger.Debugf("conn.Close error %v", err)
		}
	}()

	conn.SetPongHandler(func(msg string) error {
		//logger.Debugf("get pong msg %s", msg)
		select {
		case <-ctx.Done():
		case <-w.done:
		case w.channelAlertCh <- "":
		}
		return nil
	})

	trafficTimeout := time.Second * 30
	channelTimeout := time.Minute * 15
	channelCheckInterval := time.Second*15

	conn.SetPingHandler(func(msg string) error {
		//logger.Debugf("get ping msg %s", msg)
		select {
		case <-ctx.Done():
		case <-w.done:
		case w.channelAlertCh <- "":
		}
		deadline := time.Now().Add(10 * time.Second)
		err := conn.WriteControl(websocket.PongMessage, []byte{}, deadline)
		if err != nil {
			go w.Restart()
			return nil
		}
		return nil
	})

	trafficTimer := time.NewTimer(trafficTimeout)
	pingTimer := time.NewTimer(trafficTimeout / 2)
	channelsCheckTimer := time.NewTimer(time.Second)
	loginTimer := time.NewTimer(time.Second)
	defer trafficTimer.Stop()
	defer pingTimer.Stop()
	defer loginTimer.Stop()
	defer channelsCheckTimer.Stop()
	channelsUpdatedTimes := make(map[string]time.Time)
	for _, channel := range channels {
		channelsUpdatedTimes[channel] = time.Unix(0, 0)
	}
	loginSuccess := false

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case <-loginTimer.C:
			if !loginSuccess && credentials != nil {
				unixTime := time.Now().UTC().Unix()
				signPath := "/users/self/verify"
				hmac := common.GetHMAC(common.HashSHA256,
					[]byte(strconv.FormatInt(unixTime, 10)+http.MethodGet+signPath),
					[]byte(credentials.Secret),
				)
				base64 := common.Base64Encode(hmac)
				err := w.SendJSON(ctx, Subscription{
					Op: "login",
					Args: []string{
						credentials.Key,
						credentials.Passphrase,
						strconv.FormatInt(unixTime, 10),
						base64,
					},
				})
				if err != nil {
					logger.Debugf("login error %v, retry in 15s", err)
					loginTimer.Reset(time.Second * 15)
				} else {
					loginTimer.Reset(time.Minute)
				}
			} else {
				loginTimer.Reset(time.Minute * 10)
			}
		case <-channelsCheckTimer.C:
			nowTime := time.Now()
			args := make([]string, 0)
			for channel, t := range channelsUpdatedTimes {
				if !loginSuccess &&
					(common.StringContains(channel, "account") ||
						common.StringContains(channel, "order")) {
					continue
				}
				if nowTime.Add(-channelTimeout).Sub(t).Seconds() > 0 {
					args = append(args, channel)
				}
			}
			if len(args) > 0 {
				cutIndex := 0
				for i := 100; i < len(args); i += 100 {
					cutIndex = i
					err := w.SendJSON(ctx, Subscription{
						Op:   "subscribe",
						Args: args[i-100 : i],
					})
					if err != nil {
						logger.Debugf("SendJSON %s error %v, retry in 15s", args[i-100:i], err)
					}
				}
				if len(args[cutIndex:]) > 0 {
					err := w.SendJSON(ctx, Subscription{
						Op:   "subscribe",
						Args: args[cutIndex:],
					})
					if err != nil {
						logger.Debugf("SendJSON %s error %v, retry in 15s", args[cutIndex:], err)
					}
				}
				channelsCheckTimer.Reset(channelCheckInterval)
			} else {
				channelsCheckTimer.Reset(channelCheckInterval)
			}
		case <-pingTimer.C:
			deadline := time.Now().Add(trafficTimeout / 2)
			err := conn.WriteControl(websocket.PingMessage, []byte{}, deadline)
			if err != nil {
				w.Restart()
			}
			pingTimer.Reset(trafficTimeout / 2)

		case channel := <-w.channelAlertCh:
			//logger.Debugf("channel %v", channel)
			if channel == "login_success" {
				loginSuccess = true
			} else if channel == "login_failure" {
				loginSuccess = false
			} else if _, ok := channelsUpdatedTimes[channel]; ok {
				channelsUpdatedTimes[channel] = time.Now()
			}
			trafficTimer.Reset(trafficTimeout)

		case <-trafficTimer.C:
			logger.Warnf("no traffic in %f seconds, restart", trafficTimeout.Seconds())
			w.Restart()

		}
	}
}

func (w *Websocket) reconnect(ctx context.Context, wsUrl string, proxy string, counter int64) (*websocket.Conn, error) {

	if counter > 0 {
		logger.Debugf("ws %s reconnect, %d retries", wsUrl, counter)
	}

	var dialer *websocket.Dialer

	if proxy != "" {
		proxyUrl, err := url.Parse(proxy)
		if err != nil {
			logger.Fatalf("parse proxy error %v", err)
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
			"Accept-Encoding": []string{"gzip, deflate"},
			"Accept-Language": []string{"en-US,en;q=0.9,zh-CN;q=0.8,zh;q=0.7,fr;q=0.6,nl;q=0.5,zh-TW;q=0.4,vi;q=0.3"},
		},
	)
	if err != nil {
		logger.Warnf("dialer.DialContext error %v", err)
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

func (w *Websocket) start(ctx context.Context, urlStr string, credentials *Credentials, channels []string, proxy string) {
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

	for {
		select {
		case <-ctx.Done():
			logger.Debugf("ctx.Done()")
			return
		case <-w.reconnectCh:
			if internalCancel != nil {
				internalCancel()
			}
			internalCtx, internalCancel = context.WithCancel(ctx)
			conn, err := w.reconnect(internalCtx, urlStr, proxy, 0)
			if err != nil {
				logger.Fatalf("reconnect error %v", err)
				return
			}
			go w.startRead(internalCtx, conn)
			go w.startWrite(internalCtx, conn)
			go w.startTrafficMonitor(internalCtx, conn, credentials, channels)

		}
	}
}

func (w *Websocket) Stop() {
	select {
	case <-w.done:
		return
	default:
	}
	logger.Infof("WebSocket stopped")
	close(w.done)
}

func (w *Websocket) Restart() {
	select {
	case <-w.done:
		return
	default:
	}

	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	select {
	case <-timer.C:
		logger.Fatal("send nil to read timeout in 1s, exit ws")
	case w.DataCh <- nil:
	}

	timer = time.NewTimer(time.Second * 15)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			logger.Fatal("nil to reconnectCh timeout in 15s, exit ws")
		case w.reconnectCh <- nil:
			return
		}
	}

}

func (w *Websocket) SendJSON(ctx context.Context, data interface{}) error {
	bts, err := common.JSONEncode(data)
	if err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return nil
	case <-time.After(time.Second * 15):
		return fmt.Errorf("send message timeout in 15s")
	case <-w.Done():
		return nil
	case w.WriteCh <- bts:
		return nil
	}
}

func (w *Websocket) Done() chan interface{} {
	return w.done
}

func NewWebsocket(ctx context.Context, url string, credentials *Credentials, channels []string, proxy string, bufferLen int) *Websocket {
	ws := Websocket{
		done:           make(chan interface{}),
		channelAlertCh: make(chan string, bufferLen*100),
		reconnectCh:    make(chan interface{}),

		DataCh:  make(chan interface{}, bufferLen),
		WriteCh: make(chan interface{}),
		Url:     url,
	}
	go ws.start(ctx, ws.Url, credentials, channels, proxy)
	ws.reconnectCh <- nil

	return &ws
}
