package main

import (
	"compress/gzip"
	"encoding/binary"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"io"
	"os"
	"testing"
	"time"
)

func BenchmarkRead(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		func() {
			f, err := os.OpenFile("/Users/chenjilin/Downloads/20210910-20210915-1INCHUSDT-1INCHUSDT-24h0m0s-3s-1ms.gz", os.O_RDONLY, 0600)
			if err != nil {
				b.Fatal(err)
			}
			gr, err := gzip.NewReader(f)
			if err != nil {
				b.Fatal(err)
			}
			outputData := &common.MatchedSpread{}
			for err != io.EOF {
				err = binary.Read(gr, binary.BigEndian, outputData)
				if err != nil && err != io.EOF {
					b.Fatal(err)
				}
			}
			err = gr.Close()
			if err != nil {
				b.Fatal(err)
			}
			err = f.Close()
			if err != nil {
				b.Fatal(err)
			}
		}()
	}
}

func TestRead(t *testing.T) {
	f, err := os.OpenFile("/Users/chenjilin/Downloads/20210820-20210826-VETUSDT-VETUSDT-24h0m0s-3s-25ms.gz", os.O_RDONLY, 0600)
	//f, err := os.OpenFile("/Users/chenjilin/Downloads/20210820-20210916-VETUSDTM-VETUSDT-24h0m0s-3s-1ms.gz", os.O_RDONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer gr.Close()
	outputData := &common.MatchedSpread{}
	counter := 0
	var startTime *int64
	totalTime := int64(0)
	for err != io.EOF {
		err = binary.Read(gr, binary.BigEndian, outputData)
		if err != nil && err != io.EOF {
			t.Fatal(err)
		}
		if startTime == nil {
			startTime = new(int64)
			*startTime = outputData.EventTime
		} else {
			totalTime += outputData.EventTime - *startTime
			if outputData.EventTime - *startTime < 0 {
				logger.Debugf("bad time start %d end %d", *startTime, outputData.EventTime)
			}
			*startTime = outputData.EventTime
		}
		counter++
	}
	logger.Debugf("%v",  time.Duration(totalTime)/time.Duration(counter))
	logger.Debugf("%d", counter)
}

///*func TestAbc(t *testing.T) {
//
//	type packet struct {
//		Sensid int64
//		Locid  uint16
//		Tstamp uint32
//		Temp   int16
//	}
//
//	rand.Seed(time.Now().UnixNano())
//
//	dataIn := DataRow{
//		ServerTime: time.Now().UnixNano(),
//		LongLastEnter: time.Now().Sub(time.Now().Truncate(time.Second)).Seconds(),
//	}
//
//	buf := new(bytes.Buffer)
//
//	err := binary.Write(buf, binary.BigEndian, &dataIn)
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	offset := buf.Len()
//	logger.Debugf("buf len %d sizeof struct %d", offset, unsafe.Sizeof(dataIn))
//	f, err := os.OpenFile("/Users/chenjilin/Downloads/test_binary", os.O_RDONLY, 0600)
//	if err == nil {
//		ret, err := f.Seek(int64(-offset), 2)
//		if err != nil {
//			t.Fatal(err)
//		}
//		logger.Debugf("%d", ret)
//		var dataOut DataRow
//		err = binary.Read(f, binary.BigEndian, &dataOut)
//		if err != nil && err != io.EOF{
//			t.Fatal(err)
//		}
//		logger.Debugf("%v" , dataOut)
//	}
//
//
//	f, err = os.OpenFile("/Users/chenjilin/Downloads/test_binary", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
//	if err != nil {
//		panic(err)
//	}
//
//	err = binary.Write(f, binary.BigEndian, &dataIn)
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	err = f.Close()
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	f, err = os.OpenFile("/Users/chenjilin/Downloads/test_binary", os.O_RDONLY, 0600)
//	if err != nil {
//		t.Fatal(err)
//	}
//	counter := 0
//	for err != io.EOF{
//		var dataOut DataRow
//		err = binary.Read(f, binary.BigEndian, &dataOut)
//		if err != nil && err != io.EOF{
//			t.Fatal(err)
//			break
//		}
//		counter++
//		if err != io.EOF {
//			logger.Debugf("%d %v", counter, dataOut)
//		}
//	}
//	logger.Debugf("READ ALL")
//
//}
//*/
