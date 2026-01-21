package part

import (
	"fmt"
	"testing"
	"unsafe"
)

func TestHello(t *testing.T) {
	t.Log("ok")
}

func Benchmark1(b *testing.B) {
	data := "我"
	var a []byte = []byte(data)
	for b.Loop() {
		a = S2B(data)
	}
	fmt.Println(string(a))
}

func B2S(s []byte) string {
	return unsafe.String(unsafe.SliceData(s), len(s))
}

func S2B(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}
