package common

import (
	"compress/gzip"
	"context"
	"fmt"
	"github.com/geometrybase/hft-micro/logger"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

type RawMessage struct {
	Prefix []byte
	Data   []byte
	Time   int64
}

func RawWSMessageSaveLoop(
	ctx context.Context, cancel context.CancelFunc,
	savePath, xSymbol, ySymbol string,
	messageInputCh chan *RawMessage,
	fileSavedCh chan string,
) {
	logger.Debugf("START RawWSMessageSaveLoop %s %s", xSymbol, ySymbol)
	hourUpdateTimer := time.NewTimer(time.Second)
	var dayTime time.Time
	var outPath string
	var file *os.File
	var gw *gzip.Writer
	var msg *RawMessage
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
			outPath = fmt.Sprintf(
				"%s/%s-%s,%s.jl.gz",
				savePath,
				dayTime.Format("20060102"),
				SymbolSanitize(xSymbol),
				SymbolSanitize(ySymbol),
			)
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
			if gw != nil {
				_, err = gw.Write(msg.Prefix)
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

func ArchiveDailyJlGzFiles(ctx context.Context,  savePath string) {
	if _, err := os.Stat(path.Join(savePath, "/archive/")); err != nil && os.IsNotExist(err) {
		err = os.MkdirAll(path.Join(savePath, "/archive/"), 0775)
		if err != nil {
			logger.Fatal(err)
		}
	}else if err != nil {
		logger.Fatal(err)
	}
	timer := time.NewTimer(time.Hour)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
		case <-timer.C:
			files, err := ioutil.ReadDir(savePath)
			if err != nil {
				logger.Debugf("ioutil.ReadDir error %v", err)
			} else {
				hourTime := time.Now().Truncate(time.Hour * 24)
				for _, file := range files {
					if len(file.Name()) > 5 && file.Name()[len(file.Name())-5:] != "jl.gz" {
						continue
					}
					parts := strings.Split(file.Name(), "-")
					if len(parts) > 0 && len(parts[0]) == 8 {
						fileTime, err := time.Parse("20060102", parts[0])
						if err != nil {
							logger.Debugf("time.Parse error %v", err)
						} else if hourTime.Sub(fileTime) > time.Hour*2 {
							if _, err = os.Stat(path.Join(savePath, "/archive/", parts[0])); os.IsNotExist(err) {
								err = os.MkdirAll(path.Join(savePath, "/archive/", parts[0]), 0775)
								if err != nil {
									logger.Debugf("os.MkdirAll error %v", err)
									continue
								}
							}else if err != nil {
								logger.Debugf("os.Stat error %v", err)
								continue
							}

							err= os.Rename(
								path.Join(savePath, file.Name()),
								path.Join(savePath, "/archive/", parts[0], file.Name()),
							)
							if err != nil {
								logger.Debugf("os.Rename error %v", err)
							} else {
								logger.Debugf("%s archived", file.Name())
							}
						}
					}
				}

			}
			timer.Reset(time.Hour)
		}
	}
}

func RawWSMessageSaveLoopForSingleSymbol(
	ctx context.Context, cancel context.CancelFunc,
	savePath, xSymbol string,
	messageInputCh chan *RawMessage,
	fileSavedCh chan string,
) {
	logger.Debugf("START RawWSMessageSaveLoopForSingleSymbol %s", xSymbol)
	hourUpdateTimer := time.NewTimer(time.Second)
	var dayTime time.Time
	var outPath string
	var file *os.File
	var gw *gzip.Writer
	var msg *RawMessage
	var err error
	var nextLine = []byte{'\n'}
	defer func() {
		if gw != nil {
			_ = gw.Flush()
			_ = gw.Close()
			logger.Debugf("close gzip writer for %s", xSymbol)
			gw = nil
		}
		if file != nil {
			logger.Debugf("close file writer for %s", xSymbol)
			_ = file.Close()
			file = nil
		}
		fileSavedCh <- xSymbol
		logger.Debugf("EXIT saveLoop %s", xSymbol)
		cancel()
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case <-hourUpdateTimer.C:
			if gw != nil {
				_ = gw.Flush()
				_ = gw.Close()
				gw = nil
			}
			if file != nil {
				_ = file.Close()
				file = nil
			}
			time.Sleep(time.Second * 5)
			dayTime = time.Now().Truncate(time.Hour * 24)
			outPath = fmt.Sprintf(
				"%s/%s-%s.jl.gz",
				savePath,
				dayTime.Format("20060102"),
				SymbolSanitize(xSymbol),
			)
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
			if gw != nil {
				_, err = gw.Write(msg.Prefix)
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

