package main

import (
	"compress/gzip"
	"context"
	"fmt"
	"github.com/geometrybase/hft-micro/logger"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"
)

func saveLoop(ctx context.Context, cancel context.CancelFunc, savePath, xSymbol, ySymbol string, messageInputCh chan *Message, fileSavedCh chan string) {
	logger.Debugf("START saveLoop %s %s", xSymbol, ySymbol)
	hourUpdateTimer := time.NewTimer(time.Second)
	var dayTime time.Time
	var outPath string
	var file *os.File
	var gw *gzip.Writer
	var msg *Message
	var err error
	var nextLine = []byte{'\n'}
	defer func() {
		if gw != nil {
			gw.Flush()
			gw.Close()
			logger.Debugf("close gzip writer for %s %s", xSymbol, ySymbol)
			gw = nil
		}
		if file != nil {
			logger.Debugf("close file writer for %s %s", xSymbol, ySymbol)
			file.Close()
			file = nil
		}
		fileSavedCh <- xSymbol
		logger.Debugf("EXIT saveLoop %s %s", xSymbol, ySymbol)
		cancel()
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case <-hourUpdateTimer.C:
			if gw != nil {
				gw.Flush()
				gw.Close()
				gw = nil
			}
			if file != nil {
				file.Close()
				file = nil
			}
			time.Sleep(time.Second * 5)
			dayTime = time.Now().Truncate(time.Hour * 24)
			outPath = fmt.Sprintf("%s/%s-%s,%s.jl.gz", savePath, dayTime.Format("20060102"), strings.Replace(xSymbol, "/", "_", -1),  strings.Replace(ySymbol, "/", "_", -1))
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
			hourUpdateTimer.Reset(
				time.Now().Truncate(
					time.Hour * 24,
				).Add(
					time.Hour * 24,
				).Add(
					time.Duration(rand.Intn(60)) * time.Second,
				).Sub(time.Now()),
			)
		case msg = <-messageInputCh:
			//logger.Debugf("%s", msg)
			if gw != nil {
				_, err = gw.Write(msg.Source)
				if err != nil {
					cancel()
					logger.Debugf("gw.Write error %v, stop ws", err)
					return
				}
				_, err = gw.Write([]byte(strconv.FormatInt(msg.Time, 10)))
				if err != nil {
					cancel()
					logger.Debugf("gw.Write error %v, stop ws", err)
					return
				}
				_, err = gw.Write(msg.Data)
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
