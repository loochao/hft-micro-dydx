package main

import (
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/influx/client"
	"github.com/geometrybase/hft-micro/logger"
	"strings"
	"time"
)

func handleInternalSave() {
	if mConfig.InternalInflux.Address == "" {
		return
	}

	totalUnHedgeValue := 0.0
	makerURPnl := 0.0
	for _, makerSymbol := range mSymbols {
		fields := make(map[string]interface{})
		if makerPosition, ok := mPositions[makerSymbol]; ok {
			fields["makerSize"] = makerPosition.GetSize()
			makerValue := makerPosition.GetSize() * makerPosition.GetPrice()
			fields["makerValue"] = makerValue
			fields["makerPrice"] = makerPosition.GetPrice()
			if spread, ok := mWalkedDepths[makerSymbol]; ok {
				if makerPosition.GetPrice() != 0 {
					makerURPnl += makerPosition.GetSize() * (spread.MidPrice - makerPosition.GetPrice())
				}
			}
		}
		if filterRatio, ok := mFilterRatios[makerSymbol]; ok {
			fields["filterRatio"] = filterRatio.Value
		}
		if depth, ok := mWalkedDepths[makerSymbol]; ok {

			fields["makerMakerBid"] = depth.MakerBid
			fields["makerMakerAsk"] = depth.MakerAsk
			fields["makerTakerBid"] = depth.TakerBid
			fields["makerTakerAsk"] = depth.TakerAsk
		}
		if mSystemStatus == common.SystemStatusReady {
			fields["makerSystemStatus"] = 1.0
		} else {
			fields["makerSystemStatus"] = -1.0
		}
		pt, err := client.NewPoint(
			mConfig.InternalInflux.Measurement,
			map[string]string{
				"symbol": makerSymbol,
				"type":   "symbol",
			},
			fields,
			time.Now().UTC(),
		)
		if err != nil {
			logger.Debugf("client.NewPoint error %v", err)
		} else {
			err = mInfluxWriter.PushPoint(pt)
			if err != nil {
				logger.Debugf("mInfluxWriter.PushPoint error %v", err)
			}
		}
	}

	if mAccount != nil {
		totalBalance := mAccount.GetBalance()
		netWorth := totalBalance / mConfig.StartValue
		fields := make(map[string]interface{})
		fields["totalUnHedgeValue"] = totalUnHedgeValue
		fields["totalBalance"] = totalBalance
		fields["makerBalance"] = mAccount.GetBalance()
		fields["netWorth"] = netWorth
		fields["turnover"] = mTimedPositionChange.Sum() / totalBalance
		fields["startValue"] = mConfig.StartValue
		fields["netWorth"] = netWorth
		fields["makerAvailable"] = mAccount.GetFree()
		fields["makerURPnl"] = makerURPnl
		pt, err := client.NewPoint(
			mConfig.InternalInflux.Measurement,
			map[string]string{
				"type": "balance",
			},
			fields,
			time.Now().UTC(),
		)
		if err != nil {
			logger.Debugf("client.NewPoint error %v", err)
		} else {
			err = mInfluxWriter.PushPoint(pt)
			if err != nil {
				logger.Debugf("mInfluxWriter.PushPoint error %v", err)
			}
		}
	}
}

func handleExternalInfluxSave() {
	if mConfig.ExternalInflux.Address == "" {
		return
	}

	if mAccount != nil {
		totalBalance := mAccount.GetBalance()
		netWorth := totalBalance / mConfig.StartValue
		fields := make(map[string]interface{})
		fields["netWorth"] = netWorth
		for name, start := range mConfig.StartValues {
			if start > 0 {
				fields["currentValue_"+strings.ToLower(name)] = netWorth * start
			}
		}
		if len(fields) > 0 {
			pt, err := client.NewPoint(
				mConfig.ExternalInflux.Measurement,
				map[string]string{
					"name": *mConfig.Name,
				},
				fields,
				time.Now().UTC(),
			)
			if err != nil {
				logger.Debugf("client.NewPoint error %v", err)
			} else {
				err = mExternalInfluxWriter.PushPoint(pt)
				if err != nil {
					logger.Debugf("mExternalInfluxWriter.PushPoint error %v", err)
				}
			}
		}
	}
}
