package main

import (
	"github.com/geometrybase/hft-micro/influx/client"
	"github.com/geometrybase/hft-micro/logger"
	"strings"
	"time"
)

func handleSave() {
	if time.Now().Sub(swapGlobalSilent) < 0 {
		return
	}

	if swapAccount != nil &&
		swapAccount.MarginBalance != nil {
		totalBalance := *swapAccount.MarginBalance
		netWorth := totalBalance / *swapConfig.StartValue
		fields := make(map[string]interface{})
		fields["marginBalance"] = *swapAccount.MarginBalance
		fields["netWorth"] = netWorth
		fields["startValue"] = *swapConfig.StartValue
		fields["netWorth"] = netWorth
		if swapAccount.AvailableBalance != nil {
			fields["availableBalance"] = *swapAccount.AvailableBalance
		}
		if swapAccount.UnrealizedProfit != nil {
			fields["unrealizedProfit"] = *swapAccount.UnrealizedProfit
		}
		pt, err := client.NewPoint(
			*swapConfig.InternalInflux.Measurement,
			map[string]string{
				"type": "account",
			},
			fields,
			time.Now().UTC(),
		)
		if err != nil {
			logger.Debugf("client.NewPoint() error %v", err)
		} else {
			err = swapInternalInfluxWriter.PushPoint(pt)
			if err != nil {
				logger.Debugf("swapInternalInfluxWriter.PushPoint(pt) error %v", err)
			}
		}
	}

	for _, symbol := range swapSymbols {
		fields := make(map[string]interface{})
		if position, ok := swapPositions[symbol]; ok {
			fields["positionAmt"] = position.PositionAmt
			fields["positionValue"] = position.PositionAmt*position.EntryPrice
		}
		if mir, ok := swapMirs[symbol]; ok {
			fields["mir"] = mir.Value
			fields["mirLastPrice"] = mir.LastPrice
		}
		if time.Now().Sub(swapGlobalSilent) > 0 {
			fields["globalSilent"] = 0
		} else {
			fields["globalSilent"] = 1
		}
		pt, err := client.NewPoint(
			*swapConfig.InternalInflux.Measurement,
			map[string]string{
				"symbol": symbol,
				"type":   "symbol",
			},
			fields,
			time.Now().UTC(),
		)
		if err != nil {
			logger.Debugf("client.NewPoint() error %v", err)
		} else {
			err = swapInternalInfluxWriter.PushPoint(pt)
			if err != nil {
				logger.Debugf("swapInternalInfluxWriter.PushPoint(pt) error %v", err)
			}
		}
	}

}

func handleExternalInfluxSave() {

	if time.Now().Sub(swapGlobalSilent) < 0 {
		return
	}

	if swapAccount != nil &&
		swapAccount.MarginBalance != nil {
		totalBalance := *swapAccount.MarginBalance
		netWorth := totalBalance / *swapConfig.StartValue
		fields := make(map[string]interface{})
		fields["netWorth"] = netWorth
		for name, start := range swapConfig.StartValues {
			if start > 0 {
				fields["currentValue_"+strings.ToLower(name)] = netWorth * start
			}
		}
		if len(fields) > 0 {
			pt, err := client.NewPoint(
				*swapConfig.ExternalInflux.Measurement,
				map[string]string{
					"name": *swapConfig.Name,
				},
				fields,
				time.Now().UTC(),
			)
			if err != nil {
				logger.Debugf("client.NewPoint() error %v", err)
			} else {
				err = swapExternalInfluxWriter.PushPoint(pt)
				if err != nil {
					logger.Debugf(" swapExternalInfluxWriter.PushPoint(pt) error %v", err)
				}
			}
		}
	}
}

