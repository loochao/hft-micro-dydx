package main

import (
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/influx/client"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"strings"
	"time"
)

func handleSave(
	xAccount, yAccount common.Balance,
	xExchange, yExchange common.UsdExchange,
	strategiesMap map[string]*XYStrategy,
	xSymbols []string,
	xSystemStatus, ySystemStatus common.SystemStatus,
	xyConfig *Config,
	xCommissionAssetValue, yCommissionAssetValue *float64,
	xyInternalInfluxWriter, xyExternalInfluxWriter *common.InfluxWriter,
) {
	if yCommissionAssetValue == nil || xCommissionAssetValue == nil {
		return
	}
	totalUnHedgeValue := 0.0
	totalXSymbolValue := 0.0
	totalYSymbolValue := 0.0
	yURPnl := 0.0
	xURPnl := 0.0
	totalURPnl := 0.0
	hasAllSymbols := true

	for _, xSymbol := range xSymbols {
		st, ok := strategiesMap[xSymbol]
		if !ok {
			hasAllSymbols = false
			continue
		}
		ySymbol := st.ySymbol
		fields := make(map[string]interface{})
		if st.xPosition != nil &&
			st.yPosition != nil &&
			st.midPrice != 0 {

			unHedgeValue := math.Abs(st.xSize+st.ySize) * st.midPrice
			totalUnHedgeValue += unHedgeValue
			totalXSymbolValue += st.xAbsValue
			totalYSymbolValue += st.yAbsValue

			fields["unHedgeValue"] = unHedgeValue
			fields["xSize"] = st.xSize
			fields["xAbsValue"] = st.xAbsValue
			fields["xValue"] = st.xValue
			fields["ySize"] = st.ySize
			fields["yAbsValue"] = st.yAbsValue
			fields["yValue"] = st.yValue
			fields["xyValue"] = st.xValue + st.yValue
			totalURPnl += st.xValue + st.yValue

			if st.xPosition.GetPrice() != 0 {
				xURPnl += st.xValue * (st.midPrice - st.xPosition.GetPrice()) / st.xPosition.GetPrice()
			}
			if st.yPosition.GetPrice() != 0 {
				yURPnl += st.yValue * (st.midPrice - st.yPosition.GetPrice()) / st.yPosition.GetPrice()
			}
		} else {
			logger.Debugf(
				"%s %s save failed, okXPosition %v okYPosition %v",
				xSymbol, ySymbol, st.xPosition != nil, st.yPosition != nil,
			)
			hasAllSymbols = false
		}
		if xSystemStatus == common.SystemStatusReady {
			fields["xSystemStatus"] = 1.0
		} else {
			fields["xSystemStatus"] = -1.0
		}
		if ySystemStatus == common.SystemStatusReady {
			fields["ySystemStatus"] = 1.0
		} else {
			fields["ySystemStatus"] = -1.0
		}
		pt, err := client.NewPoint(
			xyConfig.InternalInflux.Measurement,
			map[string]string{
				"ySymbol": ySymbol,
				"xSymbol": xSymbol,
				"type":    "symbol",
			},
			fields,
			time.Now().UTC(),
		)
		if err != nil {
			logger.Debugf("client.NewPoint error %v", err)
		} else {
			err = xyInternalInfluxWriter.PushPoint(pt)
			if err != nil {
				logger.Debugf("xyInfluxWriter.PushPoint error %v", err)
			}
		}
	}

	if yAccount != nil &&
		xAccount != nil &&
		hasAllSymbols {
		xBalance := xAccount.GetBalance()
		yBalance := yAccount.GetBalance()
		if xExchange.IsSpot() {
			xBalance += totalXSymbolValue
		}
		if yExchange.IsSpot() {
			yBalance += totalYSymbolValue
		}
		totalBalance := xBalance + yBalance + *xCommissionAssetValue + *yCommissionAssetValue
		netWorth := totalBalance / xyConfig.StartValue
		fields := make(map[string]interface{})
		fields["totalUnHedgeValue"] = totalUnHedgeValue
		fields["totalBalance"] = totalBalance
		fields["xCommissionAssetValue"] = *xCommissionAssetValue
		fields["yCommissionAssetValue"] = *yCommissionAssetValue
		fields["yBalance"] = yBalance
		fields["xBalance"] = xBalance
		fields["yAvailable"] = yAccount.GetFree()
		fields["xAvailable"] = xAccount.GetFree()
		fields["xURPnl"] = xURPnl
		fields["yURPnl"] = yURPnl
		fields["xyURPnl"] = totalURPnl
		fields["netWorth"] = netWorth
		fields["startValue"] = xyConfig.StartValue

		pt, err := client.NewPoint(
			xyConfig.InternalInflux.Measurement,
			map[string]string{
				"type": "balance",
			},
			fields,
			time.Now().UTC(),
		)
		if err != nil {
			logger.Debugf("client.NewPoint error %v", err)
		} else {
			err = xyInternalInfluxWriter.PushPoint(pt)
			if err != nil {
				logger.Debugf("xyInfluxWriter.PushPoint error %v", err)
			}
		}

		fields = make(map[string]interface{})
		fields["netWorth"] = netWorth
		for name, start := range xyConfig.StartValues {
			if start > 0 {
				fields["currentValue_"+strings.ToLower(name)] = netWorth * start
			}
		}
		if len(fields) > 0 {
			pt, err := client.NewPoint(
				xyConfig.ExternalInflux.Measurement,
				map[string]string{
					"name": *xyConfig.Name,
				},
				fields,
				time.Now().UTC(),
			)
			if err != nil {
				logger.Debugf("client.NewPoint error %v", err)
			} else {
				err = xyExternalInfluxWriter.PushPoint(pt)
				if err != nil {
					logger.Debugf("xyExternalInfluxWriter.PushPoint error %v", err)
				}
			}
		}

	}
}
