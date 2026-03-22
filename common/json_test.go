package common

import (
	"github.com/geometrybase/hft-micro/logger"
	"github.com/stretchr/testify/assert"
	"math"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"unsafe"
)

// Global exported variables are used to store the
// return values of functions measured in the benchmarks.
// Storing the results in these variables prevents the compiler
// from completely optimizing the benchmarked functions away.
var (
	GlobalI int
	GlobalB bool
	GlobalF float64
)

type atofTest struct {
	in  string
	out string
	err error
}

var atoftests = []atofTest{
	{"", "0", strconv.ErrSyntax},
	{"1", "1", nil},
	{"+1", "1", nil},
	{"1x", "0", strconv.ErrSyntax},
	{"1.1.", "0", strconv.ErrSyntax},
	{"1e23", "1e+23", nil},
	{"1E23", "1e+23", nil},
	{"100000000000000000000000", "1e+23", nil},
	{"1e-100", "1e-100", nil},
	{"123456700", "1.234567e+08", nil},
	{"99999999999999974834176", "9.999999999999997e+22", nil},
	{"100000000000000000000001", "1.0000000000000001e+23", nil},
	{"100000000000000008388608", "1.0000000000000001e+23", nil},
	{"100000000000000016777215", "1.0000000000000001e+23", nil},
	{"100000000000000016777216", "1.0000000000000003e+23", nil},
	{"-1", "-1", nil},
	{"-0.1", "-0.1", nil},
	{"-0", "-0", nil},
	{"1e-20", "1e-20", nil},
	{"625e-3", "0.625", nil},

	// Hexadecimal floating-point.
	{"0x1p0", "1", nil},
	{"0x1p1", "2", nil},
	{"0x1p-1", "0.5", nil},
	{"0x1ep-1", "15", nil},
	{"-0x1ep-1", "-15", nil},
	{"-0x1_ep-1", "-15", nil},
	{"0x1p-200", "6.223015277861142e-61", nil},
	{"0x1p200", "1.6069380442589903e+60", nil},
	{"0x1fFe2.p0", "131042", nil},
	{"0x1fFe2.P0", "131042", nil},
	{"-0x2p3", "-16", nil},
	{"0x0.fp4", "15", nil},
	{"0x0.fp0", "0.9375", nil},
	{"0x1e2", "0", strconv.ErrSyntax},
	{"1p2", "0", strconv.ErrSyntax},

	// zeros
	{"0", "0", nil},
	{"0e0", "0", nil},
	{"-0e0", "-0", nil},
	{"+0e0", "0", nil},
	{"0e-0", "0", nil},
	{"-0e-0", "-0", nil},
	{"+0e-0", "0", nil},
	{"0e+0", "0", nil},
	{"-0e+0", "-0", nil},
	{"+0e+0", "0", nil},
	{"0e+01234567890123456789", "0", nil},
	{"0.00e-01234567890123456789", "0", nil},
	{"-0e+01234567890123456789", "-0", nil},
	{"-0.00e-01234567890123456789", "-0", nil},
	{"0x0p+01234567890123456789", "0", nil},
	{"0x0.00p-01234567890123456789", "0", nil},
	{"-0x0p+01234567890123456789", "-0", nil},
	{"-0x0.00p-01234567890123456789", "-0", nil},

	{"0e291", "0", nil}, // issue 15364
	{"0e292", "0", nil}, // issue 15364
	{"0e347", "0", nil}, // issue 15364
	{"0e348", "0", nil}, // issue 15364
	{"-0e291", "-0", nil},
	{"-0e292", "-0", nil},
	{"-0e347", "-0", nil},
	{"-0e348", "-0", nil},
	{"0x0p126", "0", nil},
	{"0x0p127", "0", nil},
	{"0x0p128", "0", nil},
	{"0x0p129", "0", nil},
	{"0x0p130", "0", nil},
	{"0x0p1022", "0", nil},
	{"0x0p1023", "0", nil},
	{"0x0p1024", "0", nil},
	{"0x0p1025", "0", nil},
	{"0x0p1026", "0", nil},
	{"-0x0p126", "-0", nil},
	{"-0x0p127", "-0", nil},
	{"-0x0p128", "-0", nil},
	{"-0x0p129", "-0", nil},
	{"-0x0p130", "-0", nil},
	{"-0x0p1022", "-0", nil},
	{"-0x0p1023", "-0", nil},
	{"-0x0p1024", "-0", nil},
	{"-0x0p1025", "-0", nil},
	{"-0x0p1026", "-0", nil},

	// NaNs
	{"nan", "NaN", nil},
	{"NaN", "NaN", nil},
	{"NAN", "NaN", nil},

	// Infs
	{"inf", "+Inf", nil},
	{"-Inf", "-Inf", nil},
	{"+INF", "+Inf", nil},
	{"-Infinity", "-Inf", nil},
	{"+INFINITY", "+Inf", nil},
	{"Infinity", "+Inf", nil},

	// largest float64
	{"1.7976931348623157e308", "1.7976931348623157e+308", nil},
	{"-1.7976931348623157e308", "-1.7976931348623157e+308", nil},
	{"0x1.fffffffffffffp1023", "1.7976931348623157e+308", nil},
	{"-0x1.fffffffffffffp1023", "-1.7976931348623157e+308", nil},
	{"0x1fffffffffffffp+971", "1.7976931348623157e+308", nil},
	{"-0x1fffffffffffffp+971", "-1.7976931348623157e+308", nil},
	{"0x.1fffffffffffffp1027", "1.7976931348623157e+308", nil},
	{"-0x.1fffffffffffffp1027", "-1.7976931348623157e+308", nil},

	// next float64 - too large
	{"1.7976931348623159e308", "+Inf", strconv.ErrRange},
	{"-1.7976931348623159e308", "-Inf", strconv.ErrRange},
	{"0x1p1024", "+Inf", strconv.ErrRange},
	{"-0x1p1024", "-Inf", strconv.ErrRange},
	{"0x2p1023", "+Inf", strconv.ErrRange},
	{"-0x2p1023", "-Inf", strconv.ErrRange},
	{"0x.1p1028", "+Inf", strconv.ErrRange},
	{"-0x.1p1028", "-Inf", strconv.ErrRange},
	{"0x.2p1027", "+Inf", strconv.ErrRange},
	{"-0x.2p1027", "-Inf", strconv.ErrRange},

	// the border is ...158079
	// borderline - okay
	{"1.7976931348623158e308", "1.7976931348623157e+308", nil},
	{"-1.7976931348623158e308", "-1.7976931348623157e+308", nil},
	{"0x1.fffffffffffff7fffp1023", "1.7976931348623157e+308", nil},
	{"-0x1.fffffffffffff7fffp1023", "-1.7976931348623157e+308", nil},
	// borderline - too large
	{"1.797693134862315808e308", "+Inf", strconv.ErrRange},
	{"-1.797693134862315808e308", "-Inf", strconv.ErrRange},
	{"0x1.fffffffffffff8p1023", "+Inf", strconv.ErrRange},
	{"-0x1.fffffffffffff8p1023", "-Inf", strconv.ErrRange},
	{"0x1fffffffffffff.8p+971", "+Inf", strconv.ErrRange},
	{"-0x1fffffffffffff8p+967", "-Inf", strconv.ErrRange},
	{"0x.1fffffffffffff8p1027", "+Inf", strconv.ErrRange},
	{"-0x.1fffffffffffff9p1027", "-Inf", strconv.ErrRange},

	// a little too large
	{"1e308", "1e+308", nil},
	{"2e308", "+Inf", strconv.ErrRange},
	{"1e309", "+Inf", strconv.ErrRange},
	{"0x1p1025", "+Inf", strconv.ErrRange},

	// way too large
	{"1e310", "+Inf", strconv.ErrRange},
	{"-1e310", "-Inf", strconv.ErrRange},
	{"1e400", "+Inf", strconv.ErrRange},
	{"-1e400", "-Inf", strconv.ErrRange},
	{"1e400000", "+Inf", strconv.ErrRange},
	{"-1e400000", "-Inf", strconv.ErrRange},
	{"0x1p1030", "+Inf", strconv.ErrRange},
	{"0x1p2000", "+Inf", strconv.ErrRange},
	{"0x1p2000000000", "+Inf", strconv.ErrRange},
	{"-0x1p1030", "-Inf", strconv.ErrRange},
	{"-0x1p2000", "-Inf", strconv.ErrRange},
	{"-0x1p2000000000", "-Inf", strconv.ErrRange},

	// denormalized
	{"1e-305", "1e-305", nil},
	{"1e-306", "1e-306", nil},
	{"1e-307", "1e-307", nil},
	{"1e-308", "1e-308", nil},
	{"1e-309", "1e-309", nil},
	{"1e-310", "1e-310", nil},
	{"1e-322", "1e-322", nil},
	// smallest denormal
	{"5e-324", "5e-324", nil},
	{"4e-324", "5e-324", nil},
	{"3e-324", "5e-324", nil},
	// too small
	{"2e-324", "0", nil},
	// way too small
	{"1e-350", "0", nil},
	{"1e-400000", "0", nil},

	// Near denormals and denormals.
	{"0x2.00000000000000p-1010", "1.8227805048890994e-304", nil}, // 0x00e0000000000000
	{"0x1.fffffffffffff0p-1010", "1.8227805048890992e-304", nil}, // 0x00dfffffffffffff
	{"0x1.fffffffffffff7p-1010", "1.8227805048890992e-304", nil}, // rounded down
	{"0x1.fffffffffffff8p-1010", "1.8227805048890994e-304", nil}, // rounded up
	{"0x1.fffffffffffff9p-1010", "1.8227805048890994e-304", nil}, // rounded up

	{"0x2.00000000000000p-1022", "4.450147717014403e-308", nil},  // 0x0020000000000000
	{"0x1.fffffffffffff0p-1022", "4.4501477170144023e-308", nil}, // 0x001fffffffffffff
	{"0x1.fffffffffffff7p-1022", "4.4501477170144023e-308", nil}, // rounded down
	{"0x1.fffffffffffff8p-1022", "4.450147717014403e-308", nil},  // rounded up
	{"0x1.fffffffffffff9p-1022", "4.450147717014403e-308", nil},  // rounded up

	{"0x1.00000000000000p-1022", "2.2250738585072014e-308", nil}, // 0x0010000000000000
	{"0x0.fffffffffffff0p-1022", "2.225073858507201e-308", nil},  // 0x000fffffffffffff
	{"0x0.ffffffffffffe0p-1022", "2.2250738585072004e-308", nil}, // 0x000ffffffffffffe
	{"0x0.ffffffffffffe7p-1022", "2.2250738585072004e-308", nil}, // rounded down
	{"0x1.ffffffffffffe8p-1023", "2.225073858507201e-308", nil},  // rounded up
	{"0x1.ffffffffffffe9p-1023", "2.225073858507201e-308", nil},  // rounded up

	{"0x0.00000003fffff0p-1022", "2.072261e-317", nil},  // 0x00000000003fffff
	{"0x0.00000003456780p-1022", "1.694649e-317", nil},  // 0x0000000000345678
	{"0x0.00000003456787p-1022", "1.694649e-317", nil},  // rounded down
	{"0x0.00000003456788p-1022", "1.694649e-317", nil},  // rounded down (half to even)
	{"0x0.00000003456790p-1022", "1.6946496e-317", nil}, // 0x0000000000345679
	{"0x0.00000003456789p-1022", "1.6946496e-317", nil}, // rounded up

	{"0x0.0000000345678800000000000000000000000001p-1022", "1.6946496e-317", nil}, // rounded up

	{"0x0.000000000000f0p-1022", "7.4e-323", nil}, // 0x000000000000000f
	{"0x0.00000000000060p-1022", "3e-323", nil},   // 0x0000000000000006
	{"0x0.00000000000058p-1022", "3e-323", nil},   // rounded up
	{"0x0.00000000000057p-1022", "2.5e-323", nil}, // rounded down
	{"0x0.00000000000050p-1022", "2.5e-323", nil}, // 0x0000000000000005

	{"0x0.00000000000010p-1022", "5e-324", nil},  // 0x0000000000000001
	{"0x0.000000000000081p-1022", "5e-324", nil}, // rounded up
	{"0x0.00000000000008p-1022", "0", nil},       // rounded down
	{"0x0.00000000000007fp-1022", "0", nil},      // rounded down

	// try to overflow exponent
	{"1e-4294967296", "0", nil},
	{"1e+4294967296", "+Inf", strconv.ErrRange},
	{"1e-18446744073709551616", "0", nil},
	{"1e+18446744073709551616", "+Inf", strconv.ErrRange},
	{"0x1p-4294967296", "0", nil},
	{"0x1p+4294967296", "+Inf", strconv.ErrRange},
	{"0x1p-18446744073709551616", "0", nil},
	{"0x1p+18446744073709551616", "+Inf", strconv.ErrRange},

	// Parse errors
	{"1e", "0", strconv.ErrSyntax},
	{"1e-", "0", strconv.ErrSyntax},
	{".e-1", "0", strconv.ErrSyntax},
	{"1\x00.2", "0", strconv.ErrSyntax},
	{"0x", "0", strconv.ErrSyntax},
	{"0x.", "0", strconv.ErrSyntax},
	{"0x1", "0", strconv.ErrSyntax},
	{"0x.1", "0", strconv.ErrSyntax},
	{"0x1p", "0", strconv.ErrSyntax},
	{"0x.1p", "0", strconv.ErrSyntax},
	{"0x1p+", "0", strconv.ErrSyntax},
	{"0x.1p+", "0", strconv.ErrSyntax},
	{"0x1p-", "0", strconv.ErrSyntax},
	{"0x.1p-", "0", strconv.ErrSyntax},
	{"0x1p+2", "4", nil},
	{"0x.1p+2", "0.25", nil},
	{"0x1p-2", "0.25", nil},
	{"0x.1p-2", "0.015625", nil},

	// https://www.exploringbinary.com/java-hangs-when-converting-2-2250738585072012e-308/
	{"2.2250738585072012e-308", "2.2250738585072014e-308", nil},
	// https://www.exploringbinary.com/php-hangs-on-numeric-value-2-2250738585072011e-308/
	{"2.2250738585072011e-308", "2.225073858507201e-308", nil},

	// A very large number (initially wrongly parsed by the fast algorithm).
	{"4.630813248087435e+307", "4.630813248087435e+307", nil},

	// A different kind of very large number.
	{"22.222222222222222", "22.22222222222222", nil},
	{"2." + strings.Repeat("2", 4000) + "e+1", "22.22222222222222", nil},
	{"0x1.1111111111111p222", "7.18931911124017e+66", nil},
	{"0x2.2222222222222p221", "7.18931911124017e+66", nil},
	{"0x2." + strings.Repeat("2", 4000) + "p221", "7.18931911124017e+66", nil},

	// Exactly halfway between 1 and math.Nextafter(1, 2).
	// Round to even (down).
	{"1.00000000000000011102230246251565404236316680908203125", "1", nil},
	{"0x1.00000000000008p0", "1", nil},
	// Slightly lower; still round down.
	{"1.00000000000000011102230246251565404236316680908203124", "1", nil},
	{"0x1.00000000000007Fp0", "1", nil},
	// Slightly higher; round up.
	{"1.00000000000000011102230246251565404236316680908203126", "1.0000000000000002", nil},
	{"0x1.000000000000081p0", "1.0000000000000002", nil},
	{"0x1.00000000000009p0", "1.0000000000000002", nil},
	// Slightly higher, but you have to read all the way to the end.
	{"1.00000000000000011102230246251565404236316680908203125" + strings.Repeat("0", 10000) + "1", "1.0000000000000002", nil},
	{"0x1.00000000000008" + strings.Repeat("0", 10000) + "1p0", "1.0000000000000002", nil},

	// Halfway between x := math.Nextafter(1, 2) and math.Nextafter(x, 2)
	// Round to even (up).
	{"1.00000000000000033306690738754696212708950042724609375", "1.0000000000000004", nil},
	{"0x1.00000000000018p0", "1.0000000000000004", nil},

	// Halfway between 1090544144181609278303144771584 and 1090544144181609419040633126912
	// (15497564393479157p+46, should round to even 15497564393479156p+46, issue 36657)
	{"1090544144181609348671888949248", "1.0905441441816093e+30", nil},
	// slightly above, rounds up
	{"1090544144181609348835077142190", "1.0905441441816094e+30", nil},

	// Underscores.
	{"1_23.50_0_0e+1_2", "1.235e+14", nil},
	{"-_123.5e+12", "0", strconv.ErrSyntax},
	{"+_123.5e+12", "0", strconv.ErrSyntax},
	{"_123.5e+12", "0", strconv.ErrSyntax},
	{"1__23.5e+12", "0", strconv.ErrSyntax},
	{"123_.5e+12", "0", strconv.ErrSyntax},
	{"123._5e+12", "0", strconv.ErrSyntax},
	{"123.5_e+12", "0", strconv.ErrSyntax},
	{"123.5__0e+12", "0", strconv.ErrSyntax},
	{"123.5e_+12", "0", strconv.ErrSyntax},
	{"123.5e+_12", "0", strconv.ErrSyntax},
	{"123.5e_-12", "0", strconv.ErrSyntax},
	{"123.5e-_12", "0", strconv.ErrSyntax},
	{"123.5e+1__2", "0", strconv.ErrSyntax},
	{"123.5e+12_", "0", strconv.ErrSyntax},

	{"0x_1_2.3_4_5p+1_2", "74565", nil},
	{"-_0x12.345p+12", "0", strconv.ErrSyntax},
	{"+_0x12.345p+12", "0", strconv.ErrSyntax},
	{"_0x12.345p+12", "0", strconv.ErrSyntax},
	{"0x__12.345p+12", "0", strconv.ErrSyntax},
	{"0x1__2.345p+12", "0", strconv.ErrSyntax},
	{"0x12_.345p+12", "0", strconv.ErrSyntax},
	{"0x12._345p+12", "0", strconv.ErrSyntax},
	{"0x12.3__45p+12", "0", strconv.ErrSyntax},
	{"0x12.345_p+12", "0", strconv.ErrSyntax},
	{"0x12.345p_+12", "0", strconv.ErrSyntax},
	{"0x12.345p+_12", "0", strconv.ErrSyntax},
	{"0x12.345p_-12", "0", strconv.ErrSyntax},
	{"0x12.345p-_12", "0", strconv.ErrSyntax},
	{"0x12.345p+1__2", "0", strconv.ErrSyntax},
	{"0x12.345p+12_", "0", strconv.ErrSyntax},
}

