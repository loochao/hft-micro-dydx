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

func archiveFiles(ctx context.Context, savePath string) {
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
		case <-timer.C:
			files, err := ioutil.ReadDir(savePath)
			if err != nil {
				logger.Debugf("ioutil.ReadDir error %v", err)
			} else {
				hourTime := time.Now().Truncate(time.Hour)
				for _, file := range files {
					parts := strings.Split(file.Name(), "-")
					if len(parts) > 0 && len(parts[0]) == 10 {
						fileTime, err := time.Parse("2006010215", parts[0])
						if err != nil {
							logger.Debugf("time.Parse error %v", err)
						} else if hourTime.Sub(fileTime) > time.Hour*2 {
							err = os.Rename(
								path.Join(savePath, file.Name()),
								path.Join(savePath, "/archive", file.Name()),
							)
							if err != nil {
								logger.Debugf("os.Rename error %v", err)
							}else{
								logger.Debugf("%s archived", file.Name())
							}
						}
					}
				}

			}
			timer.Reset(time.Minute*5)
		}
	}
}
