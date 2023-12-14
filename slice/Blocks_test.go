package part

import "testing"

func TestMain(t *testing.T) {
	buf := NewBlocks[byte](1024, 10)
	if tmpbuf, putBack, e := buf.Get(); e == nil {
		clear(tmpbuf)
		// do something with tmpbuf
		putBack()
	}
}
