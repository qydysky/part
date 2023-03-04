package part

import (
	"bytes"
	"testing"
	"unsafe"
)

func TestXxx(t *testing.T) {
	var (
		b = New[byte](5)
		c = New[byte]()
	)
	if !b.IsEmpty() {
		t.Fatal()
	}
	b.Append([]byte("abc"))
	if b.IsEmpty() || b.Size() != 3 {
		t.Fatal()
	}
	if _, e := b.HadModified(c.GetModified()); e == nil {
		t.Fatal()
	}
	bc := b.GetCopyBuf()
	bt := b.GetModified()
	b.RemoveFront(1)
	if b.IsEmpty() || b.Size() != 2 || !bytes.Equal(bc, []byte("abc")) ||
		!bytes.Equal(b.GetPureBuf(), []byte("bc")) || bt == b.GetModified() {
		t.Fatal()
	}
	b.RemoveBack(1)
	if b.IsEmpty() || b.Size() != 1 || !bytes.Equal(bc, []byte("abc")) || !bytes.Equal(b.GetPureBuf(), []byte("b")) {
		t.Fatal()
	}
	b.Reset()
	if !b.IsEmpty() || b.Size() != 0 || !bytes.Equal(bc, []byte("abc")) || !bytes.Equal(b.GetPureBuf(), []byte("")) {
		t.Fatal()
	}
	if e := b.RemoveFront(1); e == nil || e.Error() != "尝试移除的数值大于长度" {
		t.Fatal()
	}
	if e := b.Append([]byte("abcdef")); e == nil || e.Error() != "超出设定maxsize" {
		t.Fatal()
	}
	b.Clear()
	if !b.IsEmpty() || b.Size() != 0 {
		t.Fatal()
	}
	b.Append([]byte("abc"))
	if b.IsEmpty() || b.Size() != 3 {
		t.Fatal()
	}
	c.Append([]byte("abc"))
	if c.IsEmpty() || c.Size() != 3 {
		t.Fatal()
	}
}

func TestXxx2(t *testing.T) {
	var c = New[byte]()
	c.Append([]byte("12345"))
	c.Append([]byte("67890"))
	first := c.GetCopyBuf()
	c.Reset()
	c.Append([]byte("abc"))
	c.Append([]byte("defg"))
	second := c.GetCopyBuf()
	c.Reset()
	c.Append([]byte("akjsdhfaksdjhf"))
	c.Append([]byte("9834719203857"))
	third := c.GetCopyBuf()
	c.Reset()
	if !bytes.Equal(first, []byte("1234567890")) {
		t.Fatal()
	}
	if !bytes.Equal(second, []byte("abcdefg")) {
		t.Fatal()
	}
	if !bytes.Equal(third, []byte("akjsdhfaksdjhf9834719203857")) {
		t.Fatal()
	}
}

func TestXxx3(t *testing.T) {
	var c = New[byte]()
	var b []byte
	var bp = unsafe.Pointer(&b)
	c.Append([]byte("12345"))
	c.Append([]byte("67890"))
	c.AppendBufCopy(&b)
	c.Reset()
	if !bytes.Equal(b, []byte("1234567890")) {
		t.Fatal(string(b))
	}
	b = []byte{}
	c.Append([]byte("abc"))
	c.Append([]byte("defg"))
	c.AppendBufCopy(&b)
	c.Reset()
	if !bytes.Equal(b, []byte("abcdefg")) || unsafe.Pointer(&b) != bp {
		t.Fatal()
	}
	b = b[:0]
	c.Append([]byte("akjsdhfaksdjhf"))
	c.Append([]byte("9834719203857"))
	c.AppendBufCopy(&b)
	c.Reset()
	if !bytes.Equal(b, []byte("akjsdhfaksdjhf9834719203857")) || unsafe.Pointer(&b) != bp {
		t.Fatal()
	}
}
