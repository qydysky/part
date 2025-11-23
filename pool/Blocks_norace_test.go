//go:build !race

package part

import (
	"runtime"
	"testing"
)

func TestMain3(t *testing.T) {
	runtime.GOMAXPROCS(1)
	buf := NewPoolBlocks[byte]()

	tmpbuf := *(buf.Get())
	tmpbuf = append(tmpbuf[:0], []byte("123")...)
	buf.Put(&tmpbuf)

	{
		tmpbuf := *(buf.Get())
		if cap(tmpbuf) != 8 {
			t.Fatal(cap(tmpbuf))
		}
		tmpbuf = append(tmpbuf[:0], []byte("123")...)
		buf.Put(&tmpbuf)
	}
}
