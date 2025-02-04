package part

import (
	"bytes"
	"testing"
)

func TestFIFO(t *testing.T) {
	fifo := NewFIFO[byte](5)

	if fifo.In([]byte("012345")) != ErrFIFOOverflow {
		t.Fatal()
	}
	fifo.Clear()

	if fifo.In([]byte("012")) != nil {
		t.Fatal()
	}
	if fifo.In([]byte("345")) != ErrFIFOOverflow {
		t.Fatal()
	}
	fifo.Clear()

	if fifo.In([]byte("012")) != nil {
		t.Fatal()
	}
	if fifo.In([]byte("34")) != nil {
		t.Fatal()
	}
	fifo.Clear()

	if fifo.In([]byte("012")) != nil {
		t.Fatal()
	}
	if tmp, e := fifo.Out(); e != nil || !bytes.Equal(tmp, []byte("012")) {
		t.Fatal()
	}
	fifo.Clear()

	if fifo.In([]byte("01")) != nil {
		t.Fatal()
	}
	if fifo.Size() != 2 {
		t.Fatal()
	}
	if e := fifo.In([]byte("234")); e != nil {
		t.Fatal(e)
	}
	if fifo.Size() != 5 {
		t.Fatal()
	}
	if tmp, e := fifo.Out(); e != nil || !bytes.Equal(tmp, []byte("01")) {
		t.Fatal()
	}
	if fifo.In([]byte("56")) != nil {
		t.Fatal()
	}
	if tmp, e := fifo.Out(); e != nil || !bytes.Equal(tmp, []byte("234")) {
		t.Fatal()
	}
	if tmp, e := fifo.Out(); e != nil || !bytes.Equal(tmp, []byte("56")) {
		t.Fatal()
	}
	fifo.Clear()

	// if fifo.In([]byte("012")) != nil {
	// 	t.Fatal()
	// }
	// go func() {
	// 	time.Sleep(time.Millisecond * 500)
	// 	fifo.Out()
	// }()
	// if e := fifo.In([]byte("345")); e != nil {
	// 	t.Fatal(e)
	// }
	// time.Sleep(time.Second * 10)
	// fifo.Clear()
}

func BenchmarkFIFO(b *testing.B) {
	fifo := NewFIFO[byte](5)
	buf := []byte("12")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if e := fifo.In(buf); e != nil {
			b.FailNow()
		}
		if fifo.Num() > 1 {
			if tmp, e := fifo.Out(); e != nil || !bytes.Equal(tmp, buf) {
				b.FailNow()
			}
		}
	}
}
