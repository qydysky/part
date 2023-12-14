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
