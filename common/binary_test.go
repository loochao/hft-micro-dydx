package common

import (
	"bytes"
	"encoding/binary"
	"github.com/geometrybase/hft-micro/logger"
	"testing"
	"time"
)

func TestBinaryKline(t *testing.T) {
	buf := new(bytes.Buffer)
	kline := BinaryKline{
		CloseTime: time.Now().UnixNano(),
		Open:      1.0,
		High:      2.0,
		Low:       1.0,
		Close:     1.1,
		Volume:    10000,
	}
	err := binary.Write(buf, binary.BigEndian, kline)
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%d", len(buf.Bytes()))

	outKline := BinaryKline{}
	err = binary.Read(buf, binary.BigEndian, &outKline)
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", outKline)

}
