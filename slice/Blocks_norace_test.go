//go:build !race

package part

import (
	"runtime/debug"
	"testing"
)

func TestMain3(t *testing.T) {
	defer debug.SetGCPercent(debug.SetGCPercent(-1))
	buf := NewPoolBlocks[byte]()
	if tmpbuf, putBack, e := buf.Get(); e == nil {
		tmpbuf = append(tmpbuf[:0], []byte("123")...)
		// do something with tmpbuf
		putBack()
	} else {
		t.Fail()
	}
	if tmpbuf, putBack, e := buf.Get(); e == nil {
		if cap(tmpbuf) != 8 {
			t.Fatal()
		}
		tmpbuf = append(tmpbuf[:0], []byte("123")...)
		// do something with tmpbuf
		putBack()
	} else {
		t.Fail()
	}
}
