package part

import (
	"sync"
	"testing"
)

func TestMain4(t *testing.T) {
	buf := NewFlexBlocks[byte](1)
	if tmpbuf, putBack, e := buf.Get(); e == nil {
		tmpbuf = append(tmpbuf[:0], []byte("123")...)
		// do something with tmpbuf
		putBack(tmpbuf)
	} else {
		t.Fail()
	}
	if tmpbuf, putBack, e := buf.Get(); e == nil {
		if cap(tmpbuf) != 8 {
			t.Fatal()
		}
		tmpbuf = append(tmpbuf[:0], []byte("123")...)
		// do something with tmpbuf
		putBack(tmpbuf)
	} else {
		t.Fail()
	}
}

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

func TestMain2(t *testing.T) {
	buf := NewBlocks[byte](1024, 1)
	if tmpbuf, e := buf.GetAuto(); e == nil {
		clear(tmpbuf)
	} else {
		t.Fatal()
	}
	if tmpbuf, e := buf.GetAuto(); e == nil {
		clear(tmpbuf)
		if tmpbuf, e := buf.GetAuto(); e != nil {
			clear(tmpbuf)
			t.Fatal()
		}
	}
}

// 575.2 ns/op             7 B/op          0 allocs/op
func Benchmark5(b *testing.B) {
	type ie struct {
		a int
		b byte
		c string
	}
	buf := NewPoolBlock[ie]()
	for b.Loop() {
		{
			d := buf.Get()
			buf.Put(d)
		}
	}
}

// 567.2 ns/op             6 B/op          0 allocs/op
func Benchmark4(b *testing.B) {
	buf := NewPoolBlocks[byte]()
	for b.Loop() {
		{
			d := (buf.Get())
			buf.Put(d)
		}
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

func Benchmark6(b *testing.B) {
	var t = NewPoolBlock[int]()
	for b.Loop() {
		t.Put(t.Get())
	}
}

func Benchmark7(b *testing.B) {
	var p = sync.Pool{
		New: func() any {
			i := 1
			return &i
		},
	}
	for b.Loop() {
		p.Put(p.Get())
	}
}

func TestMain5(t *testing.T) {
	buf := NewPoolBlocks[byte]()

	tmpbuf := buf.Get()
	*tmpbuf = append((*tmpbuf)[:0], []byte("123")...)
	buf.Put(tmpbuf)

	{
		tmpbuf := buf.Get()
		if cap(*tmpbuf) != 8 {
			t.Fatal(cap(*tmpbuf))
		}
		*tmpbuf = append((*tmpbuf)[:0], []byte("123")...)
		buf.Put(tmpbuf)
	}
}