type atofSimpleTest struct {
	x float64
	s string
}

var (
	atofOnce               sync.Once
	atofRandomTests        []atofSimpleTest
	benchmarksRandomBits   [1024]string
	benchmarksRandomNormal [1024]string
)

func initAtof() {
	atofOnce.Do(initAtofOnce)
}

func initAtofOnce() {
	// The atof routines return strconv.NumErrors wrapping
	// the error and the string. Convert the table above.
	for i := range atoftests {
		test := &atoftests[i]
		if test.err != nil {
			test.err = &strconv.NumError{"ParseFloat", test.in, test.err}
		}
	}
}

func TestAtof(t *testing.T) {
	initAtof()
	for i := 0; i < len(atoftests); i++ {
		test := &atoftests[i]
		out, err := ParseDecimal([]byte(test.in))
		outs := strconv.FormatFloat(out, 'g', -1, 64)
		if outs != test.out || !reflect.DeepEqual(err, test.err) {
			logger.Debugf("%v", test)
			t.Errorf("ParseDecimal(%v) = %v, %v want %v, %v",
				test.in, out, err, test.out, test.err)
		}
	}
}

func TestParseDecimal(t *testing.T) {
	s := []byte(`3.14159265`)
	v, err := ParseDecimal(s)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, string(s), strconv.FormatFloat(v, 'f', -1, 64))

	s = []byte(`22.2222222222222`)
	v, err = ParseDecimal(s)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, string(s), strconv.FormatFloat(v, 'f', -1, 64))
}

func BenchmarkParseDecimal(b *testing.B) {
	s := []byte(`22.2222222222222`)
	x := 0.0
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		x, _ = ParseDecimal(s)
	}
	GlobalF = x
}

func BenchmarkFloat64frombits(b *testing.B) {
	s := uint64(222222222222222)
	x := 0.0
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 10000; j++ {
			x = math.Float64frombits(s)
		}
	}
	GlobalF = x
}

func BenchmarkUnsafeU64ToF64(b *testing.B) {
	s := uint64(222222222222222)
	x := 0.0
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 10000; j++ {
			x = *(*float64)(unsafe.Pointer(&s))
		}
	}
	GlobalF = x
}

func BenchmarkU64ToF64(b *testing.B) {
	s := uint64(222222222222222)
	x := 0.0
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 10000; j++ {
			x = float64(s)
		}
	}
	GlobalF = x
}

func BenchmarkStrconvParseFloat(b *testing.B) {
	s := `22.2222222222222`
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = strconv.ParseFloat(s, 64)
	}
}
