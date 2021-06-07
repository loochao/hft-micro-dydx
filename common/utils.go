package common

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"math"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"time"
	"unsafe"
)

//go:generate go run symbols_gen.go

const (
	HashSHA1 = iota
	HashSHA256
	HashSHA512
	HashSHA512384
	HashMD5
)

func FindTailZeroStart(b []byte) (bool, int) {
	l := len(b)
	t := l
	for t > 0 {
		if b[t-1] == '0' {
			t--
			continue
		}
		break
	}
	return t != l, t
}

func RemoveTailZero(b []byte) []byte {
	if hasZero, s := FindTailZeroStart(b); hasZero {
		return b[:s]
	}
	return b
}

func EncodeURLValues(urlPath string, values url.Values) string {
	u := urlPath
	if len(values) > 0 {
		u += "?" + values.Encode()
	}
	return u
}

func GetHMAC(hashType int, input, key []byte) []byte {
	var hasher func() hash.Hash

	switch hashType {
	case HashSHA1:
		hasher = sha1.New
	case HashSHA256:
		hasher = sha256.New
	case HashSHA512:
		hasher = sha512.New
	case HashSHA512384:
		hasher = sha512.New384
	case HashMD5:
		hasher = md5.New
	}

	h := hmac.New(hasher, key)
	h.Write(input)
	return h.Sum(nil)
}

//func ParseFloat(data []byte) {
//	startZero := true
//	startIndex := 0
//	length := len(data)
//	dotIndex := length
//	for i := range data {
//		if startZero && data[i] != '0' {
//			startZero = true
//			startIndex = i
//		}else if data[i] == '.' {
//			dotIndex = i
//		}
//	}
//}

var float64pow10 = []float64{
	1e0, 1e1, 1e2, 1e3, 1e4, 1e5, 1e6, 1e7, 1e8, 1e9,
	1e10, 1e11, 1e12, 1e13, 1e14, 1e15, 1e16, 1e17, 1e18, 1e19,
	1e20, 1e21, 1e22,
}

func ParseFloat(s []byte) (float64, error) {
	mantissa := uint64(0)
	exp := 0
	base := uint64(10)
	sawDot := false
	sawDigits := false
	nd := 0
	dp := 0
	negative := false
loop:
	for i := 0; i < len(s); i++ {
		switch c := s[i]; true {
		case c == '-' && i == 0:
			negative = true
			continue
		case c == '.':
			if sawDot {
				break loop
			}
			sawDot = true
			dp = nd
			continue

		case '0' <= c && c <= '9':
			sawDigits = true
			if c == '0' && nd == 0 { // ignore leading zeros
				dp--
				continue
			}
			nd++
			mantissa *= base
			mantissa += uint64(c - '0')
			continue
		}
		v, err := strconv.ParseFloat(*(*string)(unsafe.Pointer(&s)), 64)
		if err != nil {
			return 0, fmt.Errorf("ParseFloat error bad byte %v @ %d in %s", s[i], i, s)
		} else {
			return v, nil
		}
	}
	if !sawDigits {
		v, err := strconv.ParseFloat(*(*string)(unsafe.Pointer(&s)), 64)
		if err != nil {
			return 0, errors.New("ParseFloat error no digits")
		} else {
			return v, nil
		}
	}
	if !sawDot {
		dp = nd
	}
	if mantissa != 0 {
		exp = dp - nd
	}
	if -exp < 0 || -exp > 22 {
		return 0, fmt.Errorf("bad -exp %d %s", -exp, s)
	}
	if !negative {
		return float64(mantissa) / float64pow10[-exp], nil
	} else {
		return -float64(mantissa) / float64pow10[-exp], nil
	}
}

func ParseInt(s []byte) (int64, error) {
	if len(s) == 0 {
		return 0, errors.New("ParseInt error no digits")
	}
	n := int64(0)
	negative := false
	for i := 0; i < len(s); i++ {
		switch c := s[i]; true {
		case c == '-' && i == 0:
			negative = true
		case '0' <= c && c <= '9':
			n *= 10
			n += int64(c - '0')
			continue
		}
		return strconv.ParseInt(UnsafeBytesToString(s), 10, 64)
	}
	if negative {
		return -n, nil
	} else {
		return n, nil
	}
}

func RecvWindow(d time.Duration) int64 {
	return int64(d) / int64(time.Millisecond)
}

// HexEncodeToString takes in a hexadecimal byte array and returns a string
func HexEncodeToString(input []byte) string {
	return hex.EncodeToString(input)
}

func StringDataContains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

type symbolValuePair struct {
	Symbol string
	Value  float64
}

func RankSymbols(symbols []string, values []float64) (map[int]string, error) {
	if len(symbols) != len(values) {
		return nil, fmt.Errorf("len %d != %d", len(symbols), len(values))
	}
	pairs := make([]symbolValuePair, len(symbols))
	for i, symbol := range symbols {
		pairs[i] = symbolValuePair{Symbol: symbol, Value: values[i]}
	}
	sort.SliceStable(pairs, func(i, j int) bool {
		return pairs[i].Value < pairs[j].Value
	})
	out := make(map[int]string)
	for i, pair := range pairs {
		out[i] = pair.Symbol
	}
	return out, nil
}

func Base64Encode(input []byte) string {
	return base64.StdEncoding.EncodeToString(input)
}

func FormatByPrecision(f float64, p int) StringFloat {
	return StringFloat(fmt.Sprintf("%."+fmt.Sprintf("%d", p)+"f", f))
}

func MergedStepSize(stepSizeA, stepSizeB float64) float64 {
	base := 1.0
	for math.Floor(stepSizeA*base) != stepSizeA*base || math.Floor(stepSizeB*base) != stepSizeB*base {
		base *= 10.0
	}
	return float64(LCM(int(stepSizeA*base), int(stepSizeB*base))) / base
}

func GCD(a, b int) int {
	for b != 0 {
		t := b
		b = a % b
		a = t
	}
	return a
}

func LCM(a, b int, integers ...int) int {
	result := a * b / GCD(a, b)

	for i := 0; i < len(integers); i++ {
		result = LCM(result, integers[i])
	}

	return result
}

func UnsafeBytesToString(b []byte) (s string) {
	var length = len(b)
	if length == 0 {
		return ""
	}
	stringHeader := (*reflect.StringHeader)(unsafe.Pointer(&s))
	stringHeader.Data = uintptr(unsafe.Pointer(&b[0]))
	stringHeader.Len = length
	return s
}

