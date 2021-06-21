package main

import (
	"compress/gzip"
	"context"
	"fmt"
	"github.com/geometrybase/hft-micro/logger"
	"math/rand"
	"os"
	"time"
)

func saveLoop(ctx context.Context, cancel context.CancelFunc, savePath, symbol string, futureInputCh, spotInputCh chan []byte, fileSavedCh chan string) {
	logger.Debugf("START saveLoop %s", symbol)
	hourUpdateTimer := time.NewTimer(time.Second)
	var dayTime time.Time
	var outPath string
	var file *os.File
	var gw *gzip.Writer
	var msg []byte
	var err error
	var nextLine = []byte("\n")
	var swapPrefix = []byte("F")
	var spotPrefix = []byte("S")
	defer func() {
		if gw != nil {
			logger.Debugf("close gzip writer for %s", symbol)
			err = gw.Close()
			if err != nil {
				logger.Debugf("close gzip writer %s error %v, stop ws", outPath, err)
			}
		}
		if file != nil {
			logger.Debugf("close file %s", symbol)
			err = file.Close()
			if err != nil {
				logger.Debugf("close file %s error %v, stop ws", outPath, err)
			}
		}
		fileSavedCh <- symbol
		logger.Debugf("EXIT saveLoop %s", symbol)
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case <-hourUpdateTimer.C:
			if file != nil {
				err = file.Close()
				if err != nil {
					logger.Debugf("close file %s error %v, stop ws", outPath, err)
					cancel()
					return
				}
			}
			if gw != nil {
				err = gw.Close()
				if err != nil {
					logger.Debugf("close gzip writer %s error %v, stop ws", outPath, err)
					cancel()
					return
				}
			}
			dayTime = time.Now().Truncate(time.Hour * 24)
			outPath = fmt.Sprintf("%s/%s-%s.depth5.jl.gz", savePath, dayTime.Format("20060102"), symbol)
			file, err = os.OpenFile(outPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0755)
			if err != nil {
				cancel()
				logger.Debugf("os.OpenFile error %v, stop ws", err)
				return
			}
			gw, err = gzip.NewWriterLevel(file, gzip.BestCompression)
			if err != nil {
				cancel()
				logger.Debugf("gzip.NewWriterLevel error %v, stop ws", err)
				return
			}
			gw.Name = fmt.Sprintf("%s-%s.depth5.jl.gz", dayTime.Format("20060102"), symbol)
			gw.ModTime = time.Now()
			gw.Comment = fmt.Sprintf("depth5 raw json line for %s@%s", symbol, dayTime.Format("20060102"))
			hourUpdateTimer.Reset(
				time.Now().Truncate(
					time.Hour * 24,
				).Add(
					time.Hour * 24,
				).Add(
					time.Duration(rand.Intn(60)) * time.Second,
				).Sub(time.Now()),
			)
		case msg = <-futureInputCh:
			//logger.Debugf("%s", msg)
			if gw != nil {
				_, err = gw.Write(swapPrefix)
				if err != nil {
					cancel()
					logger.Debugf("gw.Write error %v, stop ws", err)
					return
				}
				_, err = gw.Write(msg)
				if err != nil {
					cancel()
					logger.Debugf("gw.Write error %v, stop ws", err)
					return
				}
				_, err = gw.Write(nextLine)
				if err != nil {
					cancel()
					logger.Debugf("gw.Write error %v, stop ws", err)
					return
				}
			}
		case msg = <-spotInputCh:
			//logger.Debugf("%s", msg)
			if gw != nil {
				_, err = gw.Write(spotPrefix)
				if err != nil {
					cancel()
					logger.Debugf("gw.Write error %v, stop ws", err)
					return
				}
				_, err = gw.Write(msg)
				if err != nil {
					cancel()
					logger.Debugf("gw.Write error %v, stop ws", err)
					return
				}
				_, err = gw.Write(nextLine)
				if err != nil {
					cancel()
					logger.Debugf("gw.Write error %v, stop ws", err)
					return
				}
			}
		}
	}
}
