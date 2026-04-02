package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	proxy := "socks5://127.0.0.1:1083"
	wsURL := "wss://indexer.dydx.trade/v4/ws"
	symbol := "BTC-USD"
	if len(os.Args) > 1 {
		symbol = os.Args[1]
	}

	proxyURL, _ := url.Parse(proxy)
	dialer := &websocket.Dialer{
		Proxy:            http.ProxyURL(proxyURL),
		HandshakeTimeout: 30 * time.Second,
	}

	fmt.Printf("Connecting to %s via %s...\n", wsURL, proxy)
	conn, _, err := dialer.Dial(wsURL, http.Header{"User-Agent": []string{"Mozilla/5.0"}})
	if err != nil {
		fmt.Printf("DIAL ERROR: %v\n", err)
		return
	}
	defer conn.Close()
	fmt.Println("Connected!")

	// Subscribe to orderbook
	sub := map[string]interface{}{
		"type":    "subscribe",
		"channel": "v4_orderbook",
		"id":      symbol,
	}
	subBytes, _ := json.Marshal(sub)
	fmt.Printf("Subscribing: %s\n", subBytes)
	err = conn.WriteMessage(websocket.TextMessage, subBytes)
	if err != nil {
		fmt.Printf("WRITE ERROR: %v\n", err)
		return
	}

	// Read messages
	for i := 0; i < 10; i++ {
		conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		_, msg, err := conn.ReadMessage()
		if err != nil {
			fmt.Printf("READ ERROR: %v\n", err)
			return
		}
		var envelope map[string]interface{}
		json.Unmarshal(msg, &envelope)
		msgType := envelope["type"]
		channel := envelope["channel"]
		id := envelope["id"]

		if msgType == "subscribed" {
			contents := envelope["contents"]
			if contentsMap, ok := contents.(map[string]interface{}); ok {
				bids, _ := contentsMap["bids"].([]interface{})
				asks, _ := contentsMap["asks"].([]interface{})
				fmt.Printf("MSG %d: type=%v channel=%v id=%v bids=%d asks=%d\n", i, msgType, channel, id, len(bids), len(asks))
				if len(bids) > 0 {
					fmt.Printf("  Best bid: %v\n", bids[0])
				}
				if len(asks) > 0 {
					fmt.Printf("  Best ask: %v\n", asks[0])
				}
			}
		} else if msgType == "channel_data" {
			contents := envelope["contents"]
			if contentsMap, ok := contents.(map[string]interface{}); ok {
				bids, _ := contentsMap["bids"].([]interface{})
				asks, _ := contentsMap["asks"].([]interface{})
				fmt.Printf("MSG %d: type=%v channel=%v id=%v bid_updates=%d ask_updates=%d len=%d\n",
					i, msgType, channel, id, len(bids), len(asks), len(msg))
			}
		} else {
			fmt.Printf("MSG %d: type=%v channel=%v id=%v len=%d\n", i, msgType, channel, id, len(msg))
			if len(msg) < 500 {
				fmt.Printf("  Raw: %s\n", msg)
			}
		}
	}
	fmt.Println("\nDone - WS data is flowing!")
}
