package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	bnuf "github.com/geometrybase/hft-micro/binance-usdtfuture"
	bnus "github.com/geometrybase/hft-micro/binance-usdtspot"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
)

func main() {
	proxy := flag.String("proxy", "socks5://127.0.0.1:1083", "SOCKS5 proxy address")
	bnApiKey := flag.String("bn-key", os.Getenv("BN_API_KEY"), "Binance API key")
	bnApiSecret := flag.String("bn-secret", os.Getenv("BN_API_SECRET"), "Binance API secret")
	dydxAddr := flag.String("dydx-addr", os.Getenv("DYDX_ADDRESS"), "dYdX v4 address")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	// === Binance Futures ===
	if *bnApiKey != "" {
		fmt.Println("=== Binance Futures ===")
		futuresApi, err := bnuf.NewAPI(&common.Credentials{
			Key:    *bnApiKey,
			Secret: *bnApiSecret,
		}, *proxy)
		if err != nil {
			logger.Warnf("Binance Futures API error: %v", err)
		} else {
			account, err := futuresApi.GetAccount(ctx)
			if err != nil {
				fmt.Printf("  Error: %v\n", err)
			} else {
				fmt.Printf("  Total Balance:    %.2f USDT\n", account.TotalWalletBalance)
				fmt.Printf("  Available:        %.2f USDT\n", account.AvailableBalance)
				fmt.Printf("  Unrealized PnL:   %.2f USDT\n", account.TotalUnrealizedProfit)
				for _, asset := range account.Assets {
					if asset.WalletBalance != nil && *asset.WalletBalance > 0.01 {
						avail := float64(0)
						if asset.AvailableBalance != nil {
							avail = *asset.AvailableBalance
						}
						fmt.Printf("  Asset %s: wallet=%.4f available=%.4f\n",
							asset.Asset, *asset.WalletBalance, avail)
					}
				}
				positions, err := futuresApi.GetPositions(ctx)
				if err != nil {
					fmt.Printf("  Positions error: %v\n", err)
				} else {
					for _, pos := range positions {
						if pos.PositionAmt != 0 {
							fmt.Printf("  Position: %s size=%.4f entry=%.4f pnl=%.4f\n",
								pos.Symbol, pos.PositionAmt, pos.EntryPrice, pos.UnrealizedProfit)
						}
					}
				}
			}
		}

		// === Binance Spot ===
		fmt.Println("\n=== Binance Spot ===")
		spotApi, err := bnus.NewAPI(&common.Credentials{
			Key:    *bnApiKey,
			Secret: *bnApiSecret,
		}, *proxy)
		if err != nil {
			logger.Warnf("Binance Spot API error: %v", err)
		} else {
			spotAccount, _, err := spotApi.GetAccount(ctx)
			if err != nil {
				fmt.Printf("  Error: %v\n", err)
			} else {
				for _, bal := range spotAccount.Balances {
					free := bal.Free
					locked := bal.Locked
					if free > 0.001 || locked > 0.001 {
						fmt.Printf("  %s: free=%.8f locked=%.8f\n", bal.Asset, free, locked)
					}
				}
			}
		}
	}

	// === dYdX v4 (REST API, no auth needed for balance check) ===
	if *dydxAddr != "" {
		fmt.Println("\n=== dYdX v4 ===")
		url := fmt.Sprintf("https://indexer.dydx.trade/v4/addresses/%s", *dydxAddr)
		client := &http.Client{Timeout: 15 * time.Second}
		req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("  Error: %v\n", err)
		} else {
			defer resp.Body.Close()
			body, _ := ioutil.ReadAll(resp.Body)
			var result map[string]interface{}
			if err := json.Unmarshal(body, &result); err != nil {
				fmt.Printf("  Parse error: %v\n  Body: %s\n", err, string(body)[:200])
			} else {
				subaccounts, ok := result["subaccounts"].([]interface{})
				if ok {
					for _, sub := range subaccounts {
						s := sub.(map[string]interface{})
						fmt.Printf("  Subaccount %v: equity=%v freeCollateral=%v\n",
							s["subaccountNumber"], s["equity"], s["freeCollateral"])
						if positions, ok := s["openPerpetualPositions"].(map[string]interface{}); ok {
							for market, pos := range positions {
								p := pos.(map[string]interface{})
								fmt.Printf("    Position: %s size=%v entry=%v side=%v\n",
									market, p["size"], p["entryPrice"], p["side"])
							}
						}
					}
				}
			}
		}
	}
}
