package part

import (
	"bytes"
	"io"
	"slices"
	"testing"

	pio "github.com/qydysky/part/io"
)

func Test_1(t *testing.T) {
	var (
		buf  = []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
		tbuf = bytes.NewBuffer([]byte{})
		b    = NewSliceIndexNoLock(buf)
	)
	b.Append(0, len(buf))
	if n, e := io.Copy(tbuf, pio.WrapIoWriteTo().SetRaw(b)); e != nil || n != 10 {
		t.Fatal()
	}
	if !slices.Equal(tbuf.Bytes(), buf) {
		t.Fatal()
	}
}

// 2692443               444.8 ns/op            16 B/op          1 allocs/op
func Benchmark6(b *testing.B) {
	var (
		buf  = []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
		tbuf = bytes.NewBuffer([]byte{})
		bi   = NewSliceIndexNoLock(buf)
		w    = pio.WrapIoWriteTo()
	)
	for b.Loop() {
		bi.Append(0, len(buf))
		tbuf.Reset()
		if n, e := io.Copy(tbuf, w.SetRaw(bi)); e != nil || n != 10 {
			b.Fatal(e)
		}
		if !slices.Equal(tbuf.Bytes(), buf) {
			b.Fatal()
		}
	}
}

// 3107595               378.0 ns/op            48 B/op          1 allocs/op
func Benchmark7(b *testing.B) {
	var (
		buf  = []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
		tbuf = bytes.NewBuffer([]byte{})
	)
	for b.Loop() {
		tbuf.Reset()
		if n, e := io.Copy(tbuf, bytes.NewReader(buf)); e != nil || n != 10 {
			b.Fatal(e)
		}
		if !slices.Equal(tbuf.Bytes(), buf) {
			b.Fatal()
		}
	}
}

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
