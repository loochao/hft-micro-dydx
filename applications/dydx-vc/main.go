package main

import (
	"context"
	"fmt"
	dydx_usdfuture "github.com/geometrybase/hft-micro/dydx-usdfuture"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"os"
	"sort"
	"strconv"
)

func main() {
	api, err := dydx_usdfuture.NewAPI(dydx_usdfuture.Credentials{
		ApiKey:        os.Getenv("DYDX_VC_KEY"),
		ApiSecret:     os.Getenv("DYDX_VC_SECRET"),
		ApiPassphrase: os.Getenv("DYDX_VC_PASSPHRASE"),
		AccountID:     os.Getenv("DYDX_VC_ACCOUNT_ID"),
	}, os.Getenv("DYDX_VC_PROXY"))
	if err != nil {
		logger.Fatal(err)
	}
	startValue, err := strconv.ParseFloat(os.Getenv("DYDX_VC_START_VALUE"), 64)
	if err != nil {
		logger.Fatal(err)
	}
	ctx := context.Background()
	account, err := api.GetAccount(ctx)
	if err != nil {
		logger.Fatal(err)
	}
	totalPosValue := 0.0
	totalUnrealizedPnl := 0.0
	fmt.Printf("\n\n")
	fmt.Printf("VC FUND12 + DYDX\n\n")
	fmt.Printf(
		"%10s  %6s  %8s  %12s  %12s  %9s\n",
		"SYMBOL", "SIDE", "VALUE", "SIZE", "PRICE", "URPNL",
	)
	markets := make([]string,0)
	posMap := map[string]dydx_usdfuture.Position{}
	for _, pos := range account.OpenPositions {
		markets = append(markets, pos.Market)
		posMap[pos.Market] = pos
	}
	sort.Strings(markets)
	for _, m := range markets {
		pos := posMap[m]
		totalPosValue += math.Abs(pos.Size) * pos.EntryPrice
		totalUnrealizedPnl += pos.UnrealizedPnl
		fmt.Printf(
			"%10s  %6s  %8s  %12s  %12s  %9s\n",
			pos.Market, pos.Side,
			fmt.Sprintf("%.0f", pos.Size*pos.EntryPrice),
			fmt.Sprintf("%.4f", pos.Size),
			fmt.Sprintf("%.4f", pos.EntryPrice),
			fmt.Sprintf("%.2f", pos.UnrealizedPnl),
		)
	}
	fmt.Printf("\n")
	fmt.Printf("资产组合价值\t%.0f\n", account.Equity)
	fmt.Printf("可用质押品\t%.0f\n", account.FreeCollateral)
	fmt.Printf("未实现盈亏\t%.3f\n", totalUnrealizedPnl)
	fmt.Printf("当前持仓量\t%.3f USDC\n", totalPosValue)
	fmt.Printf("杠杆\t\t%.3f\n", totalPosValue/account.Equity)
	fmt.Printf("DYDX净值\t%.3f\n", account.Equity/startValue)
	fmt.Printf("\n")
	rw, err := api.GetRewards(ctx, dydx_usdfuture.RewardsParam{
		Epoch: "4",
	})
	if err != nil {
		logger.Fatal(err)
	}
	fmt.Printf("本期编号\t%d\n", rw.Epoch)
	fmt.Printf("本期持仓\t%.0f USDC / %.0f USDC\n", rw.OpenInterest.AverageOpenInterest, rw.OpenInterest.TotalAverageOpenInterest)
	fmt.Printf("本期手续费\t%.0f USDC / %.0f USDC\n", rw.Fees.FeesPaid, rw.Fees.TotalFeesPaid)
	fmt.Printf("本期权重\t%.8f / %.8f\n", rw.Weight.Weight, rw.Weight.TotalWeight)
	fmt.Printf("DYDX奖励\t%.2f DYDX / %.0f DYDX\n", rw.EstimatedRewards, rw.TotalRewards)
	fmt.Printf("\n")
	user, err := api.GetUsers(ctx)
	if err != nil {
		logger.Fatal(err)
	}
	fmt.Printf("吃单费率\t%.5f\n", user.User.TakerFeeRate)
	fmt.Printf("挂单费率\t%.5f\n", user.User.MakerFeeRate)
	fmt.Printf("30天手续费\t%.0f\n", user.User.Fees30D)
	fmt.Printf("30天交易量\t%.0f\n", user.User.TakerVolume30D+user.User.MakerVolume30D)
	fmt.Printf("DYDX持有量\t%.2f\n", user.User.DydxTokenBalance)
	fmt.Printf("\n\n")
}

