package part

import (
	"bytes"
	"io"
	"testing"
)

func Test_SliceIndex(t *testing.T) {
	var (
		buf = []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
		b   = NewSliceIndex(buf)
	)
	b.Append(0, 3)
	if !b.Equal(buf[0:3]) {
		t.Fatal()
	}
	if b.Equal(buf[0:2]) {
		t.Fatal()
	}
	if b.Equal(buf[0:4]) {
		t.Fatal()
	}
	b.Merge(0, 3)
	if !b.Equal(buf[0:3]) {
		t.Fatal()
	}
	b.Append(3, 4)
	if !b.Equal(buf[0:4]) {
		t.Fatal()
	}
	buf1 := bytes.NewBuffer(make([]byte, 10))
	io.Copy(buf1, b)
	if buf1.String() == "0123" {
		t.Fatal()
	}
}

func Benchmark_SI1(b *testing.B) {
	var (
		buf = []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
		bu  = NewSliceIndex(buf)
	)
	bu.Append(0, 3)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !bu.Equal(buf[0:3]) {
			b.Fatal()
		}
	}
}

func Benchmark_SI3(b *testing.B) {
	var (
		buf = []byte("abc")
		bu  = NewSliceIndexNoLock(buf)
	)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bu.Append(0, 3)
	}
}

func Benchmark_SI2(b *testing.B) {
	var (
		buf = []byte{}
	)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf = append(buf, []byte("abccc")...)
	}
	_ = buf
}
