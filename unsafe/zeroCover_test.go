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
