//go:build !race

package part

import (
	"testing"
)

func TestMain3(t *testing.T) {
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
