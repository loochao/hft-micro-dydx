package main

import (
	"bufio"
	"compress/gzip"
	"github.com/geometrybase/hft-micro/logger"
	"io/ioutil"
	"os"
	"path"
	"strings"
)

func main() {

	targetWeights := map[string]float64{
		"ANKRUSDT":  0.72,
		"UNFIUSDT":  0.48,
		"OMGUSDT":   0.56,
		"HNTUSDT":   0.66,
		"COMPUSDT":  0.60,
		"AKROUSDT":  0.50,
		"KNCUSDT":   1.00,
		"NKNUSDT":   0.49,
		"ALPHAUSDT": 0.79,
		"CHRUSDT":   0.70,
		"ZECUSDT":   0.94,
		"IOTAUSDT":  0.76,
		"LINKUSDT":  1.00,
		"DOGEUSDT":  1.00,
		"CELRUSDT":  0.74,
		"ZILUSDT":   0.71,
		"ICXUSDT":   0.79,
		"HBARUSDT":  0.74,
		"SNXUSDT":   0.62,
		"LUNAUSDT":  1.00,
		"BZRXUSDT":  1.00,
		"GRTUSDT":   0.86,
		"XRPUSDT":   1.00,
		"AVAXUSDT":  0.79,
		"ZENUSDT":   0.78,
		"CTKUSDT":   0.56,
		"DODOUSDT":  1.00,
		"RLCUSDT":   0.64,
		"BANDUSDT":  0.73,
		"MANAUSDT":  0.73,
		"CHZUSDT":   0.94,
		"OGNUSDT":   0.77,
		"ETCUSDT":   1.00,
		"CRVUSDT":   1.00,
		"BTCUSDT":   1.00,
		"DENTUSDT":  0.72,
		"DOTUSDT":   1.00,
		"MTLUSDT":   0.67,
		"IOSTUSDT":  0.75,
		"TRXUSDT":   1.00,
		"WAVESUSDT": 0.79,
		"BTSUSDT":   0.64,
		"XTZUSDT":   0.68,
		"SANDUSDT":  0.74,
		"FILUSDT":   1.00,
		"FLMUSDT":   0.61,
		"ETHUSDT":   1.00,
		"SXPUSDT":   1.00,
		"RENUSDT":   0.92,
		"YFIIUSDT":  0.60,
		"DGBUSDT":   0.65,
		"STORJUSDT": 0.68,
		"NEOUSDT":   1.00,
		"RUNEUSDT":  1.00,
		"KSMUSDT":   0.89,
		"OCEANUSDT": 0.62,
		"REEFUSDT":  0.58,
		"XMRUSDT":   0.86,
		"AAVEUSDT":  0.78,
		"XLMUSDT":   0.81,
		"SFPUSDT":   0.61,
		"BELUSDT":   0.55,
		"BALUSDT":   0.65,
		"1INCHUSDT": 0.65,
		"COTIUSDT":  0.58,
		"MATICUSDT": 1.00,
		"RSRUSDT":   1.00,
		"LTCUSDT":   1.00,
		"ATOMUSDT":  0.85,
		"ONTUSDT":   0.78,
		"ALICEUSDT": 0.95,
		"XEMUSDT":   1.00,
		"NEARUSDT":  0.58,
		"ZRXUSDT":   0.71,
		"SOLUSDT":   0.97,
		"BATUSDT":   0.74,
		"ICPUSDT":   1.00,
		"LITUSDT":   0.45,
		"SRMUSDT":   0.66,
		"LINAUSDT":  1.00,
		"BCHUSDT":   1.00,
		"SKLUSDT":   0.50,
		"BTTUSDT":   0.68,
		"YFIUSDT":   0.89,
		"ONEUSDT":   0.66,
		"RVNUSDT":   0.61,
		"FTMUSDT":   0.79,
		"QTUMUSDT":  1.00,
		"TRBUSDT":   0.53,
		"ALGOUSDT":  0.83,
		"AXSUSDT":   0.54,
		"MKRUSDT":   0.66,
		"STMXUSDT":  0.55,
		"UNIUSDT":   0.87,
		"BAKEUSDT":  0.71,
		"EGLDUSDT":  0.74,
		"CVCUSDT":   0.42,
		"HOTUSDT":   1.00,
		"EOSUSDT":   1.00,
		"TOMOUSDT":  0.71,
		"BLZUSDT":   0.51,
		"DASHUSDT":  0.81,
		"VETUSDT":   1.00,
		"THETAUSDT": 1.00,
		"SUSHIUSDT": 1.00,
		"LRCUSDT":   0.62,
		"ADAUSDT":   1.00,
		"ENJUSDT":   0.76,
		"KAVAUSDT":  0.78,
	}

	dataPath := "/volume1/MarketData/bnus-bnuf-depth5-and-ticker"
	//dataPath = "/Users/chenjilin/MarketData/bnus-bnuf-depth5-and-ticker"

	outputPath := "/volume1/MarketData/techsharpe"
	//outputPath = "/Users/chenjilin/Downloads"

	var nextLine = []byte{'\n'}

	dateFolders, err := ioutil.ReadDir(dataPath)
	if err != nil {
		logger.Fatal(err)
	}
	for _, dateFolder := range dateFolders {
		spotBookTickerPath := path.Join(outputPath, "binance_spot", "bookticker", dateFolder.Name())
		spotDepth5Path := path.Join(outputPath, "binance_spot", "depth5", dateFolder.Name())
		futureBookTickerPath := path.Join(outputPath, "binance_future", "bookticker", dateFolder.Name())
		futureDepth5Path := path.Join(outputPath, "binance_future", "depth5", dateFolder.Name())
		err := os.MkdirAll(spotBookTickerPath, 0775)
		if err != nil && !os.IsExist(err) {
			logger.Debugf("%v", err)
			continue
		}
		err = os.MkdirAll(spotDepth5Path, 0775)
		if err != nil && !os.IsExist(err) {
			logger.Debugf("%v", err)
			continue
		}
		err = os.MkdirAll(futureBookTickerPath, 0775)
		if err != nil && !os.IsExist(err) {
			logger.Debugf("%v", err)
			continue
		}
		err = os.MkdirAll(futureDepth5Path, 0775)
		if err != nil && !os.IsExist(err) {
			logger.Debugf("%v", err)
			continue
		}
		symbolPaths, err := os.ReadDir(path.Join(dataPath, dateFolder.Name()))
		if err != nil {
			logger.Debugf("%v", err)
			continue
		}
		for _, symbolPath := range symbolPaths {
			if len(symbolPath.Name()) > 6 && symbolPath.Name()[len(symbolPath.Name())-6:] == ".jl.gz" {
				symbol := strings.Split(symbolPath.Name()[9:len(symbolPath.Name())-6], ",")[0]
				if _, ok := targetWeights[symbol]; !ok {
					targetWeights[symbol] = 0
				}
				logger.Debugf("%s %s start", dateFolder.Name(), symbol)
				file, err := os.Open(path.Join(dataPath, dateFolder.Name(), symbolPath.Name()))
				if err != nil {
					logger.Debugf("os.Open() error %v", err)
					continue
				}
				gr, err := gzip.NewReader(file)
				if err != nil {
					logger.Debugf("gzip.NewReader(file) error %v", err)
					continue
				}
				//b := make([]byte, 0, 512)
				//_, err = gr.Read(b)
				//if err != nil {
				//	logger.Debugf("gr.Read(b) error %v", err)
				//	continue
				//}
				scanner := bufio.NewScanner(gr)
				var msg []byte

				spotBookTickerFilePath := path.Join(spotBookTickerPath, symbol+".gz")
				spotDepth5FilePath := path.Join(spotDepth5Path, symbol+".gz")
				futureBookTickerFilePath := path.Join(futureBookTickerPath, symbol+".gz")
				futureDepth5FilePath := path.Join(futureDepth5Path, symbol+".gz")

				if _, err := os.Stat(spotBookTickerFilePath); err == nil || !os.IsNotExist(err) {
					logger.Debugf("%s %s ignore", dateFolder.Name(), symbol)
					if err != nil {
						logger.Debugf("%v", err)
					}
					continue
				}

				spotBookTickerFileTmpPath := path.Join(spotBookTickerPath, symbol+".gz.tmp")
				spotDepth5FileTmpPath := path.Join(spotDepth5Path, symbol+".gz.tmp")
				futureBookTickerFileTmpPath := path.Join(futureBookTickerPath, symbol+".gz.tmp")
				futureDepth5FileTmpPath := path.Join(futureDepth5Path, symbol+".gz.tmp")
				var spotBookTickerFile *os.File
				var spotDepth5File *os.File
				var futureBookTickerFile *os.File
				var futureDepth5File *os.File

				var spotBookTickerGW *gzip.Writer
				var spotDepth5GW *gzip.Writer
				var futureBookTickerGW *gzip.Writer
				var futureDepth5GW *gzip.Writer

				spotBookTickerFile, err = os.OpenFile(spotBookTickerFileTmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0775)
				if err != nil {
					logger.Debugf("%v", err)
					continue
				}
				spotBookTickerGW, err = gzip.NewWriterLevel(spotBookTickerFile, gzip.BestCompression)
				if err != nil {
					logger.Debugf("%v", err)
					continue
				}
				spotDepth5File, err = os.OpenFile(spotDepth5FileTmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0775)
				if err != nil {
					logger.Debugf("%v", err)
					continue
				}
				spotDepth5GW, err = gzip.NewWriterLevel(spotDepth5File, gzip.BestCompression)
				if err != nil {
					logger.Debugf("%v", err)
					continue
				}

				if targetWeights[symbol] >= 0.7 {

					futureBookTickerFile, err = os.OpenFile(futureBookTickerFileTmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0775)
					if err != nil {
						logger.Debugf("%v", err)
						continue
					}
					futureBookTickerGW, err = gzip.NewWriterLevel(futureBookTickerFile, gzip.BestCompression)
					if err != nil {
						logger.Debugf("%v", err)
						continue
					}
					futureDepth5File, err = os.OpenFile(futureDepth5FileTmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0775)
					if err != nil {
						logger.Debugf("%v", err)
						continue
					}
					futureDepth5GW, err = gzip.NewWriterLevel(futureDepth5File, gzip.BestCompression)
					if err != nil {
						logger.Debugf("%v", err)
						continue
					}

				}

				for scanner.Scan() {
					msg = scanner.Bytes()
					if msg[0] == 'S' && msg[1] == 'D' {
						_, err = spotDepth5GW.Write(msg[2:])
						if err != nil {
							logger.Debugf("%v", err)
						}
						_, err = spotDepth5GW.Write(nextLine)
						if err != nil {
							logger.Debugf("%v", err)
						}
					} else if msg[0] == 'S' && msg[1] == 'T' {
						_, err = spotBookTickerGW.Write(msg[2:])
						if err != nil {
							logger.Debugf("%v", err)
						}
						_, err = spotBookTickerGW.Write(nextLine)
						if err != nil {
							logger.Debugf("%v", err)
						}
					} else if msg[0] == 'F' && msg[1] == 'D' {
						if targetWeights[symbol] >= 0.7 {
							_, err = futureDepth5GW.Write(msg[2:])
							if err != nil {
								logger.Debugf("%v", err)
							}
							_, err = futureDepth5GW.Write(nextLine)
							if err != nil {
								logger.Debugf("%v", err)
							}
						}
					} else if msg[0] == 'F' && msg[1] == 'T' {
						if targetWeights[symbol] >= 0.7 {
							_, err = futureBookTickerGW.Write(msg[2:])
							if err != nil {
								logger.Debugf("%v", err)
							}
							_, err = futureBookTickerGW.Write(nextLine)
							if err != nil {
								logger.Debugf("%v", err)
							}
						}
					} else {
						continue
					}
				}

				_ = spotDepth5GW.Close()
				_ = spotBookTickerGW.Close()
				_ = spotDepth5File.Close()
				_ = spotBookTickerFile.Close()

				if targetWeights[symbol] >= 0.7 {
					_ = futureDepth5GW.Close()
					_ = futureBookTickerGW.Close()
					_ = futureDepth5File.Close()
					_ = futureBookTickerFile.Close()
				}
				_ = gr.Close()
				_ = file.Close()

				err = os.Rename(
					spotDepth5FileTmpPath,
					spotDepth5FilePath,
				)
				if err != nil {
					logger.Debugf("%v", err)
				}
				err = os.Rename(
					spotBookTickerFileTmpPath,
					spotBookTickerFilePath,
				)
				if err != nil {
					logger.Debugf("%v", err)
				}

				err = os.Chmod(spotDepth5FilePath, 0775)
				if err != nil {
					logger.Debugf("%v", err)
				}
				err = os.Chmod(spotBookTickerFilePath, 0775)
				if err != nil {
					logger.Debugf("%v", err)
				}

				if targetWeights[symbol] >= 0.7 {
					err = os.Rename(
						futureDepth5FileTmpPath,
						futureDepth5FilePath,
					)
					if err != nil {
						logger.Debugf("%v", err)
					}
					err = os.Rename(
						futureBookTickerFileTmpPath,
						futureBookTickerFilePath,
					)
					if err != nil {
						logger.Debugf("%v", err)
					}

					err = os.Chmod(futureDepth5FilePath, 0775)
					if err != nil {
						logger.Debugf("%v", err)
					}
					err = os.Chmod(futureBookTickerFilePath, 0775)
					if err != nil {
						logger.Debugf("%v", err)
					}

				}

			}
		}

	}

}
