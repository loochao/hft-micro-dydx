package main

import (
	"context"
	"github.com/geometrybase/hft-micro/logger"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"
)

func archiveFiles(ctx context.Context,  savePath string) {
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
					if len(file.Name()) > 3 && file.Name()[len(file.Name())-3:] != ".gz" {
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
