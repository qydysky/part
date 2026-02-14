package part

import (
	"bytes"
	"testing"
)

func Test1(t *testing.T) {
	var (
		s   = []byte{'1', '2', '2', '3', '4'}
		sep = []byte{'2'}
	)
	t.Log(sep, bytes.SplitAfter(s, sep))
	SplitAfter(s, sep, func(v []byte) bool { t.Log(v); return true })
}

// byte.Split Benchmark8-2    16173733                71.05 ns/op           80 B/op          1 allocs/op
// bytes.SplitSeq Benchmark8-2     9234154               126.3 ns/op            96 B/op          4 allocs/op
// SplitSeq Benchmark8-2    72654700                15.51 ns/op            0 B/op          0 allocs/op
func Benchmark8(b *testing.B) {
	var (
		s   = []byte{'1', '2', '2', '3', '4'}
		sep = []byte{'2'}
	)
	for b.Loop() {
		SplitAfter(s, sep, func(v []byte) bool { return true })
	}
}
