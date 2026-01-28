package unsafe

import (
	"testing"
)

func Test1(t *testing.T) {
	if testing.Benchmark(Benchmark1).MemAllocs != 0 {
		t.Fatal()
	}
}

func Test2(t *testing.T) {
	if testing.Benchmark(Benchmark2).MemAllocs != 0 {
		t.Fatal()
	}
}

func Test3(t *testing.T) {
	data := []byte("123")
	s := B2S(data)
	if s != "123" {
		t.Fatal()
	}
	data[0] = '3'
	if s != "323" {
		t.Fatal()
	}
}

func Test4(t *testing.T) {
	data := []byte("123")
	s := B2S(data)
	b := S2B(s)
	b[0] = '0'
	data[0] = '1'
}

func Benchmark1(b *testing.B) {
	data := "我1a?？"
	var a []byte = []byte(data)
	for b.Loop() {
		a = S2B(data)
	}
	if string(a) != "我1a?？" {
		b.Fatal()
	}
}

func Benchmark2(b *testing.B) {
	data := []byte("我1a?？")
	var a string = string(data)
	for b.Loop() {
		a = B2S(data)
	}
	if a != "我1a?？" {
		b.Fatal()
	}
}
