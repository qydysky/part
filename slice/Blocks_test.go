package part

import (
	"testing"
)

func TestMain(t *testing.T) {
	buf := NewBlocks[byte](1024, 1)
	if tmpbuf, putBack, e := buf.Get(); e == nil {
		clear(tmpbuf)
		// do something with tmpbuf
		putBack()
	} else {
		t.Fail()
	}
	if tmpbuf, putBack, e := buf.Get(); e == nil {
		clear(tmpbuf)
		if tmpbuf, putBack, e := buf.Get(); e != ErrOverflow {
			clear(tmpbuf)
			t.Fail()
			// do something with tmpbuf
			putBack()
		}
		// do something with tmpbuf
		putBack()
	}
}

// 374.4 ns/op            32 B/op          1 allocs/op
func Benchmark(b *testing.B) {
	buf := NewBlocks[byte](1024, 1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, f, e := buf.Get(); e != nil {
			b.Fatal(e)
		} else {
			f()
		}
	}
}

// 895.5 ns/op            56 B/op          2 allocs/op
func Benchmark2(b *testing.B) {
	buf := NewBlocks[byte](1, 1000000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, e := buf.GetAuto(); e != nil {
			b.Fatal(e)
		}
	}
}
