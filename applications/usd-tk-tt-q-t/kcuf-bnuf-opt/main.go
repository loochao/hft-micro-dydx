package main

import (
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
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
	counter := 0
	outputData := &common.MatchedSpread{}
	startTime := time.Now()
	for err != io.EOF {
		err = binary.Read(gr, binary.BigEndian, outputData)
		if err != nil && err != io.EOF {
			return err
		}
		counter++
	}
	takeTime := time.Now().Sub(startTime)
	fmt.Printf("\nROW COUNT: %d TAKE TIME: %v AVG TIME PER ROW: %v\n", counter, takeTime, takeTime/time.Duration(counter))
	err = gr.Close()
	if err != nil {
		return err
	}
	err = f.Close()
	if err != nil {
		return err
	}
	return nil
}

func main() {
	err := optBySymbol("SOLUSDTM", "SOLUSDT")
	if err != nil {
		logger.Debugf("optBySymbol %v", err)
	}
}
