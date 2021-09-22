package main

import (
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/montanaflynn/stats"
	"io"
	"os"
	"time"
)

func optBySymbol(xSymbol, ySymbol string) error {
	fileName := fmt.Sprintf("/Users/chenjilin/Downloads/20210820-20210916-%s-%s-24h0m0s-3s-1ms.gz", xSymbol, ySymbol)
	f, err := os.OpenFile(fileName, os.O_RDONLY, 0600)
	if err != nil {
		return err
	}
	gr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}

	startTime := time.Now()
	data := make([]*common.MatchedSpread, 40000000)
	counter := 0
	weeks2 := time.Hour * 24 * 18
	for err != io.EOF {
		ms := &common.MatchedSpread{}
		err = binary.Read(gr, binary.BigEndian, ms)
		if err != nil && err != io.EOF {
			return err
		}
		if time.Now().Sub(time.Unix(0, ms.EventTime)) < weeks2 {
			if err != io.EOF {
				data[counter] = ms
				counter++
				if counter == len(data) {
					dataNew := make([]*common.MatchedSpread, len(data)+10000000)
					copy(dataNew[:len(data)], data)
					data = dataNew
				}
			}
		}
	}
	err = gr.Close()
	if err != nil {
		return err
	}
	err = f.Close()
	if err != nil {
		return err
	}

	logger.Debugf("READ ALL DATA, TAKE %v", time.Now().Sub(startTime))

	for fr := 0.8; fr <= 0.8; fr += 0.2 {
		for j := 0.0; j <= 5.0; j += 1.0 {
			for i := 1.0; i <= 6.0; i += 1.0 {
				if i < j {
					continue
				}
				//logger.Debugf("OFFSET %f %f", 0.00111*i, 0.00111*j)
				result := strategyA(Params{
					xSymbol:        xSymbol,
					ySymbol:        ySymbol,
					enterOffset:    0.001 * i,
					leaveOffset:    0.001 * j,
					frFactor:       fr,
					startValue:     10000,
					enterStep:      0.1,
					enterInterval:  time.Second * 5,
					outputInterval: time.Minute,
					bestSizeFactor: 2.0,
					leverage:       5.0,
					tradeCost:      -0.0004,
				}, data)
				std, err := stats.StandardDeviation(result.NetWorth)
				if err != nil {
					logger.Debugf("error %v", err)
				}
				if result.NetWorth[len(result.NetWorth)-1] > 1 {
					logger.Debugf("%s %.4f %.4f FR %.2f NW %.4f SR %.4f TV %.2f",
						result.Params.xSymbol,
						result.Params.enterOffset, -result.Params.leaveOffset,
						result.Params.frFactor,
						result.NetWorth[len(result.NetWorth)-1],
						(result.NetWorth[len(result.NetWorth)-1]-1.0)/std,
						result.Turnover,
					)
				}
			}
		}
	}

	return nil
}

func main() {
	//err := optBySymbol("ICPUSDTM", "ICPUSDT")
	//err := optBySymbol("DOGEUSDTM", "DOGEUSDT")
	//err := optBySymbol("SOLUSDTM", "SOLUSDT")
	//err := optBySymbol("ADAUSDTM", "ADAUSDT")
	//err := optBySymbol("VETUSDTM", "VETUSDT")
	//err := optBySymbol("FTMUSDTM", "FTMUSDT")
	err := optBySymbol("XRPUSDTM", "XRPUSDT")
	if err != nil {
		logger.Debugf("optBySymbol %v", err)
	}
	logger.Debugf("done")
}
