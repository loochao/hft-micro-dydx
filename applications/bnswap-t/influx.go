package main

import (
	"github.com/geometrybase/hft-micro/influx/client"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func (st Strategy) handleInternalInfluxSave() {

	if st.USDTAsset.Asset == "" ||
		st.USDTAsset.MarginBalance == nil ||
		st.USDTAsset.CrossWalletBalance == nil ||
		st.USDTAsset.WalletBalance == nil ||
		st.USDTAsset.AvailableBalance == nil {
		return
	}

	totalDir := 0.0
	for _, s := range st.Signals {
		totalDir += s.Value
	}
	fields := make(map[string]interface{})
	fields["balance"] = *st.USDTAsset.MarginBalance
	fields["walletBalance"] = *st.USDTAsset.WalletBalance
	fields["availableBalance"] = *st.USDTAsset.AvailableBalance
	fields["startValue"] = *st.Config.StartValue
	fields["totalDir"] = totalDir
	fields["netWorth"] = *st.USDTAsset.MarginBalance / *st.Config.StartValue
	pt, err := client.NewPoint(
		*st.Config.InternalInflux.Measurement,
		map[string]string{
			"type": "balance",
		},
		fields,
		time.Now().UTC(),
	)
	if err != nil {
		logger.Debugf("Swap Balance NewPoint error %v", err)
	} else {
		go st.InternalInfluxWriter.Push(pt)
	}


	for symbolIndex := 0; symbolIndex < SYMBOLS_LEN; symbolIndex++ {
		fields := make(map[string]interface{})
		markPrice := st.MarkPrices[symbolIndex]
		if markPrice.Symbol == "" {
			continue
		}
		position := st.Positions[symbolIndex]
		signal := st.Signals[symbolIndex]
		fields["positionAmt"] = position.PositionAmt
		fields["entryPrice"] = position.EntryPrice
		fields["markPrice"] = markPrice.MarkPrice
		fields["indexPrice"] = markPrice.IndexPrice
		fields["fundingRate"] = markPrice.FundingRate
		if signal.Symbol != "" {
			fields["signal"] = signal.Value
			fields["buyCount"] = signal.Buy
			fields["sellCount"] = signal.Sell
			if totalDir*signal.Value > 0 {
				fields["signalAligned"] = signal.Value
			}else{
				fields["signalAligned"] = 0.0
			}
		}
		fields["realisedProfit"] = st.RealisedProfits[symbolIndex]
		pt, err := client.NewPoint(
			*st.Config.InternalInflux.Measurement,
			map[string]string{
				"symbol": markPrice.Symbol,
				"type":   "symbol",
			},
			fields,
			time.Now().UTC(),
		)
		if err != nil {
			logger.Debugf("new position point error %v", err)
		} else {
			go st.InternalInfluxWriter.Push(pt)
		}
	}
}

func (st Strategy) handleExternalInfluxSave() {
	if st.USDTAsset.Asset == "" ||
		st.USDTAsset.MarginBalance == nil ||
		st.USDTAsset.CrossWalletBalance == nil ||
		st.USDTAsset.WalletBalance == nil ||
		st.USDTAsset.AvailableBalance == nil {
		return
	}
	fields := make(map[string]interface{})
	fields["netWorth"] = *st.USDTAsset.MarginBalance / *st.Config.StartValue
	pt, err := client.NewPoint(
		*st.Config.ExternalInflux.Measurement,
		map[string]string{
			"name": *st.Config.Name,
		},
		fields,
		time.Now().UTC(),
	)
	if err != nil {
		logger.Debugf("Swap Balance NewPoint error %v", err)
	} else {
		go st.InternalInfluxWriter.Push(pt)
	}
}
