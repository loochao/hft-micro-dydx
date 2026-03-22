package common

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"
)

func BenchmarkStringConcat(b *testing.B) {
	const id = "1"
	const address = "192.168.1.1"
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		s := id
		s += " " + address
		s += " " + time.Now().String()
		_ = s
	}
}

func BenchmarkStringFprintf(b *testing.B) {
	b.ReportAllocs()
	const id = "1"
	const address = "192.168.1.1"
	for n := 0; n < b.N; n++ {
		var b bytes.Buffer
		fmt.Fprintf(&b, "%s %v %v", id, address, time.Now())
		_ = b.String()
	}
}

func BenchmarkStringSprintf(b *testing.B) {
	b.ReportAllocs()
	const id = "1"
	const address = "192.168.1.1"
	for n := 0; n < b.N; n++ {
		_ = fmt.Sprintf("%s %v %v", id, address, time.Now())
	}
}


func BenchmarkStringByteAppend(b *testing.B) {
	b.ReportAllocs()
	const id = "1"
	const address = "192.168.1.1"
	for n := 0; n < b.N; n++ {
		b := make([]byte, 0, 50)
		b = append(b, id...)
		b = append(b, ' ')
		b = append(b, address...)
		b = append(b, ' ')
		b = time.Now().AppendFormat(b, "2006-01-02 15:04:05.999999999 -0700 MST")
		_ = string(b)
	}
}

func BenchmarkStringStringBuilder(b *testing.B) {
	b.ReportAllocs()
	const id = "1"
	const address = "192.168.1.1"
	for n := 0; n < b.N; n++ {
		var b strings.Builder
		b.WriteString(id)
		b.WriteString(" ")
		b.WriteString(address)
		b.WriteString(" ")
		b.WriteString(time.Now().String())
		_ = b.String()
	}
}