package part

import (
	"testing"
)

func TestMain(t *testing.T) {
	t.Log(Md5String("sss"))
	t.Log(Md5String("sss"))
}

// 2288 ns/op             140 B/op          4 allocs/op
// 382.5 ns/op            16 B/op          1 allocs/op
// 330.2 ns/op             0 B/op          0 allocs/op
func Benchmark1(b *testing.B) {
	for b.Loop() {
		_ = Md5("sss")
	}
}
